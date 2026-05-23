#!/bin/bash
set -euo pipefail

PROJECT_ROOT="$(cd "$(dirname "$0")/.." && pwd)"

# Color output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

check_command() {
    if command -v "$1" &> /dev/null; then
        return 0
    else
        return 1
    fi
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
        if command -v apt-get &> /dev/null; then
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
else
    echo -e "${RED}✗ npm 未安装${NC}"
    install_nodejs
fi

# Check Go
if check_command go; then
    echo -e "${GREEN}✓${NC} Go: $(go version | awk '{print $3}')"
else
    install_go
fi

echo ""
echo "=== 构建项目 ==="

# Build frontend
echo "[1/2] Building frontend..."
(
    cd "$PROJECT_ROOT/packages/web"
    npm ci --include=dev
    npm run build
)

# Build server
echo "[2/2] Building server..."
(
    cd "$PROJECT_ROOT/packages/server"
    go build -o bin/auth-gate ./cmd/server
)

echo -e "${GREEN}构建完成: $PROJECT_ROOT/packages/server/bin/auth-gate${NC}"