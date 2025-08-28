// Package cli provides the command-line interface for CortexGo
package cli

import (
	"os"
)

// Run starts the CLI application
func Run() {
	rootCmd := NewRootCmd()

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
