#!/bin/bash
set -e

cd "$(dirname "$0")/.."

# Build if needed
if [ ! -f packages/server/bin/auth-gate ]; then
    echo "Binary not found, building first..."
    ./scripts/build.sh
fi

# Run
cd packages/server && ./bin/auth-gate
