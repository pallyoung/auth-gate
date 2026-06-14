package syshosts

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"
)

func TestApply_FreshEmptyFile_WritesMarkerBlockWithContent(t *testing.T) {
	dir := t.TempDir()
	hosts := filepath.Join(dir, "hosts")
	if err := os.WriteFile(hosts, []byte(""), 0644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	r := &Renderer{
		HostsPath:   hosts,
		BackupDir:   filepath.Join(dir, "backup"),
		KeepBackups: 5,
		Now: func() time.Time { return time.Date(2026, 6, 14, 12, 0, 0, 0, time.UTC) },
	}
	if err := r.Apply("127.0.0.1 api.local\n"); err != nil {
		t.Fatalf("Apply() error = %v", err)
	}

	got, err := os.ReadFile(hosts)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if !strings.Contains(string(got), BeginMarker) || !strings.Contains(string(got), EndMarker) {
		t.Fatalf("output missing markers: %q", string(got))
	}
	if !strings.Contains(string(got), "127.0.0.1 api.local") {
		t.Fatalf("output missing entry: %q", string(got))
	}
}

func TestApply_NonEmptyFileWithoutMarkers_AppendsMarkerBlock(t *testing.T) {
	dir := t.TempDir()
	hosts := filepath.Join(dir, "hosts")
	original := "127.0.0.1 localhost\n# my hand-written comment\n"
	if err := os.WriteFile(hosts, []byte(original), 0644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	r := &Renderer{
		HostsPath:   hosts,
		BackupDir:   filepath.Join(dir, "backup"),
		KeepBackups: 5,
		Now:         func() time.Time { return time.Date(2026, 6, 14, 12, 0, 0, 0, time.UTC) },
	}
	if err := r.Apply("10.0.0.1 api.local\n"); err != nil {
		t.Fatalf("Apply() error = %v", err)
	}

	got, err := os.ReadFile(hosts)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	s := string(got)
	if !strings.Contains(s, "127.0.0.1 localhost") || !strings.Contains(s, "# my hand-written comment") {
		t.Fatalf("original content lost: %q", s)
	}
	if !strings.Contains(s, BeginMarker) || !strings.Contains(s, EndMarker) {
		t.Fatalf("markers missing: %q", s)
	}
	if !strings.Contains(s, "10.0.0.1 api.local") {
		t.Fatalf("new entry not written: %q", s)
	}
}

func TestApply_ReplacesInsideMarkerBlock_PreservesOutside(t *testing.T) {
	dir := t.TempDir()
	hosts := filepath.Join(dir, "hosts")
	original := strings.Join([]string{
		"127.0.0.1 localhost",
		"# my hand-written",
		BeginMarker,
		"10.0.0.1 OLD.local",
		EndMarker,
		"192.168.1.1 router.local",
		"",
	}, "\n")
	if err := os.WriteFile(hosts, []byte(original), 0644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	r := &Renderer{
		HostsPath:   hosts,
		BackupDir:   filepath.Join(dir, "backup"),
		KeepBackups: 5,
	}
	if err := r.Apply("127.0.0.1 api.local\n127.0.0.1 db.local\n"); err != nil {
		t.Fatalf("Apply() error = %v", err)
	}

	got, err := os.ReadFile(hosts)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	s := string(got)
	if strings.Contains(s, "10.0.0.1 OLD.local") {
		t.Fatalf("old managed content not removed: %q", s)
	}
	if !strings.Contains(s, "127.0.0.1 api.local") || !strings.Contains(s, "127.0.0.1 db.local") {
		t.Fatalf("new content not written: %q", s)
	}
	if !strings.Contains(s, "127.0.0.1 localhost") || !strings.Contains(s, "# my hand-written") {
		t.Fatalf("prefix outside markers was lost: %q", s)
	}
	if !strings.Contains(s, "192.168.1.1 router.local") {
		t.Fatalf("suffix outside markers was lost: %q", s)
	}
	if !strings.Contains(s, BeginMarker) || !strings.Contains(s, EndMarker) {
		t.Fatalf("markers missing in output: %q", s)
	}
}

func TestApply_OnlyOneMarker_ReturnsMarkerMissing(t *testing.T) {
	dir := t.TempDir()
	hosts := filepath.Join(dir, "hosts")
	if err := os.WriteFile(hosts, []byte("127.0.0.1 localhost\n"+BeginMarker+"\n"), 0644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	r := &Renderer{
		HostsPath: hosts,
		BackupDir: filepath.Join(dir, "backup"),
	}
	if err := r.Apply("10.0.0.1 api.local\n"); !errors.Is(err, ErrMarkerMissing) {
		t.Fatalf("Apply() error = %v, want ErrMarkerMissing", err)
	}
}

func TestApply_CreatesBackupAndPrunes(t *testing.T) {
	dir := t.TempDir()
	hosts := filepath.Join(dir, "hosts")
	backupDir := filepath.Join(dir, "backup")
	if err := os.WriteFile(hosts, []byte(""), 0644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	r := &Renderer{
		HostsPath:   hosts,
		BackupDir:   backupDir,
		KeepBackups: 2,
	}
	for i := 0; i < 4; i++ {
		// Stagger timestamps so backup filenames sort deterministically.
		// Now must be set BEFORE Apply so writeBackup reads the new timestamp.
		step := i
		r.Now = func() time.Time { return time.Date(2026, 6, 14, 12, 0, step, 0, time.UTC) }
		if err := r.Apply(fmt.Sprintf("127.0.0.1 step%d.local\n", i)); err != nil {
			t.Fatalf("Apply(step %d) error = %v", i, err)
		}
	}

	entries, err := os.ReadDir(backupDir)
	if err != nil {
		t.Fatalf("ReadDir(backupDir) error = %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("len(backup entries) = %d, want 2 (KeepBackups=2)", len(entries))
	}
	// The two most-recent backups should remain.
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		names = append(names, e.Name())
	}
	sort.Strings(names)
	if !strings.HasPrefix(names[0], "hosts-20260614-120002") || !strings.HasPrefix(names[1], "hosts-20260614-120003") {
		t.Fatalf("kept backups = %v, want the two latest timestamps", names)
	}
}

func TestNewRenderer_Defaults(t *testing.T) {
	r := NewRenderer("/var/lib/auth-gate")
	if r.HostsPath != DefaultHostsPath {
		t.Fatalf("HostsPath = %q, want %q", r.HostsPath, DefaultHostsPath)
	}
	if r.BackupDir != filepath.Join("/var/lib/auth-gate", "hosts", "backup") {
		t.Fatalf("BackupDir = %q, want %q", r.BackupDir, filepath.Join("/var/lib/auth-gate", "hosts", "backup"))
	}
	if r.KeepBackups != 20 {
		t.Fatalf("KeepBackups = %d, want 20", r.KeepBackups)
	}
	if r.Now == nil {
		t.Fatal("Now = nil, want time.Now")
	}
}
