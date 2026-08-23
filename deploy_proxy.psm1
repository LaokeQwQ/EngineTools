# EngineTools translation-proxy deployment module.
#
# Transactional install/restore of the version.dll translation proxy next
# to Engine DJ.exe, guarded by a JSON hash journal:
#
#   Install-EngineProxy : stage version_orig.dll from the system DLL,
#       validate the proxy payload (x64 PE carrying version_orig
#       forwarders), persist the journal, then atomically swap the proxy
#       into place with hash verification and rollback.
#   Restore-EngineProxy : undo a journaled install exactly, refusing to
#       delete any version.dll whose hash the journal does not own.
#
# Every filesystem mutation goes through a sibling temp file that is hash-
# verified before being renamed into place, so an interrupted run can only
# ever leave a stray *.enginetools-tmp behind, never a half-written DLL.
Set-StrictMode -Version Latest

$script:JournalName = 'enginetools_proxy_journal.json'
$script:TmpSuffix = '.enginetools-tmp'
$script:PrevSuffix = '.enginetools-prev'

function Get-PEMachine {
    # Returns the PE header machine field (0x8664 = x64).
    param([Parameter(Mandatory)][string]$LiteralPath)

    $fs = [System.IO.File]::OpenRead($LiteralPath)
    try {
        $reader = New-Object System.IO.BinaryReader($fs)
        if ($fs.Length -lt 0x40 + 6) { throw "file too small to be a PE: $LiteralPath" }
        $null = $reader.ReadUInt16()                    # 'MZ'
        $fs.Position = 0x3C
        $peOffset = $reader.ReadInt32()
        if ($peOffset -lt 0 -or ($peOffset + 6) -gt $fs.Length) { throw "corrupt PE offset in $LiteralPath" }
        $fs.Position = $peOffset
        $signature = $reader.ReadBytes(4)
        if ($signature[0] -ne 0x50 -or $signature[1] -ne 0x45) { throw "missing PE signature in $LiteralPath" }
        return $reader.ReadUInt16()
    } finally {
        $fs.Dispose()
    }
}

function Test-RawContainsAscii {
    param([Parameter(Mandatory)][string]$LiteralPath,
          [Parameter(Mandatory)][string]$Text)

    $bytes = [System.IO.File]::ReadAllBytes($LiteralPath)
    $ascii = [System.Text.Encoding]::ASCII.GetString($bytes)
    return $ascii.Contains($Text)
}

function Copy-VerifiedFile {
    # Copies Source over Destination through a sibling temp file, verifying
    # the copied bytes before the rename. Fails closed: on any mismatch the
    # temp is deleted and Destination is never touched.
    param([Parameter(Mandatory)][string]$Source,
          [Parameter(Mandatory)][string]$Destination)

    $temp = "$Destination$script:TmpSuffix"
    try {
        Copy-Item -LiteralPath $Source -Destination $temp -Force
        $srcHash = (Get-FileHash -LiteralPath $Source -Algorithm SHA256).Hash
        $tmpHash = (Get-FileHash -LiteralPath $temp -Algorithm SHA256).Hash
        if ($srcHash -ne $tmpHash) { throw "copy verification failed for $Destination" }
        Move-Item -LiteralPath $temp -Destination $Destination -Force
    } finally {
        if (Test-Path -LiteralPath $temp) { Remove-Item -LiteralPath $temp -Force -ErrorAction SilentlyContinue }
    }
}

function Assert-NoRunningEngine {
    # Stops a confirmed running Engine DJ process only after the caller has
    # opted in with -AllowStop; otherwise fails loudly.
    param([string]$ProcessName = 'Engine DJ',
          [switch]$AllowStop)

    $processes = @(Get-Process -Name $ProcessName -ErrorAction SilentlyContinue)
    if ($processes.Count -eq 0) { return }
    if (-not $AllowStop) {
        throw "$ProcessName is running (pid $($processes[0].Id)); rerun with -AllowStop to terminate it"
    }
    $processes | Stop-Process -Force -ErrorAction Stop
    Start-Sleep -Seconds 1
    if (@(Get-Process -Name $ProcessName -ErrorAction SilentlyContinue).Count -gt 0) {
        throw "failed to terminate $ProcessName"
    }
}

