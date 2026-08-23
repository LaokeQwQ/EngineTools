// Atomic hook installation: lifecycle + fault-injection tests.
//
// The transaction/rollback logic runs against a fake backend everywhere,
// so partial failures are exercised even off-Windows. On Windows the same
// binary additionally drives real upstream MinHook over local dummies.
#include "../hook_install.h"

#include <cstdio>
#include <cstring>

#ifdef _WIN32
#include <windows.h>
#include <MinHook.h>
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

// ------------------------- fake backend -------------------------------
enum SpecId { kAlpha = 0, kBeta = 1, kGamma = 2, kSpecCount = 3 };

constexpr const char* kSpecNames[kSpecCount] = {"alpha", "beta", "gamma"};

struct FakeConfig {
  const char* fail_resolve = nullptr;
  const char* fail_create = nullptr;
  const char* fail_enable = nullptr;
  bool drop_dependency_original = false;  // alpha "creates" but leaves its
                                          // trampoline null
};

struct FakeState {
  int resolved[kSpecCount] = {};
  int created[kSpecCount] = {};
  int enabled[kSpecCount] = {};
  int rolled_back[kSpecCount] = {};
  int reports = 0;
};

std::size_t index_of(const void* target) {
  return static_cast<std::size_t>(
      reinterpret_cast<const char*>(target) -
      reinterpret_cast<const char*>(static_cast<std::size_t>(0x1000)));
}

void* target_for(std::size_t i) {
  return reinterpret_cast<void*>(0x1000 + i);
}

etproxy::InstallResult run_install(const FakeConfig& cfg, FakeState& st,
                                   etproxy::HookSpec* specs) {
  return etproxy::install_hooks_transactional(
      &cfg, specs, kSpecCount,
      [&](const FakeConfig*, const char*) -> void* {
        // Phase 1 resolves strictly in spec order; derive the index from
        // how many have already been resolved.
        const std::size_t i = static_cast<std::size_t>(st.resolved[0] +
                                                       st.resolved[1] +
                                                       st.resolved[2]);
        st.resolved[i] = 1;
        if (cfg.fail_resolve != nullptr &&
            strcmp(kSpecNames[i], cfg.fail_resolve) == 0)
          return nullptr;
        return target_for(i);
      },
      [&](void* target, void*, void** orig) -> bool {
        const std::size_t i = index_of(target);
        if (cfg.fail_create != nullptr &&
            strcmp(kSpecNames[i], cfg.fail_create) == 0)
          return false;
        st.created[i] = 1;
        if (!cfg.drop_dependency_original || i != kAlpha) *orig = target;
        return true;
      },
      [&](void* target) -> bool {
        const std::size_t i = index_of(target);
        if (cfg.fail_enable != nullptr &&
            strcmp(kSpecNames[i], cfg.fail_enable) == 0)
          return false;
        st.enabled[i] = 1;
        return true;
      },
      [&](void* target, void** orig) {
        const std::size_t i = index_of(target);
        st.rolled_back[i] = 1;
        if (orig != nullptr) *orig = nullptr;
      },
      [&](const etproxy::InstallResult&) { ++st.reports; });
}

void build_specs(void** slots, etproxy::HookSpec* specs) {
  specs[0] = {"alpha", "EXP_A", nullptr, &slots[0], -1};
  specs[1] = {"beta", "EXP_B", nullptr, &slots[1], kAlpha};
  specs[2] = {"gamma", "EXP_C", nullptr, &slots[2], -1};
}

// ------------------------- portable cases -----------------------------
void test_happy_path() {
  FakeConfig cfg;
  FakeState st;
  void* slots[kSpecCount] = {};  // backing storage for spec original_out
  etproxy::HookSpec specs[kSpecCount];
  build_specs(slots, specs);

  const auto r = run_install(cfg, st, specs);
  CHECK(r.ok);
  CHECK(r.error == etproxy::TxError::kNone);
  CHECK(st.reports == 0);
  for (int i = 0; i < kSpecCount; ++i) {
    CHECK(st.enabled[i] == 1);
    CHECK(st.rolled_back[i] == 0);
    CHECK(slots[i] != nullptr);  // every trampoline live
  }
}

