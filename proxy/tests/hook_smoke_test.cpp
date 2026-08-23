// Hook lifecycle smoke test against upstream MinHook.
//
// Windows-only: exercises create/enable/disable/remove on local dummy
// functions so a broken MinHook integration fails here instead of inside
// Engine DJ. Extended by the atomic-install task with failure injection.
#include <windows.h>
#include <MinHook.h>

#include <cstdio>

namespace {

int g_failures = 0;

#define CHECK(cond)                                                  \
  do {                                                               \
    if (!(cond)) {                                                   \
      std::printf("FAIL %s:%d: %s\n", __FILE__, __LINE__, #cond);   \
      ++g_failures;                                                  \
    }                                                                \
  } while (0)

}  // namespace

__declspec(noinline) static int __cdecl dummy_a(int v) { return v + 1; }
static int(__cdecl* p_orig_a)(int) = nullptr;
static int __cdecl hk_a(int v) { return p_orig_a(v) + 10; }

__declspec(noinline) static int __cdecl dummy_b(int v) { return v + 2; }
static int(__cdecl* p_orig_b)(int) = nullptr;
static int __cdecl hk_b(int v) { return p_orig_b(v) + 20; }

int main() {
  CHECK(MH_Initialize() == MH_OK);

  CHECK(MH_CreateHook(reinterpret_cast<void*>(&dummy_a),
                      reinterpret_cast<void*>(&hk_a),
                      reinterpret_cast<void**>(&p_orig_a)) == MH_OK);
  CHECK(MH_CreateHook(reinterpret_cast<void*>(&dummy_b),
                      reinterpret_cast<void*>(&hk_b),
                      reinterpret_cast<void**>(&p_orig_b)) == MH_OK);

  // Neither hook is enabled yet: targets must behave normally.
  CHECK(dummy_a(1) == 2);
  CHECK(dummy_b(1) == 3);

  CHECK(MH_EnableHook(reinterpret_cast<void*>(&dummy_a)) == MH_OK);
  CHECK(MH_EnableHook(reinterpret_cast<void*>(&dummy_b)) == MH_OK);

  CHECK(dummy_a(1) == 12);
  CHECK(dummy_b(1) == 23);
  CHECK(p_orig_a != nullptr && p_orig_b != nullptr);
  CHECK(p_orig_a(5) == 6);  // trampolines callable while active

  // Disabling one hook must not disturb the other.
  CHECK(MH_DisableHook(reinterpret_cast<void*>(&dummy_a)) == MH_OK);
  CHECK(dummy_a(1) == 2);
  CHECK(dummy_b(1) == 23);

  CHECK(MH_RemoveHook(reinterpret_cast<void*>(&dummy_a)) == MH_OK);
  CHECK(MH_RemoveHook(reinterpret_cast<void*>(&dummy_b)) == MH_OK);
  CHECK(MH_Uninitialize() == MH_OK);

  if (g_failures != 0) {
    std::printf("%d failure(s)\n", g_failures);
    return 1;
  }
  std::printf("hook smoke test passed\n");
  return 0;
}
