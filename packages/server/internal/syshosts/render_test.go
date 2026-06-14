package syshosts

import (
	"errors"
	"os"
	"path/filepath"
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

func TestApply_NonEmptyFileWithoutMarkers_ReturnsMarkerMissing(t *testing.T) {
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
	}
	err := r.Apply("10.0.0.1 api.local\n")
	if !errors.Is(err, ErrMarkerMissing) {
		t.Fatalf("Apply() error = %v, want ErrMarkerMissing", err)
	}

	// File must be untouched on error.
	got, readErr := os.ReadFile(hosts)
	if readErr != nil {
		t.Fatalf("ReadFile() error = %v", readErr)
	}
	if string(got) != original {
		t.Fatalf("file content changed despite error: got %q, want %q", string(got), original)
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
