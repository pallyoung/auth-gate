# Auth Gate

自研 API 网关服务，提供路由转发、鉴权和可视化配置能力。

## 功能特性

- **路由管理**: 基于 Host + PathPrefix 的路由匹配，支持路径重写
- **鉴权中间件**: 支持 API Key、Bearer Token (JWT)、Basic Auth
- **可视化配置**: Web UI 管理路由和鉴权规则
- **配置热更新**: 无需重启即可生效
- **数据持久化**: SQLite 存储，自动迁移

## 技术栈

| 层级 | 技术 |
|------|------|
| 后端 | Go 1.18+, Gin, SQLite, JWT |
| 前端 | React 18, TypeScript, Vite |
| 部署 | 单二进制 + 静态资源嵌入 |

## 快速开始

### 前置要求

- Go 1.18+
- Node.js 18+
- npm

### 开发模式

```bash
# 安装前端依赖
cd web && npm install && cd ..

# 启动服务 (自动构建前端)
make run

# 或分别运行
make web-build  # 构建前端
make dev        # 开发模式 (不构建前端)
```

服务启动后访问 http://localhost:8080

### 构建发布

```bash
make build
./bin/auth-gate
```

## 配置

配置文件位于 `configs/config.yaml`:

```yaml
server:
  host: "0.0.0.0"
  port: 8080

database:
  path: "./data/auth-gate.db"

auth:
  jwt_secret: "your-secret-key"
  token_expiry: 24h
```

## 默认账号

- 用户名: `admin`
- 密码: `admin`

## 项目结构

```
auth-gate/
├── cmd/server/          # 程序入口
├── internal/
│   ├── api/            # REST API
│   ├── auth/           # 鉴权模块
│   ├── config/         # 配置加载
│   ├── proxy/          # 反向代理
│   ├── router/         # 路由匹配
│   └── store/          # SQLite 存储
├── web/                # 前端代码
├── configs/            # 配置文件
└── e2e/                # 端到端测试
```

## API 端点

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | /api/auth/login | 登录 |
| GET | /api/routes | 路由列表 |
| POST | /api/routes | 创建路由 |
| PUT | /api/routes/:id | 更新路由 |
| DELETE | /api/routes/:id | 删除路由 |
| GET | /api/auth-rules | 鉴权规则列表 |
| POST | /api/auth-rules | 创建鉴权规则 |
| PUT | /api/auth-rules/:id | 更新鉴权规则 |
| DELETE | /api/auth-rules/:id | 删除鉴权规则 |

## License

MIT
