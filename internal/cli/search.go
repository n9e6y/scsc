package cli

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/spf13/cobra"

	"semcode/internal/embedder"
	"semcode/internal/embedder/external"
	"semcode/internal/embedder/ollama"
	"semcode/internal/embedder/placeholder"
	"semcode/internal/store"
)

func newSearchCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "search [query]",
		Short: "Search the indexed repository using natural language",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			query := args[0]
			dbPath := filepath.Join(".", ".semcode", "index.gob")

			dbStore := store.NewGobStore()
			if err := dbStore.Load(dbPath); err != nil {
				return fmt.Errorf("could not load index (did you run 'semcode index'?): %w", err)
			}

			var embedEngine embedder.Embedder
			var err error

			switch providerFlag {
			case "ollama":
				embedEngine = ollama.New(ollamaURL, ollamaModel)
			case "external":
				embedEngine, err = external.New("", "OPENAI_API_KEY", "text-embedding-3-small")
			default:
				embedEngine = placeholder.New()
			}
			if err != nil {
				return err
			}

			queryVecs, err := embedEngine.Embed(context.Background(), []string{query})
			if err != nil {
				return fmt.Errorf("failed to embed query: %w", err)
			}

			results, err := dbStore.Search(queryVecs[0], 5) // Top 5 results
			if err != nil {
				return err
			}

			fmt.Printf("\n🔍 Search Results for: %q\n", query)
			fmt.Println("--------------------------------------------------")

			if len(results) == 0 {
				fmt.Println("No matches found.")
				return nil
			}

			for i, res := range results {
				fmt.Printf("[%d] %s (Lines %d-%d) - Score: %.3f\n",
					i+1, res.Record.Chunk.FilePath,
					res.Record.Chunk.StartLine, res.Record.Chunk.EndLine,
					res.Score)
			}

			return nil
		},
	}
}
