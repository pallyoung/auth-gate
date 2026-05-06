#!/bin/bash
set -euo pipefail

PROJECT_ROOT="$(cd "$(dirname "$0")/.." && pwd)"

echo "=== Auth Gate Install & Run ==="

# Check Go
if ! command -v go &> /dev/null; then
    echo "Error: Go is not installed"
    exit 1
fi

# Check Node
if ! command -v node &> /dev/null; then
    echo "Error: Node.js is not installed"
    exit 1
fi

# Install frontend deps
echo "[1/4] Installing frontend dependencies..."
(
    cd "$PROJECT_ROOT/packages/web"
    npm ci --include=dev
)

# Build frontend
echo "[2/4] Building frontend..."
(
    cd "$PROJECT_ROOT/packages/web"
    npm run build
)

# Build server
echo "[3/4] Building server..."
(
    cd "$PROJECT_ROOT/packages/server"
    go build -o bin/auth-gate ./cmd/server
)

echo "[4/4] Build complete!"
echo ""
echo "Binary: packages/server/bin/auth-gate"
echo ""
echo "Run with: ./scripts/run.sh"
echo "Or directly: cd packages/server && ./bin/auth-gate"
