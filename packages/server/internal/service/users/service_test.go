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
