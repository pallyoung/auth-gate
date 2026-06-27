package admin

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/pallyoung/auth-gate/packages/server/internal/auth"
	"github.com/pallyoung/auth-gate/packages/server/internal/localca"
	"github.com/pallyoung/auth-gate/packages/server/internal/router"
	certservice "github.com/pallyoung/auth-gate/packages/server/internal/service/certificate"
	"github.com/pallyoung/auth-gate/packages/server/internal/store"
)

func newTestDB(t *testing.T) store.Store {
	t.Helper()

	db, err := store.NewJSONStore(t.TempDir())
	if err != nil {
		t.Fatalf("NewJSONStore() error = %v", err)
	}
	t.Cleanup(func() {
		_ = db.Close()
	})
	return db
}

func seedUser(t *testing.T, db store.Store, username, password, role string) *store.User {
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

func seedRoute(t *testing.T, db store.Store) {
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

// stubCertService implements CertService for tests. Each method can be
// overridden via the corresponding field, otherwise it returns a benign
// zero result so the harness can wire it up to RegisterRoutes.
type stubCertService struct {
	listFn        func() ([]store.Certificate, error)
	getFn         func(id string) (*store.Certificate, error)
	provisionFn   func(ctx context.Context, name, domain string, info *localca.SubjectInfo) (*store.Certificate, error)
	importFn      func(ctx context.Context, name, domain, certPEM, keyPEM string) (*store.Certificate, error)
	resignFn      func(id string) (*store.Certificate, error)
	deleteFn      func(id string) error
	caExportFn    func() (certPEM, name string, notAfter time.Time, err error)
}

func (s *stubCertService) List() ([]store.Certificate, error) {
	if s.listFn != nil {
		return s.listFn()
	}
	return nil, nil
}

func (s *stubCertService) Get(id string) (*store.Certificate, error) {
	if s.getFn != nil {
		return s.getFn(id)
	}
	return nil, nil
}

func (s *stubCertService) ProvisionLocal(ctx context.Context, name, domain string, days int, info *localca.SubjectInfo) (*store.Certificate, error) {
	if s.provisionFn != nil {
		return s.provisionFn(ctx, name, domain, info)
	}
	return nil, nil
}

func (s *stubCertService) Import(ctx context.Context, name, domain, certPEM, keyPEM string) (*store.Certificate, error) {
	if s.importFn != nil {
		return s.importFn(ctx, name, domain, certPEM, keyPEM)
	}
	return nil, nil
}

func (s *stubCertService) Resign(id string) (*store.Certificate, error) {
	if s.resignFn != nil {
		return s.resignFn(id)
	}
	return nil, nil
}

func (s *stubCertService) Delete(id string) error {
	if s.deleteFn != nil {
		return s.deleteFn(id)
	}
	return nil
}

func (s *stubCertService) GetCAExport() (string, string, time.Time, error) {
	if s.caExportFn != nil {
		return s.caExportFn()
	}
	return "", "", time.Time{}, nil
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
	RegisterRoutes(group, router.NewManager(db), db, nil, nil, nil)

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

func TestRegisterRoutes_CreateRouteAcceptsRegexPathMatchMode(t *testing.T) {
	gin.SetMode(gin.TestMode)
	auth.ConfigureJWTSecret("test-secret")

	db := newTestDB(t)
	user := seedUser(t, db, "admin", "password123", store.RoleAdmin)
	engine := gin.New()
	group := engine.Group("/_authgate/api")
	group.Use(auth.AuthMiddleware(db))
	RegisterRoutes(group, router.NewManager(db), db, nil, nil, nil)

	token, err := auth.GenerateToken(user.ID, user.Username, user.Role)
	if err != nil {
		t.Fatalf("GenerateToken() error = %v", err)
	}

	resp := performRequest(t, engine, http.MethodPost, "/_authgate/api/routes", token, map[string]any{
		"name":            "regex-route",
		"path_prefix":     "^/api/v\\d+",
		"path_match_mode": "regex",
		"backend":         "http://example.com",
		"enabled":         true,
	})
	if resp.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d, body=%s", resp.Code, http.StatusCreated, resp.Body.String())
	}

	var payload map[string]any
	if err := json.Unmarshal(resp.Body.Bytes(), &payload); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if payload["path_match_mode"] != "regex" {
		t.Fatalf("path_match_mode = %v, want %q", payload["path_match_mode"], "regex")
	}
	if payload["path_prefix"] != "^/api/v\\d+" {
		t.Fatalf("path_prefix = %v, want %q", payload["path_prefix"], "^/api/v\\d+")
	}
}

func TestRegisterRoutes_CreateRouteNormalizesExplicitPathMatchMode(t *testing.T) {
	gin.SetMode(gin.TestMode)
	auth.ConfigureJWTSecret("test-secret")

	db := newTestDB(t)
	user := seedUser(t, db, "admin", "password123", store.RoleAdmin)
	engine := gin.New()
	group := engine.Group("/_authgate/api")
	group.Use(auth.AuthMiddleware(db))
	RegisterRoutes(group, router.NewManager(db), db, nil, nil, nil)

	token, err := auth.GenerateToken(user.ID, user.Username, user.Role)
	if err != nil {
		t.Fatalf("GenerateToken() error = %v", err)
	}

	resp := performRequest(t, engine, http.MethodPost, "/_authgate/api/routes", token, map[string]any{
		"name":            "regex-route",
		"path_prefix":     "^/api/v\\d+",
		"path_match_mode": " REGEX_I ",
		"backend":         "http://example.com",
		"enabled":         true,
	})
	if resp.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d, body=%s", resp.Code, http.StatusCreated, resp.Body.String())
	}

	var payload map[string]any
	if err := json.Unmarshal(resp.Body.Bytes(), &payload); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if payload["path_match_mode"] != "regex_i" {
		t.Fatalf("path_match_mode = %v, want %q", payload["path_match_mode"], "regex_i")
	}
}

func TestRegisterRoutes_RejectsInvalidRoutePathMatchMode(t *testing.T) {
	gin.SetMode(gin.TestMode)
	auth.ConfigureJWTSecret("test-secret")

	db := newTestDB(t)
	user := seedUser(t, db, "admin", "password123", store.RoleAdmin)
	engine := gin.New()
	group := engine.Group("/_authgate/api")
	group.Use(auth.AuthMiddleware(db))
	RegisterRoutes(group, router.NewManager(db), db, nil, nil, nil)

	token, err := auth.GenerateToken(user.ID, user.Username, user.Role)
	if err != nil {
		t.Fatalf("GenerateToken() error = %v", err)
	}

	resp := performRequest(t, engine, http.MethodPost, "/_authgate/api/routes", token, map[string]any{
		"name":            "invalid-mode",
		"path_prefix":     "/api",
		"path_match_mode": "glob",
		"backend":         "http://example.com",
		"enabled":         true,
	})
	if resp.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d, body=%s", resp.Code, http.StatusBadRequest, resp.Body.String())
	}
	if resp.Body.String() != "{\"error\":{\"code\":\"invalid_route_path_match_mode\",\"message\":\"path_match_mode must be one of prefix, exact, stop, regex, or regex_i\"}}" {
		t.Fatalf("body = %s", resp.Body.String())
	}
}

