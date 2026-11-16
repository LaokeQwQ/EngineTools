# Engine DJ DLL patch

Windows x64 cache deployed only as a self-contained version.dll proxy.
EngineTools installs, uninstalls and monitors it; it does not launch, attach to,
or close Engine DJ. Python, Frida and a separate background executable are not used.

## Installation and monitoring

Close Engine DJ before installing or uninstalling the DLL. Installation adds
version.dll beside Engine DJ.exe and never overwrites a third-party proxy.
The proxy contains a checksummed configuration trailer with the verified
EXE/Qt hashes and query caller RVAs. No configuration sidecar is installed.
Updates and removal require a validated EngineTools-owned proxy.

Engine DJ starts caching automatically when launched normally. EngineTools
can be closed and reopened without affecting the cache. Status, hits,
fallbacks, refreshes and errors are read from a read-only view of
Local\\EngineToolsCache-<Engine DJ PID>. The mapping belongs to Engine DJ,
requires no external heartbeat and disappears when Engine DJ exits.
There is no temporary injection mode, launch.json or persistent engine-cache directory.

The proxy forwards the 17 Windows version.dll exports to the real System32
DLL. DllMain starts a worker only in Engine DJ, without waiting, loading Qt,
hashing or installing hooks under the loader lock. A compatibility probe
loads the same embedded proxy in an isolated process without starting that worker.

## Compatibility

Known EXE/Core/Network hash combinations use reviewed profiles (5.1.0 and
5.3.0). Unknown combinations use PE import and x64 instruction-boundary scanning.
Displayed Engine DJ version strings do not select compatibility.

Unknown builds also run an isolated Qt ABI probe before installation. A
temporary copy of the embedded version.dll is removed after the probe process
exits, including failed probes. Successful file hashes, caller RVAs, validation
time and detector identity are saved in setting.json. Matching EXE/Qt files and
detector identity reuse the result across restarts without scanning or probing.
Any target file or detector change invalidates the saved result. Failed checks
are never saved, and unrelated settings updates preserve verified profiles.
ABI checks cover object sizes, construction/copy/destruction, list traversal,
and lookup/name/index round trips. Missing capabilities, ambiguous call sites,
unsupported architectures, crashes or timeouts reject installation.

Empty interface names from verified callers return a Qt-constructed invalid
interface directly without enumerating adapters. They are logged separately
and excluded from lookup, hit and fallback counters and the hit-rate display. The ABI
probe checks this result against the target Qt library's original empty lookup.
Only verified interfaceFromName callers use the cache. Other callers, misses
and snapshots older than three seconds use the original Qt function. Snapshot
refresh runs every second. Persistent refresh failure removes the hook;
trampoline memory stays loaded until Engine DJ exits.

An Engine DJ/Qt update invalidates the installed compatibility hashes. The
proxy keeps forwarding version calls, disables caching and reports the mismatch.
Close Engine DJ and reinstall the DLL to update its configuration.

## Logging

EngineTools writes application, frontend and framework messages to
Log/YYYY-MM-DD.log beside its executable, using the system's local date.
Files are appended across restarts and rotate when the date changes.
UI history is capped at 2000 entries; daily files retain the complete collected log.

Proxy protocol 3 adds an in-memory ring of the latest 64 fallback records:
system timestamp, reason, queried interface name/index and caller RVA.
Reasons distinguish an expired snapshot, invalid name parameters, a missing
snapshot entry and a lookup exception. EngineTools drains the ring every two
seconds into the same daily log. Reopening EngineTools reads retained records;
overwritten records are reported with a lost-record count. No file I/O runs
inside the hooked Qt query. Old protocol-2 proxies still expose counters but
need reinstalling before fallback details can be collected.

## Build and verification

Run native\\build.cmd with x64 MSVC. It builds only version.dll and copies it
to assets\\version.bin for Go embedding. It does not build an EngineTools EXE.
Keep native source and this embedded asset synchronized.
The script locates Visual Studio C++ tools with vswhere, or uses an existing
x64 Native Tools Command Prompt environment.

Native directory layout:

- `native/src/`: proxy forwarding, cache hooks, shared protocol and exports.
- `native/minhook/`: vendored MinHook sources.
- `native/build/`: generated DLL and compiler intermediates (ignored by Git).
- `native/build.cmd`: x64 MSVC build entry point.
- `assets/version.bin`: embedded DLL payload used by the Go installer.

Critical install protection tests live in `test/uifix_test.go` at the
project root. Run `go test ./test` to check that foreign DLLs are preserved and
unsupported installations are rejected. These tests do not start Engine DJ.
Network switching and hardware reconnection require runtime verification.

MinHook v1.3.4 is vendored at native/minhook; its license is in THIRD_PARTY.txt.
