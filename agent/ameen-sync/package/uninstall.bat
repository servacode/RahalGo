@echo off
chcp 65001 >nul
schtasks /Delete /TN "RahalGoSync" /F
taskkill /IM RahalGoSync.exe /F >nul 2>&1
echo تم: البرنامج انشال من التشغيل التلقائي
pause
