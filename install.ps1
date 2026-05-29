# gp-cli Windows installer (PowerShell)
# Usage: irm https://raw.githubusercontent.com/noaa/patent-cli/main/install.ps1 | iex

$ErrorActionPreference = "Stop"

$Repo    = "noaa/patent-cli"
$Binary  = "gp-cli"
$Asset   = "gp-cli-windows-amd64.exe"
$InstDir = "$env:LOCALAPPDATA\gp-cli"

Write-Host ">> Fetching latest release..." -ForegroundColor Cyan

$Release = Invoke-RestMethod "https://api.github.com/repos/$Repo/releases/latest"
$Tag     = $Release.tag_name

if (-not $Tag) {
    Write-Error "Could not find latest release. Check https://github.com/$Repo/releases"
}

Write-Host ">> Installing $Binary $Tag for Windows/amd64" -ForegroundColor Cyan

$ZipName = "$Asset.zip"
$Url     = "https://github.com/$Repo/releases/download/$Tag/$ZipName"
$TmpDir  = [System.IO.Path]::GetTempPath() + [System.Guid]::NewGuid()
New-Item -ItemType Directory -Path $TmpDir | Out-Null

try {
    $ZipPath = "$TmpDir\$ZipName"
    Write-Host ">> Downloading $Url"
    Invoke-WebRequest -Uri $Url -OutFile $ZipPath -UseBasicParsing

    Expand-Archive -Path $ZipPath -DestinationPath $TmpDir -Force

    New-Item -ItemType Directory -Force -Path $InstDir | Out-Null
    Copy-Item "$TmpDir\$Asset" "$InstDir\$Binary.exe" -Force

    Write-Host ""
    Write-Host ">> Installed: $InstDir\$Binary.exe" -ForegroundColor Green
    Write-Host ""
} finally {
    Remove-Item -Recurse -Force $TmpDir -ErrorAction SilentlyContinue
}

# Add to user PATH if not already there
$UserPath = [Environment]::GetEnvironmentVariable("PATH", "User")
if ($UserPath -notlike "*$InstDir*") {
    [Environment]::SetEnvironmentVariable(
        "PATH",
        "$UserPath;$InstDir",
        "User"
    )
    Write-Host ">> Added $InstDir to your PATH." -ForegroundColor Yellow
    Write-Host ">> Please restart your terminal (or PowerShell) to use gp-cli." -ForegroundColor Yellow
} else {
    Write-Host ">> PATH already contains $InstDir" -ForegroundColor Gray
}

Write-Host ""
Write-Host ">> Done! Open a new terminal and run: $Binary --help" -ForegroundColor Green
