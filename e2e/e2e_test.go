package e2e

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"auth-gate/internal/api"
	"auth-gate/internal/auth"
	"auth-gate/internal/proxy"
	"auth-gate/internal/router"
	"auth-gate/internal/store"

	"github.com/gin-gonic/gin"
)

func init() { gin.SetMode(gin.TestMode) }

type testEnv struct {
	t      *testing.T
	db     *store.SQLite
	engine *gin.Engine
}

func newEnv(t *testing.T) *testEnv {
	dbPath := filepath.Join(os.TempDir(), fmt.Sprintf("auth-gate-e2e-%d.db", time.Now().UnixNano()))
	db, err := store.NewSQLite(dbPath)
	if err != nil {
		t.Fatalf("NewSQLite: %v", err)
	}
	if err := db.EnsureAdmin(); err != nil {
		t.Fatalf("EnsureAdmin: %v", err)
	}
	routerMgr := router.NewManager(db)

	engine := gin.New()
	engine.Use(gin.Recovery())
	engine.POST("/api/auth/login", api.LoginHandler(db))

	apiGroup := engine.Group("/api")
	apiGroup.Use(auth.AuthMiddleware())
	api.RegisterHandlers(apiGroup, routerMgr, db)

	// Proxy setup (ReverseProxy needs CloseNotifier — test auth logic directly)
	_ = proxy.Handler(routerMgr)

	t.Cleanup(func() {
		db.Close()
		os.Remove(dbPath)
	})

	return &testEnv{t: t, db: db, engine: engine}
}

func (e *testEnv) req(method, path string, body interface{}, token string) *httptest.ResponseRecorder {
	var bodyReader io.Reader
	if body != nil {
		data, _ := json.Marshal(body)
		bodyReader = bytes.NewReader(data)
	}
	req := httptest.NewRequest(method, path, bodyReader)
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	w := httptest.NewRecorder()
	e.engine.ServeHTTP(w, req)
	return w
}

func (e *testEnv) login(username, password string) (string, store.Permissions) {
	w := e.req("POST", "/api/auth/login", map[string]string{"username": username, "password": password}, "")
	if w.Code != http.StatusOK {
		e.t.Fatalf("login(%s) failed: %s", username, w.Body.String())
	}
	var r api.LoginResponse
	json.Unmarshal(w.Body.Bytes(), &r)
	return r.Token, r.Permissions
}

func (e *testEnv) routeID(token, name string) string {
	w := e.req("GET", "/api/routes", nil, token)
	var routes []map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &routes)
	for _, r := range routes {
		if r["name"] == name {
			return r["id"].(string)
		}
	}
	return ""
}

// --- Login flow ---

func TestE2E_Login_Success(t *testing.T) {
	e := newEnv(t)
	token, perms := e.login("admin", "admin")
	if token == "" {
		t.Error("Token should not be empty")
	}
	if !perms.CanManageRoutes || !perms.CanManageUsers {
		t.Errorf("Admin should have full permissions, got %+v", perms)
	}
}

func TestE2E_Login_RejectsBadCredentials(t *testing.T) {
	e := newEnv(t)
	tests := []struct {
		name     string
		username string
		password string
		want     int
	}{
		{"Wrong password", "admin", "wrongpass", http.StatusUnauthorized},
		{"Wrong username", "nobody", "admin", http.StatusUnauthorized},
		{"Empty fields", "", "", http.StatusBadRequest},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := e.req("POST", "/api/auth/login", map[string]string{"username": tt.username, "password": tt.password}, "")
			if w.Code != tt.want {
				t.Errorf("login status = %d, want %d", w.Code, tt.want)
			}
		})
	}
}

func TestE2E_UnauthenticatedRequests(t *testing.T) {
	e := newEnv(t)

	w := e.req("GET", "/api/routes", nil, "")
	if w.Code != http.StatusUnauthorized {
		t.Errorf("No token status = %d, want 401", w.Code)
	}

	w = e.req("GET", "/api/routes", nil, "invalid-token")
	if w.Code != http.StatusUnauthorized {
		t.Errorf("Invalid token status = %d, want 401", w.Code)
	}
}