func TestRegisterRoutes_RejectsInvalidRouteRegexPathPrefix(t *testing.T) {
	gin.SetMode(gin.TestMode)
	auth.ConfigureJWTSecret("test-secret")

	db := newTestDB(t)
	user := seedUser(t, db, "admin", "password123", store.RoleAdmin)
	engine := gin.New()
	group := engine.Group("/_authgate/api")
	group.Use(auth.AuthMiddleware(db))
	RegisterRoutes(group, router.NewManager(db), db, nil, nil, nil)

	token, err := auth.GenerateToken(user.ID, user.Username, user.Role)
	if err != nil {
		t.Fatalf("GenerateToken() error = %v", err)
	}

	resp := performRequest(t, engine, http.MethodPost, "/_authgate/api/routes", token, map[string]any{
		"name":            "invalid-regex-route",
		"path_prefix":     "[",
		"path_match_mode": "regex",
		"backend":         "http://example.com",
		"enabled":         true,
	})
	if resp.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d, body=%s", resp.Code, http.StatusBadRequest, resp.Body.String())
	}
	if resp.Body.String() != "{\"error\":{\"code\":\"invalid_route_path_regex\",\"message\":\"path_prefix must be a valid regular expression for the selected path match mode\"}}" {
		t.Fatalf("body = %s", resp.Body.String())
	}
}

func TestRegisterRoutes_CreateRouteNormalizesHostCase(t *testing.T) {
	gin.SetMode(gin.TestMode)
	auth.ConfigureJWTSecret("test-secret")

	db := newTestDB(t)
	user := seedUser(t, db, "admin", "password123", store.RoleAdmin)
	engine := gin.New()
	group := engine.Group("/_authgate/api")
	group.Use(auth.AuthMiddleware(db))
	RegisterRoutes(group, router.NewManager(db), db, nil, nil, nil)

	token, err := auth.GenerateToken(user.ID, user.Username, user.Role)
	if err != nil {
		t.Fatalf("GenerateToken() error = %v", err)
	}

	resp := performRequest(t, engine, http.MethodPost, "/_authgate/api/routes", token, map[string]any{
		"name":        "host-route",
		"host":        " API.EXAMPLE.COM ",
		"path_prefix": "/api",
		"backend":     "http://example.com",
		"enabled":     true,
	})
	if resp.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d, body=%s", resp.Code, http.StatusCreated, resp.Body.String())
	}

	var payload map[string]any
	if err := json.Unmarshal(resp.Body.Bytes(), &payload); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if payload["host"] != "api.example.com" {
		t.Fatalf("host = %v, want %q", payload["host"], "api.example.com")
	}
}

func TestRegisterRoutes_RejectsInvalidRouteHost(t *testing.T) {
	gin.SetMode(gin.TestMode)
	auth.ConfigureJWTSecret("test-secret")

	db := newTestDB(t)
	user := seedUser(t, db, "admin", "password123", store.RoleAdmin)
	engine := gin.New()
	group := engine.Group("/_authgate/api")
	group.Use(auth.AuthMiddleware(db))
	RegisterRoutes(group, router.NewManager(db), db, nil, nil, nil)

	token, err := auth.GenerateToken(user.ID, user.Username, user.Role)
	if err != nil {
		t.Fatalf("GenerateToken() error = %v", err)
	}

	resp := performRequest(t, engine, http.MethodPost, "/_authgate/api/routes", token, map[string]any{
		"name":        "host-route",
		"host":        "https://api.example.com",
		"path_prefix": "/api",
		"backend":     "http://example.com",
		"enabled":     true,
	})
	if resp.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d, body=%s", resp.Code, http.StatusBadRequest, resp.Body.String())
	}
	if resp.Body.String() != "{\"error\":{\"code\":\"invalid_route_host\",\"message\":\"host must be a hostname or IP address without scheme, port, or path\"}}" {
		t.Fatalf("body = %s", resp.Body.String())
	}
}

func TestRegisterRoutes_RejectsInvalidRouteBackends(t *testing.T) {
	gin.SetMode(gin.TestMode)
	auth.ConfigureJWTSecret("test-secret")

	db := newTestDB(t)
	user := seedUser(t, db, "admin", "password123", store.RoleAdmin)
	engine := gin.New()
	group := engine.Group("/_authgate/api")
	group.Use(auth.AuthMiddleware(db))
	RegisterRoutes(group, router.NewManager(db), db, nil, nil, nil)

	token, err := auth.GenerateToken(user.ID, user.Username, user.Role)
	if err != nil {
		t.Fatalf("GenerateToken() error = %v", err)
	}

	resp := performRequest(t, engine, http.MethodPost, "/_authgate/api/routes", token, map[string]any{
		"name":        "lb-route",
		"path_prefix": "/api",
		"backend":     "http://example.com",
		"backends": []map[string]any{
			{"url": "http://backend-a.example.com", "weight": 1},
			{"url": "ftp://backend-b.example.com", "weight": 1},
		},
		"enabled": true,
	})
	if resp.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d, body=%s", resp.Code, http.StatusBadRequest, resp.Body.String())
	}
	if resp.Body.String() != "{\"error\":{\"code\":\"invalid_route_backend\",\"message\":\"backend must be a valid http or https URL\"}}" {
		t.Fatalf("body = %s", resp.Body.String())
	}
}

func TestRegisterRoutes_CreateRouteAllowsBackendsWithoutLegacyBackend(t *testing.T) {
	gin.SetMode(gin.TestMode)
	auth.ConfigureJWTSecret("test-secret")

	db := newTestDB(t)
	user := seedUser(t, db, "admin", "password123", store.RoleAdmin)
	engine := gin.New()
	group := engine.Group("/_authgate/api")
	group.Use(auth.AuthMiddleware(db))
	RegisterRoutes(group, router.NewManager(db), db, nil, nil, nil)

	token, err := auth.GenerateToken(user.ID, user.Username, user.Role)
	if err != nil {
		t.Fatalf("GenerateToken() error = %v", err)
	}

	resp := performRequest(t, engine, http.MethodPost, "/_authgate/api/routes", token, map[string]any{
		"name":        "lb-route",
		"path_prefix": "/api",
		"backends": []map[string]any{
			{"url": "http://backend-a.example.com", "weight": 2},
			{"url": "http://backend-b.example.com", "weight": 1},
		},
		"enabled": true,
	})
	if resp.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d, body=%s", resp.Code, http.StatusCreated, resp.Body.String())
	}

	var payload map[string]any
	if err := json.Unmarshal(resp.Body.Bytes(), &payload); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if payload["backend"] != "" {
		t.Fatalf("response backend = %v, want empty string", payload["backend"])
	}
	backends, ok := payload["backends"].([]any)
	if !ok || len(backends) != 2 {
		t.Fatalf("response backends = %v, want 2 entries", payload["backends"])
	}
}

func TestRegisterRoutes_CreateRouteRequiresBackendOrBackends(t *testing.T) {
	gin.SetMode(gin.TestMode)
	auth.ConfigureJWTSecret("test-secret")

	db := newTestDB(t)
	user := seedUser(t, db, "admin", "password123", store.RoleAdmin)
	engine := gin.New()
	group := engine.Group("/_authgate/api")
	group.Use(auth.AuthMiddleware(db))
	RegisterRoutes(group, router.NewManager(db), db, nil, nil, nil)

	token, err := auth.GenerateToken(user.ID, user.Username, user.Role)
	if err != nil {
		t.Fatalf("GenerateToken() error = %v", err)
	}

	resp := performRequest(t, engine, http.MethodPost, "/_authgate/api/routes", token, map[string]any{
		"name":        "broken-route",
		"path_prefix": "/api",
		"enabled":     true,
	})
	if resp.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d, body=%s", resp.Code, http.StatusBadRequest, resp.Body.String())
	}
	if resp.Body.String() != "{\"error\":{\"code\":\"missing_route_fields\",\"message\":\"backend or backends required\"}}" {
		t.Fatalf("body = %s", resp.Body.String())
	}
}

