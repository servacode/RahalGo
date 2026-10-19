@echo off
chcp 65001 >nul
cd /d "%~dp0"
if not exist config.json (
  echo لازم تعمل config.json اول - انسخ config.example.json وعدّل الرمز
  pause
  exit /b 1
)
schtasks /Create /TN "RahalGoSync" /TR "\"%~dp0RahalGoSync.exe\"" /SC ONLOGON /RL LIMITED /F
start "" "%~dp0RahalGoSync.exe"
echo تم: البرنامج شغال وبيقلع لحاله مع الكمبيوتر. السجل بملف sync.log
pause
