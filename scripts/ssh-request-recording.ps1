param(
    [string]$Output = "",
    [int]$Seconds = 10,
    [int]$Fps = 15,
    [string]$QueueDir = "",
    [int]$TimeoutSeconds = 120
)

$ErrorActionPreference = "Stop"

$scriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
$projectRoot = Split-Path -Parent $scriptDir

if (-not $QueueDir) {
    $QueueDir = Join-Path $projectRoot "runtime\ssh-record\queue"
}

New-Item -ItemType Directory -Path $QueueDir -Force | Out-Null

if (-not $Output) {
    $stamp = Get-Date -Format "yyyyMMdd-HHmmss"
    $Output = Join-Path $projectRoot "recordings\ssh-d3d11-$stamp.mp4"
}

$id = [guid]::NewGuid().ToString("N")
$request = [ordered]@{
    id = $id
    output = $Output
    seconds = $Seconds
    fps = $Fps
}

$tmp = Join-Path $QueueDir "$id.tmp"
$json = Join-Path $QueueDir "$id.json"
$done = Join-Path $QueueDir "$id.done.txt"
$errorFile = Join-Path $QueueDir "$id.error.txt"

($request | ConvertTo-Json -Compress) | Set-Content -LiteralPath $tmp -Encoding UTF8
Move-Item -LiteralPath $tmp -Destination $json -Force

$deadline = (Get-Date).AddSeconds($TimeoutSeconds)
while ((Get-Date) -lt $deadline) {
    if (Test-Path -LiteralPath $done) {
        Get-Content -LiteralPath $done
        Remove-Item -LiteralPath $done -Force -ErrorAction SilentlyContinue
        exit 0
    }

    if (Test-Path -LiteralPath $errorFile) {
        $message = Get-Content -LiteralPath $errorFile -Raw
        Remove-Item -LiteralPath $errorFile -Force -ErrorAction SilentlyContinue
        throw $message
    }

    Start-Sleep -Seconds 1
}

throw "Timed out waiting for recording agent. Start record-agent.ps1 in the PVE/console session first."
