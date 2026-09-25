package cli

import (
	"path/filepath"

	"github.com/spf13/cobra"

	"semcode/internal/embedder"
)

var (
	// Global flags available to all commands
	indexPath      string
	providerFlag   string
	ollamaURL      string
	ollamaModel    string
	externalURL    string
	externalModel  string
	externalKeyEnv string
)

// NewRootCmd initializes the base CLI application.
func NewRootCmd() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:           "semcode",
		Short:         "A semantic code search CLI",
		SilenceUsage:  true, // don't dump usage text on runtime errors
		SilenceErrors: true, // main prints the error once
	}

	// Define global flags
	flags := rootCmd.PersistentFlags()
	flags.StringVar(&indexPath, "index", "", "Path to the index file (default <repo>/.semcode/index.gob)")
	flags.StringVarP(&providerFlag, "provider", "p", embedder.ProviderPlaceholder, "Embedder provider: ollama, external, or placeholder")
	flags.StringVar(&ollamaURL, "ollama-url", "http://localhost:11434", "URL for local Ollama instance")
	flags.StringVar(&ollamaModel, "ollama-model", "nomic-embed-text", "Ollama embedding model to use")
	flags.StringVar(&externalURL, "external-url", "https://api.openai.com/v1", "Base URL of an OpenAI-compatible embeddings API")
	flags.StringVar(&externalModel, "external-model", "text-embedding-3-small", "Model for the external embeddings API")
	flags.StringVar(&externalKeyEnv, "external-key-env", "OPENAI_API_KEY", "Env var holding the external API key")

	// Register subcommands
	rootCmd.AddCommand(newIndexCmd())
	rootCmd.AddCommand(newSearchCmd())

	return rootCmd
}

// resolveIndexPath returns --index if set, otherwise <repoDir>/.semcode/index.gob.
// Both index and search go through here so they always agree on the location.
func resolveIndexPath(repoDir string) string {
	if indexPath != "" {
		return indexPath
	}
	return filepath.Join(repoDir, ".semcode", "index.gob")
}

// embedderOptions collects the provider settings from the global flags.
func embedderOptions() embedder.Options {
	return embedder.Options{
		OllamaURL:       ollamaURL,
		OllamaModel:     ollamaModel,
		ExternalBaseURL: externalURL,
		ExternalModel:   externalModel,
		ExternalKeyEnv:  externalKeyEnv,
	}
}
