#!/bin/bash
set -e

cd "$(dirname "$0")/.."

VERSION=${1:-latest}
OUTPUT_DIR="dist/release"

echo "=== Building Release $VERSION ==="

mkdir -p "$OUTPUT_DIR"

# Build frontend
echo "[1/4] Building frontend..."
cd packages/web
npm install
npm run build
cd ../..

# Linux
echo "[2/4] Building Linux..."
cd packages/server
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="-s -w" -o "$OUTPUT_DIR/auth-gate-linux-amd64" ./cmd/server
GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -ldflags="-s -w" -o "$OUTPUT_DIR/auth-gate-linux-arm64" ./cmd/server
cd ..

# macOS
echo "[3/4] Building macOS..."
cd packages/server
GOOS=darwin GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="-s -w" -o "$OUTPUT_DIR/auth-gate-darwin-amd64" ./cmd/server
GOOS=darwin GOARCH=arm64 CGO_ENABLED=0 go build -ldflags="-s -w" -o "$OUTPUT_DIR/auth-gate-darwin-arm64" ./cmd/server
cd ..

# Windows
echo "[4/4] Building Windows..."
cd packages/server
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="-s -w" -o "$OUTPUT_DIR/auth-gate-windows-amd64.exe" ./cmd/server
cd ..

# Package
echo "Packaging..."
cd $OUTPUT_DIR
for f in auth-gate-*; do
    tar -czvf "${f}.tar.gz" "$f" 2>/dev/null || true
done

echo "=== Release built in $OUTPUT_DIR ==="
ls -lh $OUTPUT_DIR
