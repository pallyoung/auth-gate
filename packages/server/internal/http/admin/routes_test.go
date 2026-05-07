package admin

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/pallyoung/auth-gate/packages/server/internal/auth"
	"github.com/pallyoung/auth-gate/packages/server/internal/router"
	"github.com/pallyoung/auth-gate/packages/server/internal/store"
)

func newTestDB(t *testing.T) *store.SQLite {
	t.Helper()

	db, err := store.NewSQLite(filepath.Join(t.TempDir(), "auth-gate.db"))
	if err != nil {
		t.Fatalf("NewSQLite() error = %v", err)
	}
	t.Cleanup(func() {
		_ = db.Close()
	})
	return db
}

func seedUser(t *testing.T, db *store.SQLite, username, password, role string) *store.User {
	t.Helper()

	hash, err := store.HashPassword(password)
	if err != nil {
		t.Fatalf("HashPassword() error = %v", err)
	}
	user := &store.User{
		Username:     username,
		PasswordHash: hash,
		Role:         role,
		Enabled:      true,
	}
	if err := db.CreateUser(user); err != nil {
		t.Fatalf("CreateUser() error = %v", err)
	}
	return user
}

func seedRoute(t *testing.T, db *store.SQLite) {
	t.Helper()

	if err := db.CreateRoute(&store.Route{
		ID:         "route-1",
		Name:       "svc",
		PathPrefix: "/svc",
		Backend:    "http://example.com",
		Enabled:    true,
	}); err != nil {
		t.Fatalf("CreateRoute() error = %v", err)
	}
}

func performRequest(t *testing.T, engine *gin.Engine, method, path string, token string, body any) *httptest.ResponseRecorder {
	t.Helper()

	var payload []byte
	if body != nil {
		var err error
		payload, err = json.Marshal(body)
		if err != nil {
			t.Fatalf("json.Marshal() error = %v", err)
		}
	}
	req := httptest.NewRequest(method, path, bytes.NewReader(payload))
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp := httptest.NewRecorder()
	engine.ServeHTTP(resp, req)
	return resp
}

func TestRegisterRoutes_UsesStructuredErrorEnvelope(t *testing.T) {
	gin.SetMode(gin.TestMode)
	auth.ConfigureJWTSecret("test-secret")

	db := newTestDB(t)
	user := seedUser(t, db, "admin", "password123", store.RoleAdmin)
	engine := gin.New()
	group := engine.Group("/_authgate/api")
	group.Use(auth.AuthMiddleware(db))
	RegisterRoutes(group, router.NewManager(db), db)

	token, err := auth.GenerateToken(user.ID, user.Username, user.Role)
	if err != nil {
		t.Fatalf("GenerateToken() error = %v", err)
	}

	resp := performRequest(t, engine, http.MethodPost, "/_authgate/api/routes", token, map[string]any{
		"name":        "broken",
		"path_prefix": "svc",
		"backend":     "ftp://example.com",
	})
	if resp.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d, body=%s", resp.Code, http.StatusBadRequest, resp.Body.String())
	}
	if resp.Body.String() != "{\"error\":{\"code\":\"invalid_route_path_prefix\",\"message\":\"path_prefix must start with /\"}}" &&
		resp.Body.String() != "{\"error\":{\"code\":\"invalid_route_backend\",\"message\":\"backend must be a valid http or https URL\"}}" {
		t.Fatalf("unexpected body = %s", resp.Body.String())
	}
}

func TestRegisterRoutes_MeReturnsPermissions(t *testing.T) {
	gin.SetMode(gin.TestMode)
	auth.ConfigureJWTSecret("test-secret")

	db := newTestDB(t)
	user := seedUser(t, db, "editor", "password123", store.RoleEditor)
	engine := gin.New()
	engine.POST("/_authgate/api/auth/login", LoginRoute(db))
	group := engine.Group("/_authgate/api")
	group.Use(auth.AuthMiddleware(db))
	RegisterRoutes(group, router.NewManager(db), db)

	token, err := auth.GenerateToken(user.ID, user.Username, user.Role)
	if err != nil {
		t.Fatalf("GenerateToken() error = %v", err)
	}

	resp := performRequest(t, engine, http.MethodGet, "/_authgate/api/auth/me", token, nil)
	if resp.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body=%s", resp.Code, http.StatusOK, resp.Body.String())
	}
	want := "{\"id\":\"" + user.ID + "\",\"username\":\"editor\",\"role\":\"editor\",\"permissions\":{\"can_manage_routes\":true,\"can_manage_auth\":true,\"can_manage_users\":false,\"can_view_logs\":true}}"
	if resp.Body.String() != want {
		t.Fatalf("body = %s, want %s", resp.Body.String(), want)
	}
}

