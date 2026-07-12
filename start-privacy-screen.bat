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
echo It will close automatically after 20 seconds.
echo.
"%~dp0privacy-screen.exe" on --timeout 20 --click-through=false --hide-cursor=false
pause
