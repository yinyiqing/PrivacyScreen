@echo off
setlocal
cd /d "%~dp0"

if not exist "privacy-screen.exe" (
    echo privacy-screen.exe was not found.
    echo Please run build.bat first.
    pause
    exit /b 1
)

echo Starting privacy screen.
echo It will close automatically after 5 seconds.
echo.
if exist "%~dp0privacy-screen.png" (
    echo Using image: %~dp0privacy-screen.png
    "%~dp0privacy-screen.exe" on --timeout 5 --click-through=false --hide-cursor=false --image "%~dp0privacy-screen.png" --image-mode stretch
) else (
    echo privacy-screen.png was not found. Using black screen.
    "%~dp0privacy-screen.exe" on --timeout 5 --click-through=false --hide-cursor=false
)
pause
