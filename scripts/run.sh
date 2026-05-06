#!/bin/bash
set -e

cd "$(dirname "$0")/.."

# Install deps if needed
if [ ! -d "packages/web/node_modules" ]; then
    echo "Installing frontend dependencies..."
    cd packages/web && npm install && cd ../..
fi

# Build if needed
if [ ! -f "packages/server/bin/auth-gate" ]; then
    echo "Building..."
    cd packages/web && npm run build && cd ../..
    cd packages/server && go build -o bin/auth-gate ./cmd/server && cd ..
fi

# Run with hot-reload (go run watches code changes)
cd packages/server && go run ./cmd/server
