package main

import (
	"fmt"
	"os"

	"semcode/internal/cli"
)

func main() {
	// We delegate all the setup, flags, and subcommands to the cli package.
	rootCmd := cli.NewRootCmd()

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
