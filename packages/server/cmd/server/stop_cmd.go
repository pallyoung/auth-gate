package main

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"
)

var stopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop the Auth Gate server",
	Long:  "Gracefully stop the running Auth Gate server.",
	RunE: func(cmd *cobra.Command, args []string) error {
		pid, alive, err := cleanStalePID()
		if err != nil {
			return fmt.Errorf("check PID file: %w", err)
		}
		if !alive {
			return fmt.Errorf("Auth Gate is not running")
		}

		proc, err := findAndKillProcess(pid)
		if err != nil {
			return fmt.Errorf("stop process: %w", err)
		}

		fmt.Printf("Stopping Auth Gate (PID %d)...\n", pid)

		// Wait for the process to exit.
		if err := waitForExit(proc, 10*time.Second); err != nil {
			return fmt.Errorf("process did not exit: %w", err)
		}

		// Clean up PID file.
		if err := removePIDFile(); err != nil {
			return fmt.Errorf("remove PID file: %w", err)
		}

		fmt.Println("Auth Gate stopped")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(stopCmd)
}
