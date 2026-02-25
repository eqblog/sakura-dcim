#!/bin/bash
# Sakura DCIM — One-click development start
# Automatically installs all dependencies on a fresh server
#
# Usage:
#   bash scripts/start-dev.sh                          # defaults: 0.0.0.0:5173
#   FRONTEND_HOST=0.0.0.0 FRONTEND_PORT=3000 bash scripts/start-dev.sh
#   BACKEND_PORT=8080 bash scripts/start-dev.sh

ROOT="$(cd "$(dirname "$0")/.." && pwd)"

# Configurable bind addresses
FRONTEND_HOST="${FRONTEND_HOST:-0.0.0.0}"
FRONTEND_PORT="${FRONTEND_PORT:-5173}"
BACKEND_PORT="${BACKEND_PORT:-8080}"

echo "========================================="
echo "  Sakura DCIM — Starting Development"
echo "========================================="

# ─── Helper: detect package manager ───
install_pkg() {
  if command -v apt-get &>/dev/null; then
    apt-get update -qq && DEBIAN_FRONTEND=noninteractive apt-get install -y -qq "$@"
  elif command -v dnf &>/dev/null; then
    dnf install -y "$@"
  elif command -v yum &>/dev/null; then
    yum install -y "$@"
  elif command -v apk &>/dev/null; then
    apk add --no-cache "$@"
  else
    echo "ERROR: No supported package manager found (apt/yum/dnf/apk)"
    exit 1
  fi
}

# ─── 0. Install dependencies if missing ───
echo ""
echo "[0/5] Checking & installing dependencies..."

# curl
if ! command -v curl &>/dev/null; then
  echo "  -> Installing curl..."
  install_pkg curl ca-certificates
fi

# git
if ! command -v git &>/dev/null; then
  echo "  -> Installing git..."
  install_pkg git
fi

# make (sometimes needed for go builds)
if ! command -v make &>/dev/null; then
  echo "  -> Installing make..."
  install_pkg make
fi

# Docker
if ! command -v docker &>/dev/null; then
  echo "  -> Installing Docker..."
  curl -fsSL https://get.docker.com | sh
  systemctl start docker 2>/dev/null || true
  systemctl enable docker 2>/dev/null || true
  echo "  -> Docker installed."
fi

# Docker Compose V2
if ! docker compose version &>/dev/null 2>&1; then
  echo "  -> Installing Docker Compose plugin..."
  install_pkg docker-compose-plugin 2>/dev/null || {
    COMPOSE_VERSION=$(curl -s https://api.github.com/repos/docker/compose/releases/latest | grep '"tag_name"' | cut -d'"' -f4)
    mkdir -p /usr/local/lib/docker/cli-plugins
    curl -fsSL "https://github.com/docker/compose/releases/download/${COMPOSE_VERSION}/docker-compose-$(uname -s)-$(uname -m)" \
      -o /usr/local/lib/docker/cli-plugins/docker-compose
    chmod +x /usr/local/lib/docker/cli-plugins/docker-compose
  }
  echo "  -> Docker Compose installed."
fi

# Go
export PATH="/usr/local/go/bin:$HOME/go/bin:$PATH"
if ! command -v go &>/dev/null; then
  echo "  -> Installing Go 1.22..."
  GO_VERSION="1.22.5"
  ARCH=$(uname -m)
  case "$ARCH" in
    x86_64)  GOARCH="amd64" ;;
    aarch64) GOARCH="arm64" ;;
    armv7l)  GOARCH="armv6l" ;;
    *)       GOARCH="amd64" ;;
  esac
  rm -rf /usr/local/go
  curl -fsSL "https://go.dev/dl/go${GO_VERSION}.linux-${GOARCH}.tar.gz" | tar -C /usr/local -xzf -
  # Persist for future shells
  echo 'export PATH="/usr/local/go/bin:$PATH"' > /etc/profile.d/go.sh
  echo "  -> Go ${GO_VERSION} installed."
fi

# Node.js + npm
if ! command -v node &>/dev/null; then
  echo "  -> Installing Node.js 20..."
  if command -v apt-get &>/dev/null; then
    curl -fsSL https://deb.nodesource.com/setup_20.x | bash -
    DEBIAN_FRONTEND=noninteractive apt-get install -y -qq nodejs
  elif command -v dnf &>/dev/null || command -v yum &>/dev/null; then
    curl -fsSL https://rpm.nodesource.com/setup_20.x | bash -
    install_pkg nodejs
  else
    install_pkg nodejs npm
  fi
  echo "  -> Node.js $(node -v) installed."
fi

# Final check — abort if still missing
MISSING=""
command -v docker &>/dev/null || MISSING="$MISSING docker"
command -v go     &>/dev/null || MISSING="$MISSING go"
command -v node   &>/dev/null || MISSING="$MISSING node"
command -v npm    &>/dev/null || MISSING="$MISSING npm"
if [ -n "$MISSING" ]; then
  echo ""
  echo "ERROR: Failed to install:$MISSING"
  echo "Please install them manually and re-run this script."
  exit 1
fi

echo "  Docker : $(docker --version 2>/dev/null | cut -d' ' -f3 | tr -d ',')"
echo "  Go     : $(go version 2>/dev/null | cut -d' ' -f3)"
echo "  Node   : $(node -v 2>/dev/null)"
echo "  npm    : $(npm -v 2>/dev/null)"
echo "  OK"
echo ""

# ─── 1. Start infrastructure ───
echo "[1/5] Starting PostgreSQL, Redis, InfluxDB..."
cd "$ROOT" && docker compose up -d postgres redis influxdb

echo "[2/5] Waiting for PostgreSQL to be ready..."
for i in $(seq 1 30); do
  if docker exec sakura-postgres pg_isready -U sakura -d sakura_dcim -q 2>/dev/null; then
    echo "  PostgreSQL ready."
    break
  fi
  if [ "$i" -eq 30 ]; then
    echo "ERROR: PostgreSQL did not become ready in 30s"
    exit 1
  fi
  sleep 1
done

# ─── 2. Run migrations ───
echo "[3/5] Running database migrations..."
cd "$ROOT/backend" && go run -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest \
  -path migrations -database "postgres://sakura:sakura@localhost:5432/sakura_dcim?sslmode=disable" up 2>&1 || true

# ─── 3. Install frontend dependencies if needed ───
if [ ! -d "$ROOT/web/node_modules" ]; then
  echo "[4/5] Installing frontend dependencies..."
  cd "$ROOT/web" && npm install
else
  echo "[4/5] Frontend dependencies already installed."
fi

# ─── 4. Start backend and frontend ───
echo "[5/5] Starting backend and frontend..."
echo ""
echo "  Backend  → http://${FRONTEND_HOST}:${BACKEND_PORT}"
echo "  Frontend → http://${FRONTEND_HOST}:${FRONTEND_PORT}"
echo ""
echo "  Login: admin@sakura-dcim.local / admin123"
echo "========================================="
echo ""

# Run backend and frontend in parallel
cd "$ROOT/backend" && go run ./cmd/server &
BACKEND_PID=$!

cd "$ROOT/web" && npx vite --host "$FRONTEND_HOST" --port "$FRONTEND_PORT" &
FRONTEND_PID=$!

# Handle Ctrl+C — kill both processes
cleanup() {
  echo ""
  echo "Shutting down..."
  kill $BACKEND_PID $FRONTEND_PID 2>/dev/null
  wait $BACKEND_PID $FRONTEND_PID 2>/dev/null
  exit 0
}
trap cleanup SIGINT SIGTERM

wait
