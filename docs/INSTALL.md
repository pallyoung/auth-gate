# Auth Gate 安装指南

## 一键安装（推荐）

### Linux

```bash
curl -fsSL https://raw.githubusercontent.com/pallyoung/auth-gate/main/install.sh | sudo bash

# 或指定版本
curl -fsSL https://raw.githubusercontent.com/pallyoung/auth-gate/main/install.sh | sudo AUTH_GATE_VERSION=v1.0.0 bash
```

### macOS

```bash
curl -fsSL https://raw.githubusercontent.com/pallyoung/auth-gate/main/install.sh | bash

# 或指定版本
curl -fsSL https://raw.githubusercontent.com/pallyoung/auth-gate/main/install.sh | AUTH_GATE_VERSION=v1.0.0 bash
```

> macOS 一键安装**不需要 sudo**。脚本会自动检测系统架构（Intel / Apple Silicon），安装到 `/usr/local/opt/auth-gate/`，配置文件放到 `~/.auth-gate/`。

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

---

## 手动安装

### Linux

#### 1. 下载安装包

```bash
curl -fsSL https://github.com/pallyoung/auth-gate/releases/latest/download/auth-gate-linux-amd64.tar.gz -o auth-gate.tar.gz
tar -xzf auth-gate.tar.gz
cd auth-gate-*
```

#### 2. 安装

```bash
sudo ./install.sh
```

#### 3. 配置

```bash
sudo nano /opt/auth-gate/config.yaml
```

主要配置项：

```yaml
server:
  listen:
    - addr: ":80"
    - addr: ":443"
      tls: true
  admin:
    addr: "0.0.0.0:9000"

database:
  path: "./data"

auth:
  jwt_secret: "your-secret-key-here"
  bootstrap_admin_password: "your-password"
```

#### 4. 启动服务

```bash
# systemd（推荐）
sudo systemctl start auth-gate
sudo systemctl status auth-gate

# 或直接运行
cd /opt/auth-gate
./auth-gate
```

---

### macOS

#### 1. 下载安装包

```bash
# Apple Silicon (M1/M2/M3/M4)
curl -fsSL https://github.com/pallyoung/auth-gate/releases/latest/download/auth-gate-$(gh release view --json tagName -q .tagName)-darwin-arm64.tar.gz -o auth-gate.tar.gz

# 或手动替换版本号
curl -fsSL "https://github.com/pallyoung/auth-gate/releases/latest/download/auth-gate-v1.0.0-darwin-arm64.tar.gz" -o auth-gate.tar.gz

# Intel Mac 用 darwin-amd64
tar -xzf auth-gate.tar.gz
cd auth-gate-*
```

#### 2. 安装

```bash
./install.sh
```

安装后目录结构：

```
/usr/local/opt/auth-gate/
├── auth-gate           # 主程序
├── web/                # 前端静态文件
└── config.yaml.example

~/.auth-gate/
├── config.yaml         # 配置文件（首次安装自动从 example 复制）
└── data/               # 数据目录（自动创建）

/usr/local/bin/auth-gate  # 符号链接 → /usr/local/opt/auth-gate/auth-gate
```

#### 3. 配置

```bash
nano ~/.auth-gate/config.yaml
```

> **提示：** 端口 80/443 需要 root 权限。可以使用 8080/8443 代替，或用 `sudo` 启动。

#### 4. 启动

**前台运行（开发/调试）：**

```bash
auth-gate start -f ~/.auth-gate/config.yaml
```

**后台服务（launchd，推荐）：**

```bash
# 加载服务（开机自动启动）
launchctl load ~/Library/LaunchAgents/com.authgate.server.plist

# 查看状态
launchctl list | grep authgate

# 停止服务
launchctl unload ~/Library/LaunchAgents/com.authgate.server.plist

# 查看日志
tail -f /tmp/auth-gate.log
```

---

### Windows

#### 1. 下载安装包

```powershell
Invoke-WebRequest -Uri "https://github.com/pallyoung/auth-gate/releases/latest/download/auth-gate-windows-amd64.zip" -OutFile "auth-gate.zip"
Expand-Archive -Path "auth-gate.zip" -DestinationPath "auth-gate"
cd auth-gate-*
```

#### 2. 安装

```powershell
.\install.ps1
```

#### 3. 启动

```powershell
cd $env:LOCALAPPDATA\auth-gate
.\auth-gate.exe
```

---

## 访问管理界面

打开浏览器访问：http://localhost:9000

默认账号：
- 用户名：`admin`
- 密码：配置文件中设置的 `bootstrap_admin_password`

---

## 自定义安装

### 修改安装目录（Linux）

```bash
curl -fsSL https://raw.githubusercontent.com/pallyoung/auth-gate/main/install.sh | sudo AUTH_GATE_INSTALL_DIR=/usr/local/auth-gate bash
```

### 修改安装目录（macOS）

```bash
curl -fsSL https://raw.githubusercontent.com/pallyoung/auth-gate/main/install.sh | AUTH_GATE_INSTALL_DIR=~/auth-gate bash
```

### 修改服务用户（Linux）

```bash
curl -fsSL https://raw.githubusercontent.com/pallyoung/auth-gate/main/install.sh | sudo AUTH_GATE_USER=myuser bash
```

---

## 常用命令

### Linux (systemd)

```bash
sudo systemctl status auth-gate    # 查看状态
sudo journalctl -u auth-gate -f    # 查看日志
sudo systemctl restart auth-gate   # 重启
sudo systemctl stop auth-gate      # 停止
sudo systemctl enable auth-gate    # 开机自启
```

### macOS (launchd)

```bash
launchctl list | grep authgate                                      # 查看状态
tail -f /tmp/auth-gate.log                                          # 查看日志
launchctl unload ~/Library/LaunchAgents/com.authgate.server.plist   # 停止
launchctl load ~/Library/LaunchAgents/com.authgate.server.plist     # 启动
```

### Windows

```powershell
Get-Process auth-gate            # 查看进程
Get-Content /tmp/auth-gate.log -Tail 50  # 查看日志
```

---

## 卸载

### Linux

```bash
sudo systemctl stop auth-gate
sudo systemctl disable auth-gate
sudo rm /etc/systemd/system/auth-gate.service
sudo systemctl daemon-reload
sudo rm -rf /opt/auth-gate
sudo userdel auth-gate
```

### macOS

```bash
launchctl unload ~/Library/LaunchAgents/com.authgate.server.plist
rm ~/Library/LaunchAgents/com.authgate.server.plist
rm /usr/local/bin/auth-gate
rm -rf /usr/local/opt/auth-gate
rm -rf ~/.auth-gate
```