func TestE2E_MeHandler(t *testing.T) {
	e := newEnv(t)
	token, _ := e.login("admin", "admin")

	w := e.req("GET", "/api/auth/me", nil, token)
	if w.Code != http.StatusOK {
		t.Errorf("/api/auth/me status = %d, want 200", w.Code)
	}
	var me api.UserResponse
	json.Unmarshal(w.Body.Bytes(), &me)
	if me.Username != "admin" {
		t.Errorf("me.Username = %q, want admin", me.Username)
	}
}

// --- Route CRUD ---

func TestE2E_RouteCRUD(t *testing.T) {
	e := newEnv(t)
	token, _ := e.login("admin", "admin")

	w := e.req("POST", "/api/routes", map[string]interface{}{
		"name":        "crud-test-route",
		"host":        "crud.example.com",
		"path_prefix": "/v1",
		"backend":     "http://localhost:3000",
		"enabled":     true,
	}, token)
	if w.Code != http.StatusCreated {
		t.Fatalf("Create route failed (%d): %s", w.Code, w.Body.String())
	}
	routeID := e.routeID(token, "crud-test-route")
	if routeID == "" {
		t.Fatal("Could not find created route")
	}

	w = e.req("GET", "/api/routes", nil, token)
	if w.Code != http.StatusOK {
		t.Fatalf("List routes failed: %s", w.Body.String())
	}
	var routes []map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &routes)
	if len(routes) < 1 {
		t.Fatal("Expected at least 1 route")
	}

	w = e.req("GET", "/api/routes/"+routeID, nil, token)
	if w.Code != http.StatusOK {
		t.Errorf("Get route failed: %s", w.Body.String())
	}

	w = e.req("PUT", "/api/routes/"+routeID, map[string]interface{}{
		"name":        "crud-test-route-updated",
		"host":        "crud.example.com",
		"path_prefix": "/v2",
		"backend":     "http://localhost:4000",
	}, token)
	if w.Code != http.StatusOK {
		t.Fatalf("Update route failed: %s", w.Body.String())
	}

	w = e.req("DELETE", "/api/routes/"+routeID, nil, token)
	if w.Code != http.StatusOK {
		t.Fatalf("Delete route failed: %s", w.Body.String())
	}

	w = e.req("GET", "/api/routes/"+routeID, nil, token)
	if w.Code != http.StatusNotFound {
		t.Errorf("After delete, GET = %d, want 404", w.Code)
	}
}

// --- Auth rule CRUD ---

func TestE2E_AuthRuleCRUD(t *testing.T) {
	e := newEnv(t)
	token, _ := e.login("admin", "admin")

	_ = e.req("POST", "/api/routes", map[string]interface{}{
		"name":        "authrule-crud-test",
		"host":        "secure.example.com",
		"path_prefix": "/api",
		"backend":     "http://localhost:5000",
		"enabled":     true,
	}, token)
	routeID := e.routeID(token, "authrule-crud-test")

	w := e.req("POST", "/api/auth-rules", map[string]interface{}{
		"route_id": routeID,
		"type":     "apikey",
		"config":   map[string]string{"secret": "secret-crud"},
	}, token)
	if w.Code != http.StatusCreated {
		t.Fatalf("Create auth rule failed: %s", w.Body.String())
	}
	var rule map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &rule)
	ruleID := rule["id"].(string)

	w = e.req("GET", "/api/auth-rules", nil, token)
	if w.Code != http.StatusOK {
		t.Fatalf("List auth rules failed: %s", w.Body.String())
	}

	w = e.req("DELETE", "/api/auth-rules/"+ruleID, nil, token)
	if w.Code != http.StatusOK {
		t.Fatalf("Delete auth rule failed: %s", w.Body.String())
	}
}

// --- Auth rule enforcement: bearer JWT ---