func TestRegisterRoutes_RejectsInvalidRouteBackendWeights(t *testing.T) {
	gin.SetMode(gin.TestMode)
	auth.ConfigureJWTSecret("test-secret")

	db := newTestDB(t)
	user := seedUser(t, db, "admin", "password123", store.RoleAdmin)
	engine := gin.New()
	group := engine.Group("/_authgate/api")
	group.Use(auth.AuthMiddleware(db))
	RegisterRoutes(group, router.NewManager(db), db, nil, nil, nil)

	token, err := auth.GenerateToken(user.ID, user.Username, user.Role)
	if err != nil {
		t.Fatalf("GenerateToken() error = %v", err)
	}

	resp := performRequest(t, engine, http.MethodPost, "/_authgate/api/routes", token, map[string]any{
		"name":        "lb-route",
		"path_prefix": "/api",
		"backend":     "http://example.com",
		"backends": []map[string]any{
			{"url": "http://backend-a.example.com", "weight": 1},
			{"url": "http://backend-b.example.com", "weight": 0},
		},
		"enabled": true,
	})
	if resp.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d, body=%s", resp.Code, http.StatusBadRequest, resp.Body.String())
	}
	if resp.Body.String() != "{\"error\":{\"code\":\"invalid_route_backend_weight\",\"message\":\"backend weight must be greater than 0\"}}" {
		t.Fatalf("body = %s", resp.Body.String())
	}
}

func TestRegisterRoutes_RejectsInvalidRouteRedirectCode(t *testing.T) {
	gin.SetMode(gin.TestMode)
	auth.ConfigureJWTSecret("test-secret")

	db := newTestDB(t)
	user := seedUser(t, db, "admin", "password123", store.RoleAdmin)
	engine := gin.New()
	group := engine.Group("/_authgate/api")
	group.Use(auth.AuthMiddleware(db))
	RegisterRoutes(group, router.NewManager(db), db, nil, nil, nil)

	token, err := auth.GenerateToken(user.ID, user.Username, user.Role)
	if err != nil {
		t.Fatalf("GenerateToken() error = %v", err)
	}

	resp := performRequest(t, engine, http.MethodPost, "/_authgate/api/routes", token, map[string]any{
		"name":           "redirect-route",
		"path_prefix":    "/billing",
		"backend":        "http://example.com",
		"rewrite_target": "https://example.com/billing",
		"redirect_code":  307,
		"enabled":        true,
	})
	if resp.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d, body=%s", resp.Code, http.StatusBadRequest, resp.Body.String())
	}
	if resp.Body.String() != "{\"error\":{\"code\":\"invalid_route_redirect_code\",\"message\":\"redirect_code must be 0, 301, or 302\"}}" {
		t.Fatalf("body = %s", resp.Body.String())
	}
}

func TestRegisterRoutes_CreateRoutePersistsAndReturnsRuntimePolicyFields(t *testing.T) {
	gin.SetMode(gin.TestMode)
	auth.ConfigureJWTSecret("test-secret")

	db := newTestDB(t)
	user := seedUser(t, db, "admin", "password123", store.RoleAdmin)
	engine := gin.New()
	group := engine.Group("/_authgate/api")
	group.Use(auth.AuthMiddleware(db))
	RegisterRoutes(group, router.NewManager(db), db, nil, nil, nil)

	token, err := auth.GenerateToken(user.ID, user.Username, user.Role)
	if err != nil {
		t.Fatalf("GenerateToken() error = %v", err)
	}

	resp := performRequest(t, engine, http.MethodPost, "/_authgate/api/routes", token, map[string]any{
		"name":           "runtime-policy-route",
		"host":           "api.example.com",
		"path_prefix":    "/api",
		"backend":        "http://example.com",
		"enabled":        true,
		"timeout_ms":     4500,
		"retry_attempts": 3,
	})
	if resp.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d, body=%s", resp.Code, http.StatusCreated, resp.Body.String())
	}

	var payload map[string]any
	if err := json.Unmarshal(resp.Body.Bytes(), &payload); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if payload["timeout_ms"] != float64(4500) {
		t.Fatalf("response timeout_ms = %v, want %d", payload["timeout_ms"], 4500)
	}
	if payload["retry_attempts"] != float64(3) {
		t.Fatalf("response retry_attempts = %v, want %d", payload["retry_attempts"], 3)
	}

	routes, err := db.ListRoutes()
	if err != nil {
		t.Fatalf("ListRoutes() error = %v", err)
	}
	if len(routes) != 1 {
		t.Fatalf("len(routes) = %d, want 1", len(routes))
	}
	if routes[0].TimeoutMs != 4500 {
		t.Fatalf("stored TimeoutMs = %d, want %d", routes[0].TimeoutMs, 4500)
	}
	if routes[0].RetryAttempts != 3 {
		t.Fatalf("stored RetryAttempts = %d, want %d", routes[0].RetryAttempts, 3)
	}
}

func TestRegisterRoutes_UpdateRoutePreservesOmittedFields(t *testing.T) {
	gin.SetMode(gin.TestMode)
	auth.ConfigureJWTSecret("test-secret")

	db := newTestDB(t)
	user := seedUser(t, db, "admin", "password123", store.RoleAdmin)
	if err := db.CreateRoute(&store.Route{
		ID:            "route-preserve",
		Name:          "svc",
		Host:          "api.example.com",
		PathPrefix:    "/svc",
		Backend:       "http://example.com",
		StripPrefix:   true,
		Enabled:       true,
		Priority:      9,
		TLSCert:       "/etc/ssl/certs/site.pem",
		TLSKey:        "/etc/ssl/private/site.key",
		TLSEnabled:    true,
		TimeoutMs:     4500,
		RetryAttempts: 3,
		Backends: []store.Backend{
			{URL: "http://backend-a.example.com", Weight: 3},
			{URL: "http://backend-b.example.com", Weight: 1},
		},
		PathMatchMode: "exact",
		RewriteTarget: "/internal",
		RedirectCode:  301,
	}); err != nil {
		t.Fatalf("CreateRoute() error = %v", err)
	}

	engine := gin.New()
	group := engine.Group("/_authgate/api")
	group.Use(auth.AuthMiddleware(db))
	RegisterRoutes(group, router.NewManager(db), db, nil, nil, nil)

	token, err := auth.GenerateToken(user.ID, user.Username, user.Role)
	if err != nil {
		t.Fatalf("GenerateToken() error = %v", err)
	}

	resp := performRequest(t, engine, http.MethodPut, "/_authgate/api/routes/route-preserve", token, map[string]any{
		"name": "svc-renamed",
	})
	if resp.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body=%s", resp.Code, http.StatusOK, resp.Body.String())
	}

	updatedRoute, err := db.GetRoute("route-preserve")
	if err != nil {
		t.Fatalf("GetRoute() error = %v", err)
	}
	if updatedRoute.PathPrefix != "/svc" {
		t.Fatalf("updatedRoute.PathPrefix = %q, want %q", updatedRoute.PathPrefix, "/svc")
	}
	if updatedRoute.Backend != "http://example.com" {
		t.Fatalf("updatedRoute.Backend = %q, want %q", updatedRoute.Backend, "http://example.com")
	}
	if !updatedRoute.StripPrefix {
		t.Fatalf("updatedRoute.StripPrefix = %v, want true", updatedRoute.StripPrefix)
	}
	if !updatedRoute.Enabled {
		t.Fatalf("updatedRoute.Enabled = %v, want true", updatedRoute.Enabled)
	}
	if updatedRoute.Priority != 9 {
		t.Fatalf("updatedRoute.Priority = %d, want %d", updatedRoute.Priority, 9)
	}
	if updatedRoute.TLSCert != "/etc/ssl/certs/site.pem" {
		t.Fatalf("updatedRoute.TLSCert = %q, want %q", updatedRoute.TLSCert, "/etc/ssl/certs/site.pem")
	}
	if updatedRoute.TLSKey != "/etc/ssl/private/site.key" {
		t.Fatalf("updatedRoute.TLSKey = %q, want %q", updatedRoute.TLSKey, "/etc/ssl/private/site.key")
	}
	if !updatedRoute.TLSEnabled {
		t.Fatalf("updatedRoute.TLSEnabled = %v, want true", updatedRoute.TLSEnabled)
	}
	if updatedRoute.TimeoutMs != 4500 {
		t.Fatalf("updatedRoute.TimeoutMs = %d, want %d", updatedRoute.TimeoutMs, 4500)
	}
	if updatedRoute.RetryAttempts != 3 {
		t.Fatalf("updatedRoute.RetryAttempts = %d, want %d", updatedRoute.RetryAttempts, 3)
	}
	if len(updatedRoute.Backends) != 2 {
		t.Fatalf("len(updatedRoute.Backends) = %d, want %d", len(updatedRoute.Backends), 2)
	}

	var payload map[string]any
	if err := json.Unmarshal(resp.Body.Bytes(), &payload); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if payload["path_prefix"] != "/svc" {
		t.Fatalf("response path_prefix = %v, want %q", payload["path_prefix"], "/svc")
	}
	if payload["backend"] != "http://example.com" {
		t.Fatalf("response backend = %v, want %q", payload["backend"], "http://example.com")
	}
	if payload["enabled"] != true {
		t.Fatalf("response enabled = %v, want true", payload["enabled"])
	}
	if payload["strip_prefix"] != true {
		t.Fatalf("response strip_prefix = %v, want true", payload["strip_prefix"])
	}
	if payload["priority"] != float64(9) {
		t.Fatalf("response priority = %v, want %d", payload["priority"], 9)
	}
	if payload["tls_cert"] != "/etc/ssl/certs/site.pem" {
		t.Fatalf("response tls_cert = %v, want %q", payload["tls_cert"], "/etc/ssl/certs/site.pem")
	}
	if payload["tls_key"] != "/etc/ssl/private/site.key" {
		t.Fatalf("response tls_key = %v, want %q", payload["tls_key"], "/etc/ssl/private/site.key")
	}
	if payload["tls_enabled"] != true {
		t.Fatalf("response tls_enabled = %v, want true", payload["tls_enabled"])
	}
	if payload["timeout_ms"] != float64(4500) {
		t.Fatalf("response timeout_ms = %v, want %d", payload["timeout_ms"], 4500)
	}
	if payload["retry_attempts"] != float64(3) {
		t.Fatalf("response retry_attempts = %v, want %d", payload["retry_attempts"], 3)
	}
}

func TestRegisterRoutes_MeReturnsPermissions(t *testing.T) {
	gin.SetMode(gin.TestMode)
	auth.ConfigureJWTSecret("test-secret")

	db := newTestDB(t)
	user := seedUser(t, db, "editor", "password123", store.RoleEditor)
	engine := gin.New()
	engine.POST("/_authgate/api/auth/login", LoginRoute(db, nil))
	group := engine.Group("/_authgate/api")
	group.Use(auth.AuthMiddleware(db))
	RegisterRoutes(group, router.NewManager(db), db, nil, nil, nil)

	token, err := auth.GenerateToken(user.ID, user.Username, user.Role)
	if err != nil {
		t.Fatalf("GenerateToken() error = %v", err)
	}

	resp := performRequest(t, engine, http.MethodGet, "/_authgate/api/auth/me", token, nil)
	if resp.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body=%s", resp.Code, http.StatusOK, resp.Body.String())
	}

	var payload map[string]any
	if err := json.Unmarshal(resp.Body.Bytes(), &payload); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	perms := payload["permissions"].(map[string]any)
	if perms["can_manage_routes"] != true || perms["can_manage_auth"] != true || perms["can_manage_users"] != false || perms["can_view_logs"] != true {
		t.Fatalf("permissions = %#v", perms)
	}

	features := payload["features"].(map[string]any)
	if features["certificates"] != false {
		t.Fatalf("features.certificates = %v, want false", features["certificates"])
	}
}

func TestRegisterRoutes_MeReportsCertificateFeatureAvailability(t *testing.T) {
	testCases := []struct {
		name        string
		certSvc     CertService
		wantEnabled bool
	}{
		{
			name:        "disabled when certificate service is unavailable",
			certSvc:     nil,
			wantEnabled: false,
		},
		{
			name:        "enabled when certificate service is available",
			certSvc:     &stubCertService{},
			wantEnabled: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)
			auth.ConfigureJWTSecret("test-secret")

			db := newTestDB(t)
			user := seedUser(t, db, "editor", "password123", store.RoleEditor)
			engine := gin.New()
			group := engine.Group("/_authgate/api")
			group.Use(auth.AuthMiddleware(db))
			RegisterRoutes(group, router.NewManager(db), db, tc.certSvc, nil, nil)

			token, err := auth.GenerateToken(user.ID, user.Username, user.Role)
			if err != nil {
				t.Fatalf("GenerateToken() error = %v", err)
			}

			resp := performRequest(t, engine, http.MethodGet, "/_authgate/api/auth/me", token, nil)
			if resp.Code != http.StatusOK {
				t.Fatalf("status = %d, want %d, body=%s", resp.Code, http.StatusOK, resp.Body.String())
			}

			var payload map[string]any
			if err := json.Unmarshal(resp.Body.Bytes(), &payload); err != nil {
				t.Fatalf("json.Unmarshal() error = %v", err)
			}

			features := payload["features"].(map[string]any)
			if features["certificates"] != tc.wantEnabled {
				t.Fatalf("features.certificates = %v, want %v", features["certificates"], tc.wantEnabled)
			}
		})
	}
}

func TestRegisterRoutes_RejectsCertificateCreateWhenNameIsMissing(t *testing.T) {
	gin.SetMode(gin.TestMode)
	auth.ConfigureJWTSecret("test-secret")

	db := newTestDB(t)
	user := seedUser(t, db, "editor", "password123", store.RoleEditor)
	stub := &stubCertService{
		provisionFn: func(ctx context.Context, name, domain string, info *localca.SubjectInfo) (*store.Certificate, error) {
			return nil, certservice.NewError(certservice.ErrCodeInvalidName, "certificate name required", nil)
		},
	}
	engine := gin.New()
	group := engine.Group("/_authgate/api")
	group.Use(auth.AuthMiddleware(db))
	RegisterRoutes(group, router.NewManager(db), db, stub, nil, nil)

	token, err := auth.GenerateToken(user.ID, user.Username, user.Role)
	if err != nil {
		t.Fatalf("GenerateToken() error = %v", err)
	}

	resp := performRequest(t, engine, http.MethodPost, "/_authgate/api/certificates", token, map[string]any{
		"name":   "",
		"domain": "*.example.com",
		"source": "local_ca",
	})
	if resp.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d, body=%s", resp.Code, http.StatusBadRequest, resp.Body.String())
	}
	if resp.Body.String() != `{"error":{"code":"invalid_request","message":"Key: 'CertificateWriteRequest.Name' Error:Field validation for 'Name' failed on the 'required' tag"}}` {
		t.Fatalf("body = %s", resp.Body.String())
	}
}

func TestRegisterRoutes_RejectsCertificateCreateWhenDomainIsMissing(t *testing.T) {
	gin.SetMode(gin.TestMode)
	auth.ConfigureJWTSecret("test-secret")

	db := newTestDB(t)
	user := seedUser(t, db, "editor", "password123", store.RoleEditor)
	stub := &stubCertService{}
	engine := gin.New()
	group := engine.Group("/_authgate/api")
	group.Use(auth.AuthMiddleware(db))
	RegisterRoutes(group, router.NewManager(db), db, stub, nil, nil)

	token, err := auth.GenerateToken(user.ID, user.Username, user.Role)
	if err != nil {
		t.Fatalf("GenerateToken() error = %v", err)
	}

	resp := performRequest(t, engine, http.MethodPost, "/_authgate/api/certificates", token, map[string]any{
		"name":   "Wildcard",
		"domain": "",
		"source": "local_ca",
	})
	if resp.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d, body=%s", resp.Code, http.StatusBadRequest, resp.Body.String())
	}
	if resp.Body.String() != `{"error":{"code":"invalid_request","message":"Key: 'CertificateWriteRequest.Domain' Error:Field validation for 'Domain' failed on the 'required' tag"}}` {
		t.Fatalf("body = %s", resp.Body.String())
	}
}

func TestRegisterRoutes_RejectsCertificateCreateWhenSourceIsUnsupported(t *testing.T) {
	gin.SetMode(gin.TestMode)
	auth.ConfigureJWTSecret("test-secret")

	db := newTestDB(t)
	user := seedUser(t, db, "editor", "password123", store.RoleEditor)
	stub := &stubCertService{}
	engine := gin.New()
	group := engine.Group("/_authgate/api")
	group.Use(auth.AuthMiddleware(db))
	RegisterRoutes(group, router.NewManager(db), db, stub, nil, nil)

	token, err := auth.GenerateToken(user.ID, user.Username, user.Role)
	if err != nil {
		t.Fatalf("GenerateToken() error = %v", err)
	}

	resp := performRequest(t, engine, http.MethodPost, "/_authgate/api/certificates", token, map[string]any{
		"name":   "Wildcard",
		"domain": "*.example.com",
		"source": "acme",
	})
	if resp.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d, body=%s", resp.Code, http.StatusBadRequest, resp.Body.String())
	}
	if resp.Body.String() != `{"error":{"code":"invalid_source","message":"unknown certificate source: acme"}}` {
		t.Fatalf("body = %s", resp.Body.String())
	}
}

func TestRegisterRoutes_RejectsCertificateImportWhenPEMIsMissing(t *testing.T) {
	gin.SetMode(gin.TestMode)
	auth.ConfigureJWTSecret("test-secret")

	db := newTestDB(t)
	user := seedUser(t, db, "editor", "password123", store.RoleEditor)
	stub := &stubCertService{
		importFn: func(ctx context.Context, name, domain, certPEM, keyPEM string) (*store.Certificate, error) {
			return nil, certservice.NewError(certservice.ErrCodeInvalidPEM, "certificate PEM is required", nil)
		},
	}
	engine := gin.New()
	group := engine.Group("/_authgate/api")
	group.Use(auth.AuthMiddleware(db))
	RegisterRoutes(group, router.NewManager(db), db, stub, nil, nil)

	token, err := auth.GenerateToken(user.ID, user.Username, user.Role)
	if err != nil {
		t.Fatalf("GenerateToken() error = %v", err)
	}

	resp := performRequest(t, engine, http.MethodPost, "/_authgate/api/certificates", token, map[string]any{
		"name":   "Imported",
		"domain": "imported.example.com",
		"source": "imported",
	})
	if resp.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d, body=%s", resp.Code, http.StatusBadRequest, resp.Body.String())
	}
	if resp.Body.String() != `{"error":{"code":"invalid_pem","message":"certificate PEM is required"}}` {
		t.Fatalf("body = %s", resp.Body.String())
	}
}

func TestRegisterRoutes_RejectsCertificateImportWhenDomainMismatchesCert(t *testing.T) {
	gin.SetMode(gin.TestMode)
	auth.ConfigureJWTSecret("test-secret")

	db := newTestDB(t)
	user := seedUser(t, db, "editor", "password123", store.RoleEditor)
	stub := &stubCertService{
		importFn: func(ctx context.Context, name, domain, certPEM, keyPEM string) (*store.Certificate, error) {
			return nil, certservice.NewError(certservice.ErrCodeDomainMismatch, "domain mismatch: cert is for actual.example.com, expected other.example.com", nil)
		},
	}
	engine := gin.New()
	group := engine.Group("/_authgate/api")
	group.Use(auth.AuthMiddleware(db))
	RegisterRoutes(group, router.NewManager(db), db, stub, nil, nil)

	token, err := auth.GenerateToken(user.ID, user.Username, user.Role)
	if err != nil {
		t.Fatalf("GenerateToken() error = %v", err)
	}

	resp := performRequest(t, engine, http.MethodPost, "/_authgate/api/certificates", token, map[string]any{
		"name":     "Wrong",
		"domain":   "other.example.com",
		"source":   "imported",
		"cert_pem": "-----BEGIN CERTIFICATE-----\nMIIB...\n-----END CERTIFICATE-----",
		"key_pem":  "-----BEGIN RSA PRIVATE KEY-----\nMIIB...\n-----END RSA PRIVATE KEY-----",
	})
	if resp.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d, body=%s", resp.Code, http.StatusBadRequest, resp.Body.String())
	}
	if resp.Body.String() != `{"error":{"code":"domain_mismatch","message":"domain mismatch: cert is for actual.example.com, expected other.example.com"}}` {
		t.Fatalf("body = %s", resp.Body.String())
	}
}

func TestLoginRoute_ReturnsStructuredSession(t *testing.T) {
	gin.SetMode(gin.TestMode)
	auth.ConfigureJWTSecret("test-secret")

	db := newTestDB(t)
	seedUser(t, db, "admin", "password123", store.RoleAdmin)
	engine := gin.New()
	engine.POST("/_authgate/api/auth/login", LoginRoute(db, nil))

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
	userPayload := payload["user"].(map[string]any)
	if _, ok := userPayload["permissions"]; ok {
		t.Fatalf("login response user.permissions = %#v, want omitted", userPayload["permissions"])
	}
	features := userPayload["features"].(map[string]any)
	if features["certificates"] != false {
		t.Fatalf("user.features.certificates = %v, want false", features["certificates"])
	}
}

func TestLoginRoute_TrimsUsernameBeforeAuthenticating(t *testing.T) {
	gin.SetMode(gin.TestMode)
	auth.ConfigureJWTSecret("test-secret")

	db := newTestDB(t)
	seedUser(t, db, "admin", "password123", store.RoleAdmin)
	engine := gin.New()
	engine.POST("/_authgate/api/auth/login", LoginRoute(db, nil))

	resp := performRequest(t, engine, http.MethodPost, "/_authgate/api/auth/login", "", map[string]any{
		"username": "  admin  ",
		"password": "password123",
	})
	if resp.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body=%s", resp.Code, http.StatusOK, resp.Body.String())
	}
}