func TestLoginRoute_ReturnsStructuredSession(t *testing.T) {
	gin.SetMode(gin.TestMode)
	auth.ConfigureJWTSecret("test-secret")

	db := newTestDB(t)
	seedUser(t, db, "admin", "password123", store.RoleAdmin)
	engine := gin.New()
	engine.POST("/_authgate/api/auth/login", LoginRoute(db))

	resp := performRequest(t, engine, http.MethodPost, "/_authgate/api/auth/login", "", map[string]any{
		"username": "admin",
		"password": "password123",
	})
	if resp.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body=%s", resp.Code, http.StatusOK, resp.Body.String())
	}

	var payload map[string]any
	if err := json.Unmarshal(resp.Body.Bytes(), &payload); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if payload["token"] == "" {
		t.Fatalf("token missing in response: %s", resp.Body.String())
	}
	perms := payload["permissions"].(map[string]any)
	if perms["can_manage_users"] != true {
		t.Fatalf("permissions.can_manage_users = %v, want true", perms["can_manage_users"])
	}
}

func TestLoginRoute_RejectsRouteOnlyUserFromControlPlane(t *testing.T) {
	gin.SetMode(gin.TestMode)
	auth.ConfigureJWTSecret("test-secret")

	db := newTestDB(t)
	seedUser(t, db, "member", "password123", store.RoleMember)
	engine := gin.New()
	engine.POST("/_authgate/api/auth/login", LoginRoute(db))

	resp := performRequest(t, engine, http.MethodPost, "/_authgate/api/auth/login", "", map[string]any{
		"username": "member",
		"password": "password123",
	})
	if resp.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d, body=%s", resp.Code, http.StatusForbidden, resp.Body.String())
	}
	if resp.Body.String() != "{\"error\":{\"code\":\"control_plane_access_denied\",\"message\":\"control plane access denied\"}}" {
		t.Fatalf("body = %s", resp.Body.String())
	}
}

func TestRegisterRoutes_CreateAuthRuleRedactsSecrets(t *testing.T) {
	gin.SetMode(gin.TestMode)
	auth.ConfigureJWTSecret("test-secret")

	db := newTestDB(t)
	seedRoute(t, db)
	user := seedUser(t, db, "admin", "password123", store.RoleAdmin)
	engine := gin.New()
	group := engine.Group("/_authgate/api")
	group.Use(auth.AuthMiddleware(db))
	RegisterRoutes(group, router.NewManager(db), db)

	token, err := auth.GenerateToken(user.ID, user.Username, user.Role)
	if err != nil {
		t.Fatalf("GenerateToken() error = %v", err)
	}

	resp := performRequest(t, engine, http.MethodPost, "/_authgate/api/auth-rules", token, map[string]any{
		"route_id": "route-1",
		"type":     "bearer",
		"config": map[string]any{
			"secret": "shared-secret",
		},
	})
	if resp.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d, body=%s", resp.Code, http.StatusCreated, resp.Body.String())
	}
	if bytes.Contains(resp.Body.Bytes(), []byte("shared-secret")) {
		t.Fatalf("response leaked secret: %s", resp.Body.String())
	}
}

func TestRegisterRoutes_MeRejectsDisabledUserWithOldToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	auth.ConfigureJWTSecret("test-secret")

	db := newTestDB(t)
	user := seedUser(t, db, "viewer", "password123", store.RoleViewer)
	engine := gin.New()
	group := engine.Group("/_authgate/api")
	group.Use(auth.AuthMiddleware(db))
	RegisterRoutes(group, router.NewManager(db), db)

	token, err := auth.GenerateToken(user.ID, user.Username, user.Role)
	if err != nil {
		t.Fatalf("GenerateToken() error = %v", err)
	}

	user.Enabled = false
	if err := db.UpdateUser(user); err != nil {
		t.Fatalf("UpdateUser() error = %v", err)
	}

	resp := performRequest(t, engine, http.MethodGet, "/_authgate/api/auth/me", token, nil)
	if resp.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d, body=%s", resp.Code, http.StatusUnauthorized, resp.Body.String())
	}
}
