@echo off
cd /d "%~dp0BluetoothPhoneAlerts.Win"
where dotnet >nul 2>&1
if errorlevel 1 (
    echo Please install the .NET 8 SDK: https://dotnet.microsoft.com/download/dotnet/8.0
    pause
    exit /b 1
)
dotnet publish -c Release -r win-x64 --self-contained true -p:PublishSingleFile=true -p:PublishReadyToRun=false
if errorlevel 1 (
    echo Publish failed. Check the error messages above.
    pause
    exit /b 1
)
echo.
echo App folder: bin\Release\net8.0-windows\win-x64\publish\
pause
