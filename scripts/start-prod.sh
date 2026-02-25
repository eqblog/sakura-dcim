#!/bin/bash
# Sakura DCIM — One-click production deployment
# Automatically installs Docker if missing, then builds and starts all services
#
# Usage:
#   bash scripts/start-prod.sh                   # defaults: web=3000, api=8080
#   WEB_PORT=80 bash scripts/start-prod.sh       # bind web UI to port 80
#   WEB_PORT=443 API_PORT=9090 bash scripts/start-prod.sh

ROOT="$(cd "$(dirname "$0")/.." && pwd)"

# Configurable ports (passed to docker-compose.yml via env vars)
export WEB_PORT="${WEB_PORT:-3000}"
export API_PORT="${API_PORT:-8080}"

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

# ─── 0. Install Docker if missing ───
echo ""
echo "[0/3] Checking dependencies..."

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

# Verify
if ! command -v docker &>/dev/null || ! docker compose version &>/dev/null 2>&1; then
  echo "ERROR: Docker / Docker Compose not available. Please install manually."
  exit 1
fi

echo "  Docker  : $(docker --version 2>/dev/null | cut -d' ' -f3 | tr -d ',')"
echo "  Compose : $(docker compose version 2>/dev/null | cut -d' ' -f4)"
echo "  OK"
echo ""

# ─── 1. Build and start ───
echo "[1/3] Building and starting all services..."
cd "$ROOT" && docker compose up -d --build --remove-orphans

# ─── 2. Wait for healthy ───
echo "[2/3] Waiting for PostgreSQL to be healthy..."
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

# ─── 3. Run migrations ───
echo "[3/3] Running database migrations..."
# Install migrate tool inside container and run
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

# ─── Done ───
HOST_IP=$(hostname -I 2>/dev/null | awk '{print $1}')
HOST_IP="${HOST_IP:-localhost}"

echo ""
echo "========================================="
echo "  Sakura DCIM is running!"
echo ""
echo "  Web UI → http://${HOST_IP}:${WEB_PORT}"
echo "  API    → http://${HOST_IP}:${WEB_PORT}/api/v1"
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
