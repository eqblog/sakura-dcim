#!/bin/bash
# Sakura DCIM — One-click update
set -e

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

echo "========================================="
echo "  Sakura DCIM — Updating"
echo "========================================="

# 1. Pull latest code
echo "[1/4] Pulling latest code..."
git pull

# 2. Update dependencies
echo "[2/4] Updating backend dependencies..."
cd backend && go mod tidy && cd "$ROOT"

echo "[3/4] Updating frontend dependencies..."
cd web && npm install && cd "$ROOT"

# 4. Run migrations
echo "[4/4] Running database migrations..."
cd backend && go run -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest \
  -path migrations -database "postgres://sakura:sakura@localhost:5432/sakura_dcim?sslmode=disable" up 2>&1 || true
cd "$ROOT"

echo ""
echo "  Update complete! Now run:"
echo ""
echo "  Development:  ./scripts/start-dev.sh"
echo "  Production:   ./scripts/start-prod.sh"
echo "========================================="
