package store

import (
	"database/sql"
	"errors"
	"path/filepath"
	"testing"
)

func newTestSQLite(t *testing.T) *SQLite {
	t.Helper()

	db, err := NewSQLite(filepath.Join(t.TempDir(), "auth-gate.db"))
	if err != nil {
		t.Fatalf("NewSQLite() error = %v", err)
	}
	t.Cleanup(func() {
		_ = db.Close()
	})

	return db
}

func createTestRoute(t *testing.T, db *SQLite, id string) *Route {
	t.Helper()

	route := &Route{
		ID:         id,
		Name:       "test-route",
		PathPrefix: "/test",
		Backend:    "http://example.com",
		Enabled:    true,
	}
	if err := db.CreateRoute(route); err != nil {
		t.Fatalf("CreateRoute() error = %v", err)
	}
	return route
}

func TestCreateAuthRule_RejectsMissingRoute(t *testing.T) {
	db := newTestSQLite(t)

	err := db.CreateAuthRule(&AuthRule{
		RouteID: "missing-route",
		Type:    "apikey",
		Config:  AuthConfig{Secret: "secret"},
	})
	if err == nil {
		t.Fatal("CreateAuthRule() error = nil, want foreign key failure")
	}
}

func TestDeleteRoute_CascadesAuthRules(t *testing.T) {
	db := newTestSQLite(t)
	route := createTestRoute(t, db, "route-1")

	if err := db.CreateAuthRule(&AuthRule{
		ID:      "rule-1",
		RouteID: route.ID,
		Type:    "apikey",
		Config:  AuthConfig{Secret: "secret"},
	}); err != nil {
		t.Fatalf("CreateAuthRule() error = %v", err)
	}

	if err := db.DeleteRoute(route.ID); err != nil {
		t.Fatalf("DeleteRoute() error = %v", err)
	}

	_, err := db.GetAuthRule("rule-1")
	if !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("GetAuthRule() error = %v, want %v", err, sql.ErrNoRows)
	}
}

func TestCreateAuthRule_RejectsDuplicateRouteRule(t *testing.T) {
	db := newTestSQLite(t)
	route := createTestRoute(t, db, "route-1")

	if err := db.CreateAuthRule(&AuthRule{
		ID:      "rule-1",
		RouteID: route.ID,
		Type:    "apikey",
		Config:  AuthConfig{Secret: "secret-1"},
	}); err != nil {
		t.Fatalf("CreateAuthRule(first) error = %v", err)
	}

	err := db.CreateAuthRule(&AuthRule{
		ID:      "rule-2",
		RouteID: route.ID,
		Type:    "bearer",
		Config:  AuthConfig{Secret: "secret-2"},
	})
	if err == nil {
		t.Fatal("CreateAuthRule(second) error = nil, want duplicate-route rejection")
	}
}

func TestEnsureAdmin_CreatesBootstrapUser(t *testing.T) {
	db := newTestSQLite(t)

	created, err := db.EnsureAdmin("admin", "bootstrap-secret")
	if err != nil {
		t.Fatalf("EnsureAdmin() error = %v", err)
	}
	if !created {
		t.Fatal("EnsureAdmin() created = false, want true")
	}

	user, err := db.GetUserByUsername("admin")
	if err != nil {
		t.Fatalf("GetUserByUsername() error = %v", err)
	}
	if !db.VerifyPassword(user, "bootstrap-secret") {
		t.Fatal("VerifyPassword() = false, want true")
	}
}

func TestEnsureAdmin_RequiresPassword(t *testing.T) {
	db := newTestSQLite(t)

	created, err := db.EnsureAdmin("admin", "")
	if err == nil {
		t.Fatal("EnsureAdmin() error = nil, want error")
	}
	if created {
		t.Fatal("EnsureAdmin() created = true, want false")
	}
}
