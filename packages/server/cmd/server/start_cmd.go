package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

var foreground bool

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Start the Auth Gate server",
	Long:  "Start the Auth Gate server. By default the server runs in the background. Use --foreground to run in the current terminal.",
	RunE: func(cmd *cobra.Command, args []string) error {
		if foreground {
			startForeground()
			return nil
		}

		// Background mode: check if already running.
		pid, alive, err := cleanStalePID()
		if err != nil {
			return fmt.Errorf("check PID file: %w", err)
		}
		if alive {
			return fmt.Errorf("Auth Gate is already running (PID %d)", pid)
		}

		if err := launchBackground(); err != nil {
			return fmt.Errorf("start background server: %w", err)
		}

		// Read back the PID that the child wrote (or we wrote via launchBackground).
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
	startCmd.Flags().BoolVarP(&foreground, "foreground", "f", false, "run in the foreground (do not detach)")
	rootCmd.AddCommand(startCmd)
}
