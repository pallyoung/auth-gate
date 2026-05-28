package proxy

import (
	"crypto/tls"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/pallyoung/auth-gate/packages/server/internal/auth"
	"github.com/pallyoung/auth-gate/packages/server/internal/router"
	"github.com/pallyoung/auth-gate/packages/server/internal/store"
)

func TestAccessLoginRoute_SetsSecureCookieForHTTPSRequests(t *testing.T) {
	gin.SetMode(gin.TestMode)
	auth.ConfigureJWTSecret("test-secret")

	db := newTestDB(t)
	if err := db.CreateRoute(&store.Route{
		ID:         "route-1",
		Name:       "cloud",
		PathPrefix: "/cloud",
		Backend:    "http://example.com",
		Enabled:    true,
	}); err != nil {
		t.Fatalf("CreateRoute() error = %v", err)
	}

	hash, err := store.HashPassword("password123")
	if err != nil {
		t.Fatalf("HashPassword() error = %v", err)
	}
	if err := db.CreateUser(&store.User{
		ID:           "member-1",
		Username:     "member",
		PasswordHash: hash,
		Role:         store.RoleMember,
		Enabled:      true,
		RouteIDs:     []string{"route-1"},
	}); err != nil {
		t.Fatalf("CreateUser() error = %v", err)
	}

	engine := gin.New()
	engine.POST("/_authgate/api/access/login", AccessLoginRoute(router.NewManager(db), db))

	req := httptest.NewRequest(
		http.MethodPost,
		"https://example.com/_authgate/api/access/login",
		strings.NewReader(`{"route_id":"route-1","username":"member","password":"password123","next":"/cloud"}`),
	)
	req.Header.Set("Content-Type", "application/json")
	req.TLS = &tls.ConnectionState{}

	resp := httptest.NewRecorder()
	engine.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body=%s", resp.Code, http.StatusOK, resp.Body.String())
	}

	var routeAccessCookie *http.Cookie
	for _, cookie := range resp.Result().Cookies() {
		if cookie.Name == "auth_gate_route_token" {
			routeAccessCookie = cookie
			break
		}
	}

	if routeAccessCookie == nil {
		t.Fatal("route access cookie not set")
	}
	if !routeAccessCookie.Secure {
		t.Fatal("route access cookie Secure = false, want true for HTTPS requests")
	}
}

func TestAccessLoginRoute_TrimsUsernameBeforeAuthenticating(t *testing.T) {
	gin.SetMode(gin.TestMode)
	auth.ConfigureJWTSecret("test-secret")

	db := newTestDB(t)
	if err := db.CreateRoute(&store.Route{
		ID:         "route-1",
		Name:       "cloud",
		PathPrefix: "/cloud",
		Backend:    "http://example.com",
		Enabled:    true,
	}); err != nil {
		t.Fatalf("CreateRoute() error = %v", err)
	}

	hash, err := store.HashPassword("password123")
	if err != nil {
		t.Fatalf("HashPassword() error = %v", err)
	}
	if err := db.CreateUser(&store.User{
		ID:           "member-1",
		Username:     "member",
		PasswordHash: hash,
		Role:         store.RoleMember,
		Enabled:      true,
		RouteIDs:     []string{"route-1"},
	}); err != nil {
		t.Fatalf("CreateUser() error = %v", err)
	}

	engine := gin.New()
	engine.POST("/_authgate/api/access/login", AccessLoginRoute(router.NewManager(db), db))

	req := httptest.NewRequest(
		http.MethodPost,
		"http://example.com/_authgate/api/access/login",
		strings.NewReader(`{"route_id":"route-1","username":"  member  ","password":"password123","next":"/cloud"}`),
	)
	req.Header.Set("Content-Type", "application/json")

	resp := httptest.NewRecorder()
	engine.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body=%s", resp.Code, http.StatusOK, resp.Body.String())
	}
}

