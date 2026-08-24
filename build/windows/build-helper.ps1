# Build the one-shot elevated helper reproducibly.
# Output: build/bin/EngineTools.Helper.exe (installed next to the main EXE
# by the NSIS installer; wails.files carries it into the payload).
$ErrorActionPreference = 'Stop'

Push-Location (Join-Path $PSScriptRoot '..\..')
try {
    go build -trimpath -ldflags '-s -w' -o build\bin\EngineTools.Helper.exe .\cmd\enginetools-helper
    if ($LASTEXITCODE -ne 0) { throw "go build failed with $LASTEXITCODE" }
} finally {
    Pop-Location
}

Write-Host "Built build\bin\EngineTools.Helper.exe"