func TestLoginRoute_RejectsRouteOnlyUserFromControlPlane(t *testing.T) {
	gin.SetMode(gin.TestMode)
	auth.ConfigureJWTSecret("test-secret")

	db := newTestDB(t)
	seedUser(t, db, "member", "password123", store.RoleMember)
	engine := gin.New()
	engine.POST("/_authgate/api/auth/login", LoginRoute(db, nil))

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
	RegisterRoutes(group, router.NewManager(db), db, nil, nil, nil)

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

func TestRegisterRoutes_CreateAuthRulePersistsAndReturnsRuntimePolicyFields(t *testing.T) {
	gin.SetMode(gin.TestMode)
	auth.ConfigureJWTSecret("test-secret")

	db := newTestDB(t)
	seedRoute(t, db)
	user := seedUser(t, db, "admin", "password123", store.RoleAdmin)
	engine := gin.New()
	group := engine.Group("/_authgate/api")
	group.Use(auth.AuthMiddleware(db))
	RegisterRoutes(group, router.NewManager(db), db, nil, nil, nil)

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
		"whitelist":              []string{"127.0.0.1/32", "10.0.0.0/8"},
		"rate_limit":             15,
		"burst":                  30,
		"cors_allowed_origins":   "https://app.example.com,.example.com",
		"cors_allowed_methods":   "GET,POST,OPTIONS",
		"cors_allowed_headers":   "Authorization,Content-Type",
		"cors_allow_credentials": true,
		"cors_max_age":           7200,
	})
	if resp.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d, body=%s", resp.Code, http.StatusCreated, resp.Body.String())
	}
	if bytes.Contains(resp.Body.Bytes(), []byte("shared-secret")) {
		t.Fatalf("response leaked secret: %s", resp.Body.String())
	}

	createdRule, err := db.GetAuthRuleByRouteID("route-1")
	if err != nil {
		t.Fatalf("GetAuthRuleByRouteID() error = %v", err)
	}
	if len(createdRule.Whitelist) != 2 || createdRule.Whitelist[0] != "127.0.0.1/32" || createdRule.Whitelist[1] != "10.0.0.0/8" {
		t.Fatalf("createdRule.Whitelist = %#v, want %#v", createdRule.Whitelist, []string{"127.0.0.1/32", "10.0.0.0/8"})
	}
	if createdRule.RateLimit != 15 {
		t.Fatalf("createdRule.RateLimit = %d, want %d", createdRule.RateLimit, 15)
	}
	if createdRule.Burst != 30 {
		t.Fatalf("createdRule.Burst = %d, want %d", createdRule.Burst, 30)
	}
	if createdRule.CORSAllowedOrigins != "https://app.example.com,.example.com" {
		t.Fatalf("createdRule.CORSAllowedOrigins = %q, want %q", createdRule.CORSAllowedOrigins, "https://app.example.com,.example.com")
	}
	if createdRule.CORSAllowedMethods != "GET,POST,OPTIONS" {
		t.Fatalf("createdRule.CORSAllowedMethods = %q, want %q", createdRule.CORSAllowedMethods, "GET,POST,OPTIONS")
	}
	if createdRule.CORSAllowedHeaders != "Authorization,Content-Type" {
		t.Fatalf("createdRule.CORSAllowedHeaders = %q, want %q", createdRule.CORSAllowedHeaders, "Authorization,Content-Type")
	}
	if !createdRule.CORSAllowCredentials {
		t.Fatal("createdRule.CORSAllowCredentials = false, want true")
	}
	if createdRule.CORSMaxAge != 7200 {
		t.Fatalf("createdRule.CORSMaxAge = %d, want %d", createdRule.CORSMaxAge, 7200)
	}

	var payload map[string]any
	if err := json.Unmarshal(resp.Body.Bytes(), &payload); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if payload["rate_limit"] != float64(15) {
		t.Fatalf("response rate_limit = %v, want %d", payload["rate_limit"], 15)
	}
	if payload["burst"] != float64(30) {
		t.Fatalf("response burst = %v, want %d", payload["burst"], 30)
	}
	if payload["cors_allowed_origins"] != "https://app.example.com,.example.com" {
		t.Fatalf("response cors_allowed_origins = %v, want %q", payload["cors_allowed_origins"], "https://app.example.com,.example.com")
	}
	if payload["cors_allowed_methods"] != "GET,POST,OPTIONS" {
		t.Fatalf("response cors_allowed_methods = %v, want %q", payload["cors_allowed_methods"], "GET,POST,OPTIONS")
	}
	if payload["cors_allowed_headers"] != "Authorization,Content-Type" {
		t.Fatalf("response cors_allowed_headers = %v, want %q", payload["cors_allowed_headers"], "Authorization,Content-Type")
	}
	if payload["cors_allow_credentials"] != true {
		t.Fatalf("response cors_allow_credentials = %v, want true", payload["cors_allow_credentials"])
	}
	if payload["cors_max_age"] != float64(7200) {
		t.Fatalf("response cors_max_age = %v, want %d", payload["cors_max_age"], 7200)
	}
	whitelist, ok := payload["whitelist"].([]any)
	if !ok {
		t.Fatalf("response whitelist = %v, want array", payload["whitelist"])
	}
	if len(whitelist) != 2 || whitelist[0] != "127.0.0.1/32" || whitelist[1] != "10.0.0.0/8" {
		t.Fatalf("response whitelist = %#v, want %#v", whitelist, []any{"127.0.0.1/32", "10.0.0.0/8"})
	}
}

func TestRegisterRoutes_UpdateAuthRulePreservesOmittedFields(t *testing.T) {
	gin.SetMode(gin.TestMode)
	auth.ConfigureJWTSecret("test-secret")

	db := newTestDB(t)
	seedRoute(t, db)
	user := seedUser(t, db, "admin", "password123", store.RoleAdmin)
	if err := db.CreateAuthRule(&store.AuthRule{
		ID:      "rule-1",
		RouteID: "route-1",
		Type:    "apikey",
		Config: store.AuthConfig{
			HeaderName: "X-Original-Key",
			Secret:     "shared-secret",
		},
	}); err != nil {
		t.Fatalf("CreateAuthRule() error = %v", err)
	}

	engine := gin.New()
	group := engine.Group("/_authgate/api")
	group.Use(auth.AuthMiddleware(db))
	RegisterRoutes(group, router.NewManager(db), db, nil, nil, nil)

	token, err := auth.GenerateToken(user.ID, user.Username, user.Role)
	if err != nil {
		t.Fatalf("GenerateToken() error = %v", err)
	}

	resp := performRequest(t, engine, http.MethodPut, "/_authgate/api/auth-rules/rule-1", token, map[string]any{
		"config": map[string]any{
			"header_name": "X-Updated-Key",
		},
	})
	if resp.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body=%s", resp.Code, http.StatusOK, resp.Body.String())
	}

	updatedRule, err := db.GetAuthRule("rule-1")
	if err != nil {
		t.Fatalf("GetAuthRule() error = %v", err)
	}
	if updatedRule.RouteID != "route-1" {
		t.Fatalf("updatedRule.RouteID = %q, want %q", updatedRule.RouteID, "route-1")
	}
	if updatedRule.Type != "apikey" {
		t.Fatalf("updatedRule.Type = %q, want %q", updatedRule.Type, "apikey")
	}
	if updatedRule.Config.HeaderName != "X-Updated-Key" {
		t.Fatalf("updatedRule.Config.HeaderName = %q, want %q", updatedRule.Config.HeaderName, "X-Updated-Key")
	}
	if updatedRule.Config.Secret != "shared-secret" {
		t.Fatalf("updatedRule.Config.Secret = %q, want preserved secret", updatedRule.Config.Secret)
	}

	var payload map[string]any
	if err := json.Unmarshal(resp.Body.Bytes(), &payload); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if payload["route_id"] != "route-1" {
		t.Fatalf("response route_id = %v, want %q", payload["route_id"], "route-1")
	}
	if payload["type"] != "apikey" {
		t.Fatalf("response type = %v, want %q", payload["type"], "apikey")
	}
	config, ok := payload["config"].(map[string]any)
	if !ok {
		t.Fatalf("response config = %v, want object", payload["config"])
	}
	if config["header_name"] != "X-Updated-Key" {
		t.Fatalf("response config.header_name = %v, want %q", config["header_name"], "X-Updated-Key")
	}
	if _, hasSecret := config["secret"]; hasSecret {
		t.Fatalf("response leaked secret in config: %v", config)
	}
}

