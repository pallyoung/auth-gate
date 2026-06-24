//go:build !windows

package main

import (
	"fmt"
	"os"
	"os/exec"
	"syscall"
	"time"
)

func newSysProcAttr() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{
		Setsid: true, // detach into a new session
	}
}

func isProcessAlive(pid int) bool {
	if pid <= 0 {
		return false
	}
	// Send signal 0 to check if the process exists.
	err := syscall.Kill(pid, 0)
	return err == nil
}

func findAndKillProcess(pid int) (*os.Process, error) {
	proc, err := os.FindProcess(pid)
	if err != nil {
		return nil, fmt.Errorf("find process %d: %w", pid, err)
	}
	if err := proc.Signal(syscall.SIGTERM); err != nil {
		return nil, fmt.Errorf("send SIGTERM to PID %d: %w", pid, err)
	}
	return proc, nil
}

func waitForExit(proc *os.Process, timeout time.Duration) error {
	done := make(chan error, 1)
	go func() {
		_, err := proc.Wait()
		done <- err
	}()

	select {
	case <-done:
		return nil
	case <-time.After(timeout):
		// Force kill after timeout.
		proc.Kill()
		<-done
		return fmt.Errorf("process did not exit within %s, killed", timeout)
	}
}

func launchBackground() error {
	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("resolve executable: %w", err)
	}

	args := []string{"start", "--foreground", "--data-dir", dataDir}
	cmd := exec.Command(exe, args...)
	cmd.SysProcAttr = newSysProcAttr()
	cmd.Env = os.Environ()
	cmd.Stdin = nil
	cmd.Stdout = nil
	cmd.Stderr = nil

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start background process: %w", err)
	}

	pid := cmd.Process.Pid

	if err := cmd.Process.Release(); err != nil {
		return fmt.Errorf("release background process: %w", err)
	}

	// Give the child a moment to initialize and write its own PID file.
	time.Sleep(300 * time.Millisecond)

	childPID, _ := readPID()
	if childPID == 0 {
		if err := ensureDataDirExists(); err != nil {
			return err
		}
		if err := os.WriteFile(pidFilePath(), []byte(fmt.Sprintf("%d", pid)), 0644); err != nil {
			return fmt.Errorf("write PID file: %w", err)
		}
	}

	return nil
}
