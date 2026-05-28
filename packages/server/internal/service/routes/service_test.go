package routes

import (
	"database/sql"
	"net/http"
	"path/filepath"
	"testing"

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

func stringPtr(v string) *string {
	return &v
}

func boolPtr(v bool) *bool {
	return &v
}

func intPtr(v int) *int {
	return &v
}

func TestServiceCreateRoute_RejectsInvalidBackend(t *testing.T) {
	svc := NewService(newTestDB(t), nil)

	_, err := svc.Create(CreateInput{
		Name:        "broken",
		PathPrefix:  "/svc",
		Backend:     "ftp://example.com",
		StripPrefix: true,
		Enabled:     true,
	})
	if err == nil {
		t.Fatal("Create() error = nil, want validation error")
	}
	if Code(err) != ErrCodeInvalidRouteBackend {
		t.Fatalf("Code(err) = %q, want %q", Code(err), ErrCodeInvalidRouteBackend)
	}
}

func TestServiceCreateRoute_AllowsBackendsWithoutLegacyBackend(t *testing.T) {
	svc := NewService(newTestDB(t), nil)

	route, err := svc.Create(CreateInput{
		Name:       "load-balanced",
		PathPrefix: "/svc",
		Backends: []store.Backend{
			{URL: "http://backend-a.example.com", Weight: 2},
			{URL: "http://backend-b.example.com", Weight: 1},
		},
		Enabled: true,
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if route.Backend != "" {
		t.Fatalf("Backend = %q, want empty string", route.Backend)
	}
	if len(route.Backends) != 2 {
		t.Fatalf("len(Backends) = %d, want %d", len(route.Backends), 2)
	}
}

func TestServiceCreateRoute_RequiresBackendOrBackends(t *testing.T) {
	svc := NewService(newTestDB(t), nil)

	_, err := svc.Create(CreateInput{
		Name:       "broken",
		PathPrefix: "/svc",
		Enabled:    true,
	})
	if err == nil {
		t.Fatal("Create() error = nil, want validation error")
	}
	if Code(err) != ErrCodeMissingRouteFields {
		t.Fatalf("Code(err) = %q, want %q", Code(err), ErrCodeMissingRouteFields)
	}
}

func TestServiceCreateRoute_RejectsInvalidBackends(t *testing.T) {
	svc := NewService(newTestDB(t), nil)

	_, err := svc.Create(CreateInput{
		Name:       "broken",
		PathPrefix: "/svc",
		Backend:    "http://example.com",
		Backends: []store.Backend{
			{URL: "http://backend-a.example.com", Weight: 1},
			{URL: "ftp://backend-b.example.com", Weight: 1},
		},
		Enabled: true,
	})
	if err == nil {
		t.Fatal("Create() error = nil, want validation error")
	}
	if Code(err) != ErrCodeInvalidRouteBackend {
		t.Fatalf("Code(err) = %q, want %q", Code(err), ErrCodeInvalidRouteBackend)
	}
}

func TestServiceCreateRoute_RejectsInvalidBackendWeights(t *testing.T) {
	svc := NewService(newTestDB(t), nil)

	_, err := svc.Create(CreateInput{
		Name:       "broken",
		PathPrefix: "/svc",
		Backend:    "http://example.com",
		Backends: []store.Backend{
			{URL: "http://backend-a.example.com", Weight: 1},
			{URL: "http://backend-b.example.com", Weight: 0},
		},
		Enabled: true,
	})
	if err == nil {
		t.Fatal("Create() error = nil, want validation error")
	}
	if Code(err) != "invalid_route_backend_weight" {
		t.Fatalf("Code(err) = %q, want %q", Code(err), "invalid_route_backend_weight")
	}
}

func TestServiceCreateRoute_RejectsReservedControlPlanePrefix(t *testing.T) {
	svc := NewService(newTestDB(t), nil)

	_, err := svc.Create(CreateInput{
		Name:        "reserved",
		PathPrefix:  "/_authgate/cloud",
		Backend:     "http://example.com",
		StripPrefix: true,
		Enabled:     true,
	})
	if err == nil {
		t.Fatal("Create() error = nil, want validation error")
	}
	if Code(err) != ErrCodeReservedRoutePathPrefix {
		t.Fatalf("Code(err) = %q, want %q", Code(err), ErrCodeReservedRoutePathPrefix)
	}
}

func TestServiceCreateRoute_AcceptsRegexPathMatchMode(t *testing.T) {
	svc := NewService(newTestDB(t), nil)

	route, err := svc.Create(CreateInput{
		Name:          "regex-route",
		PathPrefix:    "^/api/v\\d+",
		Backend:       "http://example.com",
		StripPrefix:   false,
		Enabled:       true,
		PathMatchMode: "regex",
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if route.PathMatchMode != "regex" {
		t.Fatalf("PathMatchMode = %q, want %q", route.PathMatchMode, "regex")
	}
	if route.PathPrefix != "^/api/v\\d+" {
		t.Fatalf("PathPrefix = %q, want %q", route.PathPrefix, "^/api/v\\d+")
	}
}

func TestServiceCreateRoute_NormalizesExplicitPathMatchMode(t *testing.T) {
	svc := NewService(newTestDB(t), nil)

	route, err := svc.Create(CreateInput{
		Name:          "regex-route",
		PathPrefix:    "^/api/v\\d+",
		Backend:       "http://example.com",
		StripPrefix:   false,
		Enabled:       true,
		PathMatchMode: " REGEX_I ",
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if route.PathMatchMode != "regex_i" {
		t.Fatalf("PathMatchMode = %q, want %q", route.PathMatchMode, "regex_i")
	}
}

func TestServiceCreateRoute_RejectsInvalidPathMatchMode(t *testing.T) {
	svc := NewService(newTestDB(t), nil)

	_, err := svc.Create(CreateInput{
		Name:          "invalid-mode",
		PathPrefix:    "/api",
		Backend:       "http://example.com",
		Enabled:       true,
		PathMatchMode: "glob",
	})
	if err == nil {
		t.Fatal("Create() error = nil, want validation error")
	}
	if Code(err) != "invalid_route_path_match_mode" {
		t.Fatalf("Code(err) = %q, want %q", Code(err), "invalid_route_path_match_mode")
	}
}

func TestServiceCreateRoute_RejectsInvalidRegexPathPrefix(t *testing.T) {
	svc := NewService(newTestDB(t), nil)

	_, err := svc.Create(CreateInput{
		Name:          "invalid-regex-route",
		PathPrefix:    "[",
		Backend:       "http://example.com",
		Enabled:       true,
		PathMatchMode: "regex",
	})
	if err == nil {
		t.Fatal("Create() error = nil, want validation error")
	}
	if Code(err) != "invalid_route_path_regex" {
		t.Fatalf("Code(err) = %q, want %q", Code(err), "invalid_route_path_regex")
	}
}

func TestServiceCreateRoute_NormalizesHostCase(t *testing.T) {
	svc := NewService(newTestDB(t), nil)

	route, err := svc.Create(CreateInput{
		Name:       "host-route",
		Host:       " API.EXAMPLE.COM ",
		PathPrefix: "/api",
		Backend:    "http://example.com",
		Enabled:    true,
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if route.Host != "api.example.com" {
		t.Fatalf("Host = %q, want %q", route.Host, "api.example.com")
	}
}

func TestServiceCreateRoute_RejectsInvalidHostFormats(t *testing.T) {
	svc := NewService(newTestDB(t), nil)

	tests := []struct {
		name string
		host string
	}{
		{name: "scheme", host: "https://api.example.com"},
		{name: "hostname with port", host: "api.example.com:8443"},
		{name: "ipv6 with port", host: "[2001:db8::1]:8443"},
		{name: "path", host: "api.example.com/path"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := svc.Create(CreateInput{
				Name:       "host-route",
				Host:       tt.host,
				PathPrefix: "/api",
				Backend:    "http://example.com",
				Enabled:    true,
			})
			if err == nil {
				t.Fatal("Create() error = nil, want validation error")
			}
			if Code(err) != "invalid_route_host" {
				t.Fatalf("Code(err) = %q, want %q", Code(err), "invalid_route_host")
			}
		})
	}
}

func TestServiceCreateRoute_NormalizesBracketedIPv6Host(t *testing.T) {
	svc := NewService(newTestDB(t), nil)

	route, err := svc.Create(CreateInput{
		Name:       "ipv6-route",
		Host:       " [2001:db8::1] ",
		PathPrefix: "/api",
		Backend:    "http://example.com",
		Enabled:    true,
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if route.Host != "2001:db8::1" {
		t.Fatalf("Host = %q, want %q", route.Host, "2001:db8::1")
	}
}

func TestServiceCreateRoute_TrimsWhitespaceOnlyRewriteTarget(t *testing.T) {
	svc := NewService(newTestDB(t), nil)

	route, err := svc.Create(CreateInput{
		Name:          "redirect-route",
		PathPrefix:    "/billing",
		Backend:       "http://example.com",
		Enabled:       true,
		RewriteTarget: "   ",
		RedirectCode:  302,
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if route.RewriteTarget != "" {
		t.Fatalf("RewriteTarget = %q, want empty string", route.RewriteTarget)
	}
}

func TestServiceCreateRoute_RejectsInvalidRedirectCode(t *testing.T) {
	svc := NewService(newTestDB(t), nil)

	_, err := svc.Create(CreateInput{
		Name:         "redirect-route",
		PathPrefix:   "/billing",
		Backend:      "http://example.com",
		Enabled:      true,
		RedirectCode: 307,
	})
	if err == nil {
		t.Fatal("Create() error = nil, want validation error")
	}
	if Code(err) != "invalid_route_redirect_code" {
		t.Fatalf("Code(err) = %q, want %q", Code(err), "invalid_route_redirect_code")
	}
}

func TestServiceUpdateRoute_ReturnsNotFound(t *testing.T) {
	svc := NewService(newTestDB(t), nil)

	_, err := svc.Update("missing", UpdateInput{
		Name:        stringPtr("svc"),
		PathPrefix:  stringPtr("/svc"),
		Backend:     stringPtr("http://example.com"),
		StripPrefix: boolPtr(true),
		Enabled:     boolPtr(true),
	})
	if err == nil {
		t.Fatal("Update() error = nil, want not found")
	}
	if Code(err) != ErrCodeRouteNotFound {
		t.Fatalf("Code(err) = %q, want %q", Code(err), ErrCodeRouteNotFound)
	}
}

func TestServiceDeleteRoute_ReturnsNotFound(t *testing.T) {
	svc := NewService(newTestDB(t), nil)

	err := svc.Delete("missing")
	if err == nil {
		t.Fatal("Delete() error = nil, want not found")
	}
	if Code(err) != ErrCodeRouteNotFound {
		t.Fatalf("Code(err) = %q, want %q", Code(err), ErrCodeRouteNotFound)
	}
}

func TestServiceListRoutes_ReturnsStoredRoutes(t *testing.T) {
	db := newTestDB(t)
	if err := db.CreateRoute(&store.Route{
		ID:         "route-1",
		Name:       "svc",
		PathPrefix: "/svc",
		Backend:    "http://example.com",
		Enabled:    true,
	}); err != nil {
		t.Fatalf("CreateRoute() error = %v", err)
	}

	svc := NewService(db, nil)
	routes, err := svc.List()
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(routes) != 1 {
		t.Fatalf("len(routes) = %d, want 1", len(routes))
	}
	if routes[0].ID != "route-1" {
		t.Fatalf("routes[0].ID = %q, want %q", routes[0].ID, "route-1")
	}
}

func TestServiceGetRoute_NormalizesLegacyStoredPathMatchMode(t *testing.T) {
	db := newTestDB(t)
	if err := db.CreateRoute(&store.Route{
		ID:            "route-1",
		Name:          "svc",
		PathPrefix:    "^/api/v\\d+",
		PathMatchMode: " REGEX_I ",
		Backend:       "http://example.com",
		Enabled:       true,
	}); err != nil {
		t.Fatalf("CreateRoute() error = %v", err)
	}

	svc := NewService(db, nil)
	route, err := svc.Get("route-1")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if route.PathMatchMode != "regex_i" {
		t.Fatalf("PathMatchMode = %q, want %q", route.PathMatchMode, "regex_i")
	}
}

func TestServiceUpdateRoute_PreservesLegacyStoredRedirectCodeByNormalizingIt(t *testing.T) {
	db := newTestDB(t)
	if err := db.CreateRoute(&store.Route{
		ID:            "route-1",
		Name:          "svc",
		PathPrefix:    "/billing",
		Backend:       "http://example.com",
		Enabled:       true,
		RewriteTarget: "/dashboard",
		RedirectCode:  http.StatusTemporaryRedirect,
	}); err != nil {
		t.Fatalf("CreateRoute() error = %v", err)
	}

	svc := NewService(db, nil)
	updated, err := svc.Update("route-1", UpdateInput{
		Name: stringPtr("svc-renamed"),
	})
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	if updated.Name != "svc-renamed" {
		t.Fatalf("Name = %q, want %q", updated.Name, "svc-renamed")
	}
	if updated.RedirectCode != 0 {
		t.Fatalf("RedirectCode = %d, want %d", updated.RedirectCode, 0)
	}
}

func TestServiceDeleteRoute_MapsStoreNotFound(t *testing.T) {
	db := newTestDB(t)
	if err := db.DeleteRoute("missing"); err != nil && err != sql.ErrNoRows {
		t.Fatalf("DeleteRoute() precondition error = %v", err)
	}
}

func TestServiceCreateRoute_TLSConfigStored(t *testing.T) {
	svc := NewService(newTestDB(t), nil)

	route, err := svc.Create(CreateInput{
		Name:        "tls-route",
		PathPrefix:  "/api",
		Backend:     "http://backend.example.com",
		StripPrefix: false,
		Enabled:     true,
		Priority:    10,
		TLSCert:     "/etc/ssl/certs/site.pem",
		TLSKey:      "/etc/ssl/private/site.key",
		TLSEnabled:  true,
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	routes, err := svc.List()
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(routes) != 1 {
		t.Fatalf("len(routes) = %d, want 1", len(routes))
	}

	got := routes[0]
	if got.TLSCert != "/etc/ssl/certs/site.pem" {
		t.Errorf("TLSCert = %q, want %q", got.TLSCert, "/etc/ssl/certs/site.pem")
	}
	if got.TLSKey != "/etc/ssl/private/site.key" {
		t.Errorf("TLSKey = %q, want %q", got.TLSKey, "/etc/ssl/private/site.key")
	}
	if !got.TLSEnabled {
		t.Errorf("TLSEnabled = false, want true")
	}
	if got.ID != route.ID {
		t.Errorf("route.ID = %q, want %q", got.ID, route.ID)
	}
}

func TestServiceUpdateRoute_TLSConfigUpdated(t *testing.T) {
	svc := NewService(newTestDB(t), nil)

	created, err := svc.Create(CreateInput{
		Name:       "initial",
		PathPrefix: "/legacy",
		Backend:    "http://old.example.com",
		TLSCert:    "",
		TLSKey:     "",
		TLSEnabled: false,
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	updated, err := svc.Update(created.ID, UpdateInput{
		Name:       stringPtr("updated"),
		PathPrefix: stringPtr("/legacy"),
		Backend:    stringPtr("http://new.example.com"),
		TLSCert:    stringPtr("/etc/ssl/certs/updated.pem"),
		TLSKey:     stringPtr("/etc/ssl/private/updated.key"),
		TLSEnabled: boolPtr(true),
	})
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	if updated.TLSCert != "/etc/ssl/certs/updated.pem" {
		t.Errorf("TLSCert = %q, want %q", updated.TLSCert, "/etc/ssl/certs/updated.pem")
	}
	if updated.TLSKey != "/etc/ssl/private/updated.key" {
		t.Errorf("TLSKey = %q, want %q", updated.TLSKey, "/etc/ssl/private/updated.key")
	}
	if !updated.TLSEnabled {
		t.Errorf("TLSEnabled = false, want true")
	}

	routes, err := svc.List()
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	got := routes[0]
	if got.TLSCert != "/etc/ssl/certs/updated.pem" {
		t.Errorf("persisted TLSCert = %q, want %q", got.TLSCert, "/etc/ssl/certs/updated.pem")
	}
	if got.TLSKey != "/etc/ssl/private/updated.key" {
		t.Errorf("persisted TLSKey = %q, want %q", got.TLSKey, "/etc/ssl/private/updated.key")
	}
	if !got.TLSEnabled {
		t.Errorf("persisted TLSEnabled = false, want true")
	}
}

func TestServiceUpdateRoute_NormalizesHostCase(t *testing.T) {
	svc := NewService(newTestDB(t), nil)

	created, err := svc.Create(CreateInput{
		Name:       "route",
		Host:       "api.example.com",
		PathPrefix: "/api",
		Backend:    "http://example.com",
		Enabled:    true,
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	updated, err := svc.Update(created.ID, UpdateInput{
		Host: stringPtr(" Reports.EXAMPLE.COM "),
	})
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	if updated.Host != "reports.example.com" {
		t.Fatalf("Host = %q, want %q", updated.Host, "reports.example.com")
	}
}

func TestServiceUpdateRoute_RejectsInvalidBackends(t *testing.T) {
	svc := NewService(newTestDB(t), nil)

	created, err := svc.Create(CreateInput{
		Name:       "route",
		PathPrefix: "/api",
		Backend:    "http://example.com",
		Enabled:    true,
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	backends := []store.Backend{
		{URL: "http://backend-a.example.com", Weight: 1},
		{URL: "://backend-b.example.com", Weight: 1},
	}
	_, err = svc.Update(created.ID, UpdateInput{
		Backends: &backends,
	})
	if err == nil {
		t.Fatal("Update() error = nil, want validation error")
	}
	if Code(err) != ErrCodeInvalidRouteBackend {
		t.Fatalf("Code(err) = %q, want %q", Code(err), ErrCodeInvalidRouteBackend)
	}
}

func TestServiceUpdateRoute_AllowsSwitchingToBackendsWithoutLegacyBackend(t *testing.T) {
	svc := NewService(newTestDB(t), nil)

	created, err := svc.Create(CreateInput{
		Name:       "route",
		PathPrefix: "/api",
		Backend:    "http://example.com",
		Enabled:    true,
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	backends := []store.Backend{
		{URL: "http://backend-a.example.com", Weight: 3},
		{URL: "http://backend-b.example.com", Weight: 1},
	}
	emptyBackend := ""
	updated, err := svc.Update(created.ID, UpdateInput{
		Backend:  &emptyBackend,
		Backends: &backends,
	})
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	if updated.Backend != "" {
		t.Fatalf("Backend = %q, want empty string", updated.Backend)
	}
	if len(updated.Backends) != 2 {
		t.Fatalf("len(Backends) = %d, want %d", len(updated.Backends), 2)
	}
}

func TestServiceUpdateRoute_RejectsInvalidBackendWeights(t *testing.T) {
	svc := NewService(newTestDB(t), nil)

	created, err := svc.Create(CreateInput{
		Name:       "route",
		PathPrefix: "/api",
		Backend:    "http://example.com",
		Enabled:    true,
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	backends := []store.Backend{
		{URL: "http://backend-a.example.com", Weight: -1},
		{URL: "http://backend-b.example.com", Weight: 1},
	}
	_, err = svc.Update(created.ID, UpdateInput{
		Backends: &backends,
	})
	if err == nil {
		t.Fatal("Update() error = nil, want validation error")
	}
	if Code(err) != "invalid_route_backend_weight" {
		t.Fatalf("Code(err) = %q, want %q", Code(err), "invalid_route_backend_weight")
	}
}

func TestServiceUpdateRoute_PreservesOmittedFields(t *testing.T) {
	db := newTestDB(t)
	if err := db.CreateRoute(&store.Route{
		ID:            "route-preserve",
		Name:          "initial",
		Host:          "api.example.com",
		PathPrefix:    "/legacy",
		Backend:       "http://old.example.com",
		StripPrefix:   true,
		Enabled:       true,
		Priority:      7,
		TLSCert:       "/etc/ssl/certs/original.pem",
		TLSKey:        "/etc/ssl/private/original.key",
		TLSEnabled:    true,
		TimeoutMs:     4500,
		RetryAttempts: 3,
		Backends: []store.Backend{
			{URL: "http://backend-a.example.com", Weight: 2},
			{URL: "http://backend-b.example.com", Weight: 1},
		},
		PathMatchMode: "exact",
		RewriteTarget: "/internal",
		RedirectCode:  302,
	}); err != nil {
		t.Fatalf("CreateRoute() error = %v", err)
	}

	svc := NewService(db, nil)
	updated, err := svc.Update("route-preserve", UpdateInput{
		Name: stringPtr("renamed"),
	})
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	if updated.Name != "renamed" {
		t.Fatalf("Name = %q, want %q", updated.Name, "renamed")
	}
	if updated.Host != "api.example.com" {
		t.Fatalf("Host = %q, want %q", updated.Host, "api.example.com")
	}
	if updated.PathPrefix != "/legacy" {
		t.Fatalf("PathPrefix = %q, want %q", updated.PathPrefix, "/legacy")
	}
	if updated.Backend != "http://old.example.com" {
		t.Fatalf("Backend = %q, want %q", updated.Backend, "http://old.example.com")
	}
	if !updated.StripPrefix {
		t.Fatalf("StripPrefix = %v, want true", updated.StripPrefix)
	}
	if !updated.Enabled {
		t.Fatalf("Enabled = %v, want true", updated.Enabled)
	}
	if updated.Priority != 7 {
		t.Fatalf("Priority = %d, want %d", updated.Priority, 7)
	}
	if updated.TLSCert != "/etc/ssl/certs/original.pem" {
		t.Fatalf("TLSCert = %q, want %q", updated.TLSCert, "/etc/ssl/certs/original.pem")
	}
	if updated.TLSKey != "/etc/ssl/private/original.key" {
		t.Fatalf("TLSKey = %q, want %q", updated.TLSKey, "/etc/ssl/private/original.key")
	}
	if !updated.TLSEnabled {
		t.Fatalf("TLSEnabled = %v, want true", updated.TLSEnabled)
	}
	if updated.TimeoutMs != 4500 {
		t.Fatalf("TimeoutMs = %d, want %d", updated.TimeoutMs, 4500)
	}
	if updated.RetryAttempts != 3 {
		t.Fatalf("RetryAttempts = %d, want %d", updated.RetryAttempts, 3)
	}
	if len(updated.Backends) != 2 {
		t.Fatalf("len(Backends) = %d, want %d", len(updated.Backends), 2)
	}
	if updated.PathMatchMode != "exact" {
		t.Fatalf("PathMatchMode = %q, want %q", updated.PathMatchMode, "exact")
	}
	if updated.RewriteTarget != "/internal" {
		t.Fatalf("RewriteTarget = %q, want %q", updated.RewriteTarget, "/internal")
	}
	if updated.RedirectCode != 302 {
		t.Fatalf("RedirectCode = %d, want %d", updated.RedirectCode, 302)
	}
}

func TestServiceUpdateRoute_NormalizesExplicitPathMatchMode(t *testing.T) {
	svc := NewService(newTestDB(t), nil)

	created, err := svc.Create(CreateInput{
		Name:       "svc",
		PathPrefix: "/svc",
		Backend:    "http://example.com",
		Enabled:    true,
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	updated, err := svc.Update(created.ID, UpdateInput{
		PathPrefix:    stringPtr("^/api/v\\d+"),
		PathMatchMode: stringPtr(" REGEX "),
	})
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	if updated.PathMatchMode != "regex" {
		t.Fatalf("PathMatchMode = %q, want %q", updated.PathMatchMode, "regex")
	}
}

func TestServiceUpdateRoute_RejectsInvalidPathMatchMode(t *testing.T) {
	svc := NewService(newTestDB(t), nil)

	created, err := svc.Create(CreateInput{
		Name:       "svc",
		PathPrefix: "/svc",
		Backend:    "http://example.com",
		Enabled:    true,
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	_, err = svc.Update(created.ID, UpdateInput{
		PathMatchMode: stringPtr("glob"),
	})
	if err == nil {
		t.Fatal("Update() error = nil, want validation error")
	}
	if Code(err) != "invalid_route_path_match_mode" {
		t.Fatalf("Code(err) = %q, want %q", Code(err), "invalid_route_path_match_mode")
	}
}

func TestServiceUpdateRoute_RejectsInvalidRegexPathPrefix(t *testing.T) {
	svc := NewService(newTestDB(t), nil)

	created, err := svc.Create(CreateInput{
		Name:          "regex-route",
		PathPrefix:    "^/api/v\\d+",
		Backend:       "http://example.com",
		Enabled:       true,
		PathMatchMode: "regex",
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	invalidPattern := "["
	_, err = svc.Update(created.ID, UpdateInput{
		PathPrefix: &invalidPattern,
	})
	if err == nil {
		t.Fatal("Update() error = nil, want validation error")
	}
	if Code(err) != "invalid_route_path_regex" {
		t.Fatalf("Code(err) = %q, want %q", Code(err), "invalid_route_path_regex")
	}
}

func TestServiceUpdateRoute_ClearsRewriteTargetWhenWhitespaceOnly(t *testing.T) {
	svc := NewService(newTestDB(t), nil)

	created, err := svc.Create(CreateInput{
		Name:          "redirect-route",
		PathPrefix:    "/billing",
		Backend:       "http://example.com",
		Enabled:       true,
		RewriteTarget: "/dashboard",
		RedirectCode:  302,
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	updated, err := svc.Update(created.ID, UpdateInput{
		RewriteTarget: stringPtr("   "),
	})
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	if updated.RewriteTarget != "" {
		t.Fatalf("RewriteTarget = %q, want empty string", updated.RewriteTarget)
	}
}

func TestServiceUpdateRoute_RejectsInvalidRedirectCode(t *testing.T) {
	svc := NewService(newTestDB(t), nil)

	created, err := svc.Create(CreateInput{
		Name:       "route",
		PathPrefix: "/api",
		Backend:    "http://example.com",
		Enabled:    true,
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	_, err = svc.Update(created.ID, UpdateInput{
		RedirectCode: intPtr(308),
	})
	if err == nil {
		t.Fatal("Update() error = nil, want validation error")
	}
	if Code(err) != "invalid_route_redirect_code" {
		t.Fatalf("Code(err) = %q, want %q", Code(err), "invalid_route_redirect_code")
	}
}
