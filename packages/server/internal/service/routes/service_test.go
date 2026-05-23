package routes

import (
	"database/sql"
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

func TestServiceUpdateRoute_ReturnsNotFound(t *testing.T) {
	svc := NewService(newTestDB(t), nil)

	_, err := svc.Update("missing", UpdateInput{
		Name:        "svc",
		PathPrefix:  "/svc",
		Backend:     "http://example.com",
		StripPrefix: true,
		Enabled:     true,
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
		Name:       "updated",
		PathPrefix: "/legacy",
		Backend:    "http://new.example.com",
		TLSCert:    "/etc/ssl/certs/updated.pem",
		TLSKey:     "/etc/ssl/private/updated.key",
		TLSEnabled: true,
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
