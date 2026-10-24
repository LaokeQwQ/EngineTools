# Reproducible TagLib 2.0.2 + enginetools_tag.dll build.
#
# Downloads the pinned upstream archive, verifies the recorded SHA-256,
# builds TagLib and the bridge DLL with CMake, then installs both into
# build/taglib/install for the Go runtime loader to pick up.
$ErrorActionPreference = 'Stop'

$RepoRoot   = (Resolve-Path (Join-Path $PSScriptRoot '..' '..')).Path
$Pin        = Get-Content (Join-Path $RepoRoot 'build' 'taglib' 'taglib-pin.json') -Raw | ConvertFrom-Json
$UtfPin     = Get-Content (Join-Path $RepoRoot 'build' 'taglib' 'utfcpp-pin.json') -Raw | ConvertFrom-Json
$OutDir     = Join-Path $RepoRoot 'build' 'taglib'
$Tarball    = Join-Path $OutDir ("taglib-" + $Pin.version + ".tar.gz")
$SrcDir     = Join-Path $OutDir ("taglib-" + $Pin.version)
$UtfTarball = Join-Path $OutDir ("utfcpp-" + $UtfPin.version + ".tar.gz")
$UtfSrcDir  = Join-Path $OutDir ("utfcpp-" + $UtfPin.version)
$InstallDir = Join-Path $OutDir 'install'
$TagBuild   = Join-Path $OutDir 'taglib-build'
$BrBuild    = Join-Path $OutDir 'bridge-build'

New-Item -ItemType Directory -Force -Path $OutDir | Out-Null

function Get-PinnedArchive([string]$Url, [string]$Destination, [string]$Sha256, [string]$Label) {
    if (-not (Test-Path $Destination)) {
        Invoke-WebRequest -Uri $Url -OutFile $Destination
    }
    $hash = (Get-FileHash -LiteralPath $Destination -Algorithm SHA256).Hash
    if ($hash -ne $Sha256) { throw "$Label archive checksum mismatch" }
}

Get-PinnedArchive $Pin.url $Tarball $Pin.sha256 'TagLib'
Get-PinnedArchive $UtfPin.url $UtfTarball $UtfPin.sha256 'utfcpp'

if (-not (Test-Path $SrcDir)) {
    tar -xzf $Tarball -C $OutDir
    if ($LASTEXITCODE -ne 0) { throw 'TagLib archive extraction failed' }
}
if (-not (Test-Path $UtfSrcDir)) {
    tar -xzf $UtfTarball -C $OutDir
    if ($LASTEXITCODE -ne 0) { throw 'utfcpp archive extraction failed' }
}

# TagLib's CMake looks for a bundled 3rdparty/utfcpp submodule. The release
# tarball omits it, so wire the pinned copy in before configuring.
$UtfVendor = Join-Path $SrcDir '3rdparty' 'utfcpp'
if (-not (Test-Path $UtfVendor)) {
    New-Item -ItemType Directory -Force -Path (Split-Path $UtfVendor) | Out-Null
    Copy-Item -LiteralPath $UtfSrcDir -Destination $UtfVendor -Recurse
}

# The bridge must always consume the pinned install tree.  A package from a
# developer machine (or a runner image) could otherwise win find_package and
# silently change the ABI used by the release DLL.
$tagLibConfig = Join-Path $InstallDir 'lib\cmake\taglib\taglib-config.cmake'
$tagLibDll = Join-Path $InstallDir 'bin\tag.dll'
$tagLibImport = Join-Path $InstallDir 'lib\tag.lib'

# Warm CMake package discovery with the pinned utfcpp so TagLib never falls
# back to a machine-wide copy (or fails when none is installed).
$utfCmake = Join-Path $UtfSrcDir 'CMakeLists.txt'
if (-not (Test-Path $utfCmake)) { throw "pinned utfcpp is incomplete: $utfCmake" }

cmake -S $SrcDir -B $TagBuild -DBUILD_SHARED_LIBS=ON -DBUILD_TESTING=OFF `
    "-DCMAKE_INSTALL_PREFIX=$InstallDir" "-DCMAKE_PREFIX_PATH=$InstallDir" `
    "-Dutf8cpp_INCLUDE_DIR=$UtfSrcDir\source"
if ($LASTEXITCODE -ne 0) { throw 'TagLib configure failed' }
cmake --build $TagBuild --config Release --target install
if ($LASTEXITCODE -ne 0) { throw 'TagLib build failed' }
if (-not (Test-Path $tagLibDll)) { throw "TagLib runtime DLL was not installed: $tagLibDll" }
if (-not (Test-Path $tagLibImport)) { throw "TagLib import library was not installed: $tagLibImport" }
if (-not (Test-Path $tagLibConfig)) { throw "TagLib CMake package was not installed: $tagLibConfig" }

$BridgeSrc = Join-Path $RepoRoot 'native' 'taglibbridge'
cmake -S $BridgeSrc -B $BrBuild -DBUILD_TESTING=ON "-DTAGLIB_ROOT=$InstallDir" "-DCMAKE_PREFIX_PATH=$InstallDir"
if ($LASTEXITCODE -ne 0) { throw 'Bridge configure failed' }
cmake --build $BrBuild --config Release
if ($LASTEXITCODE -ne 0) { throw 'Bridge build failed' }
ctest --test-dir $BrBuild -C Release --output-on-failure
if ($LASTEXITCODE -ne 0) { throw 'Bridge tests failed' }

$BinDir = Join-Path $RepoRoot 'build' 'bin'
New-Item -ItemType Directory -Force -Path $BinDir | Out-Null
$bridgeDll = Get-ChildItem -LiteralPath $BrBuild -Filter 'enginetools_tag.dll' -File -Recurse |
    Where-Object { $_.FullName -notmatch '[\\/]CMakeFiles[\\/]' } |
    Select-Object -First 1
if (-not $bridgeDll) { throw "Bridge runtime DLL was not produced under $BrBuild" }

# Both DLLs are required at runtime and must sit beside the Wails executable;
# Windows does not search build/taglib/install when resolving dependencies.
Copy-Item -LiteralPath $bridgeDll.FullName -Destination (Join-Path $BinDir 'enginetools_tag.dll') -Force
Copy-Item -LiteralPath $tagLibDll -Destination (Join-Path $BinDir 'tag.dll') -Force
Write-Host 'Built enginetools_tag.dll and tag.dll'
