$ErrorActionPreference = "Stop"

$processes = Get-Process -Name "privacy-screen" -ErrorAction SilentlyContinue

if (-not $processes) {
    Write-Host "privacy-screen.exe is not running"
    exit 1
}

$processes | Stop-Process -Force
Write-Host "privacy-screen.exe stopped"
