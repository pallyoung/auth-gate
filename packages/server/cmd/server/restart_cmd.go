package main

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"
)

var restartCmd = &cobra.Command{
	Use:   "restart",
	Short: "Restart the Auth Gate server",
	Long:  "Stop the running Auth Gate server (if any) and start a new one in the background.",
	RunE: func(cmd *cobra.Command, args []string) error {
		pid, alive, err := cleanStalePID()
		if err != nil {
			return fmt.Errorf("check PID file: %w", err)
		}

		if alive {
			proc, err := findAndKillProcess(pid)
			if err != nil {
				return fmt.Errorf("stop process: %w", err)
			}

			fmt.Printf("Stopping Auth Gate (PID %d)...\n", pid)

			if err := waitForExit(proc, 10*time.Second); err != nil {
				return fmt.Errorf("process did not exit: %w", err)
			}

			if err := removePIDFile(); err != nil {
				return fmt.Errorf("remove PID file: %w", err)
			}

			fmt.Println("Auth Gate stopped")
		}

		if err := launchBackground(); err != nil {
			return fmt.Errorf("start background server: %w", err)
		}

		childPID, _ := readPID()
		if childPID > 0 {
			fmt.Printf("Auth Gate started (PID %d)\n", childPID)
		} else {
			fmt.Println("Auth Gate started")
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(restartCmd)
}
