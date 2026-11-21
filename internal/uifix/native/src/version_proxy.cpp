#define WIN32_LEAN_AND_MEAN
#include <windows.h>
#include <bcrypt.h>
#include <cstdint>
#include <cstring>
#include <string>
#include "cache_protocol.h"

extern "C" DWORD WINAPI RunInstalledCache();
static HMODULE selfModule;
static INIT_ONCE versionOnce = INIT_ONCE_STATIC_INIT;
static HMODULE systemVersion;
static const char* exports[] = {
    "GetFileVersionInfoA", "GetFileVersionInfoByHandle", "GetFileVersionInfoExA",
    "GetFileVersionInfoExW", "GetFileVersionInfoSizeA", "GetFileVersionInfoSizeExA",
    "GetFileVersionInfoSizeExW", "GetFileVersionInfoSizeW", "GetFileVersionInfoW",
    "VerFindFileA", "VerFindFileW", "VerInstallFileA", "VerInstallFileW",
    "VerLanguageNameA", "VerLanguageNameW", "VerQueryValueA", "VerQueryValueW"
};
static BOOL CALLBACK loadVersion(PINIT_ONCE, PVOID, PVOID*) {
    wchar_t path[MAX_PATH];
    const UINT size = GetSystemDirectoryW(path, MAX_PATH);
    if (!size || size + 13 >= MAX_PATH) return FALSE;
    wcscat_s(path, L"\\version.dll");
    systemVersion = LoadLibraryExW(path, nullptr, LOAD_LIBRARY_SEARCH_SYSTEM32);
    return systemVersion != nullptr;
}
extern "C" void* ResolveVersionProc(unsigned index) {
    if (index >= _countof(exports) || !InitOnceExecuteOnce(&versionOnce, loadVersion, nullptr, nullptr))
        RaiseFailFastException(nullptr, nullptr, 0);
    auto proc = GetProcAddress(systemVersion, exports[index]);
    if (!proc) RaiseFailFastException(nullptr, nullptr, 0);
    return reinterpret_cast<void*>(proc);
}

#pragma pack(push, 1)
struct Configuration {
    char magic[16];
    uint32_t version, count;
    unsigned char hashes[3][32]; // Engine DJ.exe, Qt6Core.dll, Qt6Network.dll
    uint32_t callers[16];
    unsigned char payloadHash[32];
    uint64_t payloadSize;
    unsigned char checksum[32];
};
#pragma pack(pop)
static_assert(sizeof(Configuration) == 256);

static bool hash(HANDLE file, uint64_t length, unsigned char* output) {
    BCRYPT_ALG_HANDLE algorithm = nullptr;
    BCRYPT_HASH_HANDLE digest = nullptr;
    if (BCryptOpenAlgorithmProvider(&algorithm, BCRYPT_SHA256_ALGORITHM, nullptr, 0) < 0) return false;
    bool ok = BCryptCreateHash(algorithm, &digest, nullptr, 0, nullptr, 0, 0) >= 0;
    unsigned char buffer[65536];
    while (ok && length) {
        DWORD read = 0;
        const DWORD chunk = length < sizeof(buffer) ? static_cast<DWORD>(length) : sizeof(buffer);
        ok = ReadFile(file, buffer, chunk, &read, nullptr) && read == chunk;
        if (ok) ok = BCryptHashData(digest, buffer, read, 0) >= 0;
        length -= read;
    }
    if (ok) ok = BCryptFinishHash(digest, output, 32, 0) >= 0;
    if (digest) BCryptDestroyHash(digest);
    BCryptCloseAlgorithmProvider(algorithm, 0);
    return ok;
}
static bool matches(const std::wstring& path, const unsigned char* expected) {
    HANDLE file = CreateFileW(path.c_str(), GENERIC_READ, FILE_SHARE_READ, nullptr, OPEN_EXISTING, 0, nullptr);
    if (file == INVALID_HANDLE_VALUE) return false;
    LARGE_INTEGER size{};
    unsigned char actual[32];
    const bool ok = GetFileSizeEx(file, &size) && size.QuadPart >= 0 &&
        hash(file, static_cast<uint64_t>(size.QuadPart), actual) && memcmp(actual, expected, 32) == 0;
    CloseHandle(file);
    return ok;
}
static bool readConfiguration(const std::wstring& path, Configuration& config) {
    HANDLE file = CreateFileW(path.c_str(), GENERIC_READ, FILE_SHARE_READ, nullptr, OPEN_EXISTING, 0, nullptr);
    if (file == INVALID_HANDLE_VALUE) return false;
    LARGE_INTEGER size{}, offset{};
    offset.QuadPart = -static_cast<LONGLONG>(sizeof(config));
    DWORD read = 0;
    bool ok = GetFileSizeEx(file, &size) && size.QuadPart > sizeof(config) &&
        SetFilePointerEx(file, offset, nullptr, FILE_END) &&
        ReadFile(file, &config, sizeof(config), &read, nullptr) && read == sizeof(config) &&
        memcmp(config.magic, "EngineToolsPrx01", 16) == 0 && config.version == 1 &&
        config.count > 0 && config.count <= 16 && config.payloadSize == size.QuadPart - sizeof(config);
    unsigned char actual[32];
    offset.QuadPart = 0;
    if (ok) ok = SetFilePointerEx(file, offset, nullptr, FILE_BEGIN) && hash(file, config.payloadSize, actual) &&
        memcmp(actual, config.payloadHash, 32) == 0;
    // Hash the trailer bytes directly with CNG; no persisted configuration file.
    if (ok) {
        BCRYPT_ALG_HANDLE algorithm = nullptr;
        ok = BCryptOpenAlgorithmProvider(&algorithm, BCRYPT_SHA256_ALGORITHM, nullptr, 0) >= 0;
        if (ok) {
            ok = BCryptHash(algorithm, nullptr, 0, reinterpret_cast<PUCHAR>(&config), 224, actual, 32) >= 0 &&
                memcmp(actual, config.checksum, 32) == 0;
            BCryptCloseAlgorithmProvider(algorithm, 0);
        }
    }
    CloseHandle(file);
    return ok;
}

