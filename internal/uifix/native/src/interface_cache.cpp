#define WIN32_LEAN_AND_MEAN
#include <windows.h>
#include <atomic>
#include <cstdint>
#include <memory>
#include <string>
#include <unordered_map>
#include <intrin.h>
#include <cstring>
#include "cache_protocol.h"
#include "../minhook/src/hde/hde64.h"
#include "../minhook/include/MinHook.h"

// MSVC x64 Qt object layout. Unknown builds must pass an isolated ABI probe.
// Call-site RVAs are supplied through the shared protocol, never guessed from a version label.
// All Qt objects are constructed/copied/destroyed by Qt's own exported functions.
struct Interface { void* d; };
struct String { void* d; const wchar_t* data; intptr_t size; };
struct List { volatile LONG* d; Interface* data; intptr_t size; };
static_assert(sizeof(Interface) == 8 && sizeof(String) == 24 && sizeof(List) == 24);
using DefaultFn = void* (*)(Interface*);
using CopyFn = void* (*)(Interface*, const Interface*);
using DestroyFn = void (*)(Interface*);
using NameFn = void* (*)(const Interface*, String*);
using IndexFn = int (*)(const Interface*);
using ListFn = void* (*)(List*);
using StringDestroyFn = void (*)(String*);
using ArrayFreeFn = void (*)(void*, intptr_t, intptr_t);
static DefaultFn makeDefault;
static CopyFn copyInterface;
static DestroyFn destroyInterface;
static NameFn getName;
static IndexFn getIndex;
static ListFn getAll;
static StringDestroyFn destroyString;
static ArrayFreeFn freeArray;
static SRWLOCK cacheLock = SRWLOCK_INIT;
static HANDLE stopEvent, worker;
static std::atomic<bool> initialized{false};
// lookups, hits, misses, refreshes, errors, lastRefreshMs, interfaceCount, workerTid
static std::atomic<uint64_t> counters[8];

struct OwnedInterface {
    Interface value{};
    explicit OwnedInterface(const Interface& source) { copyInterface(&value, &source); }
    ~OwnedInterface() { destroyInterface(&value); }
};
using Snapshot = std::unordered_map<std::wstring, std::shared_ptr<OwnedInterface>>;
// Process termination may unload Qt before CRT global destructors. StopCache
// releases entries on a controlled stop; process exit lets Windows reclaim them.
static Snapshot& cache = *new Snapshot;
using LookupFn = void* (*)(Interface*, const String*);
static std::atomic<LookupFn> originalLookup{nullptr};
static std::atomic<ULONGLONG> lastSuccess{0};

// GetTickCount64 includes system suspend time, but worker waits do not, so the
// age of a snapshot must use a clock that also stops while the system sleeps.
static uint64_t monotonicMs() {
    ULONGLONG ticks = 0;
    QueryUnbiasedInterruptTime(&ticks);
    return ticks / 10000;
}

