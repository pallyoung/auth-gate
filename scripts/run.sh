#!/bin/bash
set -e

cd "$(dirname "$0")/.."

# Install deps + build if needed
if [ ! -f packages/server/bin/auth-gate ]; then
    echo "Binary not found, running install..."
    ./scripts/install.sh
fi

# Run
cd packages/server && ./bin/auth-gate
