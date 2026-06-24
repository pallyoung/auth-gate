package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

const pidFileName = "auth-gate.pid"

func pidFilePath() string {
	return filepath.Join(dataDir, pidFileName)
}

func ensureDataDirExists() error {
	return os.MkdirAll(dataDir, 0755)
}

// writePIDFile writes the current process PID to the PID file.
func writePIDFile() error {
	if err := ensureDataDirExists(); err != nil {
		return fmt.Errorf("create data dir: %w", err)
	}
	return os.WriteFile(pidFilePath(), []byte(strconv.Itoa(os.Getpid())), 0644)
}

// removePIDFile removes the PID file if it exists.
func removePIDFile() error {
	err := os.Remove(pidFilePath())
	if os.IsNotExist(err) {
		return nil
	}
	return err
}

// readPID reads the PID from the PID file. Returns 0 if the file doesn't exist.
func readPID() (int, error) {
	data, err := os.ReadFile(pidFilePath())
	if os.IsNotExist(err) {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		return 0, fmt.Errorf("invalid PID file content: %w", err)
	}
	return pid, nil
}

// cleanStalePID removes the PID file if the recorded process is no longer running.
// Returns the PID (0 if no valid PID) and whether the process is alive.
func cleanStalePID() (int, bool, error) {
	pid, err := readPID()
	if err != nil {
		return 0, false, err
	}
	if pid == 0 {
		return 0, false, nil
	}
	if isProcessAlive(pid) {
		return pid, true, nil
	}
	// Stale PID file — clean it up.
	if err := removePIDFile(); err != nil {
		return 0, false, fmt.Errorf("remove stale PID file: %w", err)
	}
	return 0, false, nil
}
