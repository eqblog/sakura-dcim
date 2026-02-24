#!/bin/bash
# Sakura DCIM — One-click production start
set -e

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

echo "========================================="
echo "  Sakura DCIM — Production Deployment"
echo "========================================="

# Build and start all services
echo "[1/2] Building and starting all services..."
docker compose up -d --build

echo "[2/2] Waiting for services to be healthy..."
sleep 5

echo ""
echo "  Web UI → http://localhost:3000"
echo "  API    → http://localhost:3000/api/v1"
echo ""
echo "  Login: admin@sakura-dcim.local / admin123"
echo "========================================="
