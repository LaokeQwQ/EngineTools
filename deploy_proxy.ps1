# Deploy the translation proxy to an Engine DJ installation transactionally.
param(
    [string]$EngineDirectory = 'C:\Program Files\Engine DJ',
    [string]$ProxyDllPath = (Join-Path $PSScriptRoot 'proxy\build\Release\version.dll'),
    [switch]$AllowStop,
    [switch]$ClearQmlCache
)

$ErrorActionPreference = 'Stop'
Import-Module (Join-Path $PSScriptRoot 'deploy_proxy.psm1') -Force

$result = Install-EngineProxy -EngineDirectory $EngineDirectory `
    -ProxyDllPath $ProxyDllPath -AllowStop:$AllowStop -ClearQmlCache:$ClearQmlCache

Write-Host ''
Write-Host '=== Deploy complete ==='
Write-Host ("proxy   : {0}" -f $result.ProxySha256)
Write-Host ("original: {0}" -f $result.OriginalSha256)
Write-Host 'Launch Engine DJ to test.'
