#!/bin/bash
# Sakura DCIM — One-click development start
# Automatically installs all dependencies on a fresh server,
# creates a local dev agent, and starts everything.
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
export BACKEND_PORT  # passed to vite.config.ts for proxy target

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
echo "[0/8] Checking & installing dependencies..."

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

# Agent runtime tools (ipmitool, dmidecode, dnsmasq, snmp, mdadm, etc.)
install_agent_tools() {
  echo "  -> Installing agent runtime tools..."
  if command -v apt-get &>/dev/null; then
    DEBIAN_FRONTEND=noninteractive apt-get install -y -qq \
      ipmitool dmidecode dnsmasq snmp mdadm util-linux iproute2 pciutils 2>/dev/null
  elif command -v dnf &>/dev/null; then
    dnf install -y ipmitool dmidecode dnsmasq net-snmp-utils mdadm util-linux iproute pciutils 2>/dev/null
  elif command -v yum &>/dev/null; then
    yum install -y ipmitool dmidecode dnsmasq net-snmp-utils mdadm util-linux iproute pciutils 2>/dev/null
  elif command -v apk &>/dev/null; then
    apk add --no-cache ipmitool dmidecode dnsmasq net-snmp-tools mdadm util-linux iproute2 pciutils 2>/dev/null
  fi
  # Stop dnsmasq default service (we manage it ourselves)
  systemctl stop dnsmasq 2>/dev/null || true
  systemctl disable dnsmasq 2>/dev/null || true
}

if ! command -v ipmitool &>/dev/null || ! command -v dmidecode &>/dev/null || ! command -v dnsmasq &>/dev/null; then
  install_agent_tools
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
echo "[1/8] Starting PostgreSQL, Redis, InfluxDB..."
cd "$ROOT" && docker compose up -d postgres redis influxdb

echo "[2/8] Waiting for PostgreSQL to be ready..."
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
echo "[3/8] Running database migrations..."
cd "$ROOT/backend" && go run -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest \
  -path migrations -database "postgres://sakura:sakura@localhost:5432/sakura_dcim?sslmode=disable" up 2>&1 || true

# ─── 3. Install frontend dependencies if needed ───
if [ ! -d "$ROOT/web/node_modules" ]; then
  echo "[4/8] Installing frontend dependencies..."
  cd "$ROOT/web" && npm install
else
  echo "[4/8] Frontend dependencies already installed."
fi

# ─── 4. Start backend ───
echo "[5/8] Starting backend on port ${BACKEND_PORT}..."
cd "$ROOT/backend" && go run ./cmd/server &
BACKEND_PID=$!

# Wait for backend to be healthy
echo "  Waiting for backend..."
for i in $(seq 1 30); do
  if curl -s "http://127.0.0.1:${BACKEND_PORT}/health" >/dev/null 2>&1; then
    echo "  Backend ready."
    break
  fi
  if [ "$i" -eq 30 ]; then
    echo "  WARNING: Backend did not become ready in 30s, continuing..."
  fi
  sleep 1
done

# ─── 5. Build KVM Docker image ───
echo "[6/8] Building KVM browser image..."
if ! docker images --format '{{.Repository}}:{{.Tag}}' | grep -q 'sakura-dcim/kvm-browser:latest'; then
  docker build -t sakura-dcim/kvm-browser:latest "$ROOT/docker/kvm-browser/"
  echo "  KVM image built."
else
  echo "  KVM image already exists."
fi

# ─── 6. Auto-create local dev agent ───
echo "[7/8] Setting up local dev agent..."
AGENT_CONFIG="$ROOT/agent/.dev-config.yaml"
AGENT_PID=""

if [ ! -f "$AGENT_CONFIG" ]; then
  # Login as admin to get JWT
  LOGIN_RESP=$(curl -s -X POST "http://127.0.0.1:${BACKEND_PORT}/api/v1/auth/login" \
    -H 'Content-Type: application/json' \
    -d '{"email":"admin@sakura-dcim.local","password":"admin123"}')
  ACCESS_TOKEN=$(echo "$LOGIN_RESP" | grep -o '"access_token":"[^"]*"' | head -1 | cut -d'"' -f4)

  if [ -n "$ACCESS_TOKEN" ]; then
    # Create agent via API
    AGENT_RESP=$(curl -s -X POST "http://127.0.0.1:${BACKEND_PORT}/api/v1/agents" \
      -H 'Content-Type: application/json' \
      -H "Authorization: Bearer $ACCESS_TOKEN" \
      -d '{"name":"Local Dev Agent","location":"localhost","capabilities":["ipmi","kvm","pxe","inventory","discovery"]}')

    AGENT_ID=$(echo "$AGENT_RESP" | grep -o '"id":"[^"]*"' | head -1 | cut -d'"' -f4)
    AGENT_TOKEN=$(echo "$AGENT_RESP" | grep -o '"token":"[^"]*"' | head -1 | cut -d'"' -f4)

    if [ -n "$AGENT_ID" ] && [ -n "$AGENT_TOKEN" ]; then
      cat > "$AGENT_CONFIG" <<AGENTEOF
server_url: ws://127.0.0.1:${BACKEND_PORT}/api/v1/agents/ws
agent_id: ${AGENT_ID}
token: ${AGENT_TOKEN}
AGENTEOF
      echo "  Agent created: $AGENT_ID"
      echo "  Config saved: $AGENT_CONFIG"
    else
      echo "  WARNING: Failed to create agent. Response: $AGENT_RESP"
    fi
  else
    echo "  WARNING: Failed to login as admin. Response: $LOGIN_RESP"
  fi
else
  echo "  Using existing agent config: $AGENT_CONFIG"
fi

# Start agent if config exists
if [ -f "$AGENT_CONFIG" ]; then
  echo "  Starting local agent..."
  cd "$ROOT/agent" && go run ./cmd/agent -config "$AGENT_CONFIG" &
  AGENT_PID=$!
fi

# ─── 6. Start frontend ───
echo "[8/8] Starting frontend..."

# Detect host IP for display
HOST_IP=$(hostname -I 2>/dev/null | awk '{print $1}')
HOST_IP="${HOST_IP:-localhost}"

echo ""
echo "========================================="
echo "  Sakura DCIM is running!"
echo ""
echo "  Frontend → http://${HOST_IP}:${FRONTEND_PORT}"
echo "  Backend  → http://${HOST_IP}:${BACKEND_PORT}"
echo "  API proxy: frontend /api/* → backend :${BACKEND_PORT}"
echo ""
echo "  Login: admin@sakura-dcim.local / admin123"
echo "========================================="
echo ""

cd "$ROOT/web" && npx vite --host "$FRONTEND_HOST" --port "$FRONTEND_PORT" &
FRONTEND_PID=$!

# Handle Ctrl+C — kill all processes
cleanup() {
  echo ""
  echo "Shutting down..."
  kill $BACKEND_PID $FRONTEND_PID $AGENT_PID 2>/dev/null
  wait $BACKEND_PID $FRONTEND_PID $AGENT_PID 2>/dev/null
  exit 0
}
trap cleanup SIGINT SIGTERM

wait
