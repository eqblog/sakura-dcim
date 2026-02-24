#!/bin/bash
# Sakura DCIM — One-click development start
set -e

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

echo "========================================="
echo "  Sakura DCIM — Starting Development"
echo "========================================="

# 1. Start infrastructure
echo "[1/5] Starting PostgreSQL, Redis, InfluxDB..."
docker compose up -d postgres redis influxdb

echo "[2/5] Waiting for PostgreSQL to be ready..."
until docker exec sakura-postgres pg_isready -U sakura -d sakura_dcim -q 2>/dev/null; do
  sleep 1
done

# 2. Run migrations
echo "[3/5] Running database migrations..."
cd backend && go run -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest \
  -path migrations -database "postgres://sakura:sakura@localhost:5432/sakura_dcim?sslmode=disable" up 2>&1 || true
cd "$ROOT"

# 3. Install frontend dependencies if needed
if [ ! -d "web/node_modules" ]; then
  echo "[4/5] Installing frontend dependencies..."
  cd web && npm install && cd "$ROOT"
else
  echo "[4/5] Frontend dependencies already installed."
fi

# 4. Start backend and frontend
echo "[5/5] Starting backend and frontend..."
echo ""
echo "  Backend  → http://localhost:8080"
echo "  Frontend → http://localhost:5173"
echo ""
echo "  Login: admin@sakura-dcim.local / admin123"
echo "========================================="
echo ""

# Run backend and frontend in parallel
(cd backend && go run ./cmd/server) &
BACKEND_PID=$!
(cd web && npm run dev) &
FRONTEND_PID=$!

# Handle Ctrl+C
trap "kill $BACKEND_PID $FRONTEND_PID 2>/dev/null; exit 0" SIGINT SIGTERM
wait
