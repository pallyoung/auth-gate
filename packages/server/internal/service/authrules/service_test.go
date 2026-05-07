package authrules

import (
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

func createRoute(t *testing.T, db *store.SQLite, id string) {
	t.Helper()

	if err := db.CreateRoute(&store.Route{
		ID:         id,
		Name:       "svc",
		PathPrefix: "/svc",
		Backend:    "http://example.com",
		Enabled:    true,
	}); err != nil {
		t.Fatalf("CreateRoute() error = %v", err)
	}
}

func TestServiceCreateAuthRule_RejectsMissingRoute(t *testing.T) {
	svc := NewService(newTestDB(t), nil)

	_, err := svc.Create(CreateInput{
		RouteID: "missing",
		Type:    "apikey",
		Config: AuthConfigInput{
			Secret: "secret-1",
		},
	})
	if err == nil {
		t.Fatal("Create() error = nil, want route not found")
	}
	if Code(err) != ErrCodeRouteNotFound {
		t.Fatalf("Code(err) = %q, want %q", Code(err), ErrCodeRouteNotFound)
	}
}

func TestServiceCreateAuthRule_RejectsMissingSecret(t *testing.T) {
	db := newTestDB(t)
	createRoute(t, db, "route-1")
	svc := NewService(db, nil)

	_, err := svc.Create(CreateInput{
		RouteID: "route-1",
		Type:    "bearer",
		Config:  AuthConfigInput{},
	})
	if err == nil {
		t.Fatal("Create() error = nil, want validation error")
	}
	if Code(err) != ErrCodeMissingBearerSecret {
		t.Fatalf("Code(err) = %q, want %q", Code(err), ErrCodeMissingBearerSecret)
	}
}

func TestServiceUpdateAuthRule_PreservesSecretWhenOmitted(t *testing.T) {
	db := newTestDB(t)
	createRoute(t, db, "route-1")
	svc := NewService(db, nil)

	created, err := svc.Create(CreateInput{
		RouteID: "route-1",
		Type:    "bearer",
		Config: AuthConfigInput{
			Secret: "shared-secret",
		},
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	updated, err := svc.Update(created.ID, UpdateInput{
		RouteID: "route-1",
		Type:    "bearer",
		Config:  AuthConfigInput{},
	})
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	if updated.Config.Secret != "shared-secret" {
		t.Fatalf("updated.Config.Secret = %q, want preserved secret", updated.Config.Secret)
	}
}

func TestServiceCreateAuthRule_AllowsGatewayType(t *testing.T) {
	db := newTestDB(t)
	createRoute(t, db, "route-1")
	svc := NewService(db, nil)

	rule, err := svc.Create(CreateInput{
		RouteID: "route-1",
		Type:    "gateway",
		Config:  AuthConfigInput{},
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if rule.Type != "gateway" {
		t.Fatalf("rule.Type = %q, want gateway", rule.Type)
	}
	if rule.Config.LoginMode != "form" {
		t.Fatalf("rule.Config.LoginMode = %q, want form", rule.Config.LoginMode)
	}
}
