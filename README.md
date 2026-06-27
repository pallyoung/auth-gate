# Auth Gate

轻量级反向代理网关，提供路由转发、鉴权、可视化配置和系统监控。

## 功能特性

- **路由管理** — 基于 Host + PathPrefix 匹配，支持路径重写、静态文件服务
- **鉴权中间件** — API Key / Bearer Token (JWT) / Basic Auth，按路由粒度配置
- **请求头操作** — 自定义上游请求头和响应头
- **系统监控** — 实时 CPU、内存、goroutines、uptime 指标仪表盘
- **访问日志** — 带过滤、聚合视图（按路由/状态码分组）和自动清理
- **自定义错误页** — 友好的 404 / 502 页面，浏览器访问时自动展示
- **TLS/HTTPS** — 内置本地 CA，支持自动签发证书
- **配置热更新** — 无需重启即时生效
- **多语言** — 支持中文和英文界面

## 安装

### Linux / macOS

```bash
curl -fsSL https://raw.githubusercontent.com/pallyoung/auth-gate/main/install.sh | sudo bash
```

安装脚本会自动：
- 下载最新版本到 `/opt/auth-gate`
- 创建 `auth-gate` 系统用户
- 安装 systemd 服务（如可用）
- 启动服务并设置开机自启

**环境变量可选配置：**
```bash
AUTH_GATE_VERSION=v0.0.5          # 指定版本，默认 latest
AUTH_GATE_INSTALL_DIR=/usr/local  # 自定义安装目录
AUTH_GATE_USER=myuser             # 自定义服务用户
```

### Windows

```powershell
# PowerShell 中执行
irm https://raw.githubusercontent.com/pallyoung/auth-gate/main/install.ps1 | iex
```

安装到 `%LOCALAPPDATA%\auth-gate`，可通过参数自定义：
```powershell
.\install.ps1 -InstallDir "C:\auth-gate" -Version v0.0.5
```

### Docker

```bash
# 使用 docker-compose
docker compose up -d

# 或直接运行
docker run -d \
  -p 8080:8080 -p 9000:9000 \
  -v auth-gate-data:/app/data \
  pallyoung/auth-gate
```

### 手动编译

```bash
git clone https://github.com/pallyoung/auth-gate.git
cd auth-gate
make build
# 产出: packages/server/bin/auth-gate
```

## 开发

```bash
# 前置要求: Go 1.25+, Node.js 18+

# 安装前端依赖
cd packages/web && npm install && cd ../..

# 后端热重载开发
cd packages/server && make dev

# 或使用根目录脚本（前端+后端）
make dev
```

## 配置

配置文件 `config.yaml`：

```yaml
server:
  listen:
    - addr: ":80"
    - addr: ":443"
      tls: true
  admin:
    addr: "0.0.0.0:9000"    # 管理面板端口

database:
  path: "./data"             # SQLite 数据目录

auth:
  jwt_secret: ""             # 留空则自动生成
  bootstrap_admin_password: "change-this"
```

## 首次登录

- 管理面板地址：`http://<your-ip>:9000`
- 用户名：`admin`
- 密码：配置文件中 `auth.bootstrap_admin_password` 的值
- 未配置时，服务会在启动日志中生成并打印一次性密码

## 技术栈

| 层级 | 技术 |
|------|------|
| 后端 | Go 1.25, Gin, SQLite, JWT |
| 前端 | React 18, TypeScript, Vite |
| 部署 | 单二进制 + 静态资源嵌入 |

## License

MIT