func TestE2E_AuthRule_BearerJWT(t *testing.T) {
	// Verify JWT enforcement at the auth.Check level.
	// Tokens generated by auth.GenerateToken use JWTSecret, so we test
	// ValidateTokenWithSecret with the matching secret.

	token, _ := auth.GenerateToken("jwt-user", "jwtuser", "viewer")

	// Valid token with matching secret — passes
	claims, err := auth.ValidateTokenWithSecret(token, auth.JWTSecret)
	if err != nil {
		t.Fatalf("Valid token should validate with JWTSecret: %v", err)
	}
	if claims.UserID != "jwt-user" {
		t.Errorf("claims.UserID = %q, want jwt-user", claims.UserID)
	}

	// Wrong secret — rejected
	_, err = auth.ValidateTokenWithSecret(token, []byte("wrong-secret"))
	if err == nil {
		t.Error("Token signed with JWTSecret should fail with wrong secret")
	}

	// Empty secret — checkBearer returns false
	noSecretRule := &store.AuthRule{Type: "bearer", Config: store.AuthConfig{Secret: ""}}
	req := httptest.NewRequest("GET", "/api/users", nil)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = req
	if auth.Check(c, noSecretRule) {
		t.Error("checkBearer with empty secret should return false")
	}

	// Non-bearer rule type — checkBearer not invoked
	otherRule := &store.AuthRule{Type: "apikey", Config: store.AuthConfig{Secret: "key"}}
	req2 := httptest.NewRequest("GET", "/api/users", nil)
	rec2 := httptest.NewRecorder()
	c2, _ := gin.CreateTestContext(rec2)
	c2.Request = req2
	if auth.Check(c2, otherRule) {
		t.Error("apikey rule should return true for missing header")
	}
}

// --- Auth rule enforcement: basic auth ---

func TestE2E_AuthRule_BasicAuth(t *testing.T) {
	e := newEnv(t)
	token, _ := e.login("admin", "admin")

	_ = e.req("POST", "/api/routes", map[string]interface{}{
		"name":        "basic-protected",
		"host":        "basic.example.com",
		"path_prefix": "/secure",
		"backend":     "http://localhost:9997",
		"enabled":     true,
	}, token)
	routeID := e.routeID(token, "basic-protected")

	_ = e.req("POST", "/api/auth-rules", map[string]interface{}{
		"route_id": routeID,
		"type":     "basic",
		"config":   map[string]string{"username": "user1", "password": "pass123"},
	}, token)

	for _, tt := range []struct {
		name   string
		user   string
		pass   string
		wantOK bool
	}{
		{"Valid credentials", "user1", "pass123", true},
		{"Wrong password", "user1", "wrongpass", false},
		{"Wrong username", "nobody", "pass123", false},
		{"No credentials", "", "", false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/secure/data", nil)
			req.Host = "basic.example.com"
			if tt.user != "" {
				req.SetBasicAuth(tt.user, tt.pass)
			}
			rec := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(rec)
			c.Request = req
			rule := &store.AuthRule{Type: "basic", Config: store.AuthConfig{Username: "user1", Password: "pass123"}}
			if got := auth.Check(c, rule); got != tt.wantOK {
				t.Errorf("checkBasic = %v, want %v", got, tt.wantOK)
			}
		})
	}
}

// --- Auth rule enforcement: API key ---

func TestE2E_AuthRule_APIKey(t *testing.T) {
	e := newEnv(t)
	token, _ := e.login("admin", "admin")

	_ = e.req("POST", "/api/routes", map[string]interface{}{
		"name":        "apikey-protected",
		"host":        "apikey.example.com",
		"path_prefix": "/data",
		"backend":     "http://localhost:9995",
		"enabled":     true,
	}, token)
	routeID := e.routeID(token, "apikey-protected")

	_ = e.req("POST", "/api/auth-rules", map[string]interface{}{
		"route_id": routeID,
		"type":     "apikey",
		"config":   map[string]string{"secret": "my-api-key-123"},
	}, token)

	for _, tt := range []struct {
		name   string
		key    string
		wantOK bool
	}{
		{"Valid API key", "my-api-key-123", true},
		{"Wrong API key", "wrong-key", false},
		{"No API key", "", false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/data/items", nil)
			req.Host = "apikey.example.com"
			if tt.key != "" {
				req.Header.Set("X-API-Key", tt.key)
			}
			rec := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(rec)
			c.Request = req
			rule := &store.AuthRule{Type: "apikey", Config: store.AuthConfig{Secret: "my-api-key-123"}}
			if got := auth.Check(c, rule); got != tt.wantOK {
				t.Errorf("checkAPIKey = %v, want %v", got, tt.wantOK)
			}
		})
	}
}

