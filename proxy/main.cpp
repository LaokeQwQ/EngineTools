/**
 * Engine DJ Translation Proxy v3.
 *
 * version.dll proxy that translates Engine DJ UI strings at the QString
 * creation boundary.
 *
 * Hooking uses upstream MinHook (pinned v1.3.4, see CMakeLists.txt and
 * THIRD_PARTY_NOTICES.md). String matching is length-aware with no
 * NUL-termination assumption (see lookup.h). Log output always goes through
 * etproxy::quote_into so untrusted buffers are escaped and bounded before
 * they ever reach printf-style formatting.
 */
#include <windows.h>

#include <MinHook.h>

#include <cstdarg>
#include <cstdint>
#include <cstdio>
#include <cstring>

#include "hook_install.h"
#include "lookup.h"

// ============================================================
// Debug logging
// ============================================================
static FILE* g_log = nullptr;
static CRITICAL_SECTION g_log_lock;
static int g_hook_calls = 0;
static int g_trans_count = 0;

static void log_init() {
  InitializeCriticalSection(&g_log_lock);
  char path[MAX_PATH];
  const DWORD n = GetTempPathA(MAX_PATH, path);
  if (n > 0 && n + 20 < MAX_PATH) {
    memcpy(path + n, "engine_dj_proxy.log", sizeof("engine_dj_proxy.log"));
    g_log = fopen(path, "w");
  }
}

// fmt must be a trusted literal; all untrusted data reaches this function
// only through quote_into()-sanitized buffers passed as %s arguments.
static void log_write(const char* fmt, ...) {
  if (g_log == nullptr) return;
  EnterCriticalSection(&g_log_lock);
  va_list args;
  va_start(args, fmt);
  vfprintf(g_log, fmt, args);
  va_end(args);
  fflush(g_log);
  LeaveCriticalSection(&g_log_lock);
}

// ============================================================
// Hook function type: all return QString by value
// x64: void func(QString* hidden_ret, ...)
// ============================================================
typedef void (*StrFn)(void* ret, ...);

static void log_match(const char* api, const char* src, int64_t src_len,
                      const char* replacement, int64_t replacement_len) {
  char quoted_src[256];
  char quoted_cn[256];
  etproxy::quote_into(quoted_src, sizeof(quoted_src), src, src_len);
  etproxy::quote_into(quoted_cn, sizeof(quoted_cn), replacement,
                      replacement_len);
  log_write("[%s MATCH] %s -> %s\n", api, quoted_src, quoted_cn);
}

static void log_candidate(const char* api, const char* src, int64_t src_len,
                          int* budget) {
  if (*budget >= 100) return;
  if (!etproxy::looks_like_ui(src, static_cast<std::size_t>(src_len))) return;
  ++*budget;
  char quoted[256];
  etproxy::quote_into(quoted, sizeof(quoted), src, src_len);
  log_write("[%s UI? #%d] %s\n", api, *budget, quoted);
}

// ============================================================
// Hook implementations (one per function)
// ============================================================

// 1) QString::fromUtf8(const char*, qsizetype)
//    ?fromUtf8@QString@@SA?AV1@PEBD_J@Z
static StrFn g_pFromUtf8 = nullptr;
static void hkFromUtf8(void* r, const char* s, int64_t n) {
  g_hook_calls++;
  if (s != nullptr && n > 0) {
    const auto cn = etproxy::lookup_bounded(s, n);
    if (cn.has_value()) {
      g_trans_count++;
      if (g_trans_count <= 5) {
        log_match("fromUtf8", s, n, cn->data(),
                  static_cast<int64_t>(cn->size()));
      }
      g_pFromUtf8(r, cn->data(), static_cast<int64_t>(cn->size()));
      return;
    }
    static int logged = 0;
    log_candidate("fromUtf8", s, n, &logged);
  }
  g_pFromUtf8(r, s, n);
}

// 2) QString::fromLatin1(const char*, qsizetype)
//    ?fromLatin1@QString@@SA?AV1@PEBD_J@Z
//
// Note: on a translation hit we deliberately route the UTF-8 replacement
// through the fromUtf8 original, so fromLatin1 depends on g_pFromUtf8 being
// resolved first (enforced by the installer).
static StrFn g_pFromLatin1 = nullptr;
static void hkFromLatin1(void* r, const char* s, int64_t n) {
  if (s != nullptr && n > 0) {
    const auto cn = etproxy::lookup_bounded(s, n);
    if (cn.has_value() && g_pFromUtf8 != nullptr) {
      log_match("fromLatin1", s, n, cn->data(),
                static_cast<int64_t>(cn->size()));
      g_pFromUtf8(r, cn->data(), static_cast<int64_t>(cn->size()));
      return;
    }
  }
  g_pFromLatin1(r, s, n);
}

