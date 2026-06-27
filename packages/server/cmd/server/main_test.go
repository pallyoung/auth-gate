package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pallyoung/auth-gate/packages/server/internal/auth"
	adminhttp "github.com/pallyoung/auth-gate/packages/server/internal/http/admin"
	"github.com/pallyoung/auth-gate/packages/server/internal/config"
	"github.com/pallyoung/auth-gate/packages/server/internal/router"
	certservice "github.com/pallyoung/auth-gate/packages/server/internal/service/certificate"
	"github.com/pallyoung/auth-gate/packages/server/internal/store"
)

func newTestSQLite(t *testing.T) (store.Store, func()) {
	t.Helper()

	dbPath := t.TempDir()
	db, err := store.NewJSONStore(dbPath)
	if err != nil {
		t.Fatalf("NewJSONStore() error = %v", err)
	}

	return db, func() {
		_ = db.Close()
	}
}

func TestBuildEngine_ServesIndexWithoutSwallowingProxyPaths(t *testing.T) {
	db, cleanup := newTestSQLite(t)
	defer cleanup()

	webRoot := t.TempDir()
	if err := os.MkdirAll(filepath.Join(webRoot, "assets"), 0755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(webRoot, "index.html"), []byte("<html>auth gate</html>"), 0644); err != nil {
		t.Fatalf("WriteFile(index.html) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(webRoot, "favicon.ico"), []byte("ico"), 0644); err != nil {
		t.Fatalf("WriteFile(favicon.ico) error = %v", err)
	}

	engine := buildEngine(router.NewManager(db), webRoot, db, nil, nil, nil, config.DefaultConfig(), nil)

	controlPlaneReq := httptest.NewRequest(http.MethodGet, controlPlaneBasePath, nil)
	controlPlaneResp := httptest.NewRecorder()
	engine.ServeHTTP(controlPlaneResp, controlPlaneReq)

	if controlPlaneResp.Code != http.StatusOK {
		t.Fatalf("GET %s status = %d, want %d", controlPlaneBasePath, controlPlaneResp.Code, http.StatusOK)
	}
	if !strings.Contains(controlPlaneResp.Body.String(), "auth gate") {
		t.Fatalf("GET %s body = %q, want index.html content", controlPlaneBasePath, controlPlaneResp.Body.String())
	}

	rootReq := httptest.NewRequest(http.MethodGet, "/", nil)
	rootResp := httptest.NewRecorder()
	engine.ServeHTTP(rootResp, rootReq)

	if rootResp.Code != http.StatusNotFound {
		t.Fatalf("GET / status = %d, want %d", rootResp.Code, http.StatusNotFound)
	}
	if !strings.Contains(rootResp.Body.String(), "no route found") {
		t.Fatalf("GET / body = %q, want proxy 404", rootResp.Body.String())
	}

	proxyReq := httptest.NewRequest(http.MethodGet, "/unmatched", nil)
	proxyResp := httptest.NewRecorder()
	engine.ServeHTTP(proxyResp, proxyReq)

	if proxyResp.Code != http.StatusNotFound {
		t.Fatalf("GET /unmatched status = %d, want %d", proxyResp.Code, http.StatusNotFound)
	}
	if !strings.Contains(proxyResp.Body.String(), "no route found") {
		t.Fatalf("GET /unmatched body = %q, want proxy 404", proxyResp.Body.String())
	}
}

func TestBuildEngine_RegistersConfigReloadAsPostOnly(t *testing.T) {
	db, cleanup := newTestSQLite(t)
	defer cleanup()

	auth.ConfigureJWTSecret("test-secret")
	passwordHash, err := store.HashPassword("password123")
	if err != nil {
		t.Fatalf("HashPassword() error = %v", err)
	}
	if err := db.CreateUser(&store.User{
		ID:           "admin-1",
		Username:     "admin",
		PasswordHash: passwordHash,
		Role:         store.RoleAdmin,
		Enabled:      true,
	}); err != nil {
		t.Fatalf("CreateUser() error = %v", err)
	}

	webRoot := t.TempDir()
	if err := os.WriteFile(filepath.Join(webRoot, "index.html"), []byte("<html>auth gate</html>"), 0644); err != nil {
		t.Fatalf("WriteFile(index.html) error = %v", err)
	}

	engine := buildEngine(router.NewManager(db), webRoot, db, nil, nil, nil, config.DefaultConfig(), nil)

	token, err := auth.GenerateToken("admin-1", "admin", store.RoleAdmin)
	if err != nil {
		t.Fatalf("GenerateToken() error = %v", err)
	}

	getReq := httptest.NewRequest(http.MethodGet, controlPlaneAPIBasePath+"/config/reload", nil)
	getReq.Header.Set("Authorization", "Bearer "+token)
	getResp := httptest.NewRecorder()
	engine.ServeHTTP(getResp, getReq)

	if getResp.Code != http.StatusNotFound {
		t.Fatalf("GET %s/config/reload status = %d, want %d", controlPlaneAPIBasePath, getResp.Code, http.StatusNotFound)
	}

	postReq := httptest.NewRequest(http.MethodPost, controlPlaneAPIBasePath+"/config/reload", nil)
	postReq.Header.Set("Authorization", "Bearer "+token)
	postResp := httptest.NewRecorder()
	engine.ServeHTTP(postResp, postReq)

	if postResp.Code != http.StatusOK {
		t.Fatalf("POST %s/config/reload status = %d, want %d, body=%s", controlPlaneAPIBasePath, postResp.Code, http.StatusOK, postResp.Body.String())
	}
}

func TestBuildEngine_LoginResponseReportsCertificateFeatureAvailability(t *testing.T) {
	testCases := []struct {
		name        string
		certSvc     adminhttp.CertService
		wantEnabled bool
	}{
		{
			name:        "disabled when certificate service is unavailable",
			certSvc:     nil,
			wantEnabled: false,
		},
		{
			name:        "enabled when certificate service is available",
			certSvc:     &certservice.Service{},
			wantEnabled: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			db, cleanup := newTestSQLite(t)
			defer cleanup()

			auth.ConfigureJWTSecret("test-secret")
			passwordHash, err := store.HashPassword("password123")
			if err != nil {
				t.Fatalf("HashPassword() error = %v", err)
			}
			if err := db.CreateUser(&store.User{
				ID:           "admin-1",
				Username:     "admin",
				PasswordHash: passwordHash,
				Role:         store.RoleAdmin,
				Enabled:      true,
			}); err != nil {
				t.Fatalf("CreateUser() error = %v", err)
			}

			webRoot := t.TempDir()
			if err := os.WriteFile(filepath.Join(webRoot, "index.html"), []byte("<html>auth gate</html>"), 0644); err != nil {
				t.Fatalf("WriteFile(index.html) error = %v", err)
			}

			engine := buildEngine(router.NewManager(db), webRoot, db, tc.certSvc, nil, nil, config.DefaultConfig(), nil)

			req := httptest.NewRequest(http.MethodPost, controlPlaneAPIBasePath+"/auth/login", strings.NewReader(`{"username":"admin","password":"password123"}`))
			req.Header.Set("Content-Type", "application/json")
			resp := httptest.NewRecorder()
			engine.ServeHTTP(resp, req)

			if resp.Code != http.StatusOK {
				t.Fatalf("status = %d, want %d, body=%s", resp.Code, http.StatusOK, resp.Body.String())
			}

			var payload map[string]any
			if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
				t.Fatalf("json.Decode() error = %v", err)
			}

			userPayload := payload["user"].(map[string]any)
			features := userPayload["features"].(map[string]any)
			if features["certificates"] != tc.wantEnabled {
				t.Fatalf("user.features.certificates = %v, want %v", features["certificates"], tc.wantEnabled)
			}
		})
	}
}

