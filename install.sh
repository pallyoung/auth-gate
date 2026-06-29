#!/bin/bash
set -euo pipefail

# Auth Gate Installer
# Usage: curl -fsSL https://raw.githubusercontent.com/pallyoung/auth-gate/main/install.sh | bash
#
# Supported environment variables:
#   AUTH_GATE_VERSION          - Version to install (default: latest)
#   AUTH_GATE_INSTALL_DIR      - Override install directory
#   AUTH_GATE_USER             - Linux only: service user name (default: auth-gate)

REPO="pallyoung/auth-gate"
VERSION="${AUTH_GATE_VERSION:-latest}"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

info() { echo -e "${GREEN}[INFO]${NC} $1"; }
warn() { echo -e "${YELLOW}[WARN]${NC} $1"; }
error() { echo -e "${RED}[ERROR]${NC} $1"; exit 1; }

# Detect OS: linux or darwin
detect_os() {
    local os
    os=$(uname -s)
    case "$os" in
        Linux*)  echo "linux" ;;
        Darwin*) echo "darwin" ;;
        *)       error "Unsupported OS: $os" ;;
    esac
}

# Detect architecture: amd64 or arm64
detect_arch() {
    local arch
    arch=$(uname -m)
    case "$arch" in
        x86_64|amd64)   echo "amd64" ;;
        arm64|aarch64)   echo "arm64" ;;
        *)               error "Unsupported architecture: $arch" ;;
    esac
}

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

