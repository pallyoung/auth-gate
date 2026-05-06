#!/bin/bash
set -e

cd "$(dirname "$0")/.."

echo "=== Auth Gate Deploy ==="

# Build
echo "[1/3] Building..."
./scripts/build.sh

# Stop existing service
echo "[2/3] Stopping existing service..."
if systemctl is-active --quiet auth-gate; then
    sudo systemctl stop auth-gate
fi

# Install binary
echo "[3/3] Installing..."
sudo cp packages/server/bin/auth-gate /usr/local/bin/auth-gate

# Copy config if not exists
if [ ! -f /etc/auth-gate/config.yaml ]; then
    sudo mkdir -p /etc/auth-gate
    sudo cp packages/server/configs/config.yaml /etc/auth-gate/config.yaml
fi

# Start service
echo "Starting service..."
sudo systemctl daemon-reload
sudo systemctl enable auth-gate
sudo systemctl start auth-gate

echo "=== Deploy complete ==="
echo "Service status:"
sudo systemctl status auth-gate --no-pager
