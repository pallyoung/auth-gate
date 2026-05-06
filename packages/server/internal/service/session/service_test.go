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
