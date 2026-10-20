@echo off
setlocal
set "EXE=%~dp0BluetoothPhoneAlerts.exe"
if not exist "%EXE%" (
  echo BluetoothPhoneAlerts.exe not found. Extract the Windows-EXE.zip first.
  pause
  exit /b 1
)
set "TARGET=%LOCALAPPDATA%\BluetoothPhoneAlerts"
if not exist "%TARGET%" mkdir "%TARGET%"
copy /Y "%EXE%" "%TARGET%\BluetoothPhoneAlerts.exe" >nul
if errorlevel 1 (
  echo Installation failed. Please close running copies and try again.
  pause
  exit /b 1
)
powershell -NoProfile -Command "$link = (New-Object -ComObject WScript.Shell).CreateShortcut([System.IO.Path]::Combine([Environment]::GetFolderPath('Startup'), 'BluetoothPhoneAlerts.lnk')); $link.TargetPath = [System.IO.Path]::Combine($env:LOCALAPPDATA, 'BluetoothPhoneAlerts', 'BluetoothPhoneAlerts.exe'); $link.Arguments = '--background'; $link.WorkingDirectory = [System.IO.Path]::Combine($env:LOCALAPPDATA, 'BluetoothPhoneAlerts'); $link.Save()"
if errorlevel 1 (
  echo Could not create auto-start shortcut. You can still launch EXE manually.
)
echo Installed to %TARGET%
echo Windows startup enabled. Pair phone and start the Android service first.
start "" "%TARGET%\BluetoothPhoneAlerts.exe"
endlocal
