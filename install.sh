#!/bin/bash
set -euo pipefail

# Auth Gate Installer
# Usage: curl -fsSL https://raw.githubusercontent.com/pallyoung/auth-gate/main/install.sh | bash

REPO="pallyoung/auth-gate"
INSTALL_DIR="${AUTH_GATE_INSTALL_DIR:-/opt/auth-gate}"
VERSION="${AUTH_GATE_VERSION:-latest}"
SERVICE_USER="${AUTH_GATE_USER:-auth-gate}"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

info() { echo -e "${GREEN}[INFO]${NC} $1"; }
warn() { echo -e "${YELLOW}[WARN]${NC} $1"; }
error() { echo -e "${RED}[ERROR]${NC} $1"; exit 1; }

# Check dependencies
check_deps() {
    local missing=()
    command -v curl &>/dev/null || missing+=(curl)
    command -v tar &>/dev/null || missing+=(tar)

    if [ ${#missing[@]} -gt 0 ]; then
        error "Missing dependencies: ${missing[*]}. Please install them first."
    fi
}

# Get latest version from GitHub
get_latest_version() {
    curl -s "https://api.github.com/repos/$REPO/releases/latest" | grep '"tag_name"' | sed -E 's/.*"([^"]+)".*/\1/'
}

# Download and install
install() {
    local version="$VERSION"
    if [ "$version" = "latest" ]; then
        info "Fetching latest version..."
        version=$(get_latest_version)
        if [ -z "$version" ]; then
            error "Failed to fetch latest version. Check your internet connection."
        fi
    fi

    info "Installing Auth Gate $version..."

    # Create temp directory
    local tmp_dir
    tmp_dir=$(mktemp -d)
    trap "rm -rf $tmp_dir" EXIT

    # Download package
    info "Downloading package..."
    local url="https://github.com/$REPO/releases/download/$version/auth-gate-${version#v}-linux-amd64.tar.gz"
    if ! curl -fsSL "$url" -o "$tmp_dir/auth-gate.tar.gz"; then
        error "Failed to download $url"
    fi

    # Extract
    info "Extracting..."
    tar -xzf "$tmp_dir/auth-gate.tar.gz" -C "$tmp_dir"

    # Run installer
    info "Running installer..."
    local package_dir
    package_dir=$(find "$tmp_dir" -maxdepth 1 -type d -name "auth-gate-*" | head -1)

    if [ -z "$package_dir" ]; then
        error "Package directory not found"
    fi

    cd "$package_dir"

    # Check if running as root
    if [ "$EUID" -ne 0 ]; then
        error "Please run as root (use sudo)"
    fi

    # Create user if not exists
    if ! id "$SERVICE_USER" &>/dev/null; then
        info "Creating user $SERVICE_USER..."
        useradd -r -s /bin/false -d "$INSTALL_DIR" "$SERVICE_USER"
    fi

    # Create install directory
    info "Installing to $INSTALL_DIR..."
    mkdir -p "$INSTALL_DIR"
    cp -r ./* "$INSTALL_DIR/"
    chmod +x "$INSTALL_DIR/auth-gate"

    # Create config if not exists
    if [ ! -f "$INSTALL_DIR/config.yaml" ]; then
        if [ -f "$INSTALL_DIR/config.yaml.example" ]; then
            cp "$INSTALL_DIR/config.yaml.example" "$INSTALL_DIR/config.yaml"
            warn "Created config.yaml from example. Please edit $INSTALL_DIR/config.yaml"
        fi
    fi

    # Set permissions
    chown -R "$SERVICE_USER:$SERVICE_USER" "$INSTALL_DIR"

    # Install systemd service if available
    if command -v systemctl &>/dev/null && [ -d /etc/systemd/system ]; then
        info "Installing systemd service..."
        if [ -f "$INSTALL_DIR/auth-gate.service" ]; then
            cp "$INSTALL_DIR/auth-gate.service" /etc/systemd/system/
            systemctl daemon-reload
            systemctl enable auth-gate
            info "Service installed."
        fi
    fi

    echo ""
    echo -e "${GREEN}=== Installation Complete ===${NC}"
    echo ""
    echo "Binary: $INSTALL_DIR/auth-gate"
    echo "Config: $INSTALL_DIR/config.yaml"
    echo ""
    echo "Quick start:"
    echo "  cd $INSTALL_DIR"
    echo "  ./auth-gate"
    echo ""
    if command -v systemctl &>/dev/null; then
        echo "Or with systemd:"
        echo "  systemctl start auth-gate"
        echo "  systemctl status auth-gate"
        echo ""
    fi
    echo "Access admin UI at http://localhost:3000"
}

# Main
check_deps
install
