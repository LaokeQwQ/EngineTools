// Unit tests for etproxy::lookup and friends.
//
// Portable by design: the same source runs under ctest on Windows and
// directly on POSIX hosts. The guard-page cases place test buffers so they
// END at an inaccessible page, which makes any read past the declared
// length fault instead of silently succeeding.
#include "../lookup.h"

#include <cstdio>
#include <cstring>

#ifdef _WIN32
#include <windows.h>
#else
#include <sys/mman.h>
#include <unistd.h>
#endif

namespace {

int g_failures = 0;

#define CHECK(cond)                                                  \
  do {                                                               \
    if (!(cond)) {                                                   \
      std::printf("FAIL %s:%d: %s\n", __FILE__, __LINE__, #cond);   \
      ++g_failures;                                                  \
    }                                                                \
  } while (0)

std::size_t page_size() {
#ifdef _WIN32
  SYSTEM_INFO si;
  GetSystemInfo(&si);
  return static_cast<std::size_t>(si.dwPageSize);
#else
  return static_cast<std::size_t>(sysconf(_SC_PAGESIZE));
#endif
}

// Two consecutive pages: first readable/writable, second inaccessible.
// `occupy` bytes at the END of the first page are handed to the caller,
// so data + occupy lands exactly on the guard page.
struct GuardedBuffer {
  char* data = nullptr;
  std::size_t occupy = 0;
  void* base = nullptr;
  std::size_t map_size = 0;
};

GuardedBuffer make_guarded(std::size_t occupy) {
  const std::size_t ps = page_size();
  GuardedBuffer gb;
  gb.occupy = occupy;
  gb.map_size = 2 * ps;
#ifdef _WIN32
  gb.base = VirtualAlloc(nullptr, gb.map_size, MEM_RESERVE | MEM_COMMIT,
                         PAGE_READWRITE);
  if (gb.base != nullptr) {
    DWORD old_protect = 0;
    VirtualProtect(static_cast<char*>(gb.base) + ps, ps, PAGE_NOACCESS,
                   &old_protect);
  }
#else
  gb.base = mmap(nullptr, gb.map_size, PROT_READ | PROT_WRITE,
                 MAP_PRIVATE | MAP_ANONYMOUS, -1, 0);
  if (gb.base != MAP_FAILED) {
    mprotect(static_cast<char*>(gb.base) + ps, ps, PROT_NONE);
  } else {
    gb.base = nullptr;
  }
#endif
  if (gb.base == nullptr) {
    // Fallback keeps tests runnable on hosts that refuse the mapping; the
    // guard guarantee is simply not exercised there.
    gb.data = new char[occupy];
  } else {
    gb.data = static_cast<char*>(gb.base) + ps - occupy;
  }
  return gb;
}

void free_guarded(GuardedBuffer* gb) {
  if (gb->base != nullptr) {
#ifdef _WIN32
    VirtualFree(gb->base, 0, MEM_RELEASE);
#else
    munmap(gb->base, gb->map_size);
#endif
  } else if (gb->data != nullptr) {
    delete[] gb->data;
  }
  gb->data = nullptr;
}

void test_exact_match_without_terminator() {
  GuardedBuffer gb = make_guarded(11);
  memcpy(gb.data, "Preferences", 11);  // deliberately no NUL anywhere
  const auto hit = etproxy::lookup(gb.data, 11);
  CHECK(hit.has_value());
  if (hit.has_value()) CHECK(hit.value() == "\xe5\x81\x8f\xe5\xa5\xbd\xe8\xae\xbe\xe7\xbd\xae");
  free_guarded(&gb);
}

void test_embedded_nul_is_not_a_match() {
  GuardedBuffer gb = make_guarded(11);
  memcpy(gb.data, "Pre\0ferences", 11);
  CHECK(!etproxy::lookup(gb.data, 11).has_value());
  free_guarded(&gb);
}

void test_zero_length_and_null() {
  char scratch[4] = {'O', 'K', 'x', 'y'};
  CHECK(!etproxy::lookup(nullptr, 2).has_value());
  CHECK(!etproxy::lookup(scratch, 0).has_value());
  CHECK(!etproxy::lookup_bounded(nullptr, 4).has_value());
  CHECK(!etproxy::lookup_bounded(scratch, 0).has_value());
}

void test_multibyte_and_spaced_keys() {
  const auto usb = etproxy::lookup("Export to USB", 13);
  CHECK(usb.has_value());
  if (usb.has_value()) CHECK(usb.value() == "\xe5\xaf\xbc\xe5\x87\xba\xe5\x88\xb0 USB");

  const unsigned char utf8[] = {0xE5, 0x81, 0x8F};  // one CJK character
  CHECK(!etproxy::lookup(reinterpret_cast<const char*>(utf8), 3).has_value());

  const auto deck = etproxy::lookup("Deck 3", 6);
  CHECK(deck.has_value());
  if (deck.has_value()) CHECK(deck.value() == "\xe7\xa2\x9f\xe6\x9c\xba 3");
}

void test_length_shorter_than_allocation() {
  char wide[64];
  memset(wide, 0, sizeof(wide));
  memcpy(wide, "Settings", 8);
  CHECK(etproxy::lookup(wide, 8).has_value());
  CHECK(!etproxy::lookup(wide, 3).has_value());  // "Set" is not a key
  if (etproxy::lookup(wide, 8).has_value()) {
    CHECK(etproxy::lookup(wide, 8).value() == "\xe8\xae\xbe\xe7\xbd\xae");
  }
}

void test_guard_page_immediately_after_input() {
  const char* kKeys[] = {"Preferences", "Settings", "OK"};
  for (const char* key : kKeys) {
    const std::size_t len = strlen(key);
    GuardedBuffer gb = make_guarded(len);
    memcpy(gb.data, key, len);
    CHECK(etproxy::lookup(gb.data, len).has_value());
    free_guarded(&gb);
  }
}

void test_bounded_wrapper_negative_size() {
  // Terminator present well before any bound: wrapper must find it.
  char term[16];
  memcpy(term, "History", 8);  // includes the NUL
  const auto hit = etproxy::lookup_bounded(term, -1);
  CHECK(hit.has_value());
  if (hit.has_value()) CHECK(hit.value() == "\xe5\x8e\x86\xe5\x8f\xb2\xe8\xae\xb0\xe5\xbd\x95");

  // No terminator within 4096 bytes and a guard page right behind them:
  // the scan must stop at its documented maximum, never touch the guard.
  GuardedBuffer gb = make_guarded(4096);
  memset(gb.data, 'A', 4096);
  CHECK(!etproxy::lookup_bounded(gb.data, -1).has_value());
  free_guarded(&gb);

  // Oversized exact lengths are rejected before scanning (key cap).
  CHECK(!etproxy::lookup_bounded(term, 10000).has_value());
}

void test_table_invariants() {
  const auto& table = etproxy::hash_table();
  constexpr std::size_t kCount =
      sizeof(etproxy::kTranslations) / sizeof(etproxy::kTranslations[0]);
  CHECK(table.count == kCount);
  CHECK(table.count > 50);
  for (std::size_t i = 0; i < kCount; ++i) {
    CHECK(etproxy::kTranslations[i].en.size() <= etproxy::kMaxKeyLength);
    CHECK(!etproxy::kTranslations[i].cn.empty());
    for (std::size_t j = i + 1; j < kCount; ++j) {
      CHECK(etproxy::kTranslations[i].en != etproxy::kTranslations[j].en);
    }
  }
}

void expect_quote(const char* data, int64_t size, const char* expected,
                  int line) {
  char out[64] = {};
  const std::size_t n = etproxy::quote_into(out, sizeof(out), data, size);
  if (strcmp(out, expected) != 0 || n != strlen(expected) ||
      out[sizeof(out) - 1] != '\0') {
    std::printf("FAIL line %d: got %s want %s\n", line, out, expected);
    ++g_failures;
  }
}

void test_quote_into_escapes_and_bounds() {
  // Control bytes, quotes and backslashes are escaped.
  const char evil[] = {'a', 0x01, '"', '\\', '\n', 'z'};
  expect_quote(evil, sizeof(evil),
               "\"a\\x01\\\"\\\\\\nz\"", __LINE__);

  // An embedded NUL terminates rendering early.
  const char embedded[] = {'A', '\0', 'B'};
  expect_quote(embedded, 3, "\"A\"", __LINE__);

  // Null data renders as an empty pair of quotes.
  expect_quote(nullptr, 5, "\"\"", __LINE__);
  expect_quote("", 0, "\"\"", __LINE__);

  // Tiny capacity still yields a terminated, truncated string.
  char tiny[4];
  const std::size_t n =
      etproxy::quote_into(tiny, sizeof(tiny), "Preferences", 11);
  CHECK(n <= 3);
  CHECK(tiny[n] == '\0');
  CHECK(tiny[0] == '"');
}

}  // namespace

int main() {
  test_exact_match_without_terminator();
  test_embedded_nul_is_not_a_match();
  test_zero_length_and_null();
  test_multibyte_and_spaced_keys();
  test_length_shorter_than_allocation();
  test_guard_page_immediately_after_input();
  test_bounded_wrapper_negative_size();
  test_table_invariants();
  test_quote_into_escapes_and_bounds();

  if (g_failures != 0) {
    std::printf("%d failure(s)\n", g_failures);
    return 1;
  }
  std::printf("all lookup tests passed\n");
  return 0;
}
