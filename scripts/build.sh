#!/bin/bash
set -euo pipefail

PROJECT_ROOT="$(cd "$(dirname "$0")/.." && pwd)"

echo "Building frontend..."
(
    cd "$PROJECT_ROOT/packages/web"
    npm ci --include=dev
    npm run build
)

echo "Building server..."
(
    cd "$PROJECT_ROOT/packages/server"
    go build -o bin/auth-gate ./cmd/server
)

echo "Build complete: $PROJECT_ROOT/packages/server/bin/auth-gate"