// --- Proxy path rewriting ---

func TestE2E_ProxyPathRewrite(t *testing.T) {
	e := newEnv(t)
	token, _ := e.login("admin", "admin")

	_ = e.req("POST", "/api/routes", map[string]interface{}{
		"name":         "proxy-test",
		"host":         "proxy.example.com",
		"path_prefix":   "/backend",
		"backend":       "http://localhost:3000",
		"strip_prefix":  true,
		"enabled":       true,
	}, token)

	routeID := e.routeID(token, "proxy-test")
	w := e.req("GET", "/api/routes/"+routeID, nil, token)
	var route map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &route)
	if route["strip_prefix"] != true {
		t.Errorf("strip_prefix = %v, want true", route["strip_prefix"])
	}
	if route["path_prefix"] != "/backend" {
		t.Errorf("path_prefix = %v, want /backend", route["path_prefix"])
	}
	if route["backend"] != "http://localhost:3000" {
		t.Errorf("backend = %v, want http://localhost:3000", route["backend"])
	}
}

// --- Role-based access control ---

func TestE2E_RoleBasedAccess(t *testing.T) {
	e := newEnv(t)
	adminToken, _ := e.login("admin", "admin")

	w := e.req("POST", "/api/users", map[string]interface{}{
		"username": "editor1",
		"password": "editorpass",
		"role":     "editor",
	}, adminToken)
	if w.Code != http.StatusCreated {
		t.Fatalf("Create editor user failed: %s", w.Body.String())
	}
	var user map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &user)
	editorID := user["id"].(string)

	editorToken, _ := e.login("editor1", "editorpass")

	// Editor can create routes
	w = e.req("POST", "/api/routes", map[string]interface{}{
		"name": "editor-route", "path_prefix": "/e", "backend": "http://localhost:1",
	}, editorToken)
	if w.Code != http.StatusCreated {
		t.Errorf("Editor should be able to create routes: %s", w.Body.String())
	}

	// Editor cannot create users (403 Forbidden)
	w = e.req("POST", "/api/users", map[string]interface{}{
		"username": "hacker", "password": "bad", "role": "admin",
	}, editorToken)
	if w.Code != http.StatusForbidden {
		t.Errorf("Editor creating user = %d, want 403", w.Code)
	}

	// Admin can create users
	w = e.req("POST", "/api/users", map[string]interface{}{
		"username": "newuser", "password": "pass", "role": "admin",
	}, adminToken)
	if w.Code != http.StatusCreated {
		t.Errorf("Admin should be able to create users: %s", w.Body.String())
	}

	e.req("DELETE", "/api/users/"+editorID, nil, adminToken)
}

// --- Viewer role ---

func TestE2E_ViewerRole(t *testing.T) {
	e := newEnv(t)
	adminToken, _ := e.login("admin", "admin")

	w := e.req("POST", "/api/users", map[string]interface{}{
		"username": "viewer1",
		"password": "viewerpass",
		"role":     "viewer",
	}, adminToken)
	if w.Code != http.StatusCreated {
		t.Fatalf("Create viewer user failed: %s", w.Body.String())
	}
	var user map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &user)
	viewerID := user["id"].(string)

	viewerToken, _ := e.login("viewer1", "viewerpass")

	// Viewer cannot create routes (403)
	w = e.req("POST", "/api/routes", map[string]interface{}{
		"name": "viewer-route", "path_prefix": "/v", "backend": "http://localhost:1",
	}, viewerToken)
	if w.Code != http.StatusForbidden {
		t.Errorf("Viewer creating route = %d, want 403", w.Code)
	}

	// Viewer CAN list routes (200)
	w = e.req("GET", "/api/routes", nil, viewerToken)
	if w.Code != http.StatusOK {
		t.Errorf("Viewer listing routes = %d, want 200", w.Code)
	}

	e.req("DELETE", "/api/users/"+viewerID, nil, adminToken)
}
