@echo off
setlocal
cd /d "%~dp0"

if not exist "privacy-screen.exe" (
    echo privacy-screen.exe was not found.
    echo Please run build.bat first.
    pause
    exit /b 1
)

"%~dp0privacy-screen.exe" off
pause
