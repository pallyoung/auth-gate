# Auth Gate - 技术规格文档

## 1. 项目概述

**目标**: 自研 API 网关服务，替代 NGINX，提供路由转发、鉴权和可视化配置能力。

**核心价值**:
- 统一入口管理所有后端服务
- 可视化配置路由和鉴权规则
- 统一的访问控制和身份认证


## 1. 项目概述

**目标**: 自研 API 网关服务，替代 NGINX，提供路由转发、鉴权和可视化配置能力。

**核心价值**:
- 统一入口管理所有后端服务
- 可视化配置路由和鉴权规则
- 经一的访问控制和身份认证

---

## 2. 系统架构

```
┌─────────────────────────────────────────────────────────────┐
                        用户请求                               │
└─────────────────────────┬───────────────────────────────────┘
                          ▼
┌─────────────────────────────────────────────────────────────┐
                    Auth Gate 网关                            │
  ┌─────────────┐  ┌─────────────┐  ┌─────────────────────┐   │
  │ 路由匹配    │→ │ 鉴权中间件  │→ │  反向代理/转发      │   │
  └─────────────┘  └─────────────┘  └─────────────────────┘   │
         ▲                                    │               │
         │            ┌───────────────────────┘               │
         ▼            ▼                                       │
  ┌─────────────────────────────┐                             │
  │  配置存储 (SQLite/YAML)     │                             │
  └─────────────────────────────┘                             │
└─────────────────────────────────────────────────────────────┘
                          │
                          ▼
        ┌─────────────────────────────────────┐
        │         后端服务集群                 │
        │  :3000  :8080  :9000  ...          │
        └─────────────────────────────────────┘
```

---

## 3. 技术选型

### 3.1 后端

| 层级 | 技术选型 | 理由 |
------|---------|------|
| 语言 | Go 1.21+ | 高性能、单二进制部署、标准库强大 |
| HTTP 框架 | Gin / Echo | 成熟、中间件生态好 |
| 配置存储 | SQLite + YAML 文件 | 轻量、无外部依赖、可版本控制 |
| 配置热更新 | fsnotify | 监听配置文件变化 |

### 3.2 前端

| 层级 | 技术选型 | 理由 |
------|---------|------|
| 框架 | React 18 + TypeScript | 类型安全、生态成熟 |
| 构建 | Vite | 快速、现代 |
| UI 组件 | shadcn/ui + Tailwind CSS | 美观、可定制 |
| 状态管理 | Zustand | 轻量、简单 |
| HTTP 客户端 | fetch / ky | 原生 / 轻量 |

### 3.3 部署

- 单二进制文件 (Go) + 静态资源嵌入
- Docker 镜像 (可选)
- systemd 服务管理

---

## 4. 核心功能模块

### 4.1 路由转发

**数据模型**:
```go
type Route struct {
    ID          string   `json:"id"`
    Name        string   `json:"name"`
    Host        string   `json:"host"`         // 匹配的域名
    PathPrefix  string   `json:"path_prefix"`  // 路径前缀
    Backend     string   `json:"backend"`      // 后端地址 e.g. "http://127.0.0.1:3000"
    StripPrefix bool     `json:"strip_prefix"` // 是否去掉前缀
    Enabled     bool     `json:"enabled"`
    Priority    int      `json:"priority"`     // 匹配优先级
}
```

**匹配逻辑**:
1. 先按 Host 匹配
2. 再按 PathPrefix 匹配 (最长前缀优先)
3. 支持路径重写

### 4.2 鉴权中间件

**支持的鉴权方式**:

| 类型 | 说明 |
------|------|
| 无鉴权 | 公开接口 |
| API Key | Header 中的固定 Key |
| Bearer Token | JWT 验证 |
| Basic Auth | 用户名密码 |

**数据模型**:
```go
type AuthRule struct {
    ID           string     `json:"id"`
    RouteID      string     `json:"route_id"`      // 关联的路由
    Type         string     `json:"type"`          // none, apikey, bearer, basic
    Config       AuthConfig `json:"config"`        // 鉴权配置
    Whitelist    []string   `json:"whitelist"`     // IP 白名单
    RateLimit    int        `json:"rate_limit"`    // 请求频率限制 (req/min)
}

type AuthConfig struct {
    HeaderName   string `json:"header_name"`   // API Key header 名
    Secret       string `json:"secret"`        // 密钥
    JWKSUrl      string `json:"jwks_url"`      // JWT 公钥地址 (可选)
    Issuer       string `json:"issuer"`        // JWT issuer
    Audience     string `json:"audience"`      // JWT audience
}
```

### 4.3 可视化配置面板

**页面结构**:
```
/dashboard
├── /routes        # 路由管理
│   ├── 列表
│   ├── 新增/编辑
│   └── 删除
├── /auth          # 鉴权规则
│   ├── 列表
│   └── 配置
├── /monitor       # 监控面板 (MVP 可暂缓)
│   └── 请求统计
└── /settings      # 系统设置
    ├── 管理员账户
    └── 配置导入/导出
```

### 4.4 API 设计

```
# 路由管理
GET    /api/routes          # 列表
POST   /api/routes          # 创建
PUT    /api/routes/:id      # 更新
DELETE /api/routes/:id      # 删除

# 鉴权规则
GET    /api/auth-rules      # 列表
POST   /api/auth-rules      # 创建
PUT    /api/auth-rules/:id  # 更新
DELETE /api/auth-rules/:id  # 删除

# 配置
GET    /api/config/export   # 导出配置
POST   /api/config/import   # 导入配置
POST   /api/config/reload   # 热重载
```

---

## 5. 项目结构

```
auth-gate/
├── cmd/
│   └── server/
│       └── main.go              # 入口
├── internal/
│   ├── config/
│   │   ├── config.go            # 配置加载
│   │   └── watcher.go           # 热更新
│   ├── router/
│   │   ├── router.go            # 路由管理
│   │   └── matcher.go           # 匹配逻辑
│   ├── proxy/
│   │   └── proxy.go             # 反向代理
│   ├── auth/
│   │   ├── auth.go              # 鉴权中间件
│   │   ├── apikey.go
│   │   ├── bearer.go
│   │   └── basic.go
│   ├── api/
│   │   └── handlers.go          # 管理 API
│   └── store/
│       └── sqlite.go            # 数据存储
├── web/                         # 前端
│   ├── src/
│   │   ├── pages/
│   │   ├── components/
│   │   └── lib/
│   ├── package.json
│   └── vite.config.ts
├── configs/
│   └── config.yaml              # 配置文件
├── Dockerfile
└── go.mod
```

---

## 6. MVP 范围 (第一阶段)

### 必须有
- [x] 路由配置 (Host + PathPrefix 匹配)
- [x] 反向代理转发
- [x] API Key 鉴权
- [x] 基础 Web UI (路由 CRUD)
- [x] SQLite 持久化

### 暂缓
- [ ] JWT 鉴权
- [ ] 请求监控统计
- [ ] 配置版本控制
- [ ] 多用户管理

---

## 7. 后续迭代方向

| 阶段 | 功能 |
------|------|
| V1.1 | JWT 鉴权、请求日志、指标统计 |
| V1.2 | WebSocket 支持、gRPC 转发 |
| V2.0 | 分布式部署、配置同步、服务发现 |

---

## 8. 风险与限制

| 风险 | 缓解措施 |
------|---------|
| 性能不如 NGINX | Go 足够快，单机场景够用 |
| 单点故障 | 单机部署本来就是单点，可后续做 HA |
| 功能不够丰富 | 按 MVP 逐步迭代 |

---

## 9. 时间估算

| 任务 | 预估 |
------|------|
| 后端骨架 + 路由转发 | 2-3 天 |
| API Key 鉴权 | 1 天 |
| SQLite 存储 + API | 1 天 |
| Web UI | 2-3 天 |
| 测试 + 文档 | 1 天 |
| **总计** | **7-9 天** |

---

## 10. 是否开始实现？

请确认以下事项后开始开发：

1. 技术选型是否认可？
2. MVP 范围是否合适？
3. 是否需要调整优先级？
*** End Patch

---

## 13. 用户认证与权限系统

### 13.1 用户管理

**数据模型**:
```go
type User struct {
    ID           string    `json:"id"`
    Username     string    `json:"username"`
    PasswordHash string    `json:"-"`
    Role         string    `json:"role"` // admin, editor, viewer
    Enabled      bool      `json:"enabled"`
}
```

**角色权限**:
| 角色 | 路由管理 | 鉴权规则 | 用户管理 |
|------|---------|---------|---------|
| admin | ✓ CRUD | ✓ CRUD | ✓ CRUD |
| editor | ✓ CRUD | ✓ CRUD | ✗ |
| viewer | ✓ Read | ✓ Read | ✗ |

### 13.2 JWT 认证

- Token 有效期: 24小时
- Header: `Authorization: Bearer <token>`
- 登录端点: `POST /api/auth/login`

### 13.3 API 端点

```
POST   /api/auth/login          # 登录
POST   /api/auth/logout         # 登出
GET    /api/auth/me             # 当前用户信息

GET    /api/users              # 列表 (admin)
POST   /api/users              # 创建 (admin)
PUT    /api/users/:id          # 更新 (admin)
DELETE /api/users/:id          # 删除 (admin)
```

### 13.4 默认用户

- 用户名: `admin`
- 密码: `admin`
- 角色: `admin`

---

## 14. 技术栈

### 后端
- Go 1.18+
- Gin 框架
- SQLite 数据库
- JWT (golang-jwt/jwt/v5)
- bcrypt 密码加密

### 前端
- React 18
- TypeScript
- Vite
- CSS Variables 设计系统
- Lucide Icons

---

## 13. 用户认证与权限系统

### 13.1 用户管理

**数据模型**:
- `User`: id, username, password_hash, role, enabled, created_at, updated_at

**角色权限**:
- `admin`: 全部权限
- `editor`: 路由和鉴权规则管理
- `viewer`: 只读权限

### 13.2 JWT 认证

- Token 有效期: 24小时
- Header: `Authorization: Bearer <token>`
- 登录端点: `POST /api/auth/login`

### 13.3 API 端点

```
POST   /api/auth/login          # 登录
POST   /api/auth/logout         # 登出
GET    /api/auth/me             # 当前用户信息
GET    /api/users              # 列表 (admin)
POST   /api/users              # 创建 (admin)
PUT    /api/users/:id          # 更新 (admin)
DELETE /api/users/:id          # 删除 (admin)
```

### 13.4 默认用户

- 用户名: `admin`
- 密码: `admin`
- 角色: `admin`

---

## 14. 技术栈

### 后端
- Go 1.18+
- Gin 框架
- SQLite 数据库
- JWT (golang-jwt/jwt/v5)
- bcrypt 密码加密

### 前端
- React 18
- TypeScript
- Vite
- CSS Variables 设计系统
- Lucide Icons
