package hostservice

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pallyoung/auth-gate/packages/server/internal/store"
	syshosts "github.com/pallyoung/auth-gate/packages/server/internal/syshosts"
)

type failingActivationStore struct{ store.Store }

func (s failingActivationStore) SetActiveHostProfile(string) error {
	return errors.New("storage unavailable")
}

func TestService_StateSaveFailureRestoresHosts(t *testing.T) {
	svc, db := newTestService(t)
	p, _ := svc.CreateProfile(ProfileInput{Name: "dev"})
	dir := t.TempDir()
	path := filepath.Join(dir, "hosts")
	original := "127.0.0.1 localhost\r\n# user comment\r\n"
	if err := os.WriteFile(path, []byte(original), 0644); err != nil {
		t.Fatal(err)
	}
	svc.renderer = &syshosts.Renderer{HostsPath: path, BackupDir: filepath.Join(dir, "backup")}
	svc.db = failingActivationStore{db}
	if _, err := svc.ActivateProfile(p.ID); Code(err) != ErrCodeStoreFailure {
		t.Fatalf("error = %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil || string(data) != original {
		t.Fatalf("hosts not restored: %q, %v", data, err)
	}
	got, _ := db.GetHostProfile(p.ID)
	if got.IsActive {
		t.Fatal("failed state save activated profile")
	}
}

func newTestService(t *testing.T) (*Service, store.Store) {
	t.Helper()
	db, err := store.NewJSONStore(t.TempDir())
	if err != nil {
		t.Fatalf("NewJSONStore() error = %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return NewService(db, nil), db
}

func TestService_FailedActivationKeepsPreviousProfile(t *testing.T) {
	svc, db := newTestService(t)
	previous, _ := svc.CreateProfile(ProfileInput{Name: "previous"})
	next, _ := svc.CreateProfile(ProfileInput{Name: "next"})
	if err := db.SetActiveHostProfile(previous.ID); err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	path := filepath.Join(dir, "hosts")
	// A damaged marker block makes the real renderer reject the write.
	original := "127.0.0.1 localhost\n" + syshosts.BeginMarker + "\n"
	if err := os.WriteFile(path, []byte(original), 0644); err != nil {
		t.Fatal(err)
	}
	svc.renderer = &syshosts.Renderer{HostsPath: path, BackupDir: filepath.Join(dir, "backups")}
	if _, err := svc.ActivateProfile(next.ID); Code(err) != ErrCodeMarkerMissing {
		t.Fatalf("error = %v", err)
	}
	got, _ := svc.GetProfile(previous.ID)
	if !got.IsActive {
		t.Fatal("failed write cleared the previous active profile")
	}
	got, _ = svc.GetProfile(next.ID)
	if got.IsActive {
		t.Fatal("failed write activated the new profile")
	}
	data, err := os.ReadFile(path)
	if err != nil || string(data) != original {
		t.Fatalf("hosts changed: %q, %v", data, err)
	}
}

func TestService_DeactivatePreservesOtherHostsAndSavedEntries(t *testing.T) {
	svc, db := newTestService(t)
	p, _ := svc.CreateProfile(ProfileInput{Name: "dev"})
	other, _ := svc.CreateProfile(ProfileInput{Name: "other"})
	_, err := svc.CreateEntry(p.ID, EntryInput{IP: "10.0.0.1", Hostnames: []string{"dev.local"}, Enabled: true})
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	path := filepath.Join(dir, "hosts")
	original := "127.0.0.1 localhost\n" + syshosts.BeginMarker + "\n\n" + syshosts.EndMarker + "\n::1 localhost\n"
	if err := os.WriteFile(path, []byte(original), 0644); err != nil {
		t.Fatal(err)
	}
	svc.renderer = &syshosts.Renderer{HostsPath: path, BackupDir: filepath.Join(dir, "backups")}
	if _, err := svc.ActivateProfile(p.ID); err != nil {
		t.Fatal(err)
	}
	deactivator, ok := any(svc).(interface {
		DeactivateProfile(string) (*store.HostProfile, error)
	})
	if !ok {
		t.Fatal("service does not support deactivation")
	}
	// Deactivating an inactive profile must not clear the currently active one.
	if _, err := deactivator.DeactivateProfile(other.ID); err != nil {
		t.Fatal(err)
	}
	active, _ := db.GetHostProfile(p.ID)
	if !active.IsActive {
		t.Fatal("deactivated the wrong profile")
	}
	result, err := deactivator.DeactivateProfile(p.ID)
	if err != nil {
		t.Fatal(err)
	}
	if result.IsActive {
		t.Fatal("profile remains active")
	}
	data, err := os.ReadFile(path)
	if err != nil || string(data) != original {
		t.Fatalf("hosts = %q, err = %v", data, err)
	}
	entries, _ := svc.ListEntries(p.ID)
	if len(entries) != 1 {
		t.Fatal("deactivation deleted saved entries")
	}
	if _, err := deactivator.DeactivateProfile(p.ID); err != nil {
		t.Fatal("repeat deactivation:", err)
	}
	if _, err := svc.ActivateProfile(p.ID); err != nil {
		t.Fatal(err)
	}
	// Reject a damaged file without claiming successful deactivation.
	if err := os.WriteFile(path, []byte(syshosts.BeginMarker+"\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := deactivator.DeactivateProfile(p.ID); Code(err) != ErrCodeMarkerMissing {
		t.Fatalf("error = %v", err)
	}
	active, _ = db.GetHostProfile(p.ID)
	if !active.IsActive {
		t.Fatal("failed deactivation cleared status")
	}
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

func TestService_CreateEntry_RejectsInvalidIP(t *testing.T) {
	svc, _ := newTestService(t)
	p, _ := svc.CreateProfile(ProfileInput{Name: "dev"})
	_, err := svc.CreateEntry(p.ID, EntryInput{IP: "not-an-ip", Hostnames: []string{"api.local"}, Enabled: true})
	if Code(err) != ErrCodeInvalidIP {
		t.Fatalf("Code(err) = %q, want %q", Code(err), ErrCodeInvalidIP)
	}
}

func TestService_CreateEntry_RejectsInvalidHostname(t *testing.T) {
	svc, _ := newTestService(t)
	p, _ := svc.CreateProfile(ProfileInput{Name: "dev"})
	_, err := svc.CreateEntry(p.ID, EntryInput{IP: "127.0.0.1", Hostnames: []string{"bad host"}, Enabled: true})
	if Code(err) != ErrCodeInvalidHostname {
		t.Fatalf("Code(err) = %q, want %q", Code(err), ErrCodeInvalidHostname)
	}
}

func TestService_CreateEntry_RejectsDuplicateHostname(t *testing.T) {
	svc, _ := newTestService(t)
	p, _ := svc.CreateProfile(ProfileInput{Name: "dev"})
	if _, err := svc.CreateEntry(p.ID, EntryInput{IP: "127.0.0.1", Hostnames: []string{"api.local"}, Enabled: true}); err != nil {
		t.Fatalf("CreateEntry(first) error = %v", err)
	}
	_, err := svc.CreateEntry(p.ID, EntryInput{IP: "10.0.0.1", Hostnames: []string{"api.local"}, Enabled: true})
	if Code(err) != ErrCodeDuplicateHostname {
		t.Fatalf("Code(err) = %q, want %q", Code(err), ErrCodeDuplicateHostname)
	}
}

func TestService_EntriesLifecycle(t *testing.T) {
	svc, _ := newTestService(t)
	p, _ := svc.CreateProfile(ProfileInput{Name: "dev"})
	e1, err := svc.CreateEntry(p.ID, EntryInput{IP: "127.0.0.1", Hostnames: []string{"a.local"}, Enabled: true})
	if err != nil {
		t.Fatalf("CreateEntry(e1) error = %v", err)
	}
	e2, err := svc.CreateEntry(p.ID, EntryInput{IP: "127.0.0.1", Hostnames: []string{"b.local"}, Enabled: false})
	if err != nil {
		t.Fatalf("CreateEntry(e2) error = %v", err)
	}

	entries, err := svc.ListEntries(p.ID)
	if err != nil {
		t.Fatalf("ListEntries() error = %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("len(entries) = %d, want 2", len(entries))
	}

	if _, err := svc.UpdateEntry(p.ID, e1.ID, EntryInput{IP: "10.0.0.1", Hostnames: []string{"a.local", "alt.local"}, Enabled: true}); err != nil {
		t.Fatalf("UpdateEntry() error = %v", err)
	}
	if err := svc.ReorderEntries(p.ID, []string{e2.ID, e1.ID}); err != nil {
		t.Fatalf("ReorderEntries() error = %v", err)
	}
	entries, _ = svc.ListEntries(p.ID)
	if entries[0].ID != e2.ID || entries[1].ID != e1.ID {
		t.Fatalf("order after reorder = [%s, %s], want [%s, %s]", entries[0].ID, entries[1].ID, e2.ID, e1.ID)
	}

	if err := svc.DeleteEntry(p.ID, e1.ID); err != nil {
		t.Fatalf("DeleteEntry() error = %v", err)
	}
	if _, err := svc.GetEntry(p.ID, e1.ID); Code(err) != ErrCodeEntryNotFound {
		t.Fatalf("GetEntry() after delete Code(err) = %q, want %q", Code(err), ErrCodeEntryNotFound)
	}
}

func TestService_ActivateProfile_WritesFileAndSetsActive(t *testing.T) {
	db, err := store.NewJSONStore(t.TempDir())
	if err != nil {
		t.Fatalf("NewJSONStore() error = %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	dir := t.TempDir()
	hosts := filepath.Join(dir, "hosts")
	if err := os.WriteFile(hosts, []byte(""), 0644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	r := &syshosts.Renderer{
		HostsPath:   hosts,
		BackupDir:   filepath.Join(dir, "backup"),
		KeepBackups: 5,
	}

	svc := NewService(db, r)
	p, _ := svc.CreateProfile(ProfileInput{Name: "dev"})
	if _, err := svc.CreateEntry(p.ID, EntryInput{IP: "127.0.0.1", Hostnames: []string{"api.local"}, Enabled: true}); err != nil {
		t.Fatalf("CreateEntry() error = %v", err)
	}

	activated, err := svc.ActivateProfile(p.ID)
	if err != nil {
		t.Fatalf("ActivateProfile() error = %v", err)
	}
	if !activated.IsActive {
		t.Fatal("activated.IsActive = false, want true")
	}
	got, err := os.ReadFile(hosts)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	s := string(got)
	if !strings.Contains(s, syshosts.BeginMarker) || !strings.Contains(s, syshosts.EndMarker) {
		t.Fatalf("output missing markers: %q", s)
	}
	if !strings.Contains(s, "127.0.0.1 api.local") {
		t.Fatalf("output missing entry: %q", s)
	}
}

func TestService_ActivateProfile_AppendsMarkersWhenMissing(t *testing.T) {
	db, err := store.NewJSONStore(t.TempDir())
	if err != nil {
		t.Fatalf("NewJSONStore() error = %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	dir := t.TempDir()
	hosts := filepath.Join(dir, "hosts")
	if err := os.WriteFile(hosts, []byte("127.0.0.1 localhost\n# hand-written\n"), 0644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	r := &syshosts.Renderer{HostsPath: hosts, BackupDir: filepath.Join(dir, "backup")}

	svc := NewService(db, r)
	p, _ := svc.CreateProfile(ProfileInput{Name: "dev"})
	if _, err := svc.CreateEntry(p.ID, EntryInput{IP: "127.0.0.1", Hostnames: []string{"api.local"}, Enabled: true}); err != nil {
		t.Fatalf("CreateEntry() error = %v", err)
	}

	_, err = svc.ActivateProfile(p.ID)
	if err != nil {
		t.Fatalf("ActivateProfile() error = %v", err)
	}

	got, _ := db.GetHostProfile(p.ID)
	if !got.IsActive {
		t.Fatal("profile was not activated")
	}
	raw, _ := os.ReadFile(hosts)
	s := string(raw)
	if !strings.Contains(s, "api.local") {
		t.Fatalf("entry not written: %q", s)
	}
	if !strings.Contains(s, "127.0.0.1 localhost") {
		t.Fatalf("original content lost: %q", s)
	}
	if !strings.Contains(s, syshosts.BeginMarker) {
		t.Fatalf("begin marker missing: %q", s)
	}
}

func TestService_ActivateProfile_SkipsDisabledEntries(t *testing.T) {
	db, err := store.NewJSONStore(t.TempDir())
	if err != nil {
		t.Fatalf("NewJSONStore() error = %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	dir := t.TempDir()
	hosts := filepath.Join(dir, "hosts")
	if err := os.WriteFile(hosts, []byte(""), 0644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	r := &syshosts.Renderer{HostsPath: hosts, BackupDir: filepath.Join(dir, "backup")}

	svc := NewService(db, r)
	p, _ := svc.CreateProfile(ProfileInput{Name: "dev"})
	if _, err := svc.CreateEntry(p.ID, EntryInput{IP: "127.0.0.1", Hostnames: []string{"on.local"}, Enabled: true}); err != nil {
		t.Fatalf("CreateEntry(on) error = %v", err)
	}
	if _, err := svc.CreateEntry(p.ID, EntryInput{IP: "127.0.0.1", Hostnames: []string{"off.local"}, Enabled: false}); err != nil {
		t.Fatalf("CreateEntry(off) error = %v", err)
	}

	if _, err := svc.ActivateProfile(p.ID); err != nil {
		t.Fatalf("ActivateProfile() error = %v", err)
	}
	raw, _ := os.ReadFile(hosts)
	s := string(raw)
	if !strings.Contains(s, "on.local") {
		t.Fatalf("enabled entry missing: %q", s)
	}
	if strings.Contains(s, "off.local") {
		t.Fatalf("disabled entry should be skipped: %q", s)
	}
}

func TestService_ActivateProfile_UnknownID(t *testing.T) {
	svc, _ := newTestService(t)
	_, err := svc.ActivateProfile("missing")
	if Code(err) != ErrCodeProfileNotFound {
		t.Fatalf("Code(err) = %q, want %q", Code(err), ErrCodeProfileNotFound)
	}
}

func stringRepeat(s string, n int) string {
	out := make([]byte, 0, len(s)*n)
	for i := 0; i < n; i++ {
		out = append(out, s...)
	}
	return string(out)
}
