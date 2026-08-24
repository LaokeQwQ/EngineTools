# Reproducible TagLib 2.0.2 + enginetools_tag.dll build.
#
# Downloads the pinned upstream archive, verifies the recorded SHA-256,
# builds TagLib and the bridge DLL with CMake, then installs both into
# build/taglib/install for the Go runtime loader to pick up.
$ErrorActionPreference = 'Stop'

$RepoRoot   = (Resolve-Path (Join-Path $PSScriptRoot '..' '..')).Path
$Pin        = Get-Content (Join-Path $RepoRoot 'build' 'taglib' 'taglib-pin.json') -Raw | ConvertFrom-Json
$OutDir     = Join-Path $RepoRoot 'build' 'taglib'
$Tarball    = Join-Path $OutDir ("taglib-" + $Pin.version + ".tar.gz")
$SrcDir     = Join-Path $OutDir ("taglib-" + $Pin.version)
$InstallDir = Join-Path $OutDir 'install'
$TagBuild   = Join-Path $OutDir 'taglib-build'
$BrBuild    = Join-Path $OutDir 'bridge-build'

New-Item -ItemType Directory -Force -Path $OutDir | Out-Null

if (-not (Test-Path $Tarball)) {
    Invoke-WebRequest -Uri $Pin.url -OutFile $Tarball
}
$hash = (Get-FileHash -LiteralPath $Tarball -Algorithm SHA256).Hash
if ($hash -ne $Pin.sha256) { throw "TagLib archive checksum mismatch" }

if (-not (Test-Path $SrcDir)) { tar -xzf $Tarball -C $OutDir }

cmake -S $SrcDir -B $TagBuild -DBUILD_SHARED_LIBS=ON -DBUILD_TESTING=OFF "-DCMAKE_INSTALL_PREFIX=$InstallDir"
if ($LASTEXITCODE -ne 0) { throw 'TagLib configure failed' }
cmake --build $TagBuild --config Release --target install
if ($LASTEXITCODE -ne 0) { throw 'TagLib build failed' }

$BridgeSrc = Join-Path $RepoRoot 'native' 'taglibbridge'
cmake -S $BridgeSrc -B $BrBuild -DBUILD_TESTING=ON "-DTAGLIB_ROOT=$InstallDir"
if ($LASTEXITCODE -ne 0) { throw 'Bridge configure failed' }
cmake --build $BrBuild --config Release
if ($LASTEXITCODE -ne 0) { throw 'Bridge build failed' }
ctest --test-dir $BrBuild -C Release --output-on-failure
if ($LASTEXITCODE -ne 0) { throw 'Bridge tests failed' }

$BinDir = Join-Path $RepoRoot 'build' 'bin'
New-Item -ItemType Directory -Force -Path $BinDir | Out-Null
Copy-Item (Join-Path $BrBuild 'Release' 'enginetools_tag.dll') $BinDir -Force
Write-Host 'Built enginetools_tag.dll'
