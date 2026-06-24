#!/bin/bash
set -euo pipefail

PROJECT_ROOT="$(cd "$(dirname "$0")/.." && pwd)"
DIST_DIR="$PROJECT_ROOT/dist"
EXE="$DIST_DIR/auth-gate"

# Build if needed
if [ ! -f "$EXE" ]; then
    echo "Distribution not found, deploying..."
    exec "$PROJECT_ROOT/scripts/deploy.sh"
fi

# Start service
cd "$DIST_DIR"
echo "Starting Auth Gate control plane on http://localhost:8080/_authgate"
echo "Press Ctrl+C to stop"
echo ""
exec ./auth-gate start -f
