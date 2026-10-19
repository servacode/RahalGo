@echo off
chcp 65001 >nul
cd /d "%~dp0"
RahalGoSync-console.exe tables > tables.txt
echo الجداول انحفظت بملف tables.txt
pause
