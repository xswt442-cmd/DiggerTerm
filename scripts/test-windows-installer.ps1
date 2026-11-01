# Run only on a clean Windows CI account: exercises installation without starting the app.
param([Parameter(Mandatory = $true)][string]$Installer)
$ErrorActionPreference = 'Stop'
$exe = (Resolve-Path -LiteralPath $Installer).Path
$folder = Join-Path $env:RUNNER_TEMP ('diggerterm-installer-' + [guid]::NewGuid().ToString('N'))
$root = 'HKCU:\Software\DiggerTerm'
$menu = 'HKCU:\Software\Classes\Directory\shell\DiggerTerm'
$background = 'HKCU:\Software\Classes\Directory\Background\shell\DiggerTerm'
$shortcut = Join-Path ([Environment]::GetFolderPath('Programs')) 'DiggerTerm.lnk'
$desktopShortcut = Join-Path ([Environment]::GetFolderPath('Desktop')) 'DiggerTerm.lnk'
if (!$env:CI -or !$env:RUNNER_TEMP -or (Test-Path $root) -or (Test-Path $menu) -or (Test-Path $background) -or (Test-Path $shortcut) -or (Test-Path $desktopShortcut)) {
    throw 'Installer test requires a clean CI account.'
}
$process = Start-Process -FilePath $exe -ArgumentList '/S', "/D=$folder" -Wait -PassThru
if ($process.ExitCode -ne 0) { throw "Installer failed: $($process.ExitCode)" }
if (!(Test-Path "$folder\diggerterm.exe") -or !(Test-Path $shortcut)) { throw 'Installed executable or Start menu shortcut missing.' }
foreach ($entry in @(@($menu, '%1'), @($background, '%V'))) {
    $command = (Get-Item -LiteralPath "$($entry[0])\command").GetValue('')
    $expected = '"{0}\diggerterm.exe" --directory "{1}\."' -f $folder, $entry[1]
    if ($command -ne $expected) { throw "Unexpected Explorer command: $command" }
}
# Check that files the installer does not own survive uninstall.
Set-Content -LiteralPath "$folder\keep.txt" -Value 'preserve'
$process = Start-Process -FilePath "$folder\uninstall.exe" -ArgumentList '/S', "_?=$folder" -Wait -PassThru
if ($process.ExitCode -ne 0) { throw "Uninstaller failed: $($process.ExitCode)" }
if ((Test-Path "$folder\diggerterm.exe") -or (Test-Path $root) -or (Test-Path $menu) -or (Test-Path $background) -or (Test-Path $shortcut)) { throw 'Uninstall left an owned file or registration.' }
if (!(Test-Path "$folder\keep.txt")) { throw 'Uninstall removed an unowned file.' }
Remove-Item -LiteralPath "$folder\keep.txt" -Force
if (Test-Path "$folder\uninstall.exe") { Remove-Item -LiteralPath "$folder\uninstall.exe" -Force }
Remove-Item -LiteralPath $folder -Force
