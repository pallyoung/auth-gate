package users

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

func TestServiceCreateUser_NormalizesRole(t *testing.T) {
	svc := NewService(newTestDB(t))

	user, err := svc.Create(CreateInput{
		Username: "alice",
		Password: "password123",
		Role:     "",
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if user.Role != store.RoleViewer {
		t.Fatalf("user.Role = %q, want %q", user.Role, store.RoleViewer)
	}
}

func TestServiceCreateUser_RejectsInvalidRole(t *testing.T) {
	svc := NewService(newTestDB(t))

	_, err := svc.Create(CreateInput{
		Username: "alice",
		Password: "password123",
		Role:     "owner",
	})
	if err == nil {
		t.Fatal("Create() error = nil, want invalid role")
	}
	if Code(err) != ErrCodeInvalidRole {
		t.Fatalf("Code(err) = %q, want %q", Code(err), ErrCodeInvalidRole)
	}
}

func TestServiceCreateUser_AssignsRouteAccess(t *testing.T) {
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

	svc := NewService(db)
	user, err := svc.Create(CreateInput{
		Username: "member-1",
		Password: "password123",
		Role:     store.RoleMember,
		Enabled:  true,
		RouteIDs: []string{"route-1"},
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if len(user.RouteIDs) != 1 || user.RouteIDs[0] != "route-1" {
		t.Fatalf("user.RouteIDs = %v, want [route-1]", user.RouteIDs)
	}
}
