// Package syshosts rewrites a marker-protected region of a file (typically
// /etc/hosts) atomically. The renderer is dependency-free and has one job:
// splice the given content into the marker block, keep everything outside
// the block untouched, write a backup, and atomically replace the file.
package syshosts

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const (
	// BeginMarker marks the start of the managed region in the file.
	BeginMarker = "# BEGIN AUTH-GATE MANAGED BLOCK"
	// EndMarker marks the end of the managed region in the file.
	EndMarker = "# END AUTH-GATE MANAGED BLOCK"
)

// ErrMarkerMissing is returned when the file does not contain a valid managed
// marker block (both markers missing on a non-empty file, only one present,
// or the end marker appears before the begin marker). The renderer refuses to
// write a fresh block over hand-written content.
var ErrMarkerMissing = errors.New("syshosts: managed marker block missing or inconsistent")

// ErrPermissionDenied is returned when the process lacks write access to the
// hosts file (e.g. not running as Administrator on Windows).
var ErrPermissionDenied = errors.New("syshosts: permission denied")

// Renderer rewrites a marker-protected region of a file. It is safe for
// concurrent use only when Now is a function whose return value does not
// change across calls in a way that would produce identical backup
// filenames for distinct writes.
type Renderer struct {
	HostsPath   string
	BackupDir   string
	KeepBackups int
	// Now is injected for tests. nil falls back to time.Now.
	Now func() time.Time
}

// NewRenderer returns a Renderer with sensible defaults pointed at the given
// data directory. The backup directory is dataDir/hosts/backup and up to 20
// backups are retained.
func NewRenderer(dataDir string) *Renderer {
	return &Renderer{
		HostsPath:   DefaultHostsPath,
		BackupDir:   filepath.Join(dataDir, "hosts", "backup"),
		KeepBackups: 20,
		Now:         time.Now,
	}
}

// Apply reads the file at r.HostsPath, replaces the content between the
// managed markers with the given content, writes a timestamped backup of
// the original file, and atomically replaces the file. If the file does
// not contain a valid marker block and is non-empty, ErrMarkerMissing is
// returned and the file is not modified.
func (r *Renderer) Apply(content string) error {
	if r.HostsPath == "" {
		return errors.New("syshosts: HostsPath is empty")
	}

	if err := checkWriteAccess(r.HostsPath); err != nil {
		return fmt.Errorf("%w: %v", ErrPermissionDenied, err)
	}

	existing, err := os.ReadFile(r.HostsPath)
	if err != nil {
		return fmt.Errorf("read hosts file: %w", err)
	}

	prefix, suffix, err := splitAroundMarkers(existing)
	if err != nil {
		return err
	}

	var buf bytes.Buffer
	if len(prefix) > 0 {
		buf.Write(prefix)
		if !bytes.HasSuffix(prefix, []byte("\n")) {
			buf.WriteByte('\n')
		}
	}
	buf.WriteString(BeginMarker)
	buf.WriteByte('\n')
	buf.WriteString(strings.TrimRight(content, "\n"))
	buf.WriteByte('\n')
	buf.WriteString(EndMarker)
	buf.WriteByte('\n')
	if len(suffix) > 0 {
		buf.Write(suffix)
	}
	data := bytes.TrimRight(buf.Bytes(), " \t\n")
	data = append(data, '\n')

	if err := os.MkdirAll(r.BackupDir, 0755); err != nil {
		return fmt.Errorf("create backup dir: %w", err)
	}
	if err := r.writeBackup(existing); err != nil {
		return fmt.Errorf("write backup: %w", err)
	}
	if err := r.writeAtomic(data); err != nil {
		return fmt.Errorf("write hosts file: %w", err)
	}
	return nil
}

