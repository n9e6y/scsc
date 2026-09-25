package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"semcode/internal/chunker"
	"semcode/internal/embedder"
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

			// 1. Initialize the Embedder based on global flags (defined in root.go)
			embedEngine, err := embedder.New(providerFlag, embedderOptions())
			if err != nil {
				return fmt.Errorf("failed to initialize embedder: %w", err)
			}
			fmt.Printf("Using %s embedder (model %s)\n", embedEngine.Provider(), embedEngine.Model())

			// 2. Initialize Chunker and Store
			chunkEngine := chunker.NewSymbolChunker(100)
			dbStore := store.NewGobStore()

			// 3. Initialize and Run Orchestrator
			orchestrator := indexer.New(chunkEngine, embedEngine, dbStore)

			if err := orchestrator.Run(cmd.Context(), dir, resolveIndexPath(dir)); err != nil {
				return fmt.Errorf("indexing failed: %w", err)
			}

			return nil
		},
	}
}