func TestRegisterRoutes_GetLegacyStoredAuthRuleNormalizesConfigFields(t *testing.T) {
	gin.SetMode(gin.TestMode)
	auth.ConfigureJWTSecret("test-secret")

	db := newTestDB(t)
	seedRoute(t, db)
	user := seedUser(t, db, "admin", "password123", store.RoleAdmin)
	if err := db.CreateAuthRule(&store.AuthRule{
		ID:      "rule-1",
		RouteID: "route-1",
		Type:    " gateway ",
		Config: store.AuthConfig{
			HeaderName: " X-Route-Key ",
			Secret:     " shared-secret ",
			Username:   " service-user ",
			LoginMode:  " form ",
		},
	}); err != nil {
		t.Fatalf("CreateAuthRule() error = %v", err)
	}

	engine := gin.New()
	group := engine.Group("/_authgate/api")
	group.Use(auth.AuthMiddleware(db))
	RegisterRoutes(group, router.NewManager(db), db, nil, nil, nil)

	token, err := auth.GenerateToken(user.ID, user.Username, user.Role)
	if err != nil {
		t.Fatalf("GenerateToken() error = %v", err)
	}

	resp := performRequest(t, engine, http.MethodGet, "/_authgate/api/auth-rules/rule-1", token, nil)
	if resp.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body=%s", resp.Code, http.StatusOK, resp.Body.String())
	}

	var payload map[string]any
	if err := json.Unmarshal(resp.Body.Bytes(), &payload); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if payload["type"] != "gateway" {
		t.Fatalf("response type = %v, want %q", payload["type"], "gateway")
	}
	config, ok := payload["config"].(map[string]any)
	if !ok {
		t.Fatalf("response config = %v, want object", payload["config"])
	}
	if config["header_name"] != "X-Route-Key" {
		t.Fatalf("response config.header_name = %v, want %q", config["header_name"], "X-Route-Key")
	}
	if config["username"] != "service-user" {
		t.Fatalf("response config.username = %v, want %q", config["username"], "service-user")
	}
	if config["login_mode"] != "form" {
		t.Fatalf("response config.login_mode = %v, want %q", config["login_mode"], "form")
	}
	if _, hasSecret := config["secret"]; hasSecret {
		t.Fatalf("response leaked secret in config: %v", config)
	}
}

func TestRegisterRoutes_MeNormalizesLegacyStoredRoleForControlPlaneAccess(t *testing.T) {
	gin.SetMode(gin.TestMode)
	auth.ConfigureJWTSecret("test-secret")

	db := newTestDB(t)
	user := seedUser(t, db, "viewer", "password123", " VIEWER ")
	engine := gin.New()
	group := engine.Group("/_authgate/api")
	group.Use(auth.AuthMiddleware(db))
	RegisterRoutes(group, router.NewManager(db), db, nil, nil, nil)

	token, err := auth.GenerateToken(user.ID, user.Username, store.RoleViewer)
	if err != nil {
		t.Fatalf("GenerateToken() error = %v", err)
	}

	resp := performRequest(t, engine, http.MethodGet, "/_authgate/api/auth/me", token, nil)
	if resp.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body=%s", resp.Code, http.StatusOK, resp.Body.String())
	}

	var payload map[string]any
	if err := json.Unmarshal(resp.Body.Bytes(), &payload); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if payload["role"] != store.RoleViewer {
		t.Fatalf("response role = %v, want %q", payload["role"], store.RoleViewer)
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
	RegisterRoutes(group, router.NewManager(db), db, nil, nil, nil)

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

func TestRegisterRoutes_ListUsersNormalizesLegacyStoredRole(t *testing.T) {
	gin.SetMode(gin.TestMode)
	auth.ConfigureJWTSecret("test-secret")

	db := newTestDB(t)
	adminUser := seedUser(t, db, "admin", "password123", store.RoleAdmin)
	seedUser(t, db, "viewer", "password123", " VIEWER ")

	engine := gin.New()
	group := engine.Group("/_authgate/api")
	group.Use(auth.AuthMiddleware(db))
	RegisterRoutes(group, router.NewManager(db), db, nil, nil, nil)

	token, err := auth.GenerateToken(adminUser.ID, adminUser.Username, adminUser.Role)
	if err != nil {
		t.Fatalf("GenerateToken() error = %v", err)
	}

	resp := performRequest(t, engine, http.MethodGet, "/_authgate/api/users", token, nil)
	if resp.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body=%s", resp.Code, http.StatusOK, resp.Body.String())
	}

	var payload []map[string]any
	if err := json.Unmarshal(resp.Body.Bytes(), &payload); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	var found bool
	for _, entry := range payload {
		if entry["username"] == "viewer" {
			found = true
			if entry["role"] != store.RoleViewer {
				t.Fatalf("viewer role = %v, want %q", entry["role"], store.RoleViewer)
			}
		}
	}
	if !found {
		t.Fatalf("viewer user not found in payload: %v", payload)
	}
}

func TestRegisterRoutes_UpdateUserPreservesEnabledWhenOmitted(t *testing.T) {
	gin.SetMode(gin.TestMode)
	auth.ConfigureJWTSecret("test-secret")

	db := newTestDB(t)
	adminUser := seedUser(t, db, "admin", "password123", store.RoleAdmin)
	targetUser := seedUser(t, db, "viewer", "password123", store.RoleViewer)

	engine := gin.New()
	group := engine.Group("/_authgate/api")
	group.Use(auth.AuthMiddleware(db))
	RegisterRoutes(group, router.NewManager(db), db, nil, nil, nil)

	token, err := auth.GenerateToken(adminUser.ID, adminUser.Username, adminUser.Role)
	if err != nil {
		t.Fatalf("GenerateToken() error = %v", err)
	}

	resp := performRequest(t, engine, http.MethodPut, "/_authgate/api/users/"+targetUser.ID, token, map[string]any{
		"username": "viewer-renamed",
		"role":     store.RoleViewer,
	})
	if resp.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body=%s", resp.Code, http.StatusOK, resp.Body.String())
	}

	updatedUser, err := db.GetUserByID(targetUser.ID)
	if err != nil {
		t.Fatalf("GetUserByID() error = %v", err)
	}
	if !updatedUser.Enabled {
		t.Fatalf("updatedUser.Enabled = %v, want true", updatedUser.Enabled)
	}

	var payload map[string]any
	if err := json.Unmarshal(resp.Body.Bytes(), &payload); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if payload["enabled"] != true {
		t.Fatalf("response enabled = %v, want true", payload["enabled"])
	}
}

func TestRegisterRoutes_CreateUserRejectsWhitespaceOnlyUsername(t *testing.T) {
	gin.SetMode(gin.TestMode)
	auth.ConfigureJWTSecret("test-secret")

	db := newTestDB(t)
	adminUser := seedUser(t, db, "admin", "password123", store.RoleAdmin)

	engine := gin.New()
	group := engine.Group("/_authgate/api")
	group.Use(auth.AuthMiddleware(db))
	RegisterRoutes(group, router.NewManager(db), db, nil, nil, nil)

	token, err := auth.GenerateToken(adminUser.ID, adminUser.Username, adminUser.Role)
	if err != nil {
		t.Fatalf("GenerateToken() error = %v", err)
	}

	resp := performRequest(t, engine, http.MethodPost, "/_authgate/api/users", token, map[string]any{
		"username": "   ",
		"password": "password123",
		"role":     store.RoleViewer,
		"enabled":  true,
	})
	if resp.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d, body=%s", resp.Code, http.StatusBadRequest, resp.Body.String())
	}

	var payload map[string]any
	if err := json.Unmarshal(resp.Body.Bytes(), &payload); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	errorPayload, ok := payload["error"].(map[string]any)
	if !ok {
		t.Fatalf("response error = %v, want object", payload["error"])
	}
	if errorPayload["code"] != "invalid_username" {
		t.Fatalf("response error.code = %v, want %q", errorPayload["code"], "invalid_username")
	}
}

