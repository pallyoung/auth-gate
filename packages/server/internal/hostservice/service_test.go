package hostservice

import (
	"path/filepath"
	"testing"

	"github.com/pallyoung/auth-gate/packages/server/internal/store"
)

func newTestService(t *testing.T) (*Service, *store.SQLite) {
	t.Helper()
	db, err := store.NewSQLite(filepath.Join(t.TempDir(), "auth-gate.db"))
	if err != nil {
		t.Fatalf("NewSQLite() error = %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return NewService(db, nil), db
}

func TestService_CreateAndListProfiles(t *testing.T) {
	svc, _ := newTestService(t)

	_, err := svc.CreateProfile(ProfileInput{Name: "dev", Description: "dev cluster"})
	if err != nil {
		t.Fatalf("CreateProfile() error = %v", err)
	}

	profiles, err := svc.ListProfiles()
	if err != nil {
		t.Fatalf("ListProfiles() error = %v", err)
	}
	if len(profiles) != 1 || profiles[0].Name != "dev" {
		t.Fatalf("profiles = %+v, want one named dev", profiles)
	}
}

func TestService_GetProfile_NotFound(t *testing.T) {
	svc, _ := newTestService(t)
	_, err := svc.GetProfile("missing")
	if Code(err) != ErrCodeProfileNotFound {
		t.Fatalf("Code(err) = %q, want %q", Code(err), ErrCodeProfileNotFound)
	}
}

func TestService_RejectsInvalidProfileName(t *testing.T) {
	svc, _ := newTestService(t)
	cases := []struct {
		name  string
		input string
	}{
		{"empty", ""},
		{"too-long", stringRepeat("a", 33)},
		{"slash", "bad/name"},
		{"colon", "bad:name"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := svc.CreateProfile(ProfileInput{Name: c.input})
			if Code(err) != ErrCodeInvalidProfileName {
				t.Fatalf("Code(err) = %q, want %q", Code(err), ErrCodeInvalidProfileName)
			}
		})
	}
}

func TestService_RejectsDuplicateProfileName(t *testing.T) {
	svc, _ := newTestService(t)
	if _, err := svc.CreateProfile(ProfileInput{Name: "dev"}); err != nil {
		t.Fatalf("CreateProfile(first) error = %v", err)
	}
	_, err := svc.CreateProfile(ProfileInput{Name: "dev"})
	if Code(err) != ErrCodeDuplicateProfileName {
		t.Fatalf("Code(err) = %q, want %q", Code(err), ErrCodeDuplicateProfileName)
	}
}

func TestService_UpdateProfile(t *testing.T) {
	svc, db := newTestService(t)
	p, err := svc.CreateProfile(ProfileInput{Name: "dev"})
	if err != nil {
		t.Fatalf("CreateProfile() error = %v", err)
	}
	e := &store.HostEntry{ProfileID: p.ID, Position: 0, IP: "127.0.0.1", Hostnames: "api.local"}
	if err := db.CreateHostEntry(e); err != nil {
		t.Fatalf("CreateHostEntry() error = %v", err)
	}

	if _, err := svc.UpdateProfile(p.ID, ProfileInput{Name: "staging", Description: "stg"}); err != nil {
		t.Fatalf("UpdateProfile() error = %v", err)
	}
	got, err := svc.GetProfile(p.ID)
	if err != nil {
		t.Fatalf("GetProfile() error = %v", err)
	}
	if got.Name != "staging" || got.Description != "stg" {
		t.Fatalf("got = %+v, want name=staging description=stg", got)
	}
}

func TestService_DeleteProfile(t *testing.T) {
	svc, db := newTestService(t)
	p, err := svc.CreateProfile(ProfileInput{Name: "dev"})
	if err != nil {
		t.Fatalf("CreateProfile() error = %v", err)
	}
	e := &store.HostEntry{ProfileID: p.ID, Position: 0, IP: "127.0.0.1", Hostnames: "api.local"}
	if err := db.CreateHostEntry(e); err != nil {
		t.Fatalf("CreateHostEntry() error = %v", err)
	}

	if err := svc.DeleteProfile(p.ID); err != nil {
		t.Fatalf("DeleteProfile() error = %v", err)
	}
	if _, err := svc.GetProfile(p.ID); Code(err) != ErrCodeProfileNotFound {
		t.Fatalf("GetProfile() after delete Code(err) = %q, want %q", Code(err), ErrCodeProfileNotFound)
	}
	entries, _ := db.ListHostEntries(p.ID)
	if len(entries) != 0 {
		t.Fatalf("entries after profile delete = %d, want 0 (cascade)", len(entries))
	}
}

func stringRepeat(s string, n int) string {
	out := make([]byte, 0, len(s)*n)
	for i := 0; i < n; i++ {
		out = append(out, s...)
	}
	return string(out)
}
