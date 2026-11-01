# Run only on a clean Windows CI account: exercises installation without starting the app.
param([Parameter(Mandatory = $true)][string]$Installer)
$ErrorActionPreference = 'Stop'
$exe = (Resolve-Path -LiteralPath $Installer).Path
$folder = Join-Path $env:RUNNER_TEMP ('diggerterm-installer-' + [guid]::NewGuid().ToString('N'))
$root = 'HKCU:\Software\DiggerTerm'
$menu = 'HKCU:\Software\Classes\Directory\shell\DiggerTerm'
$background = 'HKCU:\Software\Classes\Directory\Background\shell\DiggerTerm'
$adminMenu = 'HKCU:\Software\Classes\Directory\shell\DiggerTermAdmin'
$adminBackground = 'HKCU:\Software\Classes\Directory\Background\shell\DiggerTermAdmin'
$shortcut = Join-Path ([Environment]::GetFolderPath('Programs')) 'DiggerTerm.lnk'
$adminShortcut = Join-Path ([Environment]::GetFolderPath('Programs')) 'DiggerTerm (Administrator).lnk'
$desktopShortcut = Join-Path ([Environment]::GetFolderPath('Desktop')) 'DiggerTerm.lnk'
if (!$env:CI -or !$env:RUNNER_TEMP -or (Test-Path $root) -or (Test-Path $menu) -or (Test-Path $background) -or (Test-Path $adminMenu) -or (Test-Path $adminBackground) -or (Test-Path $shortcut) -or (Test-Path $adminShortcut) -or (Test-Path $desktopShortcut)) {
    throw 'Installer test requires a clean CI account.'
}
$process = Start-Process -FilePath $exe -ArgumentList '/S', "/D=$folder" -Wait -PassThru
if ($process.ExitCode -ne 0) { throw "Installer failed: $($process.ExitCode)" }
if (!(Test-Path "$folder\diggerterm.exe") -or !(Test-Path $shortcut)) { throw 'Installed executable or Start menu shortcut missing.' }
if (!(Test-Path $adminShortcut)) { throw 'Administrator Start menu shortcut missing.' }
$link = (New-Object -ComObject WScript.Shell).CreateShortcut($adminShortcut)
if ($link.Arguments -ne '--elevate') { throw 'Administrator shortcut does not request elevation.' }
foreach ($entry in @(@($menu, '%1'), @($background, '%V'))) {
    $command = (Get-Item -LiteralPath "$($entry[0])\command").GetValue('')
    $expected = '"{0}\diggerterm.exe" --directory "{1}\."' -f $folder, $entry[1]
    if ($command -ne $expected) { throw "Unexpected Explorer command: $command" }
}
foreach ($entry in @(@($adminMenu, '%1'), @($adminBackground, '%V'))) {
    $command = (Get-Item -LiteralPath "$($entry[0])\command").GetValue('')
    $expected = '"{0}\diggerterm.exe" --elevate --directory "{1}\."' -f $folder, $entry[1]
    if ($command -ne $expected) { throw "Unexpected administrator Explorer command: $command" }
    if ($null -eq (Get-Item -LiteralPath $entry[0]).GetValue('HasLUAShield', $null)) { throw 'Administrator Explorer entry has no shield.' }
}
# Check that files the installer does not own survive uninstall.
Set-Content -LiteralPath "$folder\keep.txt" -Value 'preserve'
$process = Start-Process -FilePath "$folder\uninstall.exe" -ArgumentList '/S', "_?=$folder" -Wait -PassThru
if ($process.ExitCode -ne 0) { throw "Uninstaller failed: $($process.ExitCode)" }
if ((Test-Path "$folder\diggerterm.exe") -or (Test-Path $root) -or (Test-Path $menu) -or (Test-Path $background) -or (Test-Path $adminMenu) -or (Test-Path $adminBackground) -or (Test-Path $shortcut) -or (Test-Path $adminShortcut)) { throw 'Uninstall left an owned file or registration.' }
if (!(Test-Path "$folder\keep.txt")) { throw 'Uninstall removed an unowned file.' }
Remove-Item -LiteralPath "$folder\keep.txt" -Force
if (Test-Path "$folder\uninstall.exe") { Remove-Item -LiteralPath "$folder\uninstall.exe" -Force }
Remove-Item -LiteralPath $folder -Force
