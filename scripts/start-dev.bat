@echo off
chcp 65001 >nul
REM Sakura DCIM — One-click development start (Windows)
REM
REM The agent runs inside a Docker container with access to ipmitool,
REM dmidecode, and all other Linux tools needed for IPMI/KVM/PXE.
REM
REM Usage:
REM   scripts\start-dev.bat
REM   set BACKEND_PORT=9090&& scripts\start-dev.bat

cd /d "%~dp0.."
set ROOT=%cd%

if "%BACKEND_PORT%"=="" set BACKEND_PORT=8080

echo =========================================
echo   Sakura DCIM — Starting Development
echo =========================================

REM ─── 1. Start infrastructure ───
echo.
echo [1/8] Starting PostgreSQL, Redis, InfluxDB...
docker compose up -d postgres redis
docker compose up -d influxdb 2>nul || echo          InfluxDB skipped.

REM ─── 2. Wait for PostgreSQL ───
echo [2/8] Waiting for PostgreSQL...
:wait_pg
docker exec sakura-postgres pg_isready -U sakura -d sakura_dcim -q >nul 2>&1
if errorlevel 1 (
    timeout /t 1 /nobreak >nul
    goto wait_pg
)
echo          PostgreSQL ready.

REM ─── 3. Run migrations ───
echo [3/8] Running database migrations...
cd /d "%ROOT%\backend"
go run -tags "postgres" github.com/golang-migrate/migrate/v4/cmd/migrate@latest -path migrations -database "postgres://sakura:sakura@localhost:5432/sakura_dcim?sslmode=disable" up 2>nul
cd /d "%ROOT%"

REM ─── 4. Install frontend deps ───
if not exist "%ROOT%\web\node_modules" (
    echo [4/8] Installing frontend dependencies...
    cd /d "%ROOT%\web" && call npm install
    cd /d "%ROOT%"
) else (
    echo [4/8] Frontend dependencies OK.
)

REM ─── 5. Start backend ───
echo [5/8] Starting backend on port %BACKEND_PORT%...
start "Sakura Backend" cmd /k "cd /d %ROOT%\backend && go run ./cmd/server"

echo          Waiting for backend...
:wait_backend
timeout /t 1 /nobreak >nul
curl -s "http://127.0.0.1:%BACKEND_PORT%/health" >nul 2>&1
if errorlevel 1 goto wait_backend
echo          Backend ready.

REM ─── 6. Build KVM Docker image ───
echo [6/8] Building KVM browser image...
docker images --format "{{.Repository}}:{{.Tag}}" 2>nul | findstr /c:"sakura-dcim/kvm-browser:latest" >nul 2>&1
if errorlevel 1 (
    docker build -t sakura-dcim/kvm-browser:latest "%ROOT%\docker\kvm-browser\"
    echo          KVM image built.
) else (
    echo          KVM image exists.
)

REM ─── 7. Setup agent in Docker ───
echo [7/8] Setting up local dev agent...
docker rm -f sakura-dev-agent >nul 2>&1

REM Use PowerShell to handle JSON API calls and create agent
powershell -NoProfile -ExecutionPolicy Bypass -Command ^
  "$bp='%BACKEND_PORT%'; $root='%ROOT%'; $cfg=Join-Path $root 'agent\.dev-config.yaml'; " ^
  "if (Test-Path $cfg) { " ^
  "  $lines = Get-Content $cfg; " ^
  "  $aid = ($lines | Where-Object {$_ -match 'agent_id'}) -replace 'agent_id:\s*',''; " ^
  "  $tok = ($lines | Where-Object {$_ -match '^token'}) -replace 'token:\s*',''; " ^
  "  Write-Host '         Using existing agent config.'; " ^
  "} else { " ^
  "  try { " ^
  "    $login = Invoke-RestMethod -Uri \"http://127.0.0.1:$bp/api/v1/auth/login\" -Method Post -ContentType 'application/json' -Body '{\"email\":\"admin@sakura-dcim.local\",\"password\":\"admin123\"}'; " ^
  "    $token = $login.data.access_token; " ^
  "    if (-not $token) { $token = $login.access_token }; " ^
  "    $headers = @{ Authorization = \"Bearer $token\" }; " ^
  "    $body = '{\"name\":\"Local Dev Agent\",\"location\":\"localhost\",\"capabilities\":[\"ipmi\",\"kvm\",\"pxe\",\"inventory\",\"discovery\"]}'; " ^
  "    $resp = Invoke-RestMethod -Uri \"http://127.0.0.1:$bp/api/v1/agents\" -Method Post -ContentType 'application/json' -Headers $headers -Body $body; " ^
  "    $d = if ($resp.data) { $resp.data } else { $resp }; " ^
  "    $aid = $d.id; $tok = $d.token; " ^
  "    if ($aid -and $tok) { " ^
  "      \"server_url: ws://host.docker.internal:$bp/api/v1/agents/ws`nagent_id: $aid`ntoken: $tok\" | Out-File -Encoding utf8 $cfg; " ^
  "      Write-Host \"         Agent created: $aid\"; " ^
  "    } else { Write-Host '         WARNING: Failed to create agent.' } " ^
  "  } catch { Write-Host \"         WARNING: Agent setup failed: $_\" } " ^
  "} " ^
  "if ($aid -and $tok) { " ^
  "  Write-Host '         Building agent image...'; " ^
  "  docker build -t sakura-dcim/agent:dev -f \"$root\docker\agent.Dockerfile\" \"$root\agent\" 2>$null | Out-Null; " ^
  "  Write-Host '         Starting agent container...'; " ^
  "  docker run -d --name sakura-dev-agent -e SAKURA_AGENT_SERVER_URL=\"ws://host.docker.internal:$bp/api/v1/agents/ws\" -e SAKURA_AGENT_ID=$aid -e SAKURA_AGENT_TOKEN=$tok -e AGENT_HOST_GATEWAY=host.docker.internal -v /var/run/docker.sock:/var/run/docker.sock --restart unless-stopped sakura-dcim/agent:dev 2>$null | Out-Null; " ^
  "  Write-Host '         Agent running in Docker (sakura-dev-agent)'; " ^
  "}"

REM ─── 8. Start frontend ───
echo [8/8] Starting frontend...
echo.
echo =========================================
echo   Sakura DCIM is running!
echo.
echo   Frontend  : http://localhost:5173
echo   Backend   : http://localhost:%BACKEND_PORT%
echo   Agent     : Docker (sakura-dev-agent)
echo.
echo   Login: admin@sakura-dcim.local / admin123
echo.
echo   Agent logs : docker logs -f sakura-dev-agent
echo   Stop agent : docker rm -f sakura-dev-agent
echo   Stop all   : docker compose down
echo =========================================
echo.

start "Sakura Frontend" cmd /k "cd /d %ROOT%\web && npm run dev"

echo Services started in separate windows.
