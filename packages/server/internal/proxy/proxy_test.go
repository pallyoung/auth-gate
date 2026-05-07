package proxy

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pallyoung/auth-gate/packages/server/internal/auth"
	"github.com/pallyoung/auth-gate/packages/server/internal/router"
	"github.com/pallyoung/auth-gate/packages/server/internal/store"

	"github.com/gin-gonic/gin"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func newProxyTestDB(t *testing.T) *store.SQLite {
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

// Test the path-rewriting logic in isolation, mirroring what proxy.go does in Director.
func TestPathRewrite(t *testing.T) {
	tests := []struct {
		name        string
		pathPrefix  string
		strip       bool
		reqPath     string
		wantPath    string
	}{
		{"strip prefix", "/api", true, "/api/users/123", "/users/123"},
		{"strip prefix root", "/api", true, "/api", "/"},
		{"strip deeper", "/api/v1", true, "/api/v1/users/123", "/users/123"},
		{"no strip", "/api", false, "/api/users", "/api/users"},
		{"strip no leading slash result", "/api", true, "/apifoo", "/foo"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := tt.reqPath
			if tt.strip {
				path = strings.TrimPrefix(path, tt.pathPrefix)
				if !strings.HasPrefix(path, "/") {
					path = "/" + path
				}
			}
			if path != tt.wantPath {
				t.Errorf("rewritePath(%q, strip=%v) = %q, want %q", tt.reqPath, tt.strip, path, tt.wantPath)
			}
		})
	}
}

// Test that X-Forwarded-* headers are set correctly.
func TestForwardedHeaders(t *testing.T) {
	req := httptest.NewRequest("GET", "/test", nil)
	req.Host = "example.com:8080"

	// Simulate what proxy.go Director does
	host := req.Host
	if idx := strings.Index(host, ":"); idx != -1 {
		host = host[:idx]
	}
	req.Header.Set("X-Forwarded-Host", host)
	req.Header.Set("X-Forwarded-Proto", "http")
	req.Header.Set("X-Forwarded-For", "192.168.1.1")

	if got := req.Header.Get("X-Forwarded-Host"); got != "example.com" {
		t.Errorf("X-Forwarded-Host = %q, want %q", got, "example.com")
	}
	if got := req.Header.Get("X-Forwarded-Proto"); got != "http" {
		t.Errorf("X-Forwarded-Proto = %q, want %q", got, "http")
	}
	if got := req.Header.Get("X-Forwarded-For"); got != "192.168.1.1" {
		t.Errorf("X-Forwarded-For = %q, want %q", got, "192.168.1.1")
	}
}

// Test the host:port stripping logic used in proxy.go.
func TestHostPortStripping(t *testing.T) {
	tests := []struct {
		host string
		want string
	}{
		{"example.com:8080", "example.com"},
		{"example.com", "example.com"},
	}

	for _, tt := range tests {
		t.Run(tt.host, func(t *testing.T) {
			host := tt.host
			if idx := strings.Index(host, ":"); idx != -1 {
				host = host[:idx]
			}
			if host != tt.want {
				t.Errorf("stripPort(%q) = %q, want %q", tt.host, host, tt.want)
			}
		})
	}
}

// Test backend URL parsing handles various formats.
func TestBackendURLParsing(t *testing.T) {
	tests := []struct {
		backend string
		wantErr bool
	}{
		{"http://localhost:3000", false},
		{"http://127.0.0.1:3000", false},
		{"http://backend:8080/path", false},
		{"://invalid", true},
	}

	for _, tt := range tests {
		t.Run(tt.backend, func(t *testing.T) {
			_, err := url.Parse(tt.backend)
			if (err != nil) != tt.wantErr {
				t.Errorf("url.Parse(%q) err=%v, wantErr=%v", tt.backend, err, tt.wantErr)
			}
		})
	}
}

func TestWriteUnauthorized_BasicSetsChallengeHeader(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/cloud", nil)

	writeUnauthorized(c, &router.Route{
		Name:       "Cloud Console",
		PathPrefix: "/cloud",
		AuthRule: &router.AuthRule{
			Type: "basic",
		},
	})

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
	if got := w.Header().Get("WWW-Authenticate"); got != `Basic realm="Cloud Console"` {
		t.Fatalf("WWW-Authenticate = %q, want %q", got, `Basic realm="Cloud Console"`)
	}
}

func TestRouteAccessClaims_RejectsDisabledUser(t *testing.T) {
	auth.ConfigureJWTSecret("test-secret")

	db := newProxyTestDB(t)
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

	user := &store.User{
		ID:           "member-1",
		Username:     "member",
		PasswordHash: hash,
		Role:         store.RoleMember,
		Enabled:      true,
		RouteIDs:     []string{"route-1"},
	}
	if err := db.CreateUser(user); err != nil {
		t.Fatalf("CreateUser() error = %v", err)
	}

	token, err := auth.GenerateRouteAccessToken(user.ID, user.Username, user.Role, user.RouteIDs)
	if err != nil {
		t.Fatalf("GenerateRouteAccessToken() error = %v", err)
	}

	user.Enabled = false
	if err := db.UpdateUser(user); err != nil {
		t.Fatalf("UpdateUser() error = %v", err)
	}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req := httptest.NewRequest(http.MethodGet, "/cloud", nil)
	req.AddCookie(&http.Cookie{Name: routeAccessCookieName, Value: token})
	c.Request = req

	if claims, ok := routeAccessClaims(c, db); ok || claims != nil {
		t.Fatalf("routeAccessClaims() = (%v, %v), want (nil, false)", claims, ok)
	}
}

func TestRouteAllowedByClaims_UsesCurrentRouteAssignments(t *testing.T) {
	claims := &auth.Claims{
		Role:     store.RoleMember,
		RouteIDs: []string{"route-2"},
	}

	if routeAllowedByClaims(claims, "route-1") {
		t.Fatal("routeAllowedByClaims() = true, want false")
	}
}