static uint64_t snapshotAge() {
    const auto refreshed = lastSuccess.load();
    return refreshed ? monotonicMs() - refreshed : UINT64_MAX;
}
static uintptr_t engineBase;
static uint32_t allowedCallers[16]{};
static size_t allowedCallerCount = 0;
extern "C" __declspec(dllexport) void SetOriginalLookup(void* address) {
    originalLookup = reinterpret_cast<LookupFn>(address);
}
static bool monitoredCaller(uintptr_t address) {
    const auto rva = address - engineBase;
    for (size_t i = 0; i < allowedCallerCount; ++i) if (rva == allowedCallers[i]) return true;
    return false;
}
static CacheDiagnostics* diagnostics;
static SRWLOCK diagnosticsLock = SRWLOCK_INIT;
extern "C" void SetCacheDiagnostics(CacheDiagnostics* value) { diagnostics = value; }
static void copyDiagnosticName(wchar_t* output, const String* name, uint32_t reason) {
    if (reason == 2) { wcscpy_s(output, 240, L"<invalid>"); return; }
    __try {
        if (name && (name->size < 0 || name->size > 4096 || (name->size && !name->data))) { wcscpy_s(output, 240, L"<invalid>"); return; }
        if (!name || !name->size) { wcscpy_s(output, 240, L"<empty>"); return; }
        const size_t length = name->size > 239 ? 239 : static_cast<size_t>(name->size);
        memcpy(output, name->data, length * sizeof(wchar_t));
        output[length] = 0;
    } __except(EXCEPTION_EXECUTE_HANDLER) { wcscpy_s(output, 240, L"<unreadable>"); }
}
static void recordDiagnostic(const String* name, uint32_t reason, uint32_t caller) {
    if (!diagnostics) return;
    wchar_t text[240]{};
    copyDiagnosticName(text, name, reason);
    FILETIME now;
    GetSystemTimeAsFileTime(&now);
    AcquireSRWLockExclusive(&diagnosticsLock);
    const auto sequence = diagnostics->count + 1;
    auto& entry = diagnostics->entries[(sequence - 1) % CacheLogCapacity];
    InterlockedExchange64(&entry.sequence, 0);
    entry.filetime = (static_cast<uint64_t>(now.dwHighDateTime) << 32) | now.dwLowDateTime;
    entry.reason = reason;
    entry.caller = caller;
    memcpy(entry.name, text, sizeof(text));
    InterlockedExchange64(&entry.sequence, sequence);
    InterlockedExchange64(&diagnostics->count, sequence);
    ReleaseSRWLockExclusive(&diagnosticsLock);
}
static void* fallback(Interface* result, const String* name, uint32_t reason, uint32_t caller) {
    ++counters[2];
    recordDiagnostic(name, reason, caller);
    auto original = originalLookup.load();
    if (original) return original(result, name);
    makeDefault(result);
    return result;
}

struct OwnedString {
    String value{};
    ~OwnedString() { destroyString(&value); }
};
struct OwnedList {
    List value{};
    ~OwnedList() {
        if (!value.d || *value.d == -1) return;
        if (InterlockedDecrement(value.d) != 0) return;
        for (intptr_t i = 0; i < value.size; ++i) destroyInterface(value.data + i);
        freeArray(const_cast<LONG*>(value.d), sizeof(Interface), alignof(Interface));
    }
};

static bool refresh() {
    const auto begin = monotonicMs();
    try {
        OwnedList list;
        getAll(&list.value);
        if (list.value.size < 0 || list.value.size > 4096 || (list.value.size > 0 && !list.value.data)) {
            ++counters[4];
            return false; // Preserve last successful snapshot on enumeration failure.
        }
        Snapshot next;
        for (intptr_t i = 0; i < list.value.size; ++i) {
            const auto& iface = list.value.data[i];
            OwnedString name;
            getName(&iface, &name.value);
            if (!name.value.data || name.value.size < 0 || name.value.size > 4096) {
                ++counters[4]; return false;
            }
            auto object = std::make_shared<OwnedInterface>(iface);
            next.emplace(std::wstring(name.value.data, size_t(name.value.size)), object);
            // Qt also accepts a decimal interface index as its lookup key.
            const int index = getIndex(&iface);
            if (index > 0) next.emplace(std::to_wstring(index), object);
        }
        AcquireSRWLockExclusive(&cacheLock);
        cache.swap(next);
        ReleaseSRWLockExclusive(&cacheLock);
        // Old snapshot destruction happens outside the GUI reader's lock.
        counters[6] = uint64_t(list.value.size);
        counters[5] = monotonicMs() - begin;
        lastSuccess = monotonicMs();
        ++counters[3];
        return true;
    } catch (...) {
        ++counters[4];
        return false;
    }
}

static DWORD WINAPI workerMain(void*) {
    counters[7] = GetCurrentThreadId();
    while (WaitForSingleObject(stopEvent, 1000) == WAIT_TIMEOUT) refresh();
    return 0;
}

template <typename T> static bool resolve(HMODULE module, const char* name, T& target) {
    target = reinterpret_cast<T>(GetProcAddress(module, name));
    return target != nullptr;
}

static bool bindQtFunctions(HMODULE network, HMODULE core) {
    return resolve(network, "??0QNetworkInterface@@QEAA@XZ", makeDefault)
        && resolve(network, "??0QNetworkInterface@@QEAA@AEBV0@@Z", copyInterface)
        && resolve(network, "??1QNetworkInterface@@QEAA@XZ", destroyInterface)
        && resolve(network, "?name@QNetworkInterface@@QEBA?AVQString@@XZ", getName)
        && resolve(network, "?index@QNetworkInterface@@QEBAHXZ", getIndex)
        && resolve(network, "?allInterfaces@QNetworkInterface@@SA?AV?$QList@VQNetworkInterface@@@@XZ", getAll)
        && resolve(core, "??1QString@@QEAA@XZ", destroyString)
        && resolve(core, "?deallocate@QArrayData@@SAXPEAU1@_J1@Z", freeArray);
}

extern "C" __declspec(dllexport) int InitializeCache() {
    if (initialized) return 1;
    for (auto& counter : counters) counter = 0;
    lastSuccess = 0;
    engineBase = reinterpret_cast<uintptr_t>(GetModuleHandleW(nullptr));
    HMODULE network = GetModuleHandleW(L"Qt6Network.dll");
    HMODULE core = GetModuleHandleW(L"Qt6Core.dll");
    if (!network || !core) return -1;
    if (!bindQtFunctions(network, core)) return -2;
    // Initial warm-up runs on the instrumentation thread, before installing the hook.
    if (!refresh()) return -3;
    stopEvent = CreateEventW(nullptr, TRUE, FALSE, nullptr);
    if (!stopEvent) return -4;
    worker = CreateThread(nullptr, 0, workerMain, nullptr, 0, nullptr);
    if (!worker) { CloseHandle(stopEvent); stopEvent = nullptr; return -5; }
    SetThreadPriority(worker, THREAD_PRIORITY_BELOW_NORMAL);
    initialized = true;
    return 1;
}

extern "C" __declspec(dllexport) void* CachedInterfaceFromName(Interface* result, const String* name) {
    // Cache only the selected profile's verified call sites. Other callers keep Qt semantics.
    const auto address = reinterpret_cast<uintptr_t>(_ReturnAddress());
    if (!monitoredCaller(address)) {
        auto original = originalLookup.load();
        if (original) return original(result, name);
        makeDefault(result); return result;
    }
    const auto caller = static_cast<uint32_t>(address - engineBase);
    if (!name || name->size < 0 || name->size > 4096 || (name->size && !name->data)) {
        ++counters[0];
        return fallback(result, name, 2, caller);
    }
    // An unspecified interface has a stable invalid result, independent of the
    // snapshot. Construct it through Qt; do not enumerate adapters for this case.
    if (name->size == 0) {
        recordDiagnostic(name, 5, caller);
        return makeDefault(result);
    }
    ++counters[0];
    if (snapshotAge() > 3000) return fallback(result, name, 1, caller);
    try {
        const std::wstring key(name->data ? name->data : L"", size_t(name->size));
        AcquireSRWLockShared(&cacheLock);
        if (snapshotAge() > 3000) {
            ReleaseSRWLockShared(&cacheLock);
            return fallback(result, name, 1, caller);
        }
        auto found = cache.find(key);
        if (found != cache.end()) {
            // Copy only the implicitly-shared Qt interface handle under the lock.
            copyInterface(result, &found->second->value);
            ReleaseSRWLockShared(&cacheLock);
            ++counters[1];
            return result;
        }
        ReleaseSRWLockShared(&cacheLock);
    } catch (...) { return fallback(result, name, 4, caller); }
    return fallback(result, name, 3, caller);
}

extern "C" __declspec(dllexport) void GetCacheStats(uint64_t* output) {
    for (int i = 0; i < 8; ++i) output[i] = counters[i].load();
    output[8] = snapshotAge();
}

extern "C" __declspec(dllexport) int StopCache() {
    if (worker) {
        SetEvent(stopEvent);
        // Do not release objects or handles while a worker may still publish.
        if (WaitForSingleObject(worker, 10000) != WAIT_OBJECT_0) return 0;
        CloseHandle(worker); CloseHandle(stopEvent);
        worker = stopEvent = nullptr;
    }
    Snapshot old;
    AcquireSRWLockExclusive(&cacheLock);
    cache.swap(old);
    ReleaseSRWLockExclusive(&cacheLock);
    initialized = false;
    return 1;
}

// Shared status header: 16 status words followed by up to 16 caller RVAs.
// magic, version, owner PID, command, phase, error, age, lookups, hits, misses,
// refreshes, errors, interfaces, refresh duration, heartbeat, caller count.
static void publish(volatile LONG64* shared) {
    uint64_t values[9]{};
    GetCacheStats(values);
    InterlockedExchange64(shared + 6, static_cast<LONG64>(values[8]));
    for (int i = 0; i < 5; ++i) InterlockedExchange64(shared + 7 + i, static_cast<LONG64>(values[i]));
    InterlockedExchange64(shared + 12, static_cast<LONG64>(values[6]));
    InterlockedExchange64(shared + 13, static_cast<LONG64>(values[5]));
}

static std::atomic<bool> controllerRunning{false};
static bool hookCreated = false;
static void* hookTarget = nullptr;

static bool verifiedCallBoundary(uintptr_t call) {
    DWORD64 base = 0;
    const auto function = RtlLookupFunctionEntry(call, &base, nullptr);
    if (!function || base != engineBase || call < base + function->BeginAddress || call + 6 > base + function->EndAddress) return false;
    auto cursor = base + function->BeginAddress;
    size_t decoded = 0;
    while (cursor < call && decoded++ < 65536) {
        hde64s instruction{};
        hde64_disasm(reinterpret_cast<void*>(cursor), &instruction);
        if (!instruction.len || (instruction.flags & F_ERROR) || cursor + instruction.len > call) return false;
        cursor += instruction.len;
    }
    return cursor == call;
}

static MH_STATUS changeHook(bool enable) {
    for (unsigned attempt = 0; ; ++attempt) {
        const auto status = enable ? MH_EnableHook(hookTarget) : MH_DisableHook(hookTarget);
        // A thread that exited between OpenThread and SuspendThread also yields
        // THREAD_CONTROL; retry so the next freeze works on the remaining set.
        if (status != MH_ERROR_THREAD_BUSY && status != MH_ERROR_THREAD_CONTROL) return status;
        if (attempt == 7) return status;
        Sleep(1);
    }
}
extern "C" DWORD WINAPI RunInstalledCache() {
    bool expected = false;
    if (!controllerRunning.compare_exchange_strong(expected, true)) return 10;
    wchar_t mappingName[96];
    swprintf_s(mappingName, L"Local\\EngineToolsCache-%lu", GetCurrentProcessId());
    HANDLE mapping = OpenFileMappingW(FILE_MAP_ALL_ACCESS, FALSE, mappingName);
    if (!mapping) { controllerRunning = false; return 11; }
    auto shared = static_cast<volatile LONG64*>(MapViewOfFile(mapping, FILE_MAP_ALL_ACCESS, 0, 0, 256));
    if (!shared) { CloseHandle(mapping); controllerRunning = false; return 12; }
    if (shared[0] != 0x45435431 || shared[1] != CacheProtocol || shared[2] != GetCurrentProcessId()) {
        UnmapViewOfFile(const_cast<LONG64*>(shared)); CloseHandle(mapping); controllerRunning = false; return 13;
    }
    // Validate the complete supplied call list before touching Qt or installing a hook.
    engineBase = reinterpret_cast<uintptr_t>(GetModuleHandleW(nullptr));
    const auto image = reinterpret_cast<const IMAGE_DOS_HEADER*>(engineBase);
    const auto nt = reinterpret_cast<const IMAGE_NT_HEADERS64*>(engineBase + image->e_lfanew);
    const auto size = nt->OptionalHeader.SizeOfImage;
    const auto target = GetProcAddress(GetModuleHandleW(L"Qt6Network.dll"), "?interfaceFromName@QNetworkInterface@@SA?AV1@AEBVQString@@@Z");
    bool callersValid = shared[15] > 0 && shared[15] <= 16 && target;
    allowedCallerCount = 0;
    for (LONG64 i = 0; callersValid && i < shared[15]; ++i) {
        const auto rva = static_cast<uint64_t>(shared[16 + i]);
        if (rva < 6 || rva >= size) { callersValid = false; break; }
        const auto call = reinterpret_cast<const uint8_t*>(engineBase + rva - 6);
        if (call[0] != 0xff || call[1] != 0x15 || !verifiedCallBoundary(reinterpret_cast<uintptr_t>(call))) { callersValid = false; break; }
        int32_t displacement;
        memcpy(&displacement, call + 2, sizeof(displacement));
        const auto slot = static_cast<int64_t>(rva) + displacement;
        if (slot < 0 || static_cast<uint64_t>(slot) + sizeof(void*) > size ||
            *reinterpret_cast<const void* const*>(engineBase + slot) != reinterpret_cast<void*>(target)) {
            callersValid = false; break;
        }
        allowedCallers[allowedCallerCount++] = static_cast<uint32_t>(rva);
    }
    if (!callersValid) {
        InterlockedExchange64(shared + 5, 15);
        InterlockedExchange64(shared + 4, 4);
        UnmapViewOfFile(const_cast<LONG64*>(shared)); CloseHandle(mapping);
        controllerRunning = false; return 15;
    }
    DWORD error = 0;
    bool enabled = false;
    if (!error) {
        const int result = InitializeCache();
        if (result != 1) error = 100 + DWORD(-result);
    }
    if (!error) {
        if (!hookCreated) {
            const auto init = MH_Initialize();
            if (init != MH_OK && init != MH_ERROR_ALREADY_INITIALIZED) error = 200 + init;
            hookTarget = reinterpret_cast<void*>(GetProcAddress(GetModuleHandleW(L"Qt6Network.dll"), "?interfaceFromName@QNetworkInterface@@SA?AV1@AEBVQString@@@Z"));
            if (!hookTarget) error = 220;
            void* trampoline = nullptr;
            if (!error) {
                const auto created = MH_CreateHook(hookTarget, reinterpret_cast<void*>(CachedInterfaceFromName), &trampoline);
                if (created != MH_OK) error = 230 + created;
                else { SetOriginalLookup(trampoline); hookCreated = true; }
            }
        }
        if (!error) {
            const auto status = changeHook(true);
            if (status != MH_OK) error = 260 + status;
            else enabled = true;
        }
    }
    if (!error) {
        publish(shared);
        InterlockedExchange64(shared + 4, 2); // Active only after the hook is enabled.
        while (InterlockedCompareExchange64(shared + 3, 0, 0) == 0) {
            publish(shared);
            Sleep(100);
            if (snapshotAge() > 3000) { error = 302; break; }
        }
    }
    InterlockedExchange64(shared + 4, 5); // Stopping.
    if (enabled) {
        const auto status = changeHook(false);
        if (status != MH_OK) error = 330 + status;
    }
    // Do not delete the trampoline or unload this DLL: a concurrent caller may
    // still be returning through the replacement. Retain them until process exit.
    if (!StopCache()) error = 303;
    publish(shared);
    InterlockedExchange64(shared + 5, error);
    InterlockedExchange64(shared + 4, error ? 4 : 3);
    UnmapViewOfFile(const_cast<LONG64*>(shared));
    CloseHandle(mapping);
    // A stuck worker must not be restarted against the same Qt cache objects.
    if (error != 303) controllerRunning = false;
    return error;
}

// Executed only in a separate, disposable EngineTools probe process. No hook
// is installed and no Engine DJ process is opened by this entry point.
template <typename T> struct GuardedQt {
    uint64_t before = 0x9a684a7fc75d4a8bULL;
    T value{};
    uint64_t after = 0x2ab6c4d92ce4a9d1ULL;
    bool intact() const { return before == 0x9a684a7fc75d4a8bULL && after == 0x2ab6c4d92ce4a9d1ULL; }
};

