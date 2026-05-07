package session

import (
	"path/filepath"
	"testing"

	"github.com/pallyoung/auth-gate/packages/server/internal/auth"
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

func createUser(t *testing.T, db *store.SQLite, username, password, role string, enabled bool) {
	t.Helper()

	hash, err := store.HashPassword(password)
	if err != nil {
		t.Fatalf("HashPassword() error = %v", err)
	}
	if err := db.CreateUser(&store.User{
		Username:     username,
		PasswordHash: hash,
		Role:         role,
		Enabled:      enabled,
	}); err != nil {
		t.Fatalf("CreateUser() error = %v", err)
	}
}

func TestServiceLogin_ReturnsSession(t *testing.T) {
	auth.ConfigureJWTSecret("test-secret")
	db := newTestDB(t)
	createUser(t, db, "admin", "password123", store.RoleAdmin, true)
	svc := NewService(db)

	session, err := svc.Login("admin", "password123")
	if err != nil {
		t.Fatalf("Login() error = %v", err)
	}
	if session.Token == "" {
		t.Fatal("session.Token is empty")
	}
	if !session.Permissions.CanManageUsers {
		t.Fatal("session.Permissions.CanManageUsers = false, want true for admin")
	}
}

func TestServiceLogin_RejectsDisabledUser(t *testing.T) {
	auth.ConfigureJWTSecret("test-secret")
	db := newTestDB(t)
	createUser(t, db, "disabled", "password123", store.RoleViewer, false)
	svc := NewService(db)

	_, err := svc.Login("disabled", "password123")
	if err == nil {
		t.Fatal("Login() error = nil, want disabled user error")
	}
	if Code(err) != ErrCodeUserDisabled {
		t.Fatalf("Code(err) = %q, want %q", Code(err), ErrCodeUserDisabled)
	}
}

func TestServiceLogin_RejectsRouteOnlyUserForControlPlane(t *testing.T) {
	auth.ConfigureJWTSecret("test-secret")
	db := newTestDB(t)
	createUser(t, db, "member", "password123", store.RoleMember, true)
	svc := NewService(db)

	_, err := svc.Login("member", "password123")
	if err == nil {
		t.Fatal("Login() error = nil, want control plane access denied")
	}
	if Code(err) != ErrCodeControlPlaneAccessDenied {
		t.Fatalf("Code(err) = %q, want %q", Code(err), ErrCodeControlPlaneAccessDenied)
	}
}

func TestServiceLoginForRoute_ValidatesAssignedAccess(t *testing.T) {
	auth.ConfigureJWTSecret("test-secret")
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

	hash, err := store.HashPassword("password123")
	if err != nil {
		t.Fatalf("HashPassword() error = %v", err)
	}
	if err := db.CreateUser(&store.User{
		Username:     "member",
		PasswordHash: hash,
		Role:         store.RoleMember,
		Enabled:      true,
		RouteIDs:     []string{"route-1"},
	}); err != nil {
		t.Fatalf("CreateUser() error = %v", err)
	}

	svc := NewService(db)
	routeSession, err := svc.LoginForRoute("route-1", "member", "password123")
	if err != nil {
		t.Fatalf("LoginForRoute() error = %v", err)
	}
	if routeSession.Token == "" {
		t.Fatal("routeSession.Token is empty")
	}

	_, err = svc.LoginForRoute("other-route", "member", "password123")
	if err == nil {
		t.Fatal("LoginForRoute() error = nil, want route not found")
	}
	if Code(err) != ErrCodeRouteNotFound {
		t.Fatalf("Code(err) = %q, want %q", Code(err), ErrCodeRouteNotFound)
	}
}