# ── Linux install ────────────────────────────────────────────────────
install_linux() {
    local version="$1" tmp_dir="$2" pkg_dir="$3"
    local install_dir="${AUTH_GATE_INSTALL_DIR:-/opt/auth-gate}"
    local service_user="${AUTH_GATE_USER:-auth-gate}"

    # Must be root for Linux install
    if [ "$EUID" -ne 0 ]; then
        error "Please run as root (use sudo)"
    fi

    # Create service user
    if ! id "$service_user" &>/dev/null; then
        info "Creating user $service_user..."
        useradd -r -s /bin/false -d "$install_dir" "$service_user"
    fi

    # Install files
    info "Installing to $install_dir..."
    mkdir -p "$install_dir"
    cp -r "$pkg_dir"/* "$install_dir/"
    chmod +x "$install_dir/auth-gate"

    # Create config if not exists
    if [ ! -f "$install_dir/config.yaml" ]; then
        if [ -f "$install_dir/config.yaml.example" ]; then
            cp "$install_dir/config.yaml.example" "$install_dir/config.yaml"
            warn "Created config.yaml from example. Please edit $install_dir/config.yaml"
        fi
    fi

    chown -R "$service_user:$service_user" "$install_dir"

    # Install systemd service
    if command -v systemctl &>/dev/null && [ -d /etc/systemd/system ]; then
        if [ -f "$install_dir/auth-gate.service" ]; then
            info "Installing systemd service..."
            cp "$install_dir/auth-gate.service" /etc/systemd/system/
            systemctl daemon-reload
            systemctl enable auth-gate
            info "Service installed and enabled."
        fi
    fi

    echo ""
    echo -e "${GREEN}=== Installation Complete ===${NC}"
    echo ""
    echo "Binary:  $install_dir/auth-gate"
    echo "Config:  $install_dir/config.yaml"
    echo ""
    echo "Quick start:"
    echo "  cd $install_dir && ./auth-gate"
    echo ""
    if command -v systemctl &>/dev/null; then
        echo "Or with systemd:"
        echo "  systemctl start auth-gate"
        echo "  systemctl status auth-gate"
        echo ""
    fi
    echo "Admin UI: http://localhost:9000"
}

# ── macOS install ────────────────────────────────────────────────────
install_darwin() {
    local version="$1" tmp_dir="$2" pkg_dir="$3"
    local install_dir="${AUTH_GATE_INSTALL_DIR:-/usr/local/opt/auth-gate}"
    local bin_link="/usr/local/bin/auth-gate"
    local config_dir="$HOME/.auth-gate"

    info "Installing to $install_dir..."

    # Create directories
    mkdir -p "$install_dir"
    mkdir -p "$config_dir"
    mkdir -p "$HOME/Library/LaunchAgents"

    # Copy binary and web assets
    cp "$pkg_dir/auth-gate" "$install_dir/auth-gate"
    chmod +x "$install_dir/auth-gate"
    cp -r "$pkg_dir/web" "$install_dir/web" 2>/dev/null || true

    # Symlink binary to PATH
    ln -sf "$install_dir/auth-gate" "$bin_link"
    info "Symlinked $bin_link → $install_dir/auth-gate"

    # Copy config to ~/.auth-gate/ if not exists
    if [ ! -f "$config_dir/config.yaml" ]; then
        if [ -f "$pkg_dir/config.yaml.example" ]; then
            cp "$pkg_dir/config.yaml.example" "$config_dir/config.yaml"
            warn "Created $config_dir/config.yaml from example."
            warn "Please edit it before starting: nano $config_dir/config.yaml"
        fi
    fi

    # Install launchd plist
    if [ -f "$pkg_dir/com.authgate.server.plist" ]; then
        info "Installing launchd service..."
        local plist_src="$pkg_dir/com.authgate.server.plist"
        local plist_dest="$HOME/Library/LaunchAgents/com.authgate.server.plist"

        # Template the WorkingDirectory with actual home
        sed "s|__HOME__|$HOME|g" "$plist_src" > "$plist_dest"
        info "Plist installed to $plist_dest"
    fi

    echo ""
    echo -e "${GREEN}=== Installation Complete ===${NC}"
    echo ""
    echo "Binary:  $install_dir/auth-gate"
    echo "Config:  $config_dir/config.yaml"
    echo ""
    echo "Quick start:"
    echo "  auth-gate start -f $config_dir/config.yaml"
    echo ""
    echo "Or install as a background service:"
    echo "  launchctl load ~/Library/LaunchAgents/com.authgate.server.plist"
    echo "  launchctl list | grep authgate      # check status"
    echo "  launchctl unload ~/Library/LaunchAgents/com.authgate.server.plist  # stop"
    echo ""
    echo "Logs: /tmp/auth-gate.log"
    echo "Admin UI: http://localhost:9000"
    echo ""
    warn "Ports 80/443 require root. To use them, either:"
    echo "  1. Run with sudo: sudo auth-gate start -f $config_dir/config.yaml"
    echo "  2. Or change ports in config.yaml to 8080/8443 (no sudo needed)"
}

# ── Main ─────────────────────────────────────────────────────────────
check_deps

OS=$(detect_os)
ARCH=$(detect_arch)
info "Detected: $OS $ARCH"

if [ "$VERSION" = "latest" ]; then
    info "Fetching latest version..."
    VERSION=$(get_latest_version)
    if [ -z "$VERSION" ]; then
        error "Failed to fetch latest version. Check your internet connection."
    fi
fi

info "Installing Auth Gate $VERSION ($OS $ARCH)..."

# Download
tmp_dir=$(mktemp -d)
trap "rm -rf $tmp_dir" EXIT

url="https://github.com/$REPO/releases/download/$VERSION/auth-gate-${VERSION}-${OS}-${ARCH}.tar.gz"
info "Downloading $url ..."
if ! curl -fsSL "$url" -o "$tmp_dir/auth-gate.tar.gz"; then
    error "Failed to download $url"
fi

# Extract
info "Extracting..."
tar -xzf "$tmp_dir/auth-gate.tar.gz" -C "$tmp_dir"

pkg_dir=$(find "$tmp_dir" -maxdepth 1 -type d -name "auth-gate-*" | head -1)
if [ -z "$pkg_dir" ]; then
    error "Package directory not found in archive"
fi

# Platform-specific install
case "$OS" in
    linux)  install_linux "$VERSION" "$tmp_dir" "$pkg_dir" ;;
    darwin) install_darwin "$VERSION" "$tmp_dir" "$pkg_dir" ;;
esac
