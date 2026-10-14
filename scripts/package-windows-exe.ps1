# Package the production Wails Windows binary as a standalone release EXE.
# The historical scripts/package-windows.ps1 keeps the v0.0.1 ZIP recipe.
[CmdletBinding()]
param(
    [Parameter(Mandatory = $true)]
    [ValidatePattern('^v[0-9]+\.[0-9]+\.[0-9]+$')]
    [string]$Version,

    [string]$OutputDir = 'dist'
)

$ErrorActionPreference = 'Stop'
Set-StrictMode -Version Latest

$root = (Resolve-Path (Join-Path $PSScriptRoot '..')).Path
$manifest = Get-Content (Join-Path $root 'frontend/package.json') -Raw | ConvertFrom-Json
if ($Version -ne "v$($manifest.version)") {
    throw "Version $Version disagrees with frontend/package.json ($($manifest.version))"
}
$config = Get-Content (Join-Path $root 'build/config.yml') -Raw
if ($config -notmatch "(?m)^  version: `"$([regex]::Escape($manifest.version))`"") {
    throw 'build/config.yml version disagrees with frontend/package.json'
}

$binary = Join-Path $root 'bin/diggerterm.exe'
if (!(Test-Path -LiteralPath $binary -PathType Leaf)) {
    throw "Missing Windows binary: $binary. Build it with wails3 build first."
}

# Confirm a genuine Windows x64 PE, rather than an unrelated ELF or stub.
$stream = [IO.File]::OpenRead($binary)
try {
    $reader = [IO.BinaryReader]::new($stream)
    if ($reader.ReadUInt16() -ne 0x5A4D) { throw 'Not a Windows MZ binary' }
    $stream.Position = 0x3C
    $peOffset = $reader.ReadInt32()
    if ($peOffset -lt 64 -or $peOffset -gt $stream.Length - 6) {
        throw 'Invalid Windows PE header offset'
    }
    $stream.Position = $peOffset
    if ($reader.ReadUInt32() -ne 0x00004550) { throw 'Not a Windows PE binary' }
    if ($reader.ReadUInt16() -ne 0x8664) { throw 'Binary is not Windows x64 (AMD64)' }
} finally {
    $stream.Dispose()
}

$folder = if ([IO.Path]::IsPathRooted($OutputDir)) {
    $OutputDir
} else {
    Join-Path $root $OutputDir
}
New-Item -ItemType Directory -Path $folder -Force | Out-Null
$destination = Join-Path $folder "DiggerTerm-$Version-windows-amd64.exe"
Copy-Item -LiteralPath $binary -Destination $destination -Force
$info = Get-Item -LiteralPath $destination
if ($info.Length -lt 100KB) { throw "Suspiciously small Windows EXE: $($info.Length) bytes" }

Write-Host "Windows release EXE: $destination"
Write-Host "SHA256: $((Get-FileHash -LiteralPath $destination -Algorithm SHA256).Hash.ToLowerInvariant())"
