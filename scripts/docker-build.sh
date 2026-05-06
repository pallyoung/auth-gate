#!/bin/bash
set -e

cd "$(dirname "$0")/.."

echo "=== Docker Build ==="
docker build -t auth-gate:latest .
echo "Image: auth-gate:latest"
