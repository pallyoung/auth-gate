#!/bin/bash
set -e

cd "$(dirname "$0")/.."

echo "Installing frontend dependencies..."
cd packages/web && npm install && cd ..

echo "Building frontend..."
cd packages/web && npm run build && cd ..

echo "Starting server..."
cd packages/server && go run ./cmd/server
