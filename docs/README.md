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
# 启动打包产物
make run

# 或手动运行启动脚本
./scripts/run.sh

# 仅启动后端开发服务
cd packages/server && make dev
```

服务启动后访问 http://localhost:8080/_authgate

### 构建发布

```bash
make build
./bin/auth-gate
```

## 配置

配置文件位于 `configs/config.yaml`:

```yaml
server:
  addr: ":8080"

database:
  path: "./data/auth-gate.db"

auth:
  jwt_secret: "your-secret-key"
  bootstrap_admin_password: "change-this-password"
```

## 首次登录

- 用户名: `admin`
- 密码: 使用 `BOOTSTRAP_ADMIN_PASSWORD` 环境变量或 `auth.bootstrap_admin_password`
- 未配置时，服务会生成一次性密码并在启动日志中打印

## 项目结构

```
auth-gate/
├── packages/
│   ├── server/         # Go 后端服务
│   └── web/            # React 前端
├── scripts/            # 构建/部署脚本
├── docs/               # 项目文档
└── e2e/                # 端到端测试
```

## API 端点

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | /_authgate/api/auth/login | 登录 |
| GET | /_authgate/api/routes | 路由列表 |
| POST | /_authgate/api/routes | 创建路由 |
| PUT | /_authgate/api/routes/:id | 更新路由 |
| DELETE | /_authgate/api/routes/:id | 删除路由 |
| GET | /_authgate/api/auth-rules | 鉴权规则列表 |
| POST | /_authgate/api/auth-rules | 创建鉴权规则 |
| PUT | /_authgate/api/auth-rules/:id | 更新鉴权规则 |
| DELETE | /_authgate/api/auth-rules/:id | 删除鉴权规则 |

## License

MIT
