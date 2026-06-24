# Auth Gate 安装指南

## 一键安装（推荐）

### Linux / macOS

```bash
curl -fsSL https://raw.githubusercontent.com/pallyoung/auth-gate/main/install.sh | sudo bash

# 或指定版本
curl -fsSL https://raw.githubusercontent.com/pallyoung/auth-gate/main/install.sh | sudo AUTH_GATE_VERSION=v1.0.0 bash
```

### Windows (PowerShell)

```powershell
# 下载安装包
Invoke-WebRequest -Uri "https://github.com/pallyoung/auth-gate/releases/latest/download/auth-gate-windows-amd64.zip" -OutFile "auth-gate.zip"

# 解压
Expand-Archive -Path "auth-gate.zip" -DestinationPath "auth-gate"
cd auth-gate-*

# 运行安装脚本
.\install.ps1
```

## 手动安装

### 1. 下载安装包

**Linux / macOS:**
```bash
# 下载最新版本
curl -fsSL https://github.com/pallyoung/auth-gate/releases/latest/download/auth-gate-linux-amd64.tar.gz -o auth-gate.tar.gz

# 解压
tar -xzf auth-gate.tar.gz
cd auth-gate-*
```

**Windows:**
```powershell
# 下载最新版本
Invoke-WebRequest -Uri "https://github.com/pallyoung/auth-gate/releases/latest/download/auth-gate-windows-amd64.zip" -OutFile "auth-gate.zip"

# 解压
Expand-Archive -Path "auth-gate.zip" -DestinationPath "auth-gate"
cd auth-gate-*
```

### 2. 运行安装脚本

**Linux / macOS:**
```bash
sudo ./install.sh
```

**Windows:**
```powershell
.\install.ps1
```

### 3. 配置

```bash
sudo nano /opt/auth-gate/config.yaml
```

主要配置项：

```yaml
server:
  addr: ":8080"  # 监听端口

database:
  path: "./data/auth-gate.db"  # 数据库路径

auth:
  jwt_secret: "your-secret-key-here"  # JWT 密钥（必须修改）
  bootstrap_admin_password: "your-password"  # 管理员密码（必须修改）
```

### 4. 启动服务

```bash
# 使用 systemd（推荐）
sudo systemctl start auth-gate
sudo systemctl status auth-gate

# 或直接运行
cd /opt/auth-gate
./auth-gate
```

### 5. 访问管理界面

打开浏览器访问：http://localhost:8080/_authgate

默认账号：
- 用户名：`admin`
- 密码：配置文件中设置的 `bootstrap_admin_password`

## 自定义安装

### 修改安装目录

```bash
curl -fsSL https://raw.githubusercontent.com/pallyoung/auth-gate/main/install.sh | sudo AUTH_GATE_INSTALL_DIR=/usr/local/auth-gate bash
```

### 修改服务用户

```bash
curl -fsSL https://raw.githubusercontent.com/pallyoung/auth-gate/main/install.sh | sudo AUTH_GATE_USER=myuser bash
```

## 构建发布包

```bash
# 构建当前平台的发布包
make release

# 或指定版本
./scripts/release-build.sh v1.0.0
```

产出位置：`dist/release/`

## 目录结构

安装后的目录结构：

```
/opt/auth-gate/
├── auth-gate           # 主程序
├── config.yaml         # 配置文件
├── config.yaml.example # 配置示例
├── auth-gate.service   # systemd 服务文件
├── web/                # 前端静态文件
│   ├── index.html
│   └── assets/
└── data/               # 数据目录（自动创建）
    └── auth-gate.db
```

## 常用命令

### Linux / macOS (systemd)

```bash
# 查看服务状态
sudo systemctl status auth-gate

# 查看日志
sudo journalctl -u auth-gate -f

# 重启服务
sudo systemctl restart auth-gate

# 停止服务
sudo systemctl stop auth-gate

# 开机自启
sudo systemctl enable auth-gate
```

### Windows

```powershell
# 直接运行
cd $env:LOCALAPPDATA\auth-gate
.\auth-gate.exe

# 后台运行（使用 nssm 或类似工具）
nssm install auth-gate "$env:LOCALAPPDATA\auth-gate\auth-gate.exe"
nssm start auth-gate

# 查看日志
Get-Content "$env:LOCALAPPDATA\auth-gate\logs\*.log" -Tail 50
```

## 卸载

```bash
# 停止服务
sudo systemctl stop auth-gate
sudo systemctl disable auth-gate

# 删除服务文件
sudo rm /etc/systemd/system/auth-gate.service
sudo systemctl daemon-reload

# 删除安装目录
sudo rm -rf /opt/auth-gate

# 删除用户
sudo userdel auth-gate
```