static DWORD probeQt(const wchar_t* directory) {
    SetErrorMode(SEM_FAILCRITICALERRORS | SEM_NOGPFAULTERRORBOX);
    const std::wstring root(directory);
    const auto core = LoadLibraryExW((root + L"\\Qt6Core.dll").c_str(), nullptr,
        LOAD_LIBRARY_SEARCH_DLL_LOAD_DIR | LOAD_LIBRARY_SEARCH_DEFAULT_DIRS);
    const auto network = LoadLibraryExW((root + L"\\Qt6Network.dll").c_str(), nullptr,
        LOAD_LIBRARY_SEARCH_DLL_LOAD_DIR | LOAD_LIBRARY_SEARCH_DEFAULT_DIRS);
    if (!core || !network) return 401;
    using VersionFn = const char* (*)();
    auto version = reinterpret_cast<VersionFn>(GetProcAddress(core, "qVersion"));
    if (!version || strncmp(version(), "6.", 2) != 0) return 402;
    if (!bindQtFunctions(network, core)) return 403;
    auto lookup = reinterpret_cast<LookupFn>(GetProcAddress(network, "?interfaceFromName@QNetworkInterface@@SA?AV1@AEBVQString@@@Z"));
    if (!lookup) return 404;

    GuardedQt<Interface> empty, copied;
    makeDefault(&empty.value);
    if (!empty.intact() || getIndex(&empty.value) > 0) return 405;
    copyInterface(&copied.value, &empty.value);
    if (!copied.intact() || copied.value.d != empty.value.d) return 406;
    destroyInterface(&copied.value);
    destroyInterface(&empty.value);
    if (!empty.intact() || !copied.intact()) return 407;

    // Check the actual target Qt library's empty-name semantics before allowing
    // this fast path on an automatically detected build.
    GuardedQt<String> emptyName;
    GuardedQt<Interface> emptyLookup, fastEmpty;
    lookup(&emptyLookup.value, &emptyName.value);
    makeDefault(&fastEmpty.value);
    const bool equivalentEmpty = emptyName.intact() && emptyLookup.intact() && fastEmpty.intact() &&
        getIndex(&emptyLookup.value) == 0 && getIndex(&fastEmpty.value) == 0 &&
        emptyLookup.value.d == fastEmpty.value.d;
    destroyInterface(&emptyLookup.value);
    destroyInterface(&fastEmpty.value);
    if (!equivalentEmpty || !emptyLookup.intact() || !fastEmpty.intact()) return 414;

    GuardedQt<List> raw;
    getAll(&raw.value);
    if (!raw.intact() || !raw.value.d || !raw.value.data || raw.value.size < 1 || raw.value.size > 4096) return 408;
    OwnedList owned;
    owned.value = raw.value;
    raw.value = {};
    for (intptr_t i = 0; i < owned.value.size && i < 3; ++i) {
        const auto& iface = owned.value.data[i];
        const int index = getIndex(&iface);
        GuardedQt<String> name, roundtrip;
        getName(&iface, &name.value);
        if (!name.intact() || !name.value.data || name.value.size < 1 || name.value.size > 4096 || index <= 0) return 409;
        GuardedQt<Interface> result, copy;
        lookup(&result.value, &name.value);
        if (!result.intact() || getIndex(&result.value) != index) return 410;
        copyInterface(&copy.value, &result.value);
        if (!copy.intact() || copy.value.d != result.value.d) return 411;
        getName(&copy.value, &roundtrip.value);
        if (!roundtrip.intact() || roundtrip.value.size != name.value.size ||
            !roundtrip.value.data || memcmp(roundtrip.value.data, name.value.data, name.value.size * sizeof(wchar_t)) != 0) return 412;
        destroyString(&roundtrip.value);
        destroyString(&name.value);
        destroyInterface(&copy.value);
        destroyInterface(&result.value);
        if (!roundtrip.intact() || !name.intact() || !result.intact() || !copy.intact()) return 413;
    }
    return 0;
}

extern "C" __declspec(dllexport) DWORD WINAPI ProbeABI(const wchar_t* directory) {
    __try {
        return probeQt(directory);
    } __except(EXCEPTION_EXECUTE_HANDLER) {
        return 490;
    }
}
