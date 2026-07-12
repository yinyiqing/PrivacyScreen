param(
    [string]$Output = ""
)

$ErrorActionPreference = "Stop"

$projectRoot = Split-Path -Parent $MyInvocation.MyCommand.Path
$source = Join-Path $projectRoot "privacy_screen.go"

if (-not (Test-Path $source)) {
    throw "privacy_screen.go was not found in $projectRoot"
}

if (-not $Output) {
    $Output = Join-Path $projectRoot "privacy-screen.exe"
}

go version | Out-Host
go build -o $Output $source

if ($LASTEXITCODE -ne 0) {
    throw "Go build failed with exit code $LASTEXITCODE"
}

$item = Get-Item -LiteralPath $Output
Write-Host "Built: $($item.FullName)"
Write-Host "Bytes: $($item.Length)"
