# Download unmodified official Gradle 8.9 wrapper bootstrap files.
# Source: https://github.com/gradle/gradle/tree/v8.9.0
$ErrorActionPreference = 'Stop'
$root = Split-Path -Parent $MyInvocation.MyCommand.Path
$base = 'https://raw.githubusercontent.com/gradle/gradle/v8.9.0'
$files = @(
  @('gradlew', 'gradlew'),
  @('gradlew.bat', 'gradlew.bat'),
  @('gradle/wrapper/gradle-wrapper.jar', 'gradle/wrapper/gradle-wrapper.jar')
)
foreach ($file in $files) {
  $destination = Join-Path $root $file[1]
  New-Item -ItemType Directory -Path (Split-Path $destination -Parent) -Force | Out-Null
  Write-Host "Downloading $($file[0])..."
  Invoke-WebRequest -Uri "$base/$($file[0])" -OutFile $destination -UseBasicParsing
}
Write-Host 'Official Gradle Wrapper downloaded. Now open this android folder in Android Studio.'
