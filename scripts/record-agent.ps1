param(
    [string]$QueueDir = "",
    [int]$PollMilliseconds = 500,
    [switch]$Once
)

$ErrorActionPreference = "Stop"

$scriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
$projectRoot = Split-Path -Parent $scriptDir
$recordScript = Join-Path $scriptDir "record-d3d11-mp4.ps1"

if (-not $QueueDir) {
    $QueueDir = Join-Path $projectRoot "runtime\ssh-record\queue"
}

if (-not (Test-Path $recordScript)) {
    throw "record-d3d11-mp4.ps1 was not found next to this script"
}

New-Item -ItemType Directory -Path $QueueDir -Force | Out-Null
Write-Host "Recording agent is watching: $QueueDir"

while ($true) {
    $request = Get-ChildItem -LiteralPath $QueueDir -Filter "*.json" -File -ErrorAction SilentlyContinue |
        Sort-Object CreationTime |
        Select-Object -First 1

    if (-not $request) {
        Start-Sleep -Milliseconds $PollMilliseconds
        continue
    }

    $runningPath = "$($request.FullName).running"
    Move-Item -LiteralPath $request.FullName -Destination $runningPath -Force

    try {
        $body = Get-Content -LiteralPath $runningPath -Raw | ConvertFrom-Json
        $id = [string]$body.id
        $output = [string]$body.output
        $seconds = [int]$body.seconds
        $fps = [int]$body.fps

        if (-not $id) { throw "request id is empty" }
        if (-not $output) { throw "output path is empty" }
        if ($seconds -le 0) { throw "seconds must be greater than 0" }
        if ($fps -le 0) { throw "fps must be greater than 0" }

        $log = & powershell -ExecutionPolicy Bypass -File $recordScript -Output $output -Seconds $seconds -Fps $fps 2>&1
        if ($LASTEXITCODE -ne 0) {
            throw ($log | Out-String)
        }

        $item = Get-Item -LiteralPath $output
        $done = @(
            "OK"
            "Output=$($item.FullName)"
            "Bytes=$($item.Length)"
            "LastWriteTime=$($item.LastWriteTime.ToString('s'))"
        ) -join [Environment]::NewLine
        Set-Content -LiteralPath (Join-Path $QueueDir "$id.done.txt") -Value $done -Encoding UTF8
    }
    catch {
        $fallbackId = if ($id) { $id } else { [IO.Path]::GetFileNameWithoutExtension($runningPath) }
        Set-Content -LiteralPath (Join-Path $QueueDir "$fallbackId.error.txt") -Value $_.Exception.Message -Encoding UTF8
    }
    finally {
        Remove-Item -LiteralPath $runningPath -Force -ErrorAction SilentlyContinue
    }

    if ($Once) {
        break
    }
}
