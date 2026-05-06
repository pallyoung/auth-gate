#!/bin/bash
set -e

cd "$(dirname "$0")/.."

echo "=== Auth Gate Deploy ==="

# Build
echo "[1/3] Building..."
./scripts/build.sh

# Create dist directory
mkdir -p dist/web

# Copy files to dist
echo "[2/3] Copying to dist..."
cp packages/server/bin/auth-gate dist/
cp packages/server/configs/config.yaml dist/
rm -rf dist/web
cp -r packages/web/dist dist/web

# Start service
echo "[3/3] Starting service..."
cd dist
./auth-gate
