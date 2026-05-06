#!/bin/bash
set -euo pipefail

PROJECT_ROOT="$(cd "$(dirname "$0")/.." && pwd)"
VERSION=${1:-latest}
OUTPUT_DIR="$PROJECT_ROOT/dist/release"

echo "=== Building Release $VERSION ==="

mkdir -p "$OUTPUT_DIR"

# Build frontend
echo "[1/4] Building frontend..."
(
    cd "$PROJECT_ROOT/packages/web"
    npm ci --include=dev
    npm run build
)

# Linux
echo "[2/4] Building Linux..."
(
    cd "$PROJECT_ROOT/packages/server"
    GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="-s -w" -o "$OUTPUT_DIR/auth-gate-linux-amd64" ./cmd/server
    GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -ldflags="-s -w" -o "$OUTPUT_DIR/auth-gate-linux-arm64" ./cmd/server
)

# macOS
echo "[3/4] Building macOS..."
(
    cd "$PROJECT_ROOT/packages/server"
    GOOS=darwin GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="-s -w" -o "$OUTPUT_DIR/auth-gate-darwin-amd64" ./cmd/server
    GOOS=darwin GOARCH=arm64 CGO_ENABLED=0 go build -ldflags="-s -w" -o "$OUTPUT_DIR/auth-gate-darwin-arm64" ./cmd/server
)

# Windows
echo "[4/4] Building Windows..."
(
    cd "$PROJECT_ROOT/packages/server"
    GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="-s -w" -o "$OUTPUT_DIR/auth-gate-windows-amd64.exe" ./cmd/server
)

# Package
echo "Packaging..."
(
    cd "$OUTPUT_DIR"
    for f in auth-gate-*; do
        [ -f "$f" ] || continue
        case "$f" in
            *.tar.gz|*.zip) continue ;;
        esac
        tar -czvf "${f}.tar.gz" "$f"
    done
)

echo "=== Release built in $OUTPUT_DIR ==="
ls -lh "$OUTPUT_DIR"
