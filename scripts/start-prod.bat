@echo off
chcp 65001 >nul
REM Sakura DCIM — One-click production start

cd /d "%~dp0.."

echo =========================================
echo   Sakura DCIM — Production Deployment
echo =========================================

echo [1/2] Building and starting all services...
docker compose up -d --build

echo [2/2] Waiting for services to be healthy...
timeout /t 5 /nobreak >nul

echo.
echo   Web UI -^> http://localhost:3000
echo   API    -^> http://localhost:3000/api/v1
echo.
echo   Login: admin@sakura-dcim.local / admin123
echo =========================================
