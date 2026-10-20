@echo off
powershell.exe -NoProfile -ExecutionPolicy Bypass -File "%~dp0setup-wrapper.ps1"
if errorlevel 1 echo Download failed. Check internet access and try again.
pause
