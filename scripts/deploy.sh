#!/bin/bash
set -e

cd "$(dirname "$0")/.."

echo "=== Auth Gate Deploy ==="

# Build
echo "[1/3] Building..."
./scripts/build.sh

# Create dist directory
mkdir -p dist/web

# Stop existing service
echo "[2/3] Stopping existing service..."
if systemctl is-active --quiet auth-gate 2>/dev/null; then
    sudo systemctl stop auth-gate
fi

# Copy files to dist
echo "[3/3] Copying to dist..."
cp packages/server/bin/auth-gate dist/
cp packages/server/configs/config.yaml dist/
rm -rf dist/web
cp -r packages/web/dist dist/web

echo "=== Deploy complete ==="
echo ""
echo "Run: cd dist; ./auth-gate"
echo "Then visit: http://localhost:8080"
