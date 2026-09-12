package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"semcode/internal/chunker"
	"semcode/internal/embedder"
	"semcode/internal/embedder/external"
	"semcode/internal/embedder/ollama"
	"semcode/internal/embedder/placeholder"
	"semcode/internal/indexer"
	"semcode/internal/store"
)

func newIndexCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "index [directory]",
		Short: "Index a repository for semantic search",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			dir := "."
			if len(args) > 0 {
				dir = args[0]
			}

			// 1. Initialize the correct Embedder based on global flags (defined in root.go)
			var embedEngine embedder.Embedder
			var err error

			switch providerFlag {
			case "ollama":
				fmt.Printf("🧠 Using local Ollama model: %s\n", ollamaModel)
				embedEngine = ollama.New(ollamaURL, ollamaModel)
			case "external":
				fmt.Println("☁️ Using external API...")
				embedEngine, err = external.New("", "OPENAI_API_KEY", "text-embedding-3-small")
			default:
				fmt.Println("🧪 Using placeholder embedder...")
				embedEngine = placeholder.New()
			}
			if err != nil {
				return fmt.Errorf("failed to initialize embedder: %w", err)
			}

			// 2. Initialize Chunker and Store
			chunkEngine := chunker.NewSymbolChunker(100)
			dbStore := store.NewGobStore()

			// 3. Initialize and Run Orchestrator
			orchestrator := indexer.New(chunkEngine, embedEngine, dbStore)

			fmt.Printf("Starting indexing for %s using [%s] provider...\n", dir, providerFlag)
			if err := orchestrator.Run(cmd.Context(), dir); err != nil {
				return fmt.Errorf("indexing failed: %w", err)
			}

			return nil
		},
	}
}
