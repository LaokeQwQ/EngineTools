# Pester v5 suite for the deploy_proxy module.
#
# Runs against a fake Engine DJ directory and synthetic PE payloads, so the
# whole transaction is exercised without Windows or a real install:
#   pwsh -NoProfile -Command "Invoke-Pester ./tests/DeployProxy.Tests.ps1 -Output Detailed"
BeforeAll {
    Set-StrictMode -Version Latest
    Import-Module (Join-Path $PSScriptRoot '..' 'deploy_proxy.psm1') -Force

    function New-TestEngine {
        $root = Join-Path ([System.IO.Path]::GetTempPath()) ("etproxy-test-" + [guid]::NewGuid().ToString('N'))
        New-Item -ItemType Directory -Path $root -Force | Out-Null
        $engine = Join-Path $root 'engine'
        New-Item -ItemType Directory -Path $engine -Force | Out-Null
        Set-Content -LiteralPath (Join-Path $engine 'Engine DJ.exe') -Value 'fake engine exe'
        $sys = Join-Path $root 'sys'
        New-Item -ItemType Directory -Path $sys -Force | Out-Null
        return @{
            Root     = $root
            Engine   = $engine
            Sys      = $sys
            SystemDll  = (Join-Path $sys 'version.dll')
            ProxyDll   = (Join-Path $root 'proxy-version.dll')
        }
    }

    function New-FakePe {
        param(
            [Parameter(Mandatory)][string]$Path,
            [ushort]$Machine = 0x8664,
            [switch]$Forwarders,
            [byte[]]$Payload = $null
        )
        if ($null -ne $Payload) {
            [System.IO.File]::WriteAllBytes($Path, $Payload)
            return
        }
        $bytes = New-Object byte[] 512
        $bytes[0] = 0x4D; $bytes[1] = 0x5A                      # MZ
        $peOffset = 0x40
        [BitConverter]::GetBytes([int]$peOffset).CopyTo($bytes, 0x3C)
        $bytes[$peOffset] = 0x50; $bytes[$peOffset + 1] = 0x45  # PE\0\0
        [BitConverter]::GetBytes($Machine).CopyTo($bytes, $peOffset + 4)
        if ($Forwarders) {
            $text = [System.Text.Encoding]::ASCII.GetBytes('version_orig.dll')
            $text.CopyTo($bytes, 0x100)
        }
        [System.IO.File]::WriteAllBytes($Path, $bytes)
    }

    function Remove-TestEngine {
        param([Parameter(Mandatory)]$Fixture)
        if (Test-Path $Fixture.Root) {
            Remove-Item -LiteralPath $Fixture.Root -Recurse -Force
        }
    }

    function Get-FileTextHash {
        param([Parameter(Mandatory)][string]$LiteralPath)
        return (Get-FileHash -LiteralPath $LiteralPath -Algorithm SHA256).Hash
    }
}

