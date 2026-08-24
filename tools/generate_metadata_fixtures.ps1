# Generates deterministic four-format metadata fixtures for the golden
# tests. Everything is derived from a fixed 0.5 s / 44.1 kHz mono 440 Hz
# tone with bitexact flags and fixed dates, so repeated runs produce
# byte-identical metadata regions.
#
# Usage: powershell -File tools/generate_metadata_fixtures.ps1 -OutDir <dir>
param(
    [Parameter(Mandatory)] [string] $OutDir
)

$ErrorActionPreference = 'Stop'

$ffmpeg = Get-Command ffmpeg -ErrorAction SilentlyContinue
if (-not $ffmpeg) { throw 'ffmpeg is required on PATH' }
$versionLine = (& ffmpeg -version | Select-Object -First 1)
Write-Host "fixture generator using: $versionLine"

New-Item -ItemType Directory -Force -Path $OutDir | Out-Null

$common = @(
    '-hide_banner', '-loglevel', 'error',
    '-flags', '+bitexact', '-fflags', '+bitexact',
    '-f', 'lavfi', '-i', 'sine=frequency=440:duration=0.5:sample_rate=44100',
    '-ac', '1', '-ar', '44100'
)
$meta = @(
    '-metadata', 'title=Fixture Track',
    '-metadata', 'artist=Fixture Artist',
    '-metadata', 'album=Golden Album',
    '-metadata', 'track=3/12',
    '-metadata', 'date=2026-08-16',
    '-metadata', 'genre=Test',
    '-metadata', 'comment=Generated fixture'
)

function Emit-MP3([string]$path, [string]$id3Version, [switch]$V1) {
    $args = @('-y') + $common + $meta
    if ($V1) {
        $args += @('-write_id3v1', '1', '-write_id3v2', '0')
    } else {
        $args += @('-id3v2_version', $id3Version)
    }
    $args += @('-codec:a', 'libmp3lame', '-b:a', '128k', $path)
    & ffmpeg @args
    if ($LASTEXITCODE -ne 0) { throw "ffmpeg failed for $path" }
}

function Emit-FLAC([string]$path) {
    $args = @('-y') + $common + $meta + @('-codec:a', 'flac', '-bits_per_raw_sample', '16', $path)
    & ffmpeg @args
    if ($LASTEXITCODE -ne 0) { throw "ffmpeg failed for $path" }
}

function Emit-WAV([string]$path) {
    $args = @('-y') + $common + $meta + @('-codec:a', 'pcm_s16le', $path)
    & ffmpeg @args
    if ($LASTEXITCODE -ne 0) { throw "ffmpeg failed for $path" }
}

function Emit-AIFF([string]$path) {
    $args = @('-y') + $common + $meta + @('-codec:a', 'pcm_s16be', $path)
    & ffmpeg @args
    if ($LASTEXITCODE -ne 0) { throw "ffmpeg failed for $path" }
}

Emit-MP3   (Join-Path $OutDir 'mp3-v23.mp3') '3'
Emit-MP3   (Join-Path $OutDir 'mp3-v24.mp3') '4'
Emit-MP3   (Join-Path $OutDir 'mp3-v1.mp3')  '' -V1
Emit-FLAC  (Join-Path $OutDir 'fixture.flac')
Emit-WAV   (Join-Path $OutDir 'fixture.wav')
Emit-AIFF  (Join-Path $OutDir 'fixture.aiff')

Write-Host 'fixtures generated:'
Get-ChildItem $OutDir | Format-Table Name, Length
