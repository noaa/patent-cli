# gp-cli Windows uninstaller (PowerShell)
# Usage: irm https://raw.githubusercontent.com/noaa/patent-cli/main/uninstall.ps1 | iex

$ErrorActionPreference = "Stop"

$Binary    = "gp-cli"
$InstDir   = "$env:LOCALAPPDATA\gp-cli"
$ConfigDir = "$env:APPDATA\patent-cli"
$BinaryPath = "$InstDir\$Binary.exe"
$Removed   = $false

if (Test-Path $BinaryPath) {
    Remove-Item -Force $BinaryPath
    Write-Host ">> Removed: $BinaryPath" -ForegroundColor Green
    $Removed = $true
} else {
    Write-Host ">> Binary not found: $BinaryPath" -ForegroundColor Gray
}

if (Test-Path $InstDir) {
    $remaining = Get-ChildItem $InstDir -ErrorAction SilentlyContinue
    if (-not $remaining) {
        Remove-Item -Recurse -Force $InstDir
        Write-Host ">> Removed dir: $InstDir" -ForegroundColor Green
        $Removed = $true
    }
}

if (Test-Path $ConfigDir) {
    Remove-Item -Recurse -Force $ConfigDir
    Write-Host ">> Removed config: $ConfigDir" -ForegroundColor Green
    $Removed = $true
} else {
    Write-Host ">> Config dir not found: $ConfigDir" -ForegroundColor Gray
}

# Remove from user PATH
$UserPath = [Environment]::GetEnvironmentVariable("PATH", "User")
if ($UserPath -like "*$InstDir*") {
    $NewPath = ($UserPath -split ";" | Where-Object { $_ -ne $InstDir }) -join ";"
    [Environment]::SetEnvironmentVariable("PATH", $NewPath, "User")
    Write-Host ">> Removed $InstDir from PATH." -ForegroundColor Yellow
    Write-Host ">> Please restart your terminal for PATH changes to take effect." -ForegroundColor Yellow
    $Removed = $true
}

Write-Host ""
if ($Removed) {
    Write-Host ">> gp-cli uninstalled." -ForegroundColor Green
} else {
    Write-Host ">> Nothing to remove." -ForegroundColor Gray
}