function Write-ProxyJournal {
    param([Parameter(Mandatory)][hashtable]$Data,
          [Parameter(Mandatory)][string]$Path)

    $ordered = [ordered]@{
        schema          = 1
        timestampUtc    = [DateTime]::UtcNow.ToString('o')
        engineDirectory = $Data.engineDirectory
        proxySha256     = $Data.proxySha256
        proxyBytes      = $Data.proxyBytes
        originalSha256  = $Data.originalSha256
        originalBytes   = $Data.originalBytes
    }
    $ordered | ConvertTo-Json | Set-Content -LiteralPath $Path -Encoding utf8
}

function Install-EngineProxy {
    <#
    .SYNOPSIS
        Deploys the translation proxy next to Engine DJ.exe as one transaction.
    .PARAMETER AllowStop
        Explicit user approval to terminate a running Engine DJ process.
    #>
    [CmdletBinding()]
    param(
        [Parameter(Mandatory)][string]$EngineDirectory,
        [Parameter(Mandatory)][string]$ProxyDllPath,
        [string]$SystemVersionDll = (Join-Path $env:SystemRoot 'System32\version.dll'),
        [switch]$AllowStop,
        [switch]$ClearQmlCache
    )

    $ErrorActionPreference = 'Stop'

    # Explicit install path: must already exist and look like Engine DJ.
    $engine = (Resolve-Path -LiteralPath $EngineDirectory -ErrorAction Stop).Path
    if (-not (Test-Path -LiteralPath (Join-Path $engine 'Engine DJ.exe') -PathType Leaf)) {
        throw "Engine DJ.exe not found under '$engine'; refusing to guess the install location"
    }
    Assert-NoRunningEngine -AllowStop:$AllowStop

    $proxyTarget = Join-Path $engine 'version.dll'
    $originalTarget = Join-Path $engine 'version_orig.dll'
    $journalPath = Join-Path $engine $script:JournalName

    # Sweep temps left behind by an interrupted earlier run.
    foreach ($stale in @("$proxyTarget$script:TmpSuffix", "$originalTarget$script:TmpSuffix")) {
        if (Test-Path -LiteralPath $stale) { Remove-Item -LiteralPath $stale -Force }
    }

    # --- Phase 1: stage version_orig.dll from the system DLL -------------
    $systemHash = Get-FileHash -LiteralPath $SystemVersionDll -Algorithm SHA256 -ErrorAction Stop
    if (Test-Path -LiteralPath $originalTarget) {
        $existingOriginal = (Get-FileHash -LiteralPath $originalTarget -Algorithm SHA256).Hash
        if ($existingOriginal -ne $systemHash.Hash) {
            throw "existing version_orig.dll does not match the system version.dll; unknown provenance, refusing"
        }
    } else {
        Copy-VerifiedFile -Source $SystemVersionDll -Destination $originalTarget
    }
    $stagedOriginal = (Get-FileHash -LiteralPath $originalTarget -Algorithm SHA256).Hash
    if ($stagedOriginal -ne $systemHash.Hash) { throw 'version_orig.dll hash mismatch after staging' }

    # --- Phase 2: validate the proxy payload -----------------------------
    $proxyHash = Get-FileHash -LiteralPath $ProxyDllPath -Algorithm SHA256 -ErrorAction Stop
    if ((Get-PEMachine -LiteralPath $ProxyDllPath) -ne 0x8664) { throw 'proxy is not an x64 PE image' }
    if (-not (Test-RawContainsAscii -LiteralPath $ProxyDllPath -Text 'version_orig')) {
        throw 'proxy binary carries no version_orig forwarders'
    }

    # --- Phase 3: journal the intended end state -------------------------
    Write-ProxyJournal -Path $journalPath -Data @{
        engineDirectory = $engine
        proxySha256     = $proxyHash.Hash
        proxyBytes      = (Get-Item -LiteralPath $ProxyDllPath).Length
        originalSha256  = $systemHash.Hash
        originalBytes   = (Get-Item -LiteralPath $SystemVersionDll).Length
    }

    # --- Phase 4: atomic swap with rollback ------------------------------
    $previousBackup = "$proxyTarget$script:PrevSuffix"
    $hadPrevious = Test-Path -LiteralPath $proxyTarget
    if ($hadPrevious) { Copy-Item -LiteralPath $proxyTarget -Destination $previousBackup -Force }

    try {
        Copy-VerifiedFile -Source $ProxyDllPath -Destination $proxyTarget
        $swapped = (Get-FileHash -LiteralPath $proxyTarget -Algorithm SHA256).Hash
        if ($swapped -ne $proxyHash.Hash) { throw 'proxy hash mismatch after swap' }
    } catch {
        if ($hadPrevious -and (Test-Path -LiteralPath $previousBackup)) {
            Move-Item -LiteralPath $previousBackup -Destination $proxyTarget -Force
        }
        Remove-Item -LiteralPath $journalPath -Force -ErrorAction SilentlyContinue
        throw
    } finally {
        if (Test-Path -LiteralPath $previousBackup) {
            Remove-Item -LiteralPath $previousBackup -Force -ErrorAction SilentlyContinue
        }
    }

    if ($ClearQmlCache) { Clear-EngineQmlCache }

    return [pscustomobject]@{
        Installed     = $true
        EngineDir     = $engine
        ProxySha256   = $proxyHash.Hash
        OriginalSha256 = $systemHash.Hash
        JournalPath   = $journalPath
    }
}

