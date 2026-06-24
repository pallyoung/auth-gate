//go:build windows

package syshosts

import (
	"fmt"
	"os"
	"path/filepath"

	"golang.org/x/sys/windows"
)

// DefaultHostsPath is the Windows hosts file location.
var DefaultHostsPath = filepath.Join(os.Getenv("SystemRoot"), "System32", "drivers", "etc", "hosts")

// checkWriteAccess verifies the process can write to the hosts file by
// attempting to open it. On Windows this also checks for administrator
// privileges via the Windows API.
func checkWriteAccess(hostsPath string) error {
	f, err := os.OpenFile(hostsPath, os.O_WRONLY|os.O_APPEND, 0)
	if err != nil {
		return describeWriteError(err, hostsPath)
	}
	return f.Close()
}

// isAdmin checks whether the current process is running with elevated
// (Administrator) privileges on Windows.
func isAdmin() bool {
	token, err := windows.OpenCurrentProcessToken()
	if err != nil {
		return false
	}
	defer token.Close()
	return token.IsElevated()
}

// describeWriteError turns a file-open error into a human-readable message
// that distinguishes between "not admin" and "file locked".
func describeWriteError(err error, hostsPath string) error {
	if os.IsPermission(err) {
		if !isAdmin() {
			return fmt.Errorf("%w: auth-gate must be run as Administrator to modify the hosts file", ErrPermissionDenied)
		}
		return fmt.Errorf("%w: cannot write to %s (file may be locked by another process such as DNS Client)", ErrPermissionDenied, hostsPath)
	}
	return err
}

// writeReplace on Windows tries os.Rename first. If that fails (file locked
// by DNS Client or antivirus), it falls back to truncating and writing the
// target file directly. Both approaches are tested for permission/lock errors.
func writeReplace(tmp, target string) error {
	if err := renameOrCopy(tmp, target); err == nil {
		return nil
	}

	data, err := os.ReadFile(tmp)
	if err != nil {
		return err
	}
	_ = os.Remove(tmp)

	f, err := os.OpenFile(target, os.O_WRONLY|os.O_TRUNC, 0)
	if err != nil {
		return describeWriteError(err, target)
	}
	if _, err := f.Write(data); err != nil {
		_ = f.Close()
		return fmt.Errorf("%w: write failed (file may be locked by another process): %v", ErrPermissionDenied, err)
	}
	if err := f.Sync(); err != nil {
		_ = f.Close()
		return fmt.Errorf("%w: sync failed: %v", ErrPermissionDenied, err)
	}
	return f.Close()
}
