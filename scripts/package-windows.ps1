# Portable Windows release archive shared by local checks and GitHub Actions.
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

$executable = Join-Path $root 'bin/diggerterm.exe'
if (!(Test-Path $executable)) { throw "Missing executable: $executable. Run wails3 build first." }
$fullOutput = if ([IO.Path]::IsPathRooted($OutputDir)) { $OutputDir } else { Join-Path $root $OutputDir }
New-Item -ItemType Directory -Force -Path $fullOutput | Out-Null
$archiveName = "DiggerTerm-$Version-windows-amd64.zip"
$archive = Join-Path $fullOutput $archiveName
$checksum = "$archive.sha256"
$stage = Join-Path ([IO.Path]::GetTempPath()) ("diggerterm-release-" + [guid]::NewGuid().ToString("N"))

try {
    New-Item -ItemType Directory -Force -Path $stage | Out-Null
    Copy-Item -LiteralPath $executable -Destination $stage
    foreach ($name in @('README.md', 'README.en.md', 'CHANGELOG.md', 'LICENSE')) {
        Copy-Item -LiteralPath (Join-Path $root $name) -Destination $stage
    }
    # Keep the README logo available when opening the Markdown from the ZIP.
    New-Item -ItemType Directory -Path (Join-Path $stage 'icons') | Out-Null
    Copy-Item -LiteralPath (Join-Path $root 'icons/diggerterm-icon.svg') -Destination (Join-Path $stage 'icons')
    if (Test-Path $archive) { Remove-Item $archive -Force }
    Compress-Archive -LiteralPath (Get-ChildItem $stage | ForEach-Object FullName) -DestinationPath $archive -CompressionLevel Optimal

    Add-Type -AssemblyName System.IO.Compression.FileSystem
    $zip = [IO.Compression.ZipFile]::OpenRead($archive)
    try {
        $actual = @($zip.Entries | Where-Object { $_.Name -ne '' } | ForEach-Object { $_.FullName.Replace('\', '/') } | Sort-Object)
        $expected = @(@('diggerterm.exe', 'README.md', 'README.en.md', 'CHANGELOG.md', 'LICENSE', 'icons/diggerterm-icon.svg') | Sort-Object)
        if (($actual -join '|') -ne ($expected -join '|')) {
            throw "Unexpected archive content: $($actual -join ', ')"
        }
    } finally {
        $zip.Dispose()
    }

    $sha = (Get-FileHash -LiteralPath $archive -Algorithm SHA256).Hash.ToLowerInvariant()
    "$sha  $archiveName" | Set-Content -LiteralPath $checksum -Encoding Ascii
    Write-Host "Archive: $archive"
    Write-Host "SHA256: $sha"
} finally {
    Remove-Item $stage -Recurse -Force -ErrorAction SilentlyContinue
}
