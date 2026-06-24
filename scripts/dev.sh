#!/bin/bash
# Auth Gate Dev Script
# Starts Go backend + Vite dev server together.
# Press Ctrl+C to stop both.

set -euo pipefail

PROJECT_ROOT="$(cd "$(dirname "$0")/.." && pwd)"

echo "=== Auth Gate Dev ==="
echo ""

# Check prerequisites
command -v go  >/dev/null 2>&1 || { echo "Error: go not found in PATH"; exit 1; }
command -v npm >/dev/null 2>&1 || { echo "Error: npm not found in PATH"; exit 1; }

# Install web dependencies if needed
if [ ! -d "$PROJECT_ROOT/packages/web/node_modules" ]; then
    echo "[1/3] Installing web dependencies..."
    (cd "$PROJECT_ROOT/packages/web" && npm install)
fi

echo "[2/3] Starting Go backend (port 8080)..."

# Start Go backend in background
GO_LOG="$PROJECT_ROOT/dev-server.log"
(cd "$PROJECT_ROOT/packages/server" && DEBUG=true air) > "$GO_LOG" 2>&1 &
SERVER_PID=$!
echo "  Go backend started (PID $SERVER_PID)"
echo "  Log file: $GO_LOG"

# Give backend a moment to start
sleep 2

if ! kill -0 $SERVER_PID 2>/dev/null; then
    echo "  Error: Go backend exited immediately"
    cat "$GO_LOG"
    exit 1
fi

cleanup() {
    echo ""
    echo "Stopping Go backend (PID $SERVER_PID)..."
    kill $SERVER_PID 2>/dev/null || true
    wait $SERVER_PID 2>/dev/null || true
    echo "Dev environment stopped."
}
trap cleanup EXIT INT TERM

echo "[3/3] Starting Vite dev server (port 5174)..."
echo ""
echo "  Backend:  http://localhost:8080/_authgate"
echo "  Frontend: http://localhost:5174/_authgate/"
echo ""
echo "Press Ctrl+C to stop"
echo ""

# Run Vite in foreground
cd "$PROJECT_ROOT/packages/web"
npm run dev