void test_resolve_failure_aborts_before_mutation() {
  FakeConfig cfg;
  cfg.fail_resolve = "alpha";
  FakeState st;
  void* slots[kSpecCount] = {};  // backing storage for spec original_out
  etproxy::HookSpec specs[kSpecCount];
  build_specs(slots, specs);

  const auto r = run_install(cfg, st, specs);
  CHECK(!r.ok);
  CHECK(r.error == etproxy::TxError::kResolveFailed);
  CHECK(r.failed_index == 0);
  CHECK(st.reports == 1);
  for (int i = 0; i < kSpecCount; ++i) {
    CHECK(st.created[i] == 0);
    CHECK(st.enabled[i] == 0);
    CHECK(st.rolled_back[i] == 0);
  }
}

void test_create_failure_rolls_back_earlier_creates() {
  FakeConfig cfg;
  cfg.fail_create = "beta";
  FakeState st;
  void* slots[kSpecCount] = {};  // backing storage for spec original_out
  etproxy::HookSpec specs[kSpecCount];
  build_specs(slots, specs);

  const auto r = run_install(cfg, st, specs);
  CHECK(!r.ok);
  CHECK(r.error == etproxy::TxError::kCreateFailed);
  CHECK(r.failed_index == 1);
  CHECK(st.reports == 1);
  CHECK(st.created[0] == 1 && st.created[1] == 0 && st.created[2] == 0);
  CHECK(st.rolled_back[0] == 1);          // the created one is undone
  CHECK(st.rolled_back[1] == 0);
  CHECK(slots[0] == nullptr);             // no dangling trampoline
  CHECK(st.enabled[0] == 0);              // nothing ever enabled
}

void test_enable_failure_disables_and_removes_everything() {
  FakeConfig cfg;
  cfg.fail_enable = "gamma";
  FakeState st;
  void* slots[kSpecCount] = {};  // backing storage for spec original_out
  etproxy::HookSpec specs[kSpecCount];
  build_specs(slots, specs);

  const auto r = run_install(cfg, st, specs);
  CHECK(!r.ok);
  CHECK(r.error == etproxy::TxError::kEnableFailed);
  CHECK(r.failed_index == 2);
  CHECK(st.reports == 1);
  CHECK(st.enabled[0] == 1 && st.enabled[1] == 1 && st.enabled[2] == 0);
  for (int i = 0; i < kSpecCount; ++i) {
    CHECK(st.rolled_back[i] == 1);        // every hook removed...
    CHECK(slots[i] == nullptr);           // ...and no hot trampoline left
  }
}

void test_dependency_gate_blocks_dependent_hook() {
  FakeConfig cfg;
  cfg.drop_dependency_original = true;  // beta needs alpha's trampoline
  FakeState st;
  void* slots[kSpecCount] = {};  // backing storage for spec original_out
  etproxy::HookSpec specs[kSpecCount];
  build_specs(slots, specs);

  const auto r = run_install(cfg, st, specs);
  CHECK(!r.ok);
  CHECK(r.error == etproxy::TxError::kDependencyUnmet);
  CHECK(r.failed_index == kBeta);
  CHECK(st.reports == 1);
  CHECK(st.enabled[kBeta] == 0);   // dependent never enabled...
  CHECK(st.enabled[kGamma] == 0);  // ...nor anything after it
  for (int i = 0; i < kSpecCount; ++i) {
    CHECK(st.rolled_back[i] == 1);
    CHECK(slots[i] == nullptr);
  }
}

#ifdef _WIN32

// ------------------- real MinHook over local dummies -------------------
__declspec(noinline) static int __cdecl dummy_a(int v) { return v + 1; }
static int(__cdecl* p_orig_a)(int) = nullptr;
static int __cdecl hk_a(int v) { return p_orig_a(v) + 10; }

__declspec(noinline) static int __cdecl dummy_b(int v) { return v + 2; }
static int(__cdecl* p_orig_b)(int) = nullptr;
static int __cdecl hk_b(int v) { return p_orig_b(v) + 20; }

void test_real_minhook_lifecycle() {
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
}

#endif  // _WIN32

}  // namespace

int main() {
  test_happy_path();
  test_resolve_failure_aborts_before_mutation();
  test_create_failure_rolls_back_earlier_creates();
  test_enable_failure_disables_and_removes_everything();
  test_dependency_gate_blocks_dependent_hook();
#ifdef _WIN32
  test_real_minhook_lifecycle();
#endif

  if (g_failures != 0) {
    std::printf("%d failure(s)\n", g_failures);
    return 1;
  }
  std::printf("hook install tests passed\n");
  return 0;
}
