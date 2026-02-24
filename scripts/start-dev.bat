@echo off
chcp 65001 >nul
REM Sakura DCIM — One-click development start

cd /d "%~dp0.."
set ROOT=%cd%

echo =========================================
echo   Sakura DCIM — Starting Development
echo =========================================

echo [1/5] Starting PostgreSQL, Redis...
docker compose up -d postgres redis
echo          Starting InfluxDB (optional, may be slow on first pull)...
docker compose up -d influxdb 2>nul || echo          InfluxDB skipped (pull failed, bandwidth monitoring disabled)

echo [2/5] Waiting for PostgreSQL to be ready...
:wait_pg
docker exec sakura-postgres pg_isready -U sakura -d sakura_dcim -q >nul 2>&1
if errorlevel 1 (
    timeout /t 1 /nobreak >nul
    goto wait_pg
)
echo          PostgreSQL is ready.

echo [3/5] Running database migrations...
cd /d "%ROOT%\backend"
go run -tags "postgres" github.com/golang-migrate/migrate/v4/cmd/migrate@latest -path migrations -database "postgres://sakura:sakura@localhost:5432/sakura_dcim?sslmode=disable" up 2>nul
cd /d "%ROOT%"

if not exist "%ROOT%\web\node_modules" (
    echo [4/5] Installing frontend dependencies...
    cd /d "%ROOT%\web" && call npm install
    cd /d "%ROOT%"
) else (
    echo [4/5] Frontend dependencies already installed.
)

echo [5/5] Starting backend and frontend...
echo.
echo   Backend  -^> http://localhost:8080
echo   Frontend -^> http://localhost:5173
echo.
echo   Login: admin@sakura-dcim.local / admin123
echo =========================================
echo.

start "Sakura Backend" cmd /k "cd /d %ROOT%\backend && go run ./cmd/server"
start "Sakura Frontend" cmd /k "cd /d %ROOT%\web && npm run dev"

echo Services started in separate windows. Close them to stop.
