package main

import (
	"bytes"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pallyoung/auth-gate/packages/server/internal/auth"
	"github.com/pallyoung/auth-gate/packages/server/internal/config"
	"github.com/pallyoung/auth-gate/packages/server/internal/router"
	"github.com/pallyoung/auth-gate/packages/server/internal/store"
)

func newTestSQLite(t *testing.T) (*store.SQLite, func()) {
	t.Helper()

	dbPath := filepath.Join(t.TempDir(), "auth-gate.db")
	db, err := store.NewSQLite(dbPath)
	if err != nil {
		t.Fatalf("NewSQLite() error = %v", err)
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

	engine := buildEngine(router.NewManager(db), webRoot, db)

	rootReq := httptest.NewRequest(http.MethodGet, "/", nil)
	rootResp := httptest.NewRecorder()
	engine.ServeHTTP(rootResp, rootReq)

	if rootResp.Code != http.StatusOK {
		t.Fatalf("GET / status = %d, want %d", rootResp.Code, http.StatusOK)
	}
	if !strings.Contains(rootResp.Body.String(), "auth gate") {
		t.Fatalf("GET / body = %q, want index.html content", rootResp.Body.String())
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

func TestEnsureBootstrapAdmin_LogsGeneratedPassword(t *testing.T) {
	db, cleanup := newTestSQLite(t)
	defer cleanup()

	if err := os.Unsetenv("BOOTSTRAP_ADMIN_PASSWORD"); err != nil {
		t.Fatalf("Unsetenv() error = %v", err)
	}

	var buf bytes.Buffer
	previousOutput := log.Writer()
	log.SetOutput(&buf)
	t.Cleanup(func() {
		log.SetOutput(previousOutput)
	})

	if err := ensureBootstrapAdmin(db, config.AuthConfig{}); err != nil {
		t.Fatalf("ensureBootstrapAdmin() error = %v", err)
	}

	logs := buf.String()
	if !strings.Contains(logs, "Bootstrap admin created: username=admin password=") {
		t.Fatalf("ensureBootstrapAdmin() logs = %q", logs)
	}

	user, err := db.GetUserByUsername("admin")
	if err != nil {
		t.Fatalf("GetUserByUsername() error = %v", err)
	}
	if user.Username != "admin" {
		t.Fatalf("user.Username = %q, want %q", user.Username, "admin")
	}
}
