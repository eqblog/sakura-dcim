@echo off
chcp 65001 >nul
REM Sakura DCIM — One-click update

cd /d "%~dp0.."

echo =========================================
echo   Sakura DCIM — Updating
echo =========================================

echo [1/4] Pulling latest code...
git pull

echo [2/4] Updating backend dependencies...
cd backend && go mod tidy && cd ..

echo [3/4] Updating frontend dependencies...
cd web && call npm install && cd ..

echo [4/4] Running database migrations...
cd backend
go run -tags "postgres" github.com/golang-migrate/migrate/v4/cmd/migrate@latest -path migrations -database "postgres://sakura:sakura@localhost:5432/sakura_dcim?sslmode=disable" up 2>nul
cd ..

echo.
echo   Update complete! Now run:
echo.
echo   Development:  scripts\start-dev.bat
echo   Production:   scripts\start-prod.bat
echo =========================================
