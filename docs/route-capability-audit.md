# Auth-Gate 路由能力审计：对标 nginx

**目标：** 明确当前与 nginx 的能力差距，建立完整的路由特性矩阵，作为后续实现的基线。

**审计时间：** 2026-05-23
**代码版本：** commit 1d32710 ("Refactor auth flows and add route access control")

---

## 一、当前实现现状

### 1.1 路由模型（router/router.go）

```go
type Route struct {
    ID            string
    Name          string
    Host          string      // 空=匹配所有主机
    PathPrefix    string      // 前缀路径（支持 nginx 语法前缀）
    PathMatchMode string      // "prefix"|"exact"|"stop"|"regex"|"regex_i"
    PathRegex     *regexp.Regexp // 编译后的正则
    RewriteTarget string      // rewrite 目标路径
    RedirectCode  int         // 301|302 外跳重定向
    Backend       string      // 单个后端 URL
    Backends      []Backend   // 多后端负载均衡
    StripPrefix   bool        // 是否去掉 PathPrefix
    Enabled       bool
    Priority      int         // 数值越大优先级越高
    AuthRule      *AuthRule
}
```

### 1.2 数据库 Schema（store/sqlite.go）

```
routes       (id, name, host, path_prefix, backend, strip_prefix, enabled, priority, ...)
             + path_match_mode, rewrite_target, redirect_code
             + backends (JSON), cert_path, key_path, tls_enabled
auth_rules   (id, route_id, type, config, whitelist, rate_limit, burst, ...)
users
user_route_access
```

### 1.3 反向代理核心逻辑（proxy/proxy.go）

- 使用 `httputil.NewSingleHostReverseProxy`
- 路径前缀剥离（StripPrefix）
- 路径重写（rewrite_target，正则捕获组替换）
- 301/302 外跳重定向
- 转发 Header：`X-Forwarded-Host`, `X-Forwarded-Proto`, `X-Forwarded-For`
- 鉴权链：gateway token → auth.Check → (basic | bearer | none)
- 多后端加权轮询（weighted round-robin）
- 熔断器（circuit breaker）：closed → open → half-open 自动恢复
- 重试机制（retryTransport）
- WebSocket 代理（handleWebSocket hijack）
- 速率限制（Token Bucket，per-route 独立 limiter）
- 结构化访问日志（JSON → stdout）
- Prometheus metrics（5 个指标 + GET /metrics）

---

## 二、特性矩阵对照表

> **符号说明：** ✅ 已实现 | ❌ 未实现 | ⚠️ 部分实现 | 🏗️ 数据库有字段但代码未用

| 维度 | 具体特性 | nginx 等效指令 | 当前状态 | 优先级 |
|------|---------|--------------|---------|--------|
| **路由匹配** | 前缀路径匹配 | `location /path` | ✅ | P0 |
| | 精确路径匹配 | `location = /path` | ✅ stage-6 | P1 |
| | 正则路径匹配 | `location ~ /api/v\d+` | ✅ stage-6 | P1 |
| | 大小写敏感/不敏感 | `location ~*` | ✅ stage-6 | P1 |
| | 主机名匹配 | `server_name` | ✅ | P0 |
| | 按优先级排序 | `location` 评估顺序 | ✅ stage-6 | P0 |
| **路径操作** | 前缀剥离（已有） | `proxy_redirect off` + rewrite | ✅ | P0 |
| | Rewrite 目标路径 | `rewrite ^/old(.*) /new$1 break` | ✅ stage-6 | P1 |
| | 301/302 重定向 | `rewrite ... redirect/permanent` | ✅ stage-6 | P1 |
| **反向代理** | 单后端（已有） | `proxy_pass` | ✅ | P0 |
| | 多后端负载均衡 | `upstream {}` | ✅ | P0 |
| | round-robin | `round_robin`（默认） | ✅ | P0 |
| | 加权轮询 | `weight=N` | ✅ | P0 |
| | least_conn | `least_conn` | ❌ | P2 |
| | 后端重试 | `proxy_next_upstream` | ✅ | P0 |
| | upstream keepalive | `keepalive` | ❌ | P2 |
| **TLS** | TLS 终止 | `ssl_certificate` | ✅ stage-8 | P0 |
| | 证书管理 | `ssl_certificate_key` | ✅ stage-8 | P0 |
| | HTTP → HTTPS 重定向 | `return 301 https://` | ❌ | P1 |
| | 双向 TLS（mTLS） | `ssl_client_certificate` | ❌ | P2 |
| **超时控制** | 连接超时 | `proxy_connect_timeout` | ✅ | P0 |
| | 读取超时 | `proxy_read_timeout` | ✅ | P0 |
| | 写入超时 | `proxy_send_timeout` | ✅ stage-8 | P2 |
| | 自定义 Dial timeout | — | ✅ | P0 |
| **连接处理** | WebSocket 支持 | `proxy_http_version 1.1` + upgrade | ✅ stage-6 | P1 |
| | Server-Sent Events | 长连接保持 | ✅ stage-6 | P1 |
| | 连接升级 | `Upgrade` header | ✅ | P1 |
| | 连接池/keepalive | `keepalive_timeout` | ❌ | P2 |
| **Header 处理** | X-Forwarded-* | `proxy_set_header` | ✅ | P0 |
| | X-Real-IP | `proxy_set_header X-Real-IP` | ✅ stage-8 | P2 |
| | 去除 hop-by-hop | — | ❌ | P2 |
| **认证/鉴权** | 无认证 | — | ✅ | P0 |
| | Basic Auth | `auth_basic` | ✅ | P0 |
| | Bearer Token | 自定义 | ✅ | P0 |
| | Gateway Token | 自定义 | ✅ | P0 |
| | IP 白名单 | `allow/deny` | ✅ stage-7 | P2 |
| **限流** | 请求速率限制 | `limit_req` | ✅ stage-7 | P2 |
| | Burst | `burst=N` | ✅ stage-7 | P2 |
| **健康检查** | 被动健康检查 | `proxy_next_upstream error timeout` | ✅ | P0 |
| | 熔断（动态摘除） | — | ✅ stage-6 | P0 |
| **可观测性** | 访问日志 | `access_log` | ✅ stage-7 | P1 |
| | 错误日志（已有） | `error_log` | ✅ | P0 |
| | Prometheus metrics | — | ✅ stage-7 | P2 |
| | Request ID 追踪 | — | ✅ stage-7 | P1 |
| **其他** | 缓冲控制 | `proxy_buffering` | ❌ | P2 |
| | 压缩 | `gzip` | ❌ | P2 |
| | CORS header | — | ✅ stage-8 | P1 |

---

## 三、Stage 完成记录

| Stage | 内容 | 状态 | 关键文件 |
|-------|------|------|---------|
| stage-1 | 基础路由 + 鉴权 | ✅ | router/router.go, proxy/proxy.go |
| stage-2 | WebSocket 修复 | ✅ | proxy/proxy.go handleWebSocket |
| stage-3 | 多后端 + 负载均衡 | ✅ | proxy/proxy.go balancer |
| stage-4 | 后端超时控制 | ✅ | proxy/proxy.go dialTimeout/readTimeout |
| stage-5 | 后端重试机制 | ✅ | proxy/proxy.go retryTransport |
| stage-6 | nginx 路由优先级 + rewrite | ✅ | router/compiler.go, router/router.go, store/models.go |
| stage-7 P0 | Prometheus Metrics | ✅ | internal/metrics/collector.go, internal/proxy/proxy.go |
| stage-7 P1 | Token Bucket 限流 | ✅ | internal/middleware/ratelimit.go |
| stage-7 P1 | 访问日志 | ✅ | internal/proxy/proxy.go accessLogEntry |
| stage-7 P2 | GET /metrics 端点 | ✅ | internal/http/admin/routes.go |
| stage-8 P0 | X-Real-IP Header | ✅ | internal/proxy/proxy.go |
| stage-8 P0 | WriteTimeout 后端写入超时 | ✅ | internal/proxy/proxy.go writeTimeoutTransport |
| stage-8 P0 | TLS Termination | ✅ | cmd/server/main.go |
| stage-8 P1 | CORS Middleware（per-route） | ✅ | internal/proxy/proxy.go handleCORS |

---

## 四、缺失特性实现复杂度评估

### 低复杂度（1-2 天）

| 特性 | 说明 | 影响 |
|------|------|------|
| X-Real-IP header | 补充转发真实 IP | 调试友好 |
| 精确路径匹配 | `=` 前缀支持 | 精确路由 |
| 被动健康检查（基础） | 502/503 后端错误计数 | 稳定性 |

### 中复杂度（3-5 天）

| 特性 | 说明 | 影响 |
|------|------|------|
| TLS Termination | 证书加载 + HTTPS Server | HTTPS 入口 |
| Rewrite 规则 | 301/302/重写目标路径 | 迁移兼容 |
| CORS header | 自动 CORS | 前端兼容 |
| 后端写入超时 | proxy_send_timeout | 稳定性 |

### 高复杂度（1-2 周）

| 特性 | 说明 | 影响 |
|------|------|------|
| upstream keepalive | 连接池复用 | 性能 |
| least_conn | 最少连接负载策略 | 高可用 |
| 双向 TLS（mTLS） | 客户端证书验证 | 安全 |
| 缓冲控制 | proxy_buffering | 性能调优 |
| 压缩 | gzip | 性能调优 |

---

## 五、建议下一步（Stage 8）

### 🔴 P0 — 必须实现（影响生产可用性）

1. **TLS Termination** — HTTPS 是现代互联网的最低要求
2. **X-Real-IP** — 补充真实 IP 传递 header
3. **CORS Header** — 前端兼容性

### 🟡 P1 — 重要（影响完整性和可观测性）

4. **后端写入超时（WriteTimeout）** — 防止慢客户端占用后端
5. **least_conn 负载策略** — 高并发场景优化

### 🟢 P2 — 增强（生产级特性）

6. **upstream keepalive** — 连接复用提升性能
7. **proxy_buffering** — 大响应体性能调优
8. **gzip 压缩** — 带宽优化

---

## 六、Stage 8 完成记录

| 项目 | 内容 | 状态 | 关键文件 |
|------|------|------|---------|
| P0 | X-Real-IP Header | ✅ | internal/proxy/proxy.go |
| P0 | WriteTimeout 后端写入超时 | ✅ | internal/proxy/proxy.go writeTimeoutTransport |
| P1 | CORS Middleware（per-route） | ✅ | internal/proxy/proxy.go handleCORS |
| P0 | TLS Termination | ✅ 已实现于 main.go | cmd/server/main.go |

### Stage 8 备注

**TLS Termination**：main.go 中已实现 `startHTTPServers`，支持：
- 启动时按 (host, cert, key) 分组路由，合并为同一 TLS listener
- 每个分组加载 X509 证书对，无报错时启动 HTTPS server
- 路由级 TLSEnabled + TLSCert/TLSKey 配置
- HTTP 流量走 Gin 默认 HTTP server（端口 8080）
- 需要在路由层面配置 `tls_enabled=true` + 有效证书路径

**CORS Middleware**：已实现 per-route CORS，配置在 AuthRule 中：
- `cors_allowed_origins`：逗号分隔的 origin 列表，支持 `.example.com` 通配符子域名
- `cors_allowed_methods`：逗号分隔，默认 GET,POST,PUT,DELETE,PATCH,OPTIONS
- `cors_allowed_headers`：逗号分隔，默认 Authorization,Content-Type,X-Requested-With
- `cors_allow_credentials`：true 时设置 Access-Control-Allow-Credentials
- `cors_max_age`：OPTIONS 预检缓存时间（秒），默认 86400
- 预检请求（OPTIONS）直接 204 返回，不触发后端

**X-Real-IP**：proxy.go Director 中设置 `X-Real-IP: c.ClientIP()`，WebSocket 同。

**WriteTimeout**：通过 `writeTimeoutTransport`（per-request context deadline）实现，比 Transport.WriteTimeout 更精确。

---

## 七、剩余缺失特性（按优先级）

### P0 — 必须实现（TLS 已完成）

已全部完成。

### P1 — 重要

| 特性 | 说明 | 备注 |
|------|------|------|
| least_conn 负载策略 | 按最少连接数分发 | P2 优先级 |

### P2 — 增强（生产级特性）

| 特性 | 说明 |
|------|------|
| upstream keepalive | 连接池复用 |
| proxy_buffering | 大响应体性能调优 |
| gzip 压缩 | 带宽优化 |
| 去除 hop-by-hop header | 安全性增强 |

---

## 八、审计结论

当前 auth-gate 的路由能力已覆盖企业级反向代理核心场景：

**✅ 已实现（vs nginx）：**
- 路由匹配：主机 + 前缀 + 精确 + 正则 + 大小写不敏感 + 优先级
- 反向代理：单后端 + 多后端加权轮询 + 后端重试
- TLS Termination：证书管理 + HTTPS Server
- 超时控制：连接超时 + 读取超时 + 写入超时
- 连接处理：WebSocket + SSE + 连接升级
- Header 处理：X-Forwarded-* + X-Real-IP
- 认证鉴权：none + Basic + Bearer + Gateway + IP 白名单
- 限流：Token Bucket + Burst
- 熔断：被动健康检查 + 动态摘除
- 可观测性：Prometheus metrics + 结构化访问日志 + Request ID
- CORS：per-route 自动 CORS 处理
- rewrite：正则捕获 + 301/302 外跳重定向

**❌ 剩余差距：**
- least_conn 负载策略（P2）
- upstream keepalive（P2）
- proxy_buffering（P2）
- gzip 压缩（P2）
- mTLS 双向证书验证（P2）