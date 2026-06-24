package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show Auth Gate server status",
	Long:  "Display whether the Auth Gate server is running and its PID.",
	RunE: func(cmd *cobra.Command, args []string) error {
		pid, alive, err := cleanStalePID()
		if err != nil {
			return fmt.Errorf("check PID file: %w", err)
		}

		if alive {
			fmt.Printf("Auth Gate is running (PID %d)\n", pid)
		} else {
			fmt.Println("Auth Gate is not running")
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(statusCmd)
}
