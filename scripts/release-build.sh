#!/bin/bash
set -euo pipefail

PROJECT_ROOT="$(cd "$(dirname "$0")/.." && pwd)"
VERSION=${1:-latest}
OUTPUT_DIR="$PROJECT_ROOT/dist/release"
PACKAGE_DIR="$OUTPUT_DIR/auth-gate-${VERSION}"

echo "=== Building Release $VERSION ==="

rm -rf "$OUTPUT_DIR"
mkdir -p "$PACKAGE_DIR"

# Build frontend
echo "[1/5] Building frontend..."
(
    cd "$PROJECT_ROOT/packages/web"
    npm ci --include=dev
    npm run build
    cp -r dist "$PACKAGE_DIR/web"
)

# Build server for current platform
echo "[2/5] Building server..."
(
    cd "$PROJECT_ROOT/packages/server"
    CGO_ENABLED=0 go build -ldflags="-s -w" -o "$PACKAGE_DIR/auth-gate" ./cmd/server
)

# Copy config template
echo "[3/5] Copying config files..."
(
    cd "$PROJECT_ROOT"
    cp packages/server/configs/config.yaml "$PACKAGE_DIR/config.yaml.example"
    cp scripts/auth-gate.service "$PACKAGE_DIR/"
)

# Create install script
echo "[4/5] Creating install script..."
cat > "$PACKAGE_DIR/install.sh" << 'INSTALL_EOF'
#!/bin/bash
set -euo pipefail

INSTALL_DIR="${AUTH_GATE_INSTALL_DIR:-/opt/auth-gate}"
SERVICE_USER="${AUTH_GATE_USER:-auth-gate}"

echo "=== Installing Auth Gate ==="

# Check if running as root
if [ "$EUID" -ne 0 ]; then
    echo "Please run as root (use sudo)"
    exit 1
fi

# Create user if not exists
if ! id "$SERVICE_USER" &>/dev/null; then
    echo "Creating user $SERVICE_USER..."
    useradd -r -s /bin/false -d "$INSTALL_DIR" "$SERVICE_USER"
fi

# Create install directory
echo "Installing to $INSTALL_DIR..."
mkdir -p "$INSTALL_DIR"
cp -r ./* "$INSTALL_DIR/"
chmod +x "$INSTALL_DIR/auth-gate"

# Create config if not exists
if [ ! -f "$INSTALL_DIR/config.yaml" ]; then
    if [ -f "$INSTALL_DIR/config.yaml.example" ]; then
        cp "$INSTALL_DIR/config.yaml.example" "$INSTALL_DIR/config.yaml"
        echo "Created config.yaml from example. Please edit it."
    fi
fi

# Set permissions
chown -R "$SERVICE_USER:$SERVICE_USER" "$INSTALL_DIR"

# Install systemd service if available
if command -v systemctl &>/dev/null && [ -d /etc/systemd/system ]; then
    echo "Installing systemd service..."
    cp "$INSTALL_DIR/auth-gate.service" /etc/systemd/system/
    systemctl daemon-reload
    systemctl enable auth-gate
    echo "Service installed. Start with: systemctl start auth-gate"
fi

echo ""
echo "=== Installation Complete ==="
echo ""
echo "Binary: $INSTALL_DIR/auth-gate"
echo "Config: $INSTALL_DIR/config.yaml"
echo "Web UI: $INSTALL_DIR/web/"
echo ""
echo "Quick start:"
echo "  cd $INSTALL_DIR"
echo "  ./auth-gate"
echo ""
echo "Or with systemd:"
echo "  systemctl start auth-gate"
echo ""
echo "Access admin UI at http://localhost:9000"
INSTALL_EOF
chmod +x "$PACKAGE_DIR/install.sh"

# Create tar.gz package
echo "[5/5] Creating package..."
(
    cd "$OUTPUT_DIR"
    tar -czvf "auth-gate-${VERSION}-linux-amd64.tar.gz" "auth-gate-${VERSION}"
)

echo "=== Release built in $OUTPUT_DIR ==="
ls -lh "$OUTPUT_DIR"