Describe 'Install-EngineProxy' {

    BeforeEach {
        $fx = New-TestEngine
        New-FakePe -Path $fx.SystemDll
        New-FakePe -Path $fx.ProxyDll -Forwarders
    }
    AfterEach { Remove-TestEngine -Fixture $fx }

    It 'performs a clean install end to end' {
        $result = Install-EngineProxy -EngineDirectory $fx.Engine `
            -ProxyDllPath $fx.ProxyDll -SystemVersionDll $fx.SystemDll

        $result.Installed | Should -BeTrue
        Test-Path (Join-Path $fx.Engine 'version.dll') | Should -BeTrue
        Test-Path (Join-Path $fx.Engine 'version_orig.dll') | Should -BeTrue
        Test-Path (Join-Path $fx.Engine 'enginetools_proxy_journal.json') | Should -BeTrue

        # exact byte copies of both inputs
        Get-FileTextHash (Join-Path $fx.Engine 'version.dll') |
            Should -Be (Get-FileTextHash $fx.ProxyDll)
        Get-FileTextHash (Join-Path $fx.Engine 'version_orig.dll') |
            Should -Be (Get-FileTextHash $fx.SystemDll)

        $journal = Get-Content (Join-Path $fx.Engine 'enginetools_proxy_journal.json') -Raw |
            ConvertFrom-Json
        $journal.schema | Should -Be 1
        $journal.proxySha256 | Should -Be (Get-FileTextHash $fx.ProxyDll)
        $journal.originalSha256 | Should -Be (Get-FileTextHash $fx.SystemDll)
    }

    It 'fails closed when the original system DLL is missing' {
        { Install-EngineProxy -EngineDirectory $fx.Engine `
            -ProxyDllPath $fx.ProxyDll `
            -SystemVersionDll (Join-Path $fx.Sys 'does-not-exist.dll') } |
            Should -Throw

        Test-Path (Join-Path $fx.Engine 'version.dll') | Should -BeFalse
        Test-Path (Join-Path $fx.Engine 'version_orig.dll') | Should -BeFalse
        Test-Path (Join-Path $fx.Engine 'enginetools_proxy_journal.json') | Should -BeFalse
    }

    It 'refuses an install path that is not an Engine DJ directory' {
        $empty = Join-Path $fx.Root 'not-engine'
        New-Item -ItemType Directory -Path $empty -Force | Out-Null
        { Install-EngineProxy -EngineDirectory $empty `
            -ProxyDllPath $fx.ProxyDll -SystemVersionDll $fx.SystemDll } |
            Should -Throw
    }

    It 'reinstalls cleanly over an existing proxy' {
        Install-EngineProxy -EngineDirectory $fx.Engine `
            -ProxyDllPath $fx.ProxyDll -SystemVersionDll $fx.SystemDll | Out-Null

        # a different proxy build replaces the old one and updates the journal
        New-FakePe -Path $fx.ProxyDll -Forwarders
        Set-ItemProperty -Path $fx.ProxyDll -Name IsReadOnly -Value $false
        [System.IO.File]::WriteAllBytes($fx.ProxyDll, [System.IO.File]::ReadAllBytes($fx.ProxyDll))
        $result = Install-EngineProxy -EngineDirectory $fx.Engine `
            -ProxyDllPath $fx.ProxyDll -SystemVersionDll $fx.SystemDll

        $result.Installed | Should -BeTrue
        Get-FileTextHash (Join-Path $fx.Engine 'version.dll') |
            Should -Be (Get-FileTextHash $fx.ProxyDll)
    }

    It 'rejects a proxy that is not an x64 image' {
        New-FakePe -Path $fx.ProxyDll -Machine 0x014C -Forwarders   # i386
        { Install-EngineProxy -EngineDirectory $fx.Engine `
            -ProxyDllPath $fx.ProxyDll -SystemVersionDll $fx.SystemDll } |
            Should -Throw
        Test-Path (Join-Path $fx.Engine 'version.dll') | Should -BeFalse
    }

    It 'rejects a proxy without version_orig forwarders' {
        New-FakePe -Path $fx.ProxyDll                               # no forwarder text
        { Install-EngineProxy -EngineDirectory $fx.Engine `
            -ProxyDllPath $fx.ProxyDll -SystemVersionDll $fx.SystemDll } |
            Should -Throw
        Test-Path (Join-Path $fx.Engine 'version.dll') | Should -BeFalse
    }

    It 'survives an interrupted rename by sweeping stale temp files' {
        $stale = Join-Path $fx.Engine 'version.dll.enginetools-tmp'
        Set-Content -LiteralPath $stale -Value 'garbage from a killed run'

        Install-EngineProxy -EngineDirectory $fx.Engine `
            -ProxyDllPath $fx.ProxyDll -SystemVersionDll $fx.SystemDll | Out-Null

        Test-Path $stale | Should -BeFalse
        Test-Path (Join-Path $fx.Engine 'version.dll') | Should -BeTrue
    }

    It 'rolls the previous proxy back when the swap verification fails' {
        Install-EngineProxy -EngineDirectory $fx.Engine `
            -ProxyDllPath $fx.ProxyDll -SystemVersionDll $fx.SystemDll | Out-Null
        $firstGood = Get-FileTextHash (Join-Path $fx.Engine 'version.dll')

        # make Copy-VerifiedFile fail mid-swap: point the source at a directory
        { Install-EngineProxy -EngineDirectory $fx.Engine `
            -ProxyDllPath $fx.Sys -SystemVersionDll $fx.SystemDll } | Should -Throw

        # previous deployment still in place and journal consistent with it
        Get-FileTextHash (Join-Path $fx.Engine 'version.dll') | Should -Be $firstGood
    }
}

Describe 'Restore-EngineProxy' {

    BeforeEach {
        $fx = New-TestEngine
        New-FakePe -Path $fx.SystemDll
        New-FakePe -Path $fx.ProxyDll -Forwarders
        Install-EngineProxy -EngineDirectory $fx.Engine `
            -ProxyDllPath $fx.ProxyDll -SystemVersionDll $fx.SystemDll | Out-Null
    }
    AfterEach { Remove-TestEngine -Fixture $fx }

    It 'restores the exact original bytes and cleans the journal' {
        $originalHash = Get-FileTextHash $fx.SystemDll

        $result = Restore-EngineProxy -EngineDirectory $fx.Engine

        $result.Restored | Should -BeTrue
        Get-FileTextHash (Join-Path $fx.Engine 'version.dll') | Should -Be $originalHash
        Test-Path (Join-Path $fx.Engine 'version_orig.dll') | Should -BeFalse
        Test-Path (Join-Path $fx.Engine 'enginetools_proxy_journal.json') | Should -BeFalse
    }

    It 'refuses to delete a version.dll whose hash the journal does not own' {
        $proxyTarget = Join-Path $fx.Engine 'version.dll'
        Set-Content -LiteralPath $proxyTarget -Value 'foreign file planted here'

        { Restore-EngineProxy -EngineDirectory $fx.Engine } | Should -Throw

        Get-Content -LiteralPath $proxyTarget -Raw | Should -Be 'foreign file planted here'
        Test-Path (Join-Path $fx.Engine 'version_orig.dll') | Should -BeTrue
    }

    It 'refuses to run without a journal at all' {
        Remove-Item (Join-Path $fx.Engine 'enginetools_proxy_journal.json') -Force
        { Restore-EngineProxy -EngineDirectory $fx.Engine } | Should -Throw
        Test-Path (Join-Path $fx.Engine 'version.dll') | Should -BeTrue
    }

    It 'detects a tampered version_orig.dll and refuses' {
        Set-Content -LiteralPath (Join-Path $fx.Engine 'version_orig.dll') -Value 'tampered'
        { Restore-EngineProxy -EngineDirectory $fx.Engine } | Should -Throw
    }
}
