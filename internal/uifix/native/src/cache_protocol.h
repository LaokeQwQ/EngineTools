#pragma once
#include <windows.h>
#include <cstdint>

constexpr uint64_t CacheProtocol = 3;
constexpr size_t CacheStatusBytes = 256;
constexpr size_t CacheLogCapacity = 64;
struct CacheLogEntry {
    volatile LONG64 sequence;
    uint64_t filetime;
    uint32_t reason, caller;
    wchar_t name[240];
    uint64_t reserved;
};
static_assert(sizeof(CacheLogEntry) == 512);
struct CacheDiagnostics {
    volatile LONG64 count;
    uint64_t session;
    CacheLogEntry entries[CacheLogCapacity];
};
constexpr size_t CacheMappingBytes = CacheStatusBytes + sizeof(CacheDiagnostics);
extern "C" void SetCacheDiagnostics(CacheDiagnostics* diagnostics);
