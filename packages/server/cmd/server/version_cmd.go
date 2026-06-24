package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

// version is set at build time via:
//
//	go build -ldflags "-X main.version=1.2.3" ./cmd/server
var version = "dev"

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the Auth Gate version",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("auth-gate %s\n", version)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
