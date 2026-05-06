# Auth Gate

自研 API 网关服务，提供路由转发、鉴权和可视化配置能力。

## 项目结构

```
auth-gate/
├── packages/
│   ├── server/          # Go 后端服务
│   └── web/             # React 前端
├── scripts/             # 运行脚本
├── docs/                # 文档
└── Makefile             # 根目录构建入口
```

## 功能特性

- **路由管理**: 基于 Host + PathPrefix 的路由匹配，支持路径重写
- **鉴权中间件**: 支持 API Key、Bearer Token (JWT)、Basic Auth
- **可视化配置**: Web UI 管理路由和鉴权规则
- **配置热更新**: 无需重启即可生效
- **数据持久化**: SQLite 存储，自动迁移

## 快速开始

### 前置要求

- Go 1.18+
- Node.js 18+
- npm

### 开发模式

```bash
# 方式一: 使用根目录 Makefile
make dev

# 方式二: 使用脚本
./scripts/dev.sh
```

### 构建发布

```bash
make build
# 产出: packages/server/bin/auth-gate
```

服务启动后访问 http://localhost:8080

## 配置

配置文件位于 `packages/server/configs/config.yaml`:

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

## 技术栈

| 层级 | 技术 |
|------|------|
| 后端 | Go 1.18+, Gin, SQLite, JWT |
| 前端 | React 18, TypeScript, Vite |
| 部署 | 单二进制 + 静态资源嵌入 |

## License

MIT
