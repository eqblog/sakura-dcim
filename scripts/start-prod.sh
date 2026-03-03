#!/bin/bash
# Sakura DCIM — One-click production deployment
# Automatically installs Docker if missing, then builds and starts all services
#
# Usage:
#   bash scripts/start-prod.sh                   # defaults: web=3000, api=8080
#   WEB_PORT=80 bash scripts/start-prod.sh       # bind web UI to port 80
#   WEB_PORT=443 API_PORT=9090 bash scripts/start-prod.sh
#   WITH_AGENT=1 bash scripts/start-prod.sh      # also deploy a local agent

ROOT="$(cd "$(dirname "$0")/.." && pwd)"

# Configurable ports (passed to docker-compose.yml via env vars)
export WEB_PORT="${WEB_PORT:-3000}"
export API_PORT="${API_PORT:-8080}"
WITH_AGENT="${WITH_AGENT:-0}"

echo "========================================="
echo "  Sakura DCIM — Production Deployment"
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

# ─── Helper: install agent runtime tools ───
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

# ─── 0. Install dependencies ───
echo ""
echo "[0/4] Checking dependencies..."

if ! command -v curl &>/dev/null; then
  echo "  -> Installing curl..."
  install_pkg curl ca-certificates
fi

if ! command -v docker &>/dev/null; then
  echo "  -> Installing Docker..."
  curl -fsSL https://get.docker.com | sh
  systemctl start docker 2>/dev/null || true
  systemctl enable docker 2>/dev/null || true
  echo "  -> Docker installed."
fi

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

# Agent tools + Go (only if WITH_AGENT=1)
if [ "$WITH_AGENT" = "1" ]; then
  if ! command -v ipmitool &>/dev/null || ! command -v dmidecode &>/dev/null || ! command -v dnsmasq &>/dev/null; then
    install_agent_tools
  fi

  export PATH="/usr/local/go/bin:$HOME/go/bin:/usr/sbin:/usr/local/sbin:$PATH"

  # Check Go version, not just existence
  REQUIRED_GO_MINOR=25
  GO_VERSION_INSTALL="1.25.3"
  go_needs_install=false
  if ! command -v go &>/dev/null; then
    go_needs_install=true
  else
    CURRENT_GO_MINOR=$(go version 2>/dev/null | grep -oP 'go1\.(\d+)' | grep -oP '\d+$')
    if [ -n "$CURRENT_GO_MINOR" ] && [ "$CURRENT_GO_MINOR" -lt "$REQUIRED_GO_MINOR" ]; then
      echo "  -> Go version too old (1.${CURRENT_GO_MINOR}), upgrading..."
      go_needs_install=true
    fi
  fi

  if [ "$go_needs_install" = true ]; then
    echo "  -> Installing Go ${GO_VERSION_INSTALL}..."
    ARCH=$(uname -m)
    case "$ARCH" in
      x86_64)  GOARCH="amd64" ;;
      aarch64) GOARCH="arm64" ;;
      *)       GOARCH="amd64" ;;
    esac
    rm -rf /usr/local/go
    curl -fsSL "https://go.dev/dl/go${GO_VERSION_INSTALL}.linux-${GOARCH}.tar.gz" | tar -C /usr/local -xzf -
    echo 'export PATH="/usr/local/go/bin:$PATH"' > /etc/profile.d/go.sh
    echo "  -> Go ${GO_VERSION_INSTALL} installed."
  fi

  # Kill old agent processes
  pkill -f 'sakura-agent' 2>/dev/null || true
fi

# Verify
if ! command -v docker &>/dev/null || ! docker compose version &>/dev/null 2>&1; then
  echo "ERROR: Docker / Docker Compose not available. Please install manually."
  exit 1
fi

echo "  Docker  : $(docker --version 2>/dev/null | cut -d' ' -f3 | tr -d ',')"
echo "  Compose : $(docker compose version 2>/dev/null | cut -d' ' -f4)"
if [ "$WITH_AGENT" = "1" ]; then
  echo "  Go      : $(go version 2>/dev/null | cut -d' ' -f3)"
  echo "  ipmitool: $(ipmitool -V 2>&1 | head -1 || echo 'N/A')"
  echo "  dnsmasq : $(dnsmasq --version 2>/dev/null | head -1 | cut -d' ' -f3 || echo 'N/A')"
fi
echo "  OK"
echo ""

# ─── Build KVM browser image ───
# Always rebuild so that changes to cdp-redirect.py / start.sh are picked up
echo "[1/5] Building KVM browser image..."
docker build -t sakura-dcim/kvm-browser:latest "$ROOT/docker/kvm-browser/"
echo "  KVM image built."

# ─── 2. Build and start ───
echo "[2/5] Building and starting all services..."
cd "$ROOT" && docker compose up -d --build --remove-orphans

# ─── 3. Wait for healthy ───
echo "[3/5] Waiting for PostgreSQL to be healthy..."
for i in $(seq 1 60); do
  if docker exec sakura-postgres pg_isready -U sakura -d sakura_dcim -q 2>/dev/null; then
    echo "  PostgreSQL ready."
    break
  fi
  if [ "$i" -eq 60 ]; then
    echo "  WARNING: PostgreSQL health check timed out."
  fi
  sleep 1
