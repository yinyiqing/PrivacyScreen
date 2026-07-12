param(
    [string]$Output = "",
    [int]$Seconds = 10,
    [int]$Fps = 15,
    [int]$MonitorIndex = -1
)

$ErrorActionPreference = "Stop"

$scriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
$projectRoot = Split-Path -Parent $scriptDir

if (-not $Output) {
    $stamp = Get-Date -Format "yyyyMMdd-HHmmss"
    $Output = Join-Path $projectRoot "recordings\d3d11-$stamp.mp4"
}

$GstRoot = "C:\Program Files\gstreamer\1.0\mingw_x86_64"
$GstLaunch = Join-Path $GstRoot "bin\gst-launch-1.0.exe"

if (-not (Test-Path $GstLaunch)) {
    throw "gst-launch-1.0.exe was not found at $GstLaunch"
}

$outputDir = Split-Path -Parent $Output
if ($outputDir) {
    New-Item -ItemType Directory -Path $outputDir -Force | Out-Null
}

Remove-Item Env:GST_PLUGIN_PATH -ErrorAction SilentlyContinue
$env:GST_PLUGIN_SYSTEM_PATH_1_0 = Join-Path $GstRoot "lib\gstreamer-1.0"
$env:GST_PLUGIN_SCANNER = Join-Path $GstRoot "libexec\gstreamer-1.0\gst-plugin-scanner.exe"
$env:GST_REGISTRY = Join-Path $outputDir "gst-registry-official.bin"
$env:PATH = "$GstRoot\bin;C:\Windows\system32;C:\Windows;$env:PATH"

$buffers = [Math]::Max(1, $Seconds * $Fps)
$gstOutput = $Output -replace "\\", "/"

& $GstLaunch -e `
    d3d11screencapturesrc monitor-index=$MonitorIndex num-buffers=$buffers show-cursor=true `
    ! "video/x-raw(memory:D3D11Memory),framerate=$Fps/1" `
    ! d3d11download `
    ! videoconvert `
    ! "video/x-raw,format=I420" `
    ! x264enc tune=zerolatency speed-preset=ultrafast bitrate=2000 key-int-max=$Fps `
    ! h264parse `
    ! mp4mux faststart=true `
    ! filesink location="$gstOutput"

if ($LASTEXITCODE -ne 0) {
    throw "GStreamer recording failed with exit code $LASTEXITCODE"
}

Get-Item -LiteralPath $Output
