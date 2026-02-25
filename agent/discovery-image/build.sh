#!/bin/bash
# Build the Sakura DCIM discovery initramfs
set -e

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
OUTPUT_DIR="$SCRIPT_DIR/output"

echo "=== Building Sakura DCIM Discovery Image ==="

mkdir -p "$OUTPUT_DIR"

# Build using Docker
docker build -t sakura-discovery-build -f "$SCRIPT_DIR/Dockerfile.build" "$SCRIPT_DIR"

# Extract artifacts
CONTAINER_ID=$(docker create sakura-discovery-build)
docker cp "$CONTAINER_ID:/output/initrd-discovery.img" "$OUTPUT_DIR/initrd-discovery.img"
docker cp "$CONTAINER_ID:/output/vmlinuz-discovery" "$OUTPUT_DIR/vmlinuz-discovery" 2>/dev/null || true
docker rm "$CONTAINER_ID"

echo ""
echo "=== Build complete ==="
echo "Output files:"
ls -lh "$OUTPUT_DIR/"
echo ""
echo "Deploy to agent:"
echo "  cp $OUTPUT_DIR/vmlinuz-discovery /srv/tftp/"
echo "  cp $OUTPUT_DIR/initrd-discovery.img /srv/tftp/"