func TestAccessLoginRoute_SetsSecureCookieWhenReverseProxyReportsHTTPS(t *testing.T) {
	gin.SetMode(gin.TestMode)
	auth.ConfigureJWTSecret("test-secret")

	db := newTestDB(t)
	if err := db.CreateRoute(&store.Route{
		ID:         "route-1",
		Name:       "cloud",
		PathPrefix: "/cloud",
		Backend:    "http://example.com",
		Enabled:    true,
	}); err != nil {
		t.Fatalf("CreateRoute() error = %v", err)
	}

	hash, err := store.HashPassword("password123")
	if err != nil {
		t.Fatalf("HashPassword() error = %v", err)
	}
	if err := db.CreateUser(&store.User{
		ID:           "member-1",
		Username:     "member",
		PasswordHash: hash,
		Role:         store.RoleMember,
		Enabled:      true,
		RouteIDs:     []string{"route-1"},
	}); err != nil {
		t.Fatalf("CreateUser() error = %v", err)
	}

	engine := gin.New()
	engine.POST("/_authgate/api/access/login", AccessLoginRoute(router.NewManager(db), db))

	req := httptest.NewRequest(
		http.MethodPost,
		"http://example.com/_authgate/api/access/login",
		strings.NewReader(`{"route_id":"route-1","username":"member","password":"password123","next":"/cloud"}`),
	)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Forwarded-Proto", "https")

	resp := httptest.NewRecorder()
	engine.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body=%s", resp.Code, http.StatusOK, resp.Body.String())
	}

	var routeAccessCookie *http.Cookie
	for _, cookie := range resp.Result().Cookies() {
		if cookie.Name == "auth_gate_route_token" {
			routeAccessCookie = cookie
			break
		}
	}

	if routeAccessCookie == nil {
		t.Fatal("route access cookie not set")
	}
	if !routeAccessCookie.Secure {
		t.Fatal("route access cookie Secure = false, want true when reverse proxy reports https")
	}
}

func TestAccessLoginRoute_FallsBackWhenNextUsesSchemeRelativePath(t *testing.T) {
	gin.SetMode(gin.TestMode)
	auth.ConfigureJWTSecret("test-secret")

	db := newTestDB(t)
	if err := db.CreateRoute(&store.Route{
		ID:         "route-1",
		Name:       "cloud",
		PathPrefix: "/cloud",
		Backend:    "http://example.com",
		Enabled:    true,
	}); err != nil {
		t.Fatalf("CreateRoute() error = %v", err)
	}

	hash, err := store.HashPassword("password123")
	if err != nil {
		t.Fatalf("HashPassword() error = %v", err)
	}
	if err := db.CreateUser(&store.User{
		ID:           "member-1",
		Username:     "member",
		PasswordHash: hash,
		Role:         store.RoleMember,
		Enabled:      true,
		RouteIDs:     []string{"route-1"},
	}); err != nil {
		t.Fatalf("CreateUser() error = %v", err)
	}

	engine := gin.New()
	engine.POST("/_authgate/api/access/login", AccessLoginRoute(router.NewManager(db), db))

	req := httptest.NewRequest(
		http.MethodPost,
		"http://example.com/_authgate/api/access/login",
		strings.NewReader(`{"route_id":"route-1","username":"member","password":"password123","next":"///evil.com/path"}`),
	)
	req.Header.Set("Content-Type", "application/json")

	resp := httptest.NewRecorder()
	engine.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body=%s", resp.Code, http.StatusOK, resp.Body.String())
	}

	var payload struct {
		Next string `json:"next"`
	}
	if err := json.Unmarshal(resp.Body.Bytes(), &payload); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	if payload.Next != "/cloud" {
		t.Fatalf("payload.Next = %q, want %q", payload.Next, "/cloud")
	}
}

func TestAccessLoginRoute_FallsBackWhenNextUsesBackslashHostPath(t *testing.T) {
	gin.SetMode(gin.TestMode)
	auth.ConfigureJWTSecret("test-secret")

	db := newTestDB(t)
	if err := db.CreateRoute(&store.Route{
		ID:         "route-1",
		Name:       "cloud",
		PathPrefix: "/cloud",
		Backend:    "http://example.com",
		Enabled:    true,
	}); err != nil {
		t.Fatalf("CreateRoute() error = %v", err)
	}

	hash, err := store.HashPassword("password123")
	if err != nil {
		t.Fatalf("HashPassword() error = %v", err)
	}
	if err := db.CreateUser(&store.User{
		ID:           "member-1",
		Username:     "member",
		PasswordHash: hash,
		Role:         store.RoleMember,
		Enabled:      true,
		RouteIDs:     []string{"route-1"},
	}); err != nil {
		t.Fatalf("CreateUser() error = %v", err)
	}

	engine := gin.New()
	engine.POST("/_authgate/api/access/login", AccessLoginRoute(router.NewManager(db), db))

	req := httptest.NewRequest(
		http.MethodPost,
		"http://example.com/_authgate/api/access/login",
		strings.NewReader("{\"route_id\":\"route-1\",\"username\":\"member\",\"password\":\"password123\",\"next\":\"/\\\\evil.com/path\"}"),
	)
	req.Header.Set("Content-Type", "application/json")

	resp := httptest.NewRecorder()
	engine.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body=%s", resp.Code, http.StatusOK, resp.Body.String())
	}

	var payload struct {
		Next string `json:"next"`
	}
	if err := json.Unmarshal(resp.Body.Bytes(), &payload); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	if payload.Next != "/cloud" {
		t.Fatalf("payload.Next = %q, want %q", payload.Next, "/cloud")
	}
}
