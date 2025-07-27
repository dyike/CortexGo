package main

import (
	"fmt"
	"os"

	"github.com/dyike/CortexGo/internal/cli"
)

func main() {
	// Execute the root command
	rootCmd := cli.NewRootCmd()
	
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}