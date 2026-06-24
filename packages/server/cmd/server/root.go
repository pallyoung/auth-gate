package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var dataDir string

var rootCmd = &cobra.Command{
	Use:   "auth-gate",
	Short: "Auth Gate — self-hosted API gateway with auth and web UI",
	Long:  "Auth Gate is a self-hosted API gateway that provides routing, authentication, and a visual management console.",
}

func init() {
	defaultDataDir := os.Getenv("AUTH_GATE_DATA_DIR")
	if defaultDataDir == "" {
		defaultDataDir = "data"
	}
	rootCmd.PersistentFlags().StringVar(&dataDir, "data-dir", defaultDataDir, "data directory for PID file, config, and store")
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