func TestBuildTLSHostGroups_FormatsIPv6ListenHost(t *testing.T) {
	groups := buildTLSHostGroups([]router.Route{
		{
			Name:       "ipv6-route",
			Host:       "2001:db8::1",
			Enabled:    true,
			TLSEnabled: true,
			TLSCert:    "/tmp/site.pem",
			TLSKey:     "/tmp/site.key",
		},
	}, 443)

	if len(groups) != 1 {
		t.Fatalf("len(groups) = %d, want 1", len(groups))
	}
	if groups[0].Host != ":443" {
		t.Fatalf("groups[0].Host = %q, want %q", groups[0].Host, ":443")
	}
}

func TestConfigureJWTSecret_UsesConfigValue(t *testing.T) {
	previous := os.Getenv("JWT_SECRET")
	if err := os.Unsetenv("JWT_SECRET"); err != nil {
		t.Fatalf("Unsetenv() error = %v", err)
	}
	t.Cleanup(func() {
		if previous == "" {
			_ = os.Unsetenv("JWT_SECRET")
			return
		}
		_ = os.Setenv("JWT_SECRET", previous)
	})

	auth.ConfigureJWTSecret("seed-secret")
	configureJWTSecret(config.AuthConfig{JWTSecret: "config-secret"})

	token, err := auth.GenerateToken("id", "user", "admin")
	if err != nil {
		t.Fatalf("GenerateToken() error = %v", err)
	}
	if _, err := auth.ValidateTokenWithSecret(token, []byte("config-secret")); err != nil {
		t.Fatalf("ValidateTokenWithSecret() error = %v", err)
	}
}
