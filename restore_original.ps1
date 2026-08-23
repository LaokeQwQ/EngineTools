# Restore Engine DJ to its original version.dll exactly as journaled.
param(
    [string]$EngineDirectory = 'C:\Program Files\Engine DJ',
    [switch]$AllowStop,
    [switch]$ClearQmlCache
)

$ErrorActionPreference = 'Stop'
Import-Module (Join-Path $PSScriptRoot 'deploy_proxy.psm1') -Force

$result = Restore-EngineProxy -EngineDirectory $EngineDirectory `
    -AllowStop:$AllowStop -ClearQmlCache:$ClearQmlCache

Write-Host ''
Write-Host 'RESTORE_OK'
Write-Host ("restored sha256: {0}" -f $result.RestoredSha256)
