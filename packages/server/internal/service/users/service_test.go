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

func TestServiceCreateUser_DefaultsEmptyRoleToMember(t *testing.T) {
	svc := NewService(newTestDB(t))

	user, err := svc.Create(CreateInput{
		Username: "alice",
		Password: "password123",
		Enabled:  true,
		Role:     "",
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if user.Role != store.RoleMember {
		t.Fatalf("user.Role = %q, want %q", user.Role, store.RoleMember)
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

func TestServiceCreateUser_RejectsWhitespaceOnlyUsername(t *testing.T) {
	svc := NewService(newTestDB(t))

	_, err := svc.Create(CreateInput{
		Username: "   ",
		Password: "password123",
		Role:     store.RoleViewer,
	})
	if err == nil {
		t.Fatal("Create() error = nil, want invalid username")
	}
	if Code(err) != ErrCodeInvalidUsername {
		t.Fatalf("Code(err) = %q, want %q", Code(err), ErrCodeInvalidUsername)
	}
}

func TestServiceUpdateUser_RejectsWhitespaceOnlyUsername(t *testing.T) {
	db := newTestDB(t)
	svc := NewService(db)

	user, err := svc.Create(CreateInput{
		Username: "alice",
		Password: "password123",
		Role:     store.RoleViewer,
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	role := store.RoleViewer
	enabled := true
	username := "   "
	_, err = svc.Update(user.ID, UpdateInput{
		Username: &username,
		Role:     &role,
		Enabled:  &enabled,
	})
	if err == nil {
		t.Fatal("Update() error = nil, want invalid username")
	}
	if Code(err) != ErrCodeInvalidUsername {
		t.Fatalf("Code(err) = %q, want %q", Code(err), ErrCodeInvalidUsername)
	}
}

func TestServiceUpdateUser_PreservesUsernameWhenOmitted(t *testing.T) {
	db := newTestDB(t)
	svc := NewService(db)

	user, err := svc.Create(CreateInput{
		Username: "alice",
		Password: "password123",
		Role:     store.RoleViewer,
		Enabled:  true,
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	enabled := false
	updated, err := svc.Update(user.ID, UpdateInput{
		Enabled: &enabled,
	})
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	if updated.Username != "alice" {
		t.Fatalf("updated.Username = %q, want %q", updated.Username, "alice")
	}
	if updated.Enabled != false {
		t.Fatalf("updated.Enabled = %v, want false", updated.Enabled)
	}
}

func TestServiceUpdateUser_RejectsDuplicateUsername(t *testing.T) {
	db := newTestDB(t)
	svc := NewService(db)

	if _, err := svc.Create(CreateInput{
		Username: "alice",
		Password: "password123",
		Role:     store.RoleViewer,
	}); err != nil {
		t.Fatalf("Create(alice) error = %v", err)
	}

	bob, err := svc.Create(CreateInput{
		Username: "bob",
		Password: "password123",
		Role:     store.RoleViewer,
	})
	if err != nil {
		t.Fatalf("Create(bob) error = %v", err)
	}

	role := store.RoleViewer
	enabled := true
	username := "alice"
	_, err = svc.Update(bob.ID, UpdateInput{
		Username: &username,
		Role:     &role,
		Enabled:  &enabled,
	})
	if err == nil {
		t.Fatal("Update() error = nil, want duplicate username")
	}
	if Code(err) != ErrCodeDuplicateUser {
		t.Fatalf("Code(err) = %q, want %q", Code(err), ErrCodeDuplicateUser)
	}
}
