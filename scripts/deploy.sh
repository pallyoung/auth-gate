#!/bin/bash
set -euo pipefail

PROJECT_ROOT="$(cd "$(dirname "$0")/.." && pwd)"
DIST_DIR="$PROJECT_ROOT/dist"
WEB_DIST_DIR="$DIST_DIR/web"

echo "=== Auth Gate Deploy ==="

# Build
echo "[1/3] Building..."
 "$PROJECT_ROOT/scripts/build.sh"

# Create dist directory
mkdir -p "$DIST_DIR"

# Copy files to dist
echo "[2/3] Copying to dist..."
cp "$PROJECT_ROOT/packages/server/bin/auth-gate" "$DIST_DIR/"
cp "$PROJECT_ROOT/packages/server/configs/config.yaml" "$DIST_DIR/"
rm -rf "$WEB_DIST_DIR"
cp -r "$PROJECT_ROOT/packages/web/dist" "$WEB_DIST_DIR"

# Start service
echo "[3/3] Starting service..."
cd "$DIST_DIR"
exec ./auth-gate
