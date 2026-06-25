package router

import (
	"testing"

	"github.com/pallyoung/auth-gate/packages/server/internal/store"
)

// newTestManager creates a Manager backed by a JSON store.
func newTestManager(t *testing.T) (*Manager, func()) {
	db, err := store.NewJSONStore(t.TempDir())
	if err != nil {
		t.Fatalf("NewJSONStore: %v", err)
	}
	return &Manager{db: db}, func() {
		db.Close()
	}
}

func TestMatch_Basic(t *testing.T) {
	m, cleanup := newTestManager(t)
	defer cleanup()

	for _, r := range []store.Route{
		{ID: "1", Name: "api", Host: "example.com", PathPrefix: "/api", Backend: "http://localhost:3000", Enabled: true, Priority: 0},
	} {
		m.db.CreateRoute(&r)
	}
	m.loadRoutes()

	tests := []struct {
		host string
		path string
		want string
	}{
		{"example.com", "/api/users", "1"},
		{"example.com", "/api", "1"},
		{"example.com", "/apifoo", ""},
		{"example.com", "/other", ""},
		{"other.com", "/api/users", ""},
	}

	for _, tt := range tests {
		t.Run(tt.host+tt.path, func(t *testing.T) {
			r := m.Match(tt.host, tt.path, nil)
			if tt.want == "" {
				if r != nil {
					t.Errorf("Match(%q, %q) = %v, want nil", tt.host, tt.path, r.ID)
				}
			} else {
				if r == nil || r.ID != tt.want {
					t.Errorf("Match(%q, %q) = %v, want %q", tt.host, tt.path, r, tt.want)
				}
			}
		})
	}
}

func TestMatch_EmptyHostMatchesAll(t *testing.T) {
	m, cleanup := newTestManager(t)
	defer cleanup()

	for _, r := range []store.Route{
		{ID: "api", Host: "api.example.com", PathPrefix: "/v1", Backend: "http://localhost:8080", Enabled: true, Priority: 0},
		{ID: "catchall", Host: "", PathPrefix: "/", Backend: "http://default", Enabled: true, Priority: 0},
	} {
		m.db.CreateRoute(&r)
	}
	m.loadRoutes()

	tests := []struct {
		host string
		path string
		want string
	}{
		{"any-host.com", "/foo", "catchall"},
		{"api.example.com", "/v1/users", "api"},
		{"api.example.com", "/other", "catchall"},
	}

	for _, tt := range tests {
		t.Run(tt.host+tt.path, func(t *testing.T) {
			r := m.Match(tt.host, tt.path, nil)
			if r == nil || r.ID != tt.want {
				t.Errorf("Match(%q, %q) = %v, want %q", tt.host, tt.path, r, tt.want)
			}
		})
	}
}

func TestMatch_HostIsCaseInsensitive(t *testing.T) {
	m, cleanup := newTestManager(t)
	defer cleanup()

	for _, r := range []store.Route{
		{ID: "1", Host: "api.example.com", PathPrefix: "/api", Backend: "http://localhost:3000", Enabled: true, Priority: 0},
	} {
		m.db.CreateRoute(&r)
	}
	m.loadRoutes()

	r := m.Match("API.EXAMPLE.COM", "/api/users", nil)
	if r == nil || r.ID != "1" {
		t.Fatalf("Match(%q, %q) = %v, want route 1", "API.EXAMPLE.COM", "/api/users", r)
	}
}

func TestMatch_DisabledRoutes(t *testing.T) {
	m, cleanup := newTestManager(t)
	defer cleanup()

	for _, r := range []store.Route{
		{ID: "disabled", Host: "example.com", PathPrefix: "/api", Backend: "http://localhost:3000", Enabled: false, Priority: 10},
		{ID: "fallback", Host: "example.com", PathPrefix: "/", Backend: "http://localhost:4000", Enabled: true, Priority: 0},
	} {
		m.db.CreateRoute(&r)
	}
	m.loadRoutes()

	r := m.Match("example.com", "/api/users", nil)
	if r == nil || r.ID != "fallback" {
		t.Errorf("Match returned %v, want fallback (disabled route skipped)", r)
	}
}

func TestMatch_PriorityOrdering(t *testing.T) {
	m, cleanup := newTestManager(t)
	defer cleanup()

	// Insert in opposite order to prove loadRoutes sorts correctly
	for _, r := range []store.Route{
		{ID: "low", Host: "example.com", PathPrefix: "/api", Backend: "http://low", Enabled: true, Priority: 1},
		{ID: "high", Host: "example.com", PathPrefix: "/api", Backend: "http://high", Enabled: true, Priority: 10},
	} {
		m.db.CreateRoute(&r)
	}
	m.loadRoutes()

	r := m.Match("example.com", "/api/users", nil)
	if r == nil || r.ID != "high" {
		t.Errorf("Match returned %v, want high priority route", r)
	}
}

