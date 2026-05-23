#!/bin/bash
set -euo pipefail

PROJECT_ROOT="$(cd "$(dirname "$0")/.." && pwd)"
DIST_DIR="$PROJECT_ROOT/dist"
WEB_DIST_DIR="$DIST_DIR/web"

# Color output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

check_command() {
    command -v "$1" &> /dev/null
}

install_nodejs() {
    echo -e "${YELLOW}Node.js 未安装，正在安装...${NC}"
    export HOMEBREW_NO_AUTO_UPDATE=1
    export HOMEBREW_NO_ENV_HINTS=1
    if [[ "$OSTYPE" == "darwin"* ]]; then
        if command -v brew &> /dev/null; then
            brew install node
        else
            echo -e "${RED}Homebrew 未安装。请从 https://brew.sh 安装 Homebrew 后再运行此脚本${NC}"
            exit 1
        fi
    elif [[ "$OSTYPE" == "linux-gnu"* ]]; then
        if command -v apt-get apt-get &> /dev/null; then
            sudo apt-get update && sudo apt-get install -y nodejs npm
        elif command -v yum &> /dev/null; then
            sudo yum install -y nodejs npm
        elif command -v apk &> /dev/null; then
            sudo apk add --no-cache nodejs npm
        else
            echo -e "${RED}请手动安装 Node.js: https://nodejs.org/${NC}"
            exit 1
        fi
    else
        echo -e "${RED}请手动安装 Node.js: https://nodejs.org/${NC}"
        exit 1
    fi
}

install_go() {
    echo -e "${YELLOW}Go 未安装，正在安装...${NC}"
    export HOMEBREW_NO_AUTO_UPDATE=1
    export HOMEBREW_NO_ENV_HINTS=1
    if [[ "$OSTYPE" == "darwin"* ]]; then
        if command -v brew &> /dev/null; then
            brew install go
        else
            echo -e "${RED}Homebrew 未安装。请从 https://brew.sh 安装 Homebrew 后再运行此脚本${NC}"
            exit 1
        fi
    elif [[ "$OSTYPE" == "linux-gnu"* ]]; then
        if command -v apt-get &> /dev/null; then
            sudo apt-get update && sudo apt-get install -y golang-go
        elif command -v yum &> /dev/null; then
            sudo yum install -y golang
        else
            echo -e "${RED}请手动安装 Go: https://go.dev/dl/${NC}"
            exit 1
        fi
    else
        echo -e "${RED}请手动安装 Go: https://go.dev/dl/${NC}"
        exit 1
    fi
}

# Kill existing auth-gate processes using lsof on port 8080
kill_port_8080() {
    local pids=$(lsof -ti:8080 2>/dev/null || true)
    if [ -n "$pids" ]; then
        echo -e "${YELLOW}Killing process on port 8080: $pids${NC}"
        echo "$pids" | xargs kill -9 2>/dev/null || true
        sleep 1
    fi
}

echo "=== Auth Gate Deploy ==="
echo ""

echo "=== 检查环境 ==="

# Check Node.js
if check_command node; then
    echo -e "${GREEN}✓${NC} Node.js: $(node --version)"
else
    install_nodejs
fi

# Check npm
if check_command npm; then
    echo -e "${GREEN}✓${NC} npm: $(npm --version)"
fi

# Check Go
if check_command go; then
    echo -e "${GREEN}✓${NC} Go: $(go version | awk '{print $3}')"
else
    install_go
fi

echo ""

# Build
echo "[1/4] Building..."
(
    cd "$PROJECT_ROOT/packages/web"
    npm ci --include=dev
    npm run build
)

(
    cd "$PROJECT_ROOT/packages/server"
    go build -o bin/auth-gate ./cmd/server
)

# Create dist directory (clean first to avoid stale files)
echo "[2/4] Copying to dist..."
rm -rf "$DIST_DIR"
mkdir -p "$DIST_DIR"

# Copy files to dist
cp "$PROJECT_ROOT/packages/server/bin/auth-gate" "$DIST_DIR/"
cp "$PROJECT_ROOT/packages/server/configs/config.yaml" "$DIST_DIR/"
cp -r "$PROJECT_ROOT/packages/web/dist" "$WEB_DIST_DIR"

# Kill existing process on port 8080
echo "[3/4] Cleaning up old process..."
kill_port_8080

# Start service in background
echo "[4/4] Starting service..."
cd "$DIST_DIR"
./auth-gate > auth-gate.log 2>&1 &
AUTH_GATE_PID=$!
sleep 2

# Check if running
if kill -0 $AUTH_GATE_PID 2>/dev/null; then
    echo ""
    echo -e "${GREEN}✓ Auth Gate 已启动 (PID: $AUTH_GATE_PID, 端口 8080)${NC}"
    echo "日志文件: $DIST_DIR/auth-gate.log"
    echo "查看日志: tail -f $DIST_DIR/auth-gate.log"
else
    echo -e "${RED}✗ 启动失败，查看日志: cat $DIST_DIR/auth-gate.log${NC}"
    cat auth-gate.log
    exit 1
fi