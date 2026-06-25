package proxy

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/pallyoung/auth-gate/packages/server/internal/auth"
	httpresponse "github.com/pallyoung/auth-gate/packages/server/internal/http/response"
	"github.com/pallyoung/auth-gate/packages/server/internal/metrics"
	"github.com/pallyoung/auth-gate/packages/server/internal/middleware"
	"github.com/pallyoung/auth-gate/packages/server/internal/router"
	"github.com/pallyoung/auth-gate/packages/server/internal/store"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// balancer holds per-route weighted round-robin state.
type balancer struct {
	backends []store.Backend
	mu       sync.Mutex
	index    int
	cs       []int
}

// circuitBreaker is a per-backend failure tracker that opens (rejects) traffic
// after consecutive failures exceed the threshold, and recovers automatically.
type circuitBreaker struct {
	mu          sync.Mutex
	failures    int
	lastFailure time.Time
	state       int // 0=closed, 1=open, 2=half-open
}

type circuitState struct {
	mu       sync.Mutex
	breakers map[string]*circuitBreaker
}

const (
	circuitFailureThreshold = 5
	circuitRecoveryWindow   = 30 * time.Second
	circuitClosed           = 0
	circuitOpen             = 1
	circuitHalfOpen         = 2
)

// accessLogEntry is a structured JSON log entry written to stdout on every proxied request.
type accessLogEntry struct {
	RequestID        string `json:"request_id"`
	RouteID          string `json:"route_id"`
	Method           string `json:"method"`
	Path             string `json:"path"`
	BackendURL       string `json:"backend_url"`
	BackendLatencyMs int64  `json:"backend_latency_ms"`
	StatusCode       int    `json:"status_code"`
	ClientIP         string `json:"client_ip"`
	UserAgent        string `json:"user_agent"`
	Username         string `json:"username,omitempty"`
	AuthResult       string `json:"auth_result"`
}

var cs *circuitState

// accessWriter delegates to the current os.Stdout on every Write call,
// so tests can redirect os.Stdout after package init and still capture output.
type accessWriter struct{}

func (accessWriter) Write(p []byte) (int, error) {
	return os.Stdout.Write(p)
}

// accessLogger writes structured JSON access logs to stdout (not stderr).
var accessLogger = log.New(accessWriter{}, "", log.LstdFlags)

var (
	errInvalidRuntimeBackend = errors.New("invalid runtime backend")
	errNoBackendConfigured   = errors.New("no backend configured")
)

func init() {
	cs = &circuitState{breakers: map[string]*circuitBreaker{}}
}

// transportKey identifies a reusable base http.Transport by its effective
// timeout and connection-pool settings.
type transportKey struct {
	DialTimeout  time.Duration
	ReadTimeout  time.Duration
	MaxIdleConns int
}

// transportCache pools *http.Transport instances so that identical backend
// configurations reuse the same underlying TCP connection pool.
type transportCache struct {
	mu    sync.Mutex
	cache map[transportKey]*http.Transport
}

func newTransportCache() *transportCache {
	return &transportCache{cache: map[transportKey]*http.Transport{}}
}

func (tc *transportCache) get(key transportKey) *http.Transport {
	tc.mu.Lock()
	defer tc.mu.Unlock()
	if t, ok := tc.cache[key]; ok {
		return t
	}
	maxIdle := key.MaxIdleConns
	if maxIdle <= 0 {
		maxIdle = 2
	}
	t := &http.Transport{
		DialContext:           (&net.Dialer{Timeout: key.DialTimeout}).DialContext,
		ResponseHeaderTimeout: key.ReadTimeout,
		MaxIdleConnsPerHost:   maxIdle,
		ExpectContinueTimeout: 1 * time.Second,
	}
	tc.cache[key] = t
	return t
}

var defaultTransportCache = newTransportCache()

func (cs *circuitState) recordFailure(backendURL string) {
	if backendURL == "" {
		return
	}
	cs.mu.Lock()
	defer cs.mu.Unlock()
	cb, ok := cs.breakers[backendURL]
	if !ok {
		cb = &circuitBreaker{}
		cs.breakers[backendURL] = cb
	}
	cb.mu.Lock()
	defer cb.mu.Unlock()
	cb.failures++
	cb.lastFailure = time.Now()
	if cb.failures >= circuitFailureThreshold {
		cb.state = circuitOpen
		metrics.RecordCircuitState(backendURL, circuitOpen)
	}
}

func (cs *circuitState) recordSuccess(backendURL string) {
	if backendURL == "" {
		return
	}
	cs.mu.Lock()
	defer cs.mu.Unlock()
	cb, ok := cs.breakers[backendURL]
	if !ok {
		return
	}
	cb.mu.Lock()
	defer cb.mu.Unlock()
	if cb.state != circuitClosed {
		cb.state = circuitClosed
		metrics.RecordCircuitState(backendURL, circuitClosed)
	}
	cb.failures = 0
}

func (cs *circuitState) isOpen(backendURL string) bool {
	if backendURL == "" {
		return false
	}
	cs.mu.Lock()
	defer cs.mu.Unlock()
	cb, ok := cs.breakers[backendURL]
	if !ok {
		return false
	}
	cb.mu.Lock()
	defer cb.mu.Unlock()
	switch cb.state {
	case circuitOpen:
		if time.Since(cb.lastFailure) > circuitRecoveryWindow {
			cb.state = circuitHalfOpen
			metrics.RecordCircuitState(backendURL, circuitHalfOpen)
			return false // allow one probe through
		}
		return true
	case circuitHalfOpen:
		return false
	default:
		return false
	}
}

func newBalancer(backends []store.Backend) *balancer {
	if len(backends) == 0 {
		return nil
	}
	cs := make([]int, len(backends))
	w := 0
	for i, b := range backends {
		w += b.Weight
		cs[i] = w
	}
	return &balancer{backends: backends, index: 0, cs: cs}
}

func (b *balancer) pick() (store.Backend, bool) {
	if len(b.backends) == 1 {
		bk := b.backends[0]
		if cs.isOpen(bk.URL) {
			return store.Backend{}, false
		}
		return bk, true
	}
	// Weighted round-robin, skipping open backends.
	b.mu.Lock()
	limit := b.cs[len(b.cs)-1]
	attempts := 0
	selected := 0
	// Try up to 2× backends to find an open one; if all are open, return the last one.
	for attempts < len(b.backends)*2 {
		idx := b.index % limit
		b.index++
		for i, w := range b.cs {
			if idx < w {
				selected = i
				break
			}
		}
		if !cs.isOpen(b.backends[selected].URL) {
			b.mu.Unlock()
			return b.backends[selected], true
		}
		attempts++
	}
	b.mu.Unlock()
	// All open — return last pick anyway so the request is counted.
	return b.backends[selected], true
}

// retryTransport wraps a base http.RoundTripper and retries failed requests
// up to maxRetries times for idempotent methods on transient errors.
type retryTransport struct {
	base       http.RoundTripper
	maxRetries int
}

func newRetryTransport(base http.RoundTripper, maxRetries int) http.RoundTripper {
	if maxRetries <= 0 {
		return base
	}
	return &retryTransport{base: base, maxRetries: maxRetries}
}

func (rt *retryTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	var lastErr error
	for attempt := 0; attempt <= rt.maxRetries; attempt++ {
		resp, err := rt.base.RoundTrip(req)
		if err == nil {
			return resp, nil
		}
		lastErr = err
		// Only retry GET/HEAD/OPTIONS — others are not safe
		if req.Method != http.MethodGet && req.Method != http.MethodHead && req.Method != http.MethodOptions {
			return resp, err
		}
		// Transient errors worth retrying
		if !isRetryableError(err) {
			return resp, err
		}
	}
	return nil, lastErr
}

func isRetryableError(err error) bool {
	if err == nil {
		return false
	}
	if _, ok := err.(net.Error); ok {
		return true
	}
	s := err.Error()
	retryable := []string{
		"connection refused",
		"no such host",
		"timeout",
		"i/o timeout",
		"network is unreachable",
		"connection reset",
		"broken pipe",
	}
	for _, p := range retryable {
		if strings.Contains(strings.ToLower(s), p) {
			return true
		}
	}
	return false
}

// writeTimeoutTransport wraps a base RoundTripper and enforces a per-request
// write deadline. This compensates for the fact that http.Transport has no
// WriteTimeout field in Go 1.24.
type writeTimeoutTransport struct {
	base         http.RoundTripper
	writeTimeout time.Duration
}

func newWriteTimeoutTransport(base http.RoundTripper, writeTimeout time.Duration) http.RoundTripper {
	if writeTimeout <= 0 {
		return base
	}
	return &writeTimeoutTransport{base: base, writeTimeout: writeTimeout}
}

func (wt *writeTimeoutTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// Set a write deadline before sending the request.
	if err := req.Context().Err(); err != nil {
		return nil, err
	}
	deadline := time.Now().Add(wt.writeTimeout)
	if existing, ok := req.Context().Deadline(); ok && existing.Before(deadline) {
		deadline = existing
	}
	ctx, cancel := context.WithDeadline(req.Context(), deadline)
	defer cancel()
	req = req.WithContext(ctx)
	return wt.base.RoundTrip(req)
}

// isWebSocketRequest returns true if the request is a WebSocket upgrade.
func isWebSocketRequest(h *http.Header) bool {
	return strings.EqualFold(h.Get("Upgrade"), "websocket") &&
		strings.Contains(strings.ToLower(h.Get("Connection")), "upgrade")
}

func forwardedProto(req *http.Request) string {
	if req != nil {
		if req.TLS != nil {
			return "https"
		}
		if header := strings.TrimSpace(req.Header.Get("X-Forwarded-Proto")); header != "" {
			if first, _, _ := strings.Cut(header, ","); first != "" {
				switch strings.ToLower(strings.TrimSpace(first)) {
				case "https", "wss":
					return "https"
				case "http", "ws":
					return "http"
				}
			}
		}
		if req.URL != nil {
			switch strings.ToLower(req.URL.Scheme) {
			case "https", "wss":
				return "https"
			}
		}
	}
	return "http"
}

func requestMatchHost(authority string) string {
	authority = strings.TrimSpace(authority)
	if authority == "" {
		return ""
	}
	if strings.HasPrefix(authority, "[") {
		if idx := strings.LastIndex(authority, "]"); idx != -1 {
			return authority[1:idx]
		}
	}
	if host, _, err := net.SplitHostPort(authority); err == nil {
		return host
	}
	return authority
}

func parseRuntimeBackendURL(raw string) (*url.URL, error) {
	backendURL, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || backendURL.Scheme == "" || backendURL.Host == "" {
		return nil, errInvalidRuntimeBackend
	}
	if backendURL.Scheme != "http" && backendURL.Scheme != "https" {
		return nil, errInvalidRuntimeBackend
	}
	return backendURL, nil
}

func selectRuntimeBackend(route *router.Route) (*url.URL, store.Backend, error) {
	backends := route.EffectiveBackends()
	if len(backends) == 0 {
		return nil, store.Backend{}, errNoBackendConfigured
	}

	if len(backends) == 1 {
		backendURL, err := parseRuntimeBackendURL(backends[0].URL)
		if err != nil {
			return nil, store.Backend{}, err
		}
		return backendURL, backends[0], nil
	}

	for _, backend := range backends {
		if backend.Weight <= 0 {
			return nil, store.Backend{}, errInvalidRuntimeBackend
		}
	}

	bl := newBalancer(backends)
	if bl == nil {
		return nil, store.Backend{}, errInvalidRuntimeBackend
	}

	picked, ok := bl.pick()
	if !ok {
		return nil, store.Backend{}, errNoBackendConfigured
	}

	backendURL, err := parseRuntimeBackendURL(picked.URL)
	if err != nil {
		return nil, store.Backend{}, err
	}
	return backendURL, picked, nil
}

func Handler(routerMgr *router.Manager, accessLogStore *store.AccessLogStore) gin.HandlerFunc {
	setAccessLogStore(accessLogStore)

	return func(c *gin.Context) {
		startedAt := time.Now()
		host := requestMatchHost(c.Request.Host)
		path := c.Request.URL.Path

		route := routerMgr.Match(host, path, c.Request.Header)
		if route == nil {
			log.Printf("proxy match miss host=%s path=%s", host, path)
			c.JSON(http.StatusNotFound, httpresponse.ErrorEnvelope{
				Error: httpresponse.ErrorDetail{
					Code:    "route_not_found",
					Message: "no route found",
				},
			})
			return
		}
		log.Printf("proxy match route_id=%s host=%s path=%s", route.ID, host, path)

		// HTTP -> HTTPS redirect
		if route.HTTPSRedirect && forwardedProto(c.Request) == "http" {
			target := "https://" + c.Request.Host + c.Request.URL.RequestURI()
			c.Redirect(http.StatusMovedPermanently, target)
			c.Abort()
			return
		}

		// 外跳重定向（301/302）
		if route.RedirectCode > 0 && route.RewriteTarget != "" {
			c.Redirect(route.RedirectCode, route.RewriteTarget)
			c.Abort()
			return
		}

		// 鉴权（OR 逻辑：任一启用的认证方式通过即放行）
		authenticated := false
		if route.AuthConfig != nil && route.AuthConfig.HasAuth() {
			cfg := route.AuthConfig

			// 1. API Key 认证
			if !authenticated && cfg.ApiKeyEnabled {
				keyValue := c.GetHeader(cfg.ApiKeyHeader)
				if keyValue == "" {
					keyValue = c.Query("api_key")
				}
				if keyValue != "" {
					for _, key := range route.ApiKeys {
						if key.Status == "active" && key.Secret == keyValue {
							// 检查过期
							expired := false
							if key.ExpiresAt != nil {
								if t, err := time.Parse(time.RFC3339, *key.ExpiresAt); err == nil && t.Before(time.Now()) {
									expired = true
								}
							}
							if !expired {
								authenticated = true
								go func(keyID string) {
									if k, err := routerMgr.DB().GetApiKey(keyID); err == nil {
										now := time.Now()
										k.LastUsedAt = &now
										routerMgr.DB().UpdateApiKey(k)
									}
								}(key.ID)
								break
							}
						}
					}
				}
			}

			// 2. Basic Auth 认证
			if !authenticated && cfg.BasicEnabled {
				username, password, ok := c.Request.BasicAuth()
				if ok && username == cfg.BasicUsername && password == cfg.BasicPassword {
					authenticated = true
				}
			}

			// 3. 网关登录认证
			if !authenticated && cfg.GatewayEnabled {
				if claims, ok := routeAccessClaims(c, routerMgr.DB()); ok && routeAllowedByClaims(claims, route.ID) {
					c.Set("jwt_subject", claims.UserID)
					c.Set("jwt_username", claims.Username)
					c.Set("jwt_role", claims.Role)
					authenticated = true
				}
			}

			// 全部未通过
			if !authenticated {
				if cfg.GatewayEnabled {
					recordAuthFailure(c, route, http.StatusFound)
					c.Redirect(http.StatusFound, buildAccessLoginURL(route, c.Request.URL.RequestURI()))
					c.Abort()
				} else {
					writeUnauthorized(c, route)
				}
				return
			}

			// 速率限制（使用路由级共享策略）
			if cfg.RateLimit > 0 || cfg.Burst > 0 {
				allowed, retryAfter := middleware.Check(route.ID, c.ClientIP(), cfg.RateLimit, cfg.Burst, cfg.Whitelist)
				if !allowed {
					metrics.RecordRateLimitExceeded(route.ID)
					recordAuthFailure(c, route, http.StatusTooManyRequests)
					c.Header("Retry-After", fmt.Sprintf("%.0f", retryAfter.Seconds()))
					c.AbortWithStatusJSON(http.StatusTooManyRequests, httpresponse.ErrorEnvelope{
						Error: httpresponse.ErrorDetail{
							Code:    "rate_limit_exceeded",
							Message: "too many requests",
						},
					})
					return
				}
			}

			// CORS：检测 OPTIONS 预检请求并尽早响应
			if cfg.CORSAllowedOrigins != "" {
				corsAllowed, preflightAbort := handleCORS(cfg, c)
				if preflightAbort {
					return
				}
				_ = corsAllowed
			}
		}

		requestID := uuid.New().String()

		// WebSocket detection: before backend URL parsing so empty-backend routes work
		if isWebSocketRequest(&c.Request.Header) {
			backendURL, _, err := selectRuntimeBackend(route)
			if err != nil {
				if errors.Is(err, errNoBackendConfigured) {
					handleWebSocket(c, nil, route)
					return
				}
				c.JSON(http.StatusInternalServerError, httpresponse.ErrorEnvelope{
					Error: httpresponse.ErrorDetail{
						Code:    "invalid_backend",
						Message: "invalid backend",
					},
				})
				return
			}
			handleWebSocket(c, backendURL, route)
			return
		}

		backendURL, picked, err := selectRuntimeBackend(route)
		if err != nil {
			c.JSON(http.StatusInternalServerError, httpresponse.ErrorEnvelope{
				Error: httpresponse.ErrorDetail{
					Code:    "invalid_backend",
					Message: "invalid backend",
				},
			})
			return
		}
		backendHost := backendURL.Host

		// 重试次数
		maxRetries := 1
		if route.RetryAttempts > 0 {
			maxRetries = route.RetryAttempts
		}

		// 超时：优先用 per-backend 配置，否则用 route 级别
		dialTimeout := 5 * time.Second
		readTimeout := 30 * time.Second
		if picked.DialTimeoutMs > 0 {
			dialTimeout = time.Duration(picked.DialTimeoutMs) * time.Millisecond
		} else if route.TimeoutMs > 0 {
			dialTimeout = time.Duration(route.TimeoutMs) * time.Millisecond
		}
		if picked.ReadTimeoutMs > 0 {
			readTimeout = time.Duration(picked.ReadTimeoutMs) * time.Millisecond
		} else if route.TimeoutMs > 0 {
			readTimeout = time.Duration(route.TimeoutMs) * time.Millisecond
		}
		if picked.DialTimeoutMs > 0 {
			dialTimeout = time.Duration(picked.DialTimeoutMs) * time.Millisecond
		} else if route.TimeoutMs > 0 {
			dialTimeout = time.Duration(route.TimeoutMs) * time.Millisecond
		}
		if picked.ReadTimeoutMs > 0 {
			readTimeout = time.Duration(picked.ReadTimeoutMs) * time.Millisecond
		} else if route.TimeoutMs > 0 {
			readTimeout = time.Duration(route.TimeoutMs) * time.Millisecond
		}

		proxy := httputil.NewSingleHostReverseProxy(backendURL)

		baseTransport := defaultTransportCache.get(transportKey{
			DialTimeout:  dialTimeout,
			ReadTimeout:  readTimeout,
			MaxIdleConns: picked.MaxIdleConns,
		})

		proxy.Transport = newRetryTransport(baseTransport, maxRetries)

		// 修改请求
		originalDirector := proxy.Director
		proxy.Director = func(req *http.Request) {
			originalDirector(req)
			req.Host = backendHost

			// 传递原始请求信息
			req.Header.Set("X-Forwarded-Host", c.Request.Host)
			req.Header.Set("X-Forwarded-Proto", forwardedProto(c.Request))
			req.Header.Set("X-Forwarded-For", c.ClientIP())
			req.Header.Set("X-Real-IP", c.ClientIP())

			// 去掉前缀
			if route.StripPrefix {
				req.URL.Path = strings.TrimPrefix(req.URL.Path, route.PathPrefix)
				if !strings.HasPrefix(req.URL.Path, "/") {
					req.URL.Path = "/" + req.URL.Path
				}
			}

			// 正则 rewrite
			if route.RewriteTarget != "" && (route.PathMatchMode == "regex" || route.PathMatchMode == "regex_i") && route.PathRegex != nil {
				newPath := route.PathRegex.ReplaceAllString(req.URL.Path, route.RewriteTarget)
				if newPath != req.URL.Path {
					log.Printf("rewrite route_id=%s: %s -> %s", route.ID, req.URL.Path, newPath)
					req.URL.Path = newPath
				}
			}

			// 自定义请求头：在转发前设置/删除
			for k, v := range route.SetRequestHeaders {
				req.Header.Set(k, v)
			}
			for _, k := range route.RemoveRequestHeaders {
				req.Header.Del(k)
			}
		}

		// 自定义响应头：在返回客户端前添加/删除
		if len(route.AddResponseHeaders) > 0 || len(route.RemoveResponseHeaders) > 0 {
			proxy.ModifyResponse = func(resp *http.Response) error {
				for k, v := range route.AddResponseHeaders {
					resp.Header.Set(k, v)
				}
				for _, k := range route.RemoveResponseHeaders {
					resp.Header.Del(k)
				}
				return nil
			}
		}

		// 错误处理：记录熔断失败
		backendHostForError := backendHost
		proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
			log.Printf("proxy upstream error route_id=%s backend=%s err=%v", route.ID, backendHostForError, err)
			cs.recordFailure(backendHostForError)
			c.JSON(http.StatusBadGateway, httpresponse.ErrorEnvelope{
				Error: httpresponse.ErrorDetail{
					Code:    "backend_error",
					Message: "upstream request failed",
				},
			})
		}

		proxyWriter := &proxyResponseWriter{ResponseWriter: c.Writer}
		proxy.ServeHTTP(proxyWriter, c.Request)
		finalStatus := c.Writer.Status()
		// 正常 2xx 响应记录成功（用于半开状态的恢复探测）
		if finalStatus >= 200 && finalStatus < 300 {
			cs.recordSuccess(backendHost)
		}

		latency := time.Since(startedAt)
		metrics.RecordRequest(route.ID, c.Request.Method, statusLabel(finalStatus), float64(latency.Milliseconds()))
		if backendHost != "" {
			metrics.RecordBackendHealth(route.ID, backendHost, finalStatus < 500)
		}

		// 结构化访问日志
		username, _ := c.Get("jwt_username")
		authResult := "none"
		if authenticated {
			authResult = "pass"
		}

		accessLog := accessLogEntry{
			RequestID:        requestID,
			RouteID:          route.ID,
			Method:           c.Request.Method,
			Path:             c.Request.URL.Path,
			BackendURL:       backendHost,
			BackendLatencyMs: latency.Milliseconds(),
			StatusCode:       finalStatus,
			ClientIP:         c.ClientIP(),
			UserAgent:        c.Request.UserAgent(),
			Username:         safeString(username),
			AuthResult:       authResult,
		}
		if accessLogBytes, err := json.Marshal(accessLog); err == nil {
			accessLogger.Printf("access %s", string(accessLogBytes))
		}

		// Record to access log store
		if accessLogStore != nil {
			accessLogStore.Record(store.AccessLogEntry{
				RequestID:        accessLog.RequestID,
				RouteID:          accessLog.RouteID,
				Method:           accessLog.Method,
				Path:             accessLog.Path,
				BackendURL:       accessLog.BackendURL,
				BackendLatencyMs: accessLog.BackendLatencyMs,
				StatusCode:       accessLog.StatusCode,
				ClientIP:         accessLog.ClientIP,
				UserAgent:        accessLog.UserAgent,
				Username:         accessLog.Username,
				AuthResult:       accessLog.AuthResult,
				Timestamp:        startedAt,
			})
		}

		c.Abort()
	}
}

// handleWebSocket proxies a WebSocket connection by hijacking the client TCP connection
// and piping it bidirectionally to the backend. This requires a hijackable connection
// (not available with httptest.ResponseRecorder in tests).
func handleWebSocket(c *gin.Context, backendURL *url.URL, route *router.Route) {
	if backendURL == nil || backendURL.Host == "" {
		log.Printf("WebSocket route_id=%s: no backend configured", route.ID)
		c.JSON(http.StatusBadGateway, httpresponse.ErrorEnvelope{
			Error: httpresponse.ErrorDetail{
				Code:    "invalid_backend",
				Message: "WebSocket backend not configured",
			},
		})
		return
	}

	// Check hijack support before attempting to avoid panics.
	// httptest.ResponseRecorder does not implement http.Hijacker; real TCP connections do.
	if _, ok := c.Writer.(http.Hijacker); !ok {
		log.Printf("WebSocket route_id=%s: ResponseWriter does not support Hijack", route.ID)
		c.Writer.WriteHeader(http.StatusUpgradeRequired)
		c.Abort()
		return
	}

	hijacker := c.Writer.(http.Hijacker)
	clientConn, _, err := hijacker.Hijack()
	if err != nil {
		log.Printf("WebSocket route_id=%s: hijack failed: %v", route.ID, err)
		c.JSON(http.StatusInternalServerError, httpresponse.ErrorEnvelope{
			Error: httpresponse.ErrorDetail{
				Code:    "internal_error",
				Message: "websocket upgrade failed",
			},
		})
		return
	}
	defer clientConn.Close()

	// Connect to backend
	var dialer net.Dialer
	backendConn, err := dialer.Dial("tcp", backendURL.Host)
	if err != nil {
		log.Printf("WebSocket route_id=%s: dial backend %s failed: %v", route.ID, backendURL.Host, err)
		return
	}
	defer backendConn.Close()

	// Build WebSocket upgrade request with all original headers preserved
	req := &http.Request{
		Method:     "GET",
		URL:        backendURL,
		Host:       backendURL.Host,
		Proto:      "HTTP/1.1",
		ProtoMajor: 1,
		ProtoMinor: 1,
		Header:     make(http.Header),
	}
	for k, v := range c.Request.Header {
		req.Header[k] = v
	}
	req.Header.Set("X-Forwarded-Host", c.Request.Host)
	req.Header.Set("X-Forwarded-Proto", forwardedProto(c.Request))
	req.Header.Set("X-Forwarded-For", c.ClientIP())
	req.Header.Set("X-Real-IP", c.ClientIP())

	if err := req.Write(backendConn); err != nil {
		log.Printf("WebSocket route_id=%s: write request to backend failed: %v", route.ID, err)
		return
	}

	// Read backend response (expect 101 Switching Protocols)
	resp, err := http.ReadResponse(bufio.NewReader(backendConn), req)
	if err != nil {
		log.Printf("WebSocket route_id=%s: read backend response failed: %v", route.ID, err)
		return
	}

	if resp.StatusCode != http.StatusSwitchingProtocols {
		log.Printf("WebSocket route_id=%s: backend returned %d instead of 101", route.ID, resp.StatusCode)
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		clientConn.Write([]byte(fmt.Sprintf("HTTP/1.1 %d %s\r\n\r\n%s", resp.StatusCode, resp.Status, string(body))))
		return
	}

	// 101 Switching Protocols — forward to client and start bidirectional proxy
	resp.Write(clientConn)
	pipeWebSocket(clientConn, backendConn)
}

// pipeWebSocket proxies data bidirectionally between two connections.
// proxyResponseWriter preserves Gin's native status tracking while smoothing
// over optional interfaces that httptest.ResponseRecorder does not implement.
type proxyResponseWriter struct {
	gin.ResponseWriter
}

func (w *proxyResponseWriter) CloseNotify() <-chan bool {
	var result <-chan bool
	func() {
		defer func() { recover() }()
		if cn, ok := w.ResponseWriter.(http.CloseNotifier); ok {
			result = cn.CloseNotify()
		}
	}()
	if result != nil {
		return result
	}
	ch := make(chan bool, 1)
	return ch
}

func (w *proxyResponseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	if h, ok := w.ResponseWriter.(http.Hijacker); ok {
		return h.Hijack()
	}
	return nil, nil, http.ErrNotSupported
}

// statusLabel returns a short label for HTTP status codes for use in metrics.
func safeString(v interface{}) string {
	if v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

func statusLabel(code int) string {
	switch {
	case code >= 500:
		return "500"
	case code >= 400:
		return "400"
	case code >= 300:
		return "300"
	case code >= 200:
		return "200"
	default:
		return "other"
	}
}

func pipeWebSocket(a, b net.Conn) {
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		io.Copy(a, b)
		a.Close()
		b.Close()
	}()
	go func() {
		defer wg.Done()
		io.Copy(b, a)
		a.Close()
		b.Close()
	}()
	wg.Wait()
}

