#!/bin/bash
set -e

cd "$(dirname "$0")/.."

DIST_DIR="$(pwd)/dist"
EXE="$DIST_DIR/auth-gate"

# Build if needed
if [ ! -f "$EXE" ]; then
    echo "Binary not found, building..."
    ./scripts/deploy.sh
fi

# Start service
cd "$DIST_DIR"
echo "Starting Auth Gate on http://localhost:8080"
echo "Press Ctrl+C to stop"
echo ""
./auth-gate
