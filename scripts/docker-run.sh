#!/bin/bash
set -e

cd "$(dirname "$0")/.."

echo "=== Docker Run ==="
docker run -d \
    --name auth-gate \
    -p 8080:8080 \
    -v auth-gate-data:/app/data \
    -v "$(pwd)/packages/server/configs/config.yaml:/app/config.yaml:ro" \
    --restart unless-stopped \
    auth-gate:latest

echo "Container started"
docker logs auth-gate
