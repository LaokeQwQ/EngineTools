//
// All-or-nothing hook installation.
//
// Pure, portable, header-only. The installer is generic over its backend
// (real WinAPI + MinHook in main.cpp, fault-injecting fakes in tests), so
// the transaction and rollback logic is testable on any platform.
//
// Phases, each fully rolled back when any later step fails:
//   1. resolve every required export up front;
//   2. create every hook DISABLED (a failure rolls back earlier creates);
//   3. verify declared dependencies' originals are live before enabling a
//      dependent hook, then enable every hook; any enable failure disables
//      and removes EVERY created hook and clears every original pointer,
//      so the process can never be observed half-hooked and no caller can
//      invoke a null trampoline.
//
// The `report` callback runs at most once per attempt, with the single
// terminal error.
//
#pragma once

#include <cstddef>

namespace etproxy {

struct HookSpec {
  const char* label;            // human-readable name for diagnostics
  const char* mangled_export;   // export resolved via the resolve callback
  void* detour;                 // replacement function
  void** original_out;          // receives the trampoline; cleared on rollback
  int requires_original_of;     // index of an entry whose original must be
                                // live before this one enables, or -1
};

enum class TxError {
  kNone = 0,
  kResolveFailed,
  kCreateFailed,
  kEnableFailed,
  kDependencyUnmet,
};

struct InstallResult {
  bool ok = false;
  TxError error = TxError::kNone;
  int failed_index = -1;
};

template <typename ModuleHandle, typename ResolveFn, typename CreateFn,
          typename EnableFn, typename RollbackFn, typename ReportFn>
InstallResult install_hooks_transactional(ModuleHandle module,
                                          const HookSpec* specs,
                                          std::size_t count, ResolveFn resolve,
                                          CreateFn create, EnableFn enable,
                                          RollbackFn rollback_one,
                                          ReportFn report) {
  InstallResult result;

  constexpr std::size_t kMaxHooks = 64;
  if (specs == nullptr || count == 0 || count > kMaxHooks) {
    result.error = TxError::kCreateFailed;
    result.failed_index = -1;
    report(result);
    return result;
  }

  void* targets[kMaxHooks] = {};

  // rollback_one(target, original_out) must disable and remove the hook if
  // it exists and set *original_out to nullptr. It must tolerate being
  // called on a target that was never created or never enabled.
  auto rollback_range = [&](std::size_t begin, std::size_t end) {
    for (std::size_t i = begin; i < end && i < count; ++i) {
      rollback_one(targets[i], specs[i].original_out);
    }
  };
  // Phase 1: every export resolves before anything is mutated.
  for (std::size_t i = 0; i < count; ++i) {
    targets[i] = resolve(module, specs[i].mangled_export);
    if (targets[i] == nullptr) {
      result.error = TxError::kResolveFailed;
      result.failed_index = static_cast<int>(i);
      report(result);
      return result;
    }
  }

  // Phase 2: create disabled, stopping at the first failure.
  std::size_t made = 0;
  while (made < count &&
         create(targets[made], specs[made].detour, specs[made].original_out)) {
    ++made;
  }
  if (made != count) {
    rollback_range(0, made);
    result.error = TxError::kCreateFailed;
    result.failed_index = static_cast<int>(made);
    report(result);
    return result;
  }

  // Phase 3: dependency gates, then enable. One failure rolls back all.
  for (std::size_t i = 0; i < count; ++i) {
    const int dep = specs[i].requires_original_of;
    if (dep >= 0) {
      const std::size_t di = static_cast<std::size_t>(dep);
      if (di >= count || di >= i || specs[di].original_out == nullptr ||
          *specs[di].original_out == nullptr) {
        result.error = TxError::kDependencyUnmet;
        result.failed_index = static_cast<int>(i);
        rollback_range(0, count);
        report(result);
        return result;
      }
    }
    if (!enable(targets[i])) {
      result.error = TxError::kEnableFailed;
      result.failed_index = static_cast<int>(i);
      rollback_range(0, count);
      report(result);
      return result;
    }
  }

  result.ok = true;
  return result;
}

}  // namespace etproxy