// splitAroundMarkers finds the begin and end markers in the existing file
// and returns the content before the begin marker (prefix) and the content
// after the end-marker line (suffix). The content between the markers is
// discarded by the caller.
//
// If both markers are missing and the file is empty (or whitespace only),
// it returns (nil, nil, nil) so a fresh write is allowed. If both markers
// are missing on a non-empty file, the entire file is returned as prefix
// so the markers are appended at the end without disturbing existing
// content. If only one marker is present, or the end marker appears before
// the begin marker, ErrMarkerMissing is returned.
func splitAroundMarkers(existing []byte) (prefix, suffix []byte, err error) {
	beginIdx := bytes.Index(existing, []byte(BeginMarker))
	endIdx := bytes.Index(existing, []byte(EndMarker))

	if beginIdx < 0 && endIdx < 0 {
		if len(bytes.TrimSpace(existing)) == 0 {
			return nil, nil, nil
		}
		// No markers on a non-empty file: treat the entire content as
		// prefix so the managed block is appended at the end.
		return existing, nil, nil
	}

	if beginIdx < 0 || endIdx < 0 {
		return nil, nil, ErrMarkerMissing
	}
	if endIdx < beginIdx {
		return nil, nil, ErrMarkerMissing
	}

	lineStart := bytes.LastIndexByte(existing[:beginIdx], '\n') + 1
	prefix = existing[:lineStart]

	endLineEnd := bytes.IndexByte(existing[endIdx:], '\n')
	if endLineEnd < 0 {
		suffix = nil
	} else {
		suffix = existing[endIdx+endLineEnd+1:]
	}
	return prefix, suffix, nil
}

// writeBackup writes the current file contents to a timestamped file in
// the backup directory and prunes old backups. A collision on the
// timestamped name is resolved by appending the PID.
func (r *Renderer) writeBackup(existing []byte) error {
	stamp := r.now().UTC().Format("20060102-150405")
	path := filepath.Join(r.BackupDir, "hosts-"+stamp+".bak")
	if _, err := os.Stat(path); err == nil {
		path = filepath.Join(r.BackupDir, fmt.Sprintf("hosts-%s-%d.bak", stamp, os.Getpid()))
	}
	if err := os.WriteFile(path, existing, 0644); err != nil {
		return err
	}
	return r.pruneBackups()
}

// pruneBackups removes the oldest hosts-*.bak files so that no more than
// KeepBackups remain. KeepBackups <= 0 disables pruning.
func (r *Renderer) pruneBackups() error {
	if r.KeepBackups <= 0 {
		return nil
	}
	entries, err := os.ReadDir(r.BackupDir)
	if err != nil {
		return err
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		if !e.IsDir() && strings.HasPrefix(e.Name(), "hosts-") {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names)
	for len(names) > r.KeepBackups {
		if err := os.Remove(filepath.Join(r.BackupDir, names[0])); err != nil {
			return err
		}
		names = names[1:]
	}
	return nil
}

// writeAtomic writes data to a temp file adjacent to HostsPath, fsyncs it,
// closes it, and replaces HostsPath. The actual replacement is delegated to
// writeReplace which handles platform-specific quirks (e.g. Windows file
// locking by the DNS Client service).
func (r *Renderer) writeAtomic(data []byte) error {
	tmp := r.HostsPath + ".tmp"
	f, err := os.OpenFile(tmp, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	if _, err := f.Write(data); err != nil {
		_ = f.Close()
		return err
	}
	if err := f.Sync(); err != nil {
		_ = f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	return writeReplace(tmp, r.HostsPath)
}

// renameOrCopy tries os.Rename first, falling back to a copy+remove if
// rename fails (e.g. cross-device or platform restrictions).
func renameOrCopy(tmp, target string) error {
	if err := os.Rename(tmp, target); err == nil {
		return nil
	}

	data, err := os.ReadFile(tmp)
	if err != nil {
		return err
	}
	_ = os.Remove(tmp)
	return os.WriteFile(target, data, 0644)
}

func (r *Renderer) now() time.Time {
	if r.Now != nil {
		return r.Now()
	}
	return time.Now()
}
