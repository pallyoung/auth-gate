#!/bin/bash
set -e

cd "$(dirname "$0")/.."

echo "Building frontend..."
cd packages/web && npm install && npm run build && cd ..

echo "Building server..."
cd packages/server && go build -o bin/auth-gate ./cmd/server

echo "Build complete: packages/server/bin/auth-gate"