// 3) QString::fromUtf8(QByteArrayView)
//    ?fromUtf8@QString@@SA?AV1@VQByteArrayView@@@Z
//    QByteArrayView = { const char* data; qsizetype size; } (16 bytes on
//    x64). On x64, small trivially-copyable structs are passed split across
//    two registers: RDX = data, R8 = size.
static StrFn g_pFromUtf8BAV = nullptr;
static void hkFromUtf8BAV(void* r, const char* s, int64_t n) {
  g_hook_calls++;
  if (s != nullptr && n > 0) {
    const auto cn = etproxy::lookup_bounded(s, n);
    if (cn.has_value()) {
      g_trans_count++;
      if (g_trans_count <= 5) {
        log_match("fromUtf8BAV", s, n, cn->data(),
                  static_cast<int64_t>(cn->size()));
      }
      g_pFromUtf8BAV(r, cn->data(), static_cast<int64_t>(cn->size()));
      return;
    }
    static int logged = 0;
    log_candidate("fromUtf8BAV", s, n, &logged);
  }
  g_pFromUtf8BAV(r, s, n);
}

// Wide-string logging helper: converts up to 100 UTF-16 code units into a
// bounded buffer before quoting, so nothing unbounded reaches the logger.
static void log_wide(const char* api, const uint16_t* s, int64_t n,
                     int calls) {
  char utf8[256] = {};
  std::size_t j = 0;
  const int units = (n < 100) ? static_cast<int>(n) : 100;
  for (int i = 0; i < units && j + 4 < sizeof(utf8); ++i) {
    uint16_t c = s[i];
    if (c < 0x80) {
      utf8[j++] = static_cast<char>(c);
    } else if (c < 0x800) {
      utf8[j++] = static_cast<char>(0xC0 | (c >> 6));
      utf8[j++] = static_cast<char>(0x80 | (c & 0x3F));
    } else {
      utf8[j++] = static_cast<char>(0xE0 | (c >> 12));
      utf8[j++] = static_cast<char>(0x80 | ((c >> 6) & 0x3F));
      utf8[j++] = static_cast<char>(0x80 | (c & 0x3F));
    }
  }
  char quoted[512];
  etproxy::quote_into(quoted, sizeof(quoted), utf8,
                      static_cast<int64_t>(j));
  log_write("[%s #%d] n=%lld first=%s\n", api, calls,
            static_cast<long long>(n), quoted);
}

// 4) QString::fromUtf16(const char16_t*, qsizetype)
static StrFn g_pFromUtf16 = nullptr;
static bool g_wide_logged = false;
static void hkFromUtf16(void* r, const uint16_t* s, int64_t n) {
  g_hook_calls++;
  if (!g_wide_logged && s != nullptr && n > 0 && n < 500) {
    g_wide_logged = true;
    log_wide("fromUtf16", s, n, g_hook_calls);
  }
  g_pFromUtf16(r, s, n);
}

// 5) QString(const QChar*, qsizetype) ctor
//    ??0QString@@QEAA@PEB_S_J@Z
static StrFn g_pQStringCtor = nullptr;
static void hkQStringCtor(void* r, const uint16_t* s, int64_t n) {
  g_hook_calls++;
  if (!g_wide_logged && s != nullptr && n > 0 && n < 500) {
    g_wide_logged = true;
    log_wide("QStringCtor", s, n, g_hook_calls);
  }
  g_pQStringCtor(r, s, n);
}

// ============================================================
// Hook installation: one all-or-nothing transaction (hook_install.h).
// ============================================================
static bool mh_create(void* target, void* detour, void** original_out) {
  if (MH_CreateHook(target, detour, original_out) != MH_OK) return false;
  return *original_out != nullptr;
}

static bool mh_enable(void* target) {
  return MH_EnableHook(target) == MH_OK;
}

