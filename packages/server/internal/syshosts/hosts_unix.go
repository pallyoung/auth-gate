//go:build !windows

package syshosts

import "os"

const DefaultHostsPath = "/etc/hosts"

// checkWriteAccess verifies the process can write to the hosts file.
func checkWriteAccess(hostsPath string) error {
	f, err := os.OpenFile(hostsPath, os.O_WRONLY|os.O_APPEND, 0)
	if err != nil {
		return err
	}
	return f.Close()
}

// writeReplace is not needed on Unix — os.Rename works atomically.
func writeReplace(tmp, target string) error {
	return renameOrCopy(tmp, target)
}