func TestMatch_LongestPathPrefixWins(t *testing.T) {
	m, cleanup := newTestManager(t)
	defer cleanup()

	for _, r := range []store.Route{
		{ID: "root", Host: "example.com", PathPrefix: "/", Backend: "http://root", Enabled: true, Priority: 0},
		{ID: "deep", Host: "example.com", PathPrefix: "/api/v1/users", Backend: "http://deep", Enabled: true, Priority: 0},
		{ID: "medium", Host: "example.com", PathPrefix: "/api/v1", Backend: "http://medium", Enabled: true, Priority: 0},
	} {
		m.db.CreateRoute(&r)
	}
	m.loadRoutes()

	tests := []struct {
		path string
		want string
	}{
		{"/api/v1/users/123", "deep"},
		{"/api/v1/users", "deep"},
		{"/api/v1/foo", "medium"},
		{"/api", "root"},
		{"/", "root"},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			r := m.Match("example.com", tt.path, nil)
			if r == nil || r.ID != tt.want {
				t.Errorf("Match(%q) = %v, want %q", tt.path, r, tt.want)
			}
		})
	}
}

func TestMatch_PriorityTieBrokenByPathLength(t *testing.T) {
	m, cleanup := newTestManager(t)
	defer cleanup()

	for _, r := range []store.Route{
		{ID: "short", Host: "example.com", PathPrefix: "/api", Backend: "http://short", Enabled: true, Priority: 5},
		{ID: "longer", Host: "example.com", PathPrefix: "/api/internal", Backend: "http://longer", Enabled: true, Priority: 5},
	} {
		m.db.CreateRoute(&r)
	}
	m.loadRoutes()

	r := m.Match("example.com", "/api/internal/users", nil)
	if r == nil || r.ID != "longer" {
		t.Errorf("Match returned %v, want longer (same priority, longer path)", r)
	}
}

func TestMatch_SkipsRouteWithUnsupportedStoredPathMatchMode(t *testing.T) {
	m, cleanup := newTestManager(t)
	defer cleanup()

	for _, r := range []store.Route{
		{ID: "invalid", Host: "example.com", PathPrefix: "/admin", Backend: "http://invalid", Enabled: true, PathMatchMode: "glob"},
		{ID: "fallback", Host: "example.com", PathPrefix: "/", Backend: "http://fallback", Enabled: true},
	} {
		if err := m.db.CreateRoute(&r); err != nil {
			t.Fatalf("CreateRoute() error = %v", err)
		}
	}
	m.loadRoutes()

	r := m.Match("example.com", "/admin/users", nil)
	if r == nil || r.ID != "fallback" {
		t.Fatalf("Match() = %v, want fallback route when stored path_match_mode is unsupported", r)
	}
}

func TestGetRoutes_ReturnsCopy(t *testing.T) {
	m, cleanup := newTestManager(t)
	defer cleanup()

	m.db.CreateRoute(&store.Route{ID: "1", Host: "", PathPrefix: "/", Backend: "http://x", Enabled: true, Priority: 0})
	m.loadRoutes()

	routes := m.GetRoutes()
	if len(routes) != 1 {
		t.Fatalf("len(routes) = %d, want 1", len(routes))
	}
	routes[0].ID = "modified"
	if m.routes[0].ID != "1" {
		t.Error("GetRoutes should return a copy, not the internal slice")
	}
}

func TestReload(t *testing.T) {
	m, cleanup := newTestManager(t)
	defer cleanup()

	m.db.CreateRoute(&store.Route{ID: "1", Host: "", PathPrefix: "/", Backend: "http://x", Enabled: true, Priority: 0})
	m.loadRoutes()

	m.db.CreateRoute(&store.Route{ID: "2", Host: "", PathPrefix: "/api", Backend: "http://y", Enabled: true, Priority: 0})
	m.Reload()

	routes := m.GetRoutes()
	if len(routes) != 2 {
		t.Errorf("After reload len(routes) = %d, want 2", len(routes))
	}
}

func TestMatch_NoRoute(t *testing.T) {
	m, cleanup := newTestManager(t)
	defer cleanup()

	m.db.CreateRoute(&store.Route{ID: "1", Host: "example.com", PathPrefix: "/api", Backend: "http://x", Enabled: true, Priority: 0})
	m.loadRoutes()

	r := m.Match("other.com", "/other", nil)
	if r != nil {
		t.Errorf("Match for non-matching host/path = %v, want nil", r)
	}
}

func TestRoute_AuthConfig(t *testing.T) {
	cfg := &RouteAuthConfig{ApiKeyEnabled: true, ApiKeyHeader: "X-API-Key"}
	r := Route{ID: "1", AuthConfig: cfg}
	if r.AuthConfig == nil || !r.AuthConfig.ApiKeyEnabled {
		t.Error("Route.AuthConfig not set correctly")
	}
}