// Must tolerate targets that were never created or never enabled; clears
// the trampoline pointer so no stale caller can use it.
static void mh_rollback_one(void* target, void** original_out) {
  MH_DisableHook(target);
  MH_RemoveHook(target);
  *original_out = nullptr;
}

static bool g_hooks_ready = false;

static const etproxy::HookSpec kHookSpecs[] = {
    // 1) fromUtf8(const char*, qsizetype)
    {"fromUtf8(ptr,sz)", "?fromUtf8@QString@@SA?AV1@PEBD_J@Z",
     reinterpret_cast<void*>(&hkFromUtf8),
     reinterpret_cast<void**>(&g_pFromUtf8), -1},
    // 2) fromLatin1(const char*, qsizetype) -- routes hits through the
    // fromUtf8 trampoline, so its dependency gates on entry 0.
    {"fromLatin1(ptr,sz)", "?fromLatin1@QString@@SA?AV1@PEBD_J@Z",
     reinterpret_cast<void*>(&hkFromLatin1),
     reinterpret_cast<void**>(&g_pFromLatin1), 0},
    // 3) fromUtf8(QByteArrayView)
    {"fromUtf8(BAV)", "?fromUtf8@QString@@SA?AV1@VQByteArrayView@@@Z",
     reinterpret_cast<void*>(&hkFromUtf8BAV),
     reinterpret_cast<void**>(&g_pFromUtf8BAV), -1},
    // 4) fromUtf16(const char16_t*, qsizetype)
    {"fromUtf16", "?fromUtf16@QString@@SA?AV1@PEB_S_J@Z",
     reinterpret_cast<void*>(&hkFromUtf16),
     reinterpret_cast<void**>(&g_pFromUtf16), -1},
    // 5) QString(const QChar*, qsizetype) ctor
    {"QString(QChar*,sz)", "??0QString@@QEAA@PEB_S_J@Z",
     reinterpret_cast<void*>(&hkQStringCtor),
     reinterpret_cast<void**>(&g_pQStringCtor), -1},
};

static bool install_hooks() {
  HMODULE qt6 = GetModuleHandleW(L"Qt6Core.dll");
  log_write("Qt6Core: %p
", reinterpret_cast<void*>(qt6));
  if (qt6 == nullptr) {
    log_write("FAIL: no Qt6Core
");
    return false;
  }
  if (MH_Initialize() != MH_OK) {
    log_write("FAIL: MH_Initialize
");
    return false;
  }

  const etproxy::InstallResult r = etproxy::install_hooks_transactional(
      qt6, kHookSpecs, sizeof(kHookSpecs) / sizeof(kHookSpecs[0]),
      [](HMODULE mod, const char* mangled) {
        return reinterpret_cast<void*>(GetProcAddress(mod, mangled));
      },
      &mh_create, &mh_enable, &mh_rollback_one,
      [](const etproxy::InstallResult& res) {
        log_write("FAIL: install aborted (%d at index %d)
",
                  static_cast<int>(res.error), res.failed_index);
      });

  // Publish readiness only after every enable succeeded.
  g_hooks_ready = r.ok;
  log_write("Hooks: %s
", r.ok ? "installed" : "aborted");
  return r.ok;
}

static DWORD WINAPI InitThread(LPVOID) {
  log_write("InitThread: waiting for Qt6Core...\n");
  for (int i = 0; i < 300; ++i) {
    if (GetModuleHandleW(L"Qt6Core.dll") != nullptr) break;
    Sleep(10);
  }
  install_hooks();
  return 0;
}

extern "C" BOOL WINAPI DllMain(HINSTANCE hDll, DWORD reason, LPVOID) {
  if (reason == DLL_PROCESS_ATTACH) {
    DisableThreadLibraryCalls(hDll);
    log_init();
    log_write("=== version.dll v3 (PID=%d) ===\n", GetCurrentProcessId());
    CreateThread(nullptr, 0, InitThread, nullptr, 0, nullptr);
  }
  if (reason == DLL_PROCESS_DETACH) {
    if (g_hooks_ready) {
      MH_DisableHook(MH_ALL_HOOKS);
      MH_Uninitialize();
      g_hooks_ready = false;
    }
    log_write("=== unloaded (%d calls, %d translated) ===\n", g_hook_calls,
              g_trans_count);
    if (g_log != nullptr) fclose(g_log);
    DeleteCriticalSection(&g_log_lock);
  }
  return TRUE;
}