func routeAllowedByClaims(claims *auth.Claims, routeID string) bool {
	if claims == nil {
		return false
	}
	// 管理员默认拥有所有项目的访问权限
	if claims.Role == store.RoleAdmin {
		return true
	}
	for _, allowedRouteID := range claims.RouteIDs {
		if allowedRouteID == routeID {
			return true
		}
	}
	return false
}

// accessLogStoreInstance is the package-level access log store instance.
var accessLogStoreInstance *store.AccessLogStore

// setAccessLogStore sets the package-level access log store.
func setAccessLogStore(s *store.AccessLogStore) {
	accessLogStoreInstance = s
}

// recordAuthFailure records a failed authentication attempt.
func recordAuthFailure(c *gin.Context, route *router.Route, statusCode int) {
	if accessLogStoreInstance == nil {
		return
	}

	accessLogStoreInstance.Record(store.AccessLogEntry{
		RequestID:  uuid.New().String(),
		RouteID:    route.ID,
		Method:     c.Request.Method,
		Path:       c.Request.URL.Path,
		StatusCode: statusCode,
		ClientIP:   c.ClientIP(),
		UserAgent:  c.Request.UserAgent(),
		AuthResult: "fail",
		Timestamp:  time.Now(),
	})
}

func writeUnauthorized(c *gin.Context, route *router.Route) {
	recordAuthFailure(c, route, http.StatusUnauthorized)

	if route.AuthConfig != nil {
		// 优先返回 Basic Auth 的 WWW-Authenticate 头
		if route.AuthConfig.BasicEnabled {
			realm := route.Name
			if strings.TrimSpace(realm) == "" {
				realm = route.PathPrefix
			}
			realm = strings.ReplaceAll(realm, `"`, `'`)
			auth.RequireAuth(fmt.Sprintf(`Basic realm="%s"`, realm))(c)
			return
		}
		// 有网关登录时重定向
		if route.AuthConfig.GatewayEnabled {
			c.Redirect(http.StatusFound, buildAccessLoginURL(route, c.Request.URL.RequestURI()))
			c.Abort()
			return
		}
	}

	c.JSON(http.StatusUnauthorized, httpresponse.ErrorEnvelope{
		Error: httpresponse.ErrorDetail{
			Code:    "unauthorized",
			Message: "unauthorized",
		},
	})
	c.Abort()
}

