#!/bin/bash
set -e

cd "$(dirname "$0")/.."

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
cd packages/web && npm install && cd ../..

# Build frontend
echo "[2/4] Building frontend..."
cd packages/web && npm run build && cd ../..

# Build server
echo "[3/4] Building server..."
cd packages/server && go build -o bin/auth-gate ./cmd/server && cd ..

echo "[4/4] Build complete!"
echo ""
echo "Binary: packages/server/bin/auth-gate"
echo ""
echo "Run with: ./scripts/run.sh"
echo "Or directly: cd packages/server && ./bin/auth-gate"