function Restore-EngineProxy {
    <#
    .SYNOPSIS
        Undoes a journaled proxy install exactly.
    .DESCRIPTION
        Restores version.dll from the journaled version_orig.dll only when
        both hashes still match the journal; refuses to delete an unknown
        version.dll.
    #>
    [CmdletBinding()]
    param(
        [Parameter(Mandatory)][string]$EngineDirectory,
        [switch]$AllowStop,
        [switch]$ClearQmlCache
    )

    $ErrorActionPreference = 'Stop'

    $engine = (Resolve-Path -LiteralPath $EngineDirectory -ErrorAction Stop).Path
    Assert-NoRunningEngine -AllowStop:$AllowStop

    $proxyTarget = Join-Path $engine 'version.dll'
    $originalTarget = Join-Path $engine 'version_orig.dll'
    $journalPath = Join-Path $engine $script:JournalName

    if (-not (Test-Path -LiteralPath $journalPath -PathType Leaf)) {
        throw "no deployment journal under '$engine'; refusing to touch version.dll"
    }
    $journal = Get-Content -LiteralPath $journalPath -Raw | ConvertFrom-Json
    if ($journal.schema -ne 1) { throw "unsupported journal schema $($journal.schema)" }

    if (-not (Test-Path -LiteralPath $originalTarget -PathType Leaf)) {
        throw 'version_orig.dll is missing; cannot restore'
    }
    $originalNow = (Get-FileHash -LiteralPath $originalTarget -Algorithm SHA256).Hash
    if ($originalNow -ne $journal.originalSha256) { throw 'version_orig.dll hash differs from journal; refusing' }

    if (Test-Path -LiteralPath $proxyTarget -PathType Leaf) {
        $proxyNow = (Get-FileHash -LiteralPath $proxyTarget -Algorithm SHA256).Hash
        if ($proxyNow -ne $journal.proxySha256) {
            throw 'installed version.dll hash differs from journal; refusing to delete an unknown file'
        }
        Remove-Item -LiteralPath $proxyTarget -Force
    }

    Copy-VerifiedFile -Source $originalTarget -Destination $proxyTarget
    $restored = (Get-FileHash -LiteralPath $proxyTarget -Algorithm SHA256).Hash
    if ($restored -ne $journal.originalSha256) { throw 'restore verification failed' }

    Remove-Item -LiteralPath $originalTarget -Force
    Remove-Item -LiteralPath $journalPath -Force

    if ($ClearQmlCache) { Clear-EngineQmlCache }

    return [pscustomobject]@{
        Restored       = $true
        EngineDir      = $engine
        RestoredSha256 = $restored
    }
}

function Clear-EngineQmlCache {
    # Best-effort QML cache clear; never fatal.
    if ([string]::IsNullOrEmpty($env:LOCALAPPDATA)) { return }
    $cache = Join-Path $env:LOCALAPPDATA 'AIR Music Technology\EnginePrime\cache'
    if (Test-Path -LiteralPath $cache) {
        Get-ChildItem -LiteralPath $cache -Recurse -File -ErrorAction SilentlyContinue |
            Remove-Item -Force -ErrorAction SilentlyContinue
    }
}

Export-ModuleMember -Function Install-EngineProxy, Restore-EngineProxy, Clear-EngineQmlCache, Get-PEMachine