// handleCORS applies CORS headers for a route.
// Returns (allowedOrigin, true) if a preflight (OPTIONS) was handled and the request should abort.
// Returns ("", false) if normal request processing should continue.
func handleCORS(cfg *router.RouteAuthConfig, c *gin.Context) (string, bool) {
	origin := c.Request.Header.Get("Origin")
	if origin == "" {
		return "", false
	}

	allowedOrigins := cfg.CORSAllowedOrigins
	allowAll := strings.TrimSpace(allowedOrigins) == "*"
	if !allowAll && !originMatches(origin, allowedOrigins) {
		return "", false
	}

	maxAge := cfg.CORSMaxAge
	if maxAge <= 0 {
		maxAge = 86400
	}
	allowedMethods := cfg.CORSAllowedMethods
	if allowedMethods == "" {
		allowedMethods = "GET,POST,PUT,DELETE,PATCH,OPTIONS"
	}
	allowedHeaders := cfg.CORSAllowedHeaders
	if allowedHeaders == "" {
		allowedHeaders = "Authorization,Content-Type,X-Requested-With"
	}

	c.Header("Access-Control-Allow-Origin", origin)
	if cfg.CORSAllowCredentials {
		c.Header("Access-Control-Allow-Credentials", "true")
	}
	c.Header("Access-Control-Allow-Methods", allowedMethods)
	c.Header("Access-Control-Allow-Headers", allowedHeaders)
	c.Header("Access-Control-Max-Age", fmt.Sprintf("%d", maxAge))

	if c.Request.Method == http.MethodOptions {
		c.AbortWithStatus(http.StatusNoContent)
		return origin, true
	}
	return origin, false
}

// originMatches checks if origin matches any pattern (exact or .domain wildcard).
func originMatches(origin, allowed string) bool {
	allowedList := strings.Split(allowed, ",")
	for _, pattern := range allowedList {
		pattern = strings.TrimSpace(pattern)
		if pattern == "" {
			continue
		}
		if pattern[0] == '.' {
			domain := strings.ToLower(pattern[1:])
			lowerOrigin := strings.ToLower(origin)
			if strings.HasSuffix(lowerOrigin, domain) {
				return true
			}
		} else if strings.EqualFold(pattern, origin) {
			return true
		}
	}
	return false
}
