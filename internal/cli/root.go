package cli

import (
	"github.com/spf13/cobra"
)

var (
	// Global flags available to all commands
	providerFlag string
	ollamaURL    string
	ollamaModel  string
)

// NewRootCmd initializes the base CLI application.
func NewRootCmd() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "semcode",
		Short: "A semantic code search CLI",
	}

	// Define global flags
	rootCmd.PersistentFlags().StringVarP(&providerFlag, "provider", "p", "placeholder", "Embedder provider: ollama, external, or placeholder")
	rootCmd.PersistentFlags().StringVar(&ollamaURL, "ollama-url", "http://localhost:11434", "URL for local Ollama instance")
	rootCmd.PersistentFlags().StringVar(&ollamaModel, "ollama-model", "nomic-embed-text", "Ollama embedding model to use")

	// Register subcommands
	rootCmd.AddCommand(newIndexCmd())
	rootCmd.AddCommand(newSearchCmd())

	return rootCmd
}