done

# ─── 4. Run migrations ───
echo "[4/5] Running database migrations..."
docker exec sakura-backend sh -c '
  if [ -d /app/migrations ] && ls /app/migrations/*.up.sql >/dev/null 2>&1; then
    apk add --no-cache --quiet curl 2>/dev/null
    if [ ! -f /usr/local/bin/migrate ]; then
      curl -fsSL https://github.com/golang-migrate/migrate/releases/download/v4.17.0/migrate.linux-amd64.tar.gz | tar -xz -C /usr/local/bin/ 2>/dev/null || true
    fi
    if command -v migrate >/dev/null 2>&1 || [ -f /usr/local/bin/migrate ]; then
      /usr/local/bin/migrate -path /app/migrations -database "postgres://sakura:sakura@postgres:5432/sakura_dcim?sslmode=disable" up 2>&1 || true
    else
      echo "  Migration tool unavailable — please run migrations manually."
    fi
  fi
' 2>&1 || true

# ─── 5. Optional: set up local agent ───
AGENT_PID=""
if [ "$WITH_AGENT" = "1" ]; then
  echo "[5/5] Setting up local agent..."

  AGENT_CONFIG="$ROOT/agent/.dev-config.yaml"

  # Wait for backend API to be ready
  for i in $(seq 1 30); do
    if curl -s "http://127.0.0.1:${API_PORT}/health" >/dev/null 2>&1; then
      break
    fi
    sleep 1
  done

  if [ ! -f "$AGENT_CONFIG" ]; then
    LOGIN_RESP=$(curl -s -X POST "http://127.0.0.1:${API_PORT}/api/v1/auth/login" \
      -H 'Content-Type: application/json' \
      -d '{"email":"admin@sakura-dcim.local","password":"admin123"}')
    ACCESS_TOKEN=$(echo "$LOGIN_RESP" | grep -o '"access_token":"[^"]*"' | head -1 | cut -d'"' -f4)

    if [ -n "$ACCESS_TOKEN" ]; then
      AGENT_RESP=$(curl -s -X POST "http://127.0.0.1:${API_PORT}/api/v1/agents" \
        -H 'Content-Type: application/json' \
        -H "Authorization: Bearer $ACCESS_TOKEN" \
        -d '{"name":"Local Agent","location":"localhost","capabilities":["ipmi","kvm","pxe","inventory","discovery"]}')

      AGENT_ID=$(echo "$AGENT_RESP" | grep -o '"id":"[^"]*"' | head -1 | cut -d'"' -f4)
      AGENT_TOKEN=$(echo "$AGENT_RESP" | grep -o '"token":"[^"]*"' | head -1 | cut -d'"' -f4)

      if [ -n "$AGENT_ID" ] && [ -n "$AGENT_TOKEN" ]; then
        cat > "$AGENT_CONFIG" <<AGENTEOF
server_url: ws://127.0.0.1:${API_PORT}/api/v1/agents/ws
agent_id: ${AGENT_ID}
token: ${AGENT_TOKEN}
AGENTEOF
        echo "  Agent created: $AGENT_ID"
      else
        echo "  WARNING: Failed to create agent."
      fi
    else
      echo "  WARNING: Failed to login as admin."
    fi
  else
    echo "  Using existing agent config."
  fi

  if [ -f "$AGENT_CONFIG" ]; then
    echo "  Building and starting agent..."
    cd "$ROOT/agent" && go build -o "$ROOT/agent/bin/sakura-agent" ./cmd/agent && \
      "$ROOT/agent/bin/sakura-agent" -config "$AGENT_CONFIG" &
    AGENT_PID=$!
  fi
else
  echo "[4/4] Agent not requested (use WITH_AGENT=1 to enable)."
fi

# ─── Done ───
HOST_IP=$(hostname -I 2>/dev/null | awk '{print $1}')
HOST_IP="${HOST_IP:-localhost}"

echo ""
echo "========================================="
echo "  Sakura DCIM is running!"
echo ""
echo "  Web UI → http://${HOST_IP}:${WEB_PORT}"
echo "  API    → http://${HOST_IP}:${WEB_PORT}/api/v1"
if [ -n "$AGENT_PID" ]; then
echo "  Agent  → PID $AGENT_PID (local)"
fi
echo ""
echo "  Login: admin@sakura-dcim.local / admin123"
echo ""
echo "  Useful commands:"
echo "    docker compose logs -f          # view all logs"
echo "    docker compose logs -f backend  # backend logs only"
echo "    docker compose down             # stop all"
echo "    docker compose restart backend  # restart backend"
echo "    docker compose pull && docker compose up -d --build  # update"
echo "========================================="

# If agent is running, wait for it and handle Ctrl+C
if [ -n "$AGENT_PID" ]; then
  cleanup() {
    echo ""
    echo "Stopping agent..."
    kill $AGENT_PID 2>/dev/null
    wait $AGENT_PID 2>/dev/null
    exit 0
  }
  trap cleanup SIGINT SIGTERM
  wait $AGENT_PID
fi
