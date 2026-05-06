#!/bin/bash
set -e

cd "$(dirname "$0")/.."

echo "=== Docker Deploy ==="

# Stop and remove old container
if docker ps -a | grep -q auth-gate; then
    echo "[1/4] Removing old container..."
    docker stop auth-gate 2>/dev/null || true
    docker rm auth-gate 2>/dev/null || true
fi

# Build image
echo "[2/4] Building image..."
docker build -t auth-gate:latest .

# Start container
echo "[3/4] Starting container..."
docker run -d \
    --name auth-gate \
    -p 8080:8080 \
    -v auth-gate-data:/app/data \
    -v "$(pwd)/packages/server/configs/config.yaml:/app/config.yaml:ro" \
    --restart unless-stopped \
    auth-gate:latest

# Cleanup unused images
echo "[4/4] Cleanup..."
docker image prune -f

echo "=== Deploy complete ==="
docker ps | grep auth-gate