func TestRegisterRoutes_CreateUserDefaultsOmittedRoleToMember(t *testing.T) {
	gin.SetMode(gin.TestMode)
	auth.ConfigureJWTSecret("test-secret")

	db := newTestDB(t)
	adminUser := seedUser(t, db, "admin", "password123", store.RoleAdmin)

	engine := gin.New()
	group := engine.Group("/_authgate/api")
	group.Use(auth.AuthMiddleware(db))
	RegisterRoutes(group, router.NewManager(db), db, nil, nil, nil)

	token, err := auth.GenerateToken(adminUser.ID, adminUser.Username, adminUser.Role)
	if err != nil {
		t.Fatalf("GenerateToken() error = %v", err)
	}

	resp := performRequest(t, engine, http.MethodPost, "/_authgate/api/users", token, map[string]any{
		"username": "member-default",
		"password": "password123",
		"enabled":  true,
	})
	if resp.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d, body=%s", resp.Code, http.StatusCreated, resp.Body.String())
	}

	var payload map[string]any
	if err := json.Unmarshal(resp.Body.Bytes(), &payload); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if payload["role"] != store.RoleMember {
		t.Fatalf("response role = %v, want %q", payload["role"], store.RoleMember)
	}
}

func TestRegisterRoutes_UpdateUserPreservesRoleWhenOmitted(t *testing.T) {
	gin.SetMode(gin.TestMode)
	auth.ConfigureJWTSecret("test-secret")

	db := newTestDB(t)
	adminUser := seedUser(t, db, "admin", "password123", store.RoleAdmin)
	targetUser := seedUser(t, db, "editor-user", "password123", store.RoleEditor)

	engine := gin.New()
	group := engine.Group("/_authgate/api")
	group.Use(auth.AuthMiddleware(db))
	RegisterRoutes(group, router.NewManager(db), db, nil, nil, nil)

	token, err := auth.GenerateToken(adminUser.ID, adminUser.Username, adminUser.Role)
	if err != nil {
		t.Fatalf("GenerateToken() error = %v", err)
	}

	resp := performRequest(t, engine, http.MethodPut, "/_authgate/api/users/"+targetUser.ID, token, map[string]any{
		"username": "editor-user-renamed",
		"enabled":  true,
	})
	if resp.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body=%s", resp.Code, http.StatusOK, resp.Body.String())
	}

	updatedUser, err := db.GetUserByID(targetUser.ID)
	if err != nil {
		t.Fatalf("GetUserByID() error = %v", err)
	}
	if updatedUser.Role != store.RoleEditor {
		t.Fatalf("updatedUser.Role = %q, want %q", updatedUser.Role, store.RoleEditor)
	}

	var payload map[string]any
	if err := json.Unmarshal(resp.Body.Bytes(), &payload); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if payload["role"] != store.RoleEditor {
		t.Fatalf("response role = %v, want %q", payload["role"], store.RoleEditor)
	}
}

func TestRegisterRoutes_UpdateUserPreservesUsernameWhenOmitted(t *testing.T) {
	gin.SetMode(gin.TestMode)
	auth.ConfigureJWTSecret("test-secret")

	db := newTestDB(t)
	adminUser := seedUser(t, db, "admin", "password123", store.RoleAdmin)
	targetUser := seedUser(t, db, "viewer", "password123", store.RoleViewer)

	engine := gin.New()
	group := engine.Group("/_authgate/api")
	group.Use(auth.AuthMiddleware(db))
	RegisterRoutes(group, router.NewManager(db), db, nil, nil, nil)

	token, err := auth.GenerateToken(adminUser.ID, adminUser.Username, adminUser.Role)
	if err != nil {
		t.Fatalf("GenerateToken() error = %v", err)
	}

	resp := performRequest(t, engine, http.MethodPut, "/_authgate/api/users/"+targetUser.ID, token, map[string]any{
		"enabled": false,
	})
	if resp.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body=%s", resp.Code, http.StatusOK, resp.Body.String())
	}

	updatedUser, err := db.GetUserByID(targetUser.ID)
	if err != nil {
		t.Fatalf("GetUserByID() error = %v", err)
	}
	if updatedUser.Username != "viewer" {
		t.Fatalf("updatedUser.Username = %q, want %q", updatedUser.Username, "viewer")
	}
	if updatedUser.Enabled != false {
		t.Fatalf("updatedUser.Enabled = %v, want false", updatedUser.Enabled)
	}

	var payload map[string]any
	if err := json.Unmarshal(resp.Body.Bytes(), &payload); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if payload["username"] != "viewer" {
		t.Fatalf("response username = %v, want %q", payload["username"], "viewer")
	}
	if payload["enabled"] != false {
		t.Fatalf("response enabled = %v, want false", payload["enabled"])
	}
}

func TestRegisterRoutes_UpdateUserPreservesRouteAssignmentsWhenOmitted(t *testing.T) {
	gin.SetMode(gin.TestMode)
	auth.ConfigureJWTSecret("test-secret")

	db := newTestDB(t)
	seedRoute(t, db)
	adminUser := seedUser(t, db, "admin", "password123", store.RoleAdmin)
	targetUser := seedUser(t, db, "member-user", "password123", store.RoleMember)
	targetUser.RouteIDs = []string{"route-1"}
	if err := db.UpdateUser(targetUser); err != nil {
		t.Fatalf("UpdateUser() error = %v", err)
	}

	engine := gin.New()
	group := engine.Group("/_authgate/api")
	group.Use(auth.AuthMiddleware(db))
	RegisterRoutes(group, router.NewManager(db), db, nil, nil, nil)

	token, err := auth.GenerateToken(adminUser.ID, adminUser.Username, adminUser.Role)
	if err != nil {
		t.Fatalf("GenerateToken() error = %v", err)
	}

	resp := performRequest(t, engine, http.MethodPut, "/_authgate/api/users/"+targetUser.ID, token, map[string]any{
		"username": "member-user-renamed",
		"role":     store.RoleMember,
		"enabled":  true,
	})
	if resp.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body=%s", resp.Code, http.StatusOK, resp.Body.String())
	}

	updatedUser, err := db.GetUserByID(targetUser.ID)
	if err != nil {
		t.Fatalf("GetUserByID() error = %v", err)
	}
	if len(updatedUser.RouteIDs) != 1 || updatedUser.RouteIDs[0] != "route-1" {
		t.Fatalf("updatedUser.RouteIDs = %v, want [route-1]", updatedUser.RouteIDs)
	}

	var payload map[string]any
	if err := json.Unmarshal(resp.Body.Bytes(), &payload); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	routeIDs, ok := payload["route_ids"].([]any)
	if !ok || len(routeIDs) != 1 || routeIDs[0] != "route-1" {
		t.Fatalf("response route_ids = %v, want [route-1]", payload["route_ids"])
	}
}

func TestRegisterRoutes_UpdateUserRejectsDuplicateUsername(t *testing.T) {
	gin.SetMode(gin.TestMode)
	auth.ConfigureJWTSecret("test-secret")

	db := newTestDB(t)
	adminUser := seedUser(t, db, "admin", "password123", store.RoleAdmin)
	seedUser(t, db, "alice", "password123", store.RoleViewer)
	targetUser := seedUser(t, db, "bob", "password123", store.RoleViewer)

	engine := gin.New()
	group := engine.Group("/_authgate/api")
	group.Use(auth.AuthMiddleware(db))
	RegisterRoutes(group, router.NewManager(db), db, nil, nil, nil)

	token, err := auth.GenerateToken(adminUser.ID, adminUser.Username, adminUser.Role)
	if err != nil {
		t.Fatalf("GenerateToken() error = %v", err)
	}

	resp := performRequest(t, engine, http.MethodPut, "/_authgate/api/users/"+targetUser.ID, token, map[string]any{
		"username": "alice",
		"role":     store.RoleViewer,
		"enabled":  true,
	})
	if resp.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d, body=%s", resp.Code, http.StatusBadRequest, resp.Body.String())
	}
	if resp.Body.String() != "{\"error\":{\"code\":\"duplicate_user\",\"message\":\"username already exists\"}}" {
		t.Fatalf("body = %s", resp.Body.String())
	}
}
