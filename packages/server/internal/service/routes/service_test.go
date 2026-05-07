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