static DWORD WINAPI standaloneMain(void*) {
    wchar_t executable[32768], proxy[32768];
    if (!GetModuleFileNameW(nullptr, executable, _countof(executable)) ||
        !GetModuleFileNameW(selfModule, proxy, _countof(proxy))) return 0;
    const std::wstring exe(executable), own(proxy);
    const auto split = exe.find_last_of(L"\\/");
    if (split == std::wstring::npos || _wcsicmp(exe.c_str() + split + 1, L"Engine DJ.exe") != 0) return 0;
    const auto directory = exe.substr(0, split + 1);
    if (_wcsicmp(own.c_str(), (directory + L"version.dll").c_str()) != 0) return 0;

    wchar_t name[96];
    swprintf_s(name, L"Local\\EngineToolsCache-%lu", GetCurrentProcessId());
    HANDLE mapping = CreateFileMappingW(INVALID_HANDLE_VALUE, nullptr, PAGE_READWRITE, 0, static_cast<DWORD>(CacheMappingBytes), name);
    if (!mapping) return 0;
    if (GetLastError() == ERROR_ALREADY_EXISTS) { CloseHandle(mapping); return 0; }
    auto shared = static_cast<volatile LONG64*>(MapViewOfFile(mapping, FILE_MAP_ALL_ACCESS, 0, 0, CacheMappingBytes));
    if (!shared) { CloseHandle(mapping); return 0; }
    shared[0] = 0x45435431; shared[1] = CacheProtocol; shared[2] = GetCurrentProcessId();
    auto diagnostics = reinterpret_cast<CacheDiagnostics*>(reinterpret_cast<uintptr_t>(shared) + CacheStatusBytes);
    FILETIME session;
    GetSystemTimeAsFileTime(&session);
    diagnostics->session = (static_cast<uint64_t>(session.dwHighDateTime) << 32) | session.dwLowDateTime;
    SetCacheDiagnostics(diagnostics);
    InterlockedExchange64(shared + 4, 1);
    Configuration config{};
    DWORD error = readConfiguration(own, config) ? 0 : 500;
    const wchar_t* files[] = {L"Engine DJ.exe", L"Qt6Core.dll", L"Qt6Network.dll"};
    for (int i = 0; !error && i < 3; ++i) if (!matches(directory + files[i], config.hashes[i])) error = 501;
    for (int i = 1; !error && i < 3; ++i) {
        wchar_t loaded[32768];
        const auto module = GetModuleHandleW(files[i]);
        if (!module || !GetModuleFileNameW(module, loaded, _countof(loaded)) ||
            _wcsicmp(loaded, (directory + files[i]).c_str()) != 0) error = 501;
    }
    if (!error) {
        shared[15] = config.count;
        for (uint32_t i = 0; i < config.count; ++i) shared[16 + i] = config.callers[i];
        error = RunInstalledCache();
    }
    if (error) { InterlockedExchange64(shared + 5, error); InterlockedExchange64(shared + 4, 4); }
    // Keep diagnostics readable for a later EngineTools instance, including failures.
    Sleep(INFINITE);
    return 0;
}

BOOL WINAPI DllMain(HINSTANCE module, DWORD reason, LPVOID) {
    if (reason == DLL_PROCESS_ATTACH) {
        selfModule = module;
        // Loading this DLL for an isolated ABI probe must not create a worker
        // which could still execute after the probe unloads the module.
        wchar_t executable[32768];
        if (!GetModuleFileNameW(nullptr, executable, _countof(executable))) return TRUE;
        const auto filename = wcsrchr(executable, L'\\');
        if (!filename || _wcsicmp(filename + 1, L"Engine DJ.exe") != 0) return TRUE;
        // The entry thread cannot execute until process DLL initialization ends.
        // Never wait or call Qt/LoadLibrary/scan/hash while holding the loader lock.
        HANDLE thread = CreateThread(nullptr, 0, standaloneMain, nullptr, 0, nullptr);
        if (thread) CloseHandle(thread);
    }
    return TRUE;
}
