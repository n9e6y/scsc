package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"semcode/internal/embedder"
	"semcode/internal/store"
)

func newSearchCmd() *cobra.Command {
	var (
		repoDir string
		topK    int
	)

	cmd := &cobra.Command{
		Use:   "search [query]",
		Short: "Search the indexed repository using natural language",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			query := args[0]
			dbPath := resolveIndexPath(repoDir)

			dbStore := store.NewGobStore()
			if err := dbStore.Load(dbPath); err != nil {
				return err
			}
			meta := dbStore.Meta()
			if meta.Provider == "" {
				return fmt.Errorf("no index found at %s (run 'semcode index %s' first)", dbPath, repoDir)
			}

			// Without an explicit -p, use whatever the index was built with.
			opts := embedderOptions()
			provider := providerFlag
			if !cmd.Flags().Changed("provider") {
				provider = meta.Provider
				switch provider {
				case embedder.ProviderOllama:
					opts.OllamaModel = meta.Model
				case embedder.ProviderExternal:
					opts.ExternalModel = meta.Model
				}
			}

			embedEngine, err := embedder.New(provider, opts)
			if err != nil {
				return err
			}
			if embedEngine.Provider() != meta.Provider || embedEngine.Model() != meta.Model {
				return fmt.Errorf("index was built with %s/%s (%dd) but search is using %s/%s; re-run with %s",
					meta.Provider, meta.Model, meta.Dims,
					embedEngine.Provider(), embedEngine.Model(), flagsFor(meta))
			}

			queryVecs, err := embedEngine.Embed(cmd.Context(), []string{query})
			if err != nil {
				return fmt.Errorf("failed to embed query: %w", err)
			}
			if got := len(queryVecs[0]); got != meta.Dims {
				return fmt.Errorf("query vector has %d dims but the index has %d; was the model changed on the server?", got, meta.Dims)
			}

			results, err := dbStore.Search(queryVecs[0], topK)
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
				c := res.Record.Chunk
				fmt.Printf("[%d] %s (Lines %d-%d) %s %s - Score: %.3f\n",
					i+1, c.FilePath, c.StartLine, c.EndLine, c.SymbolKind, c.SymbolName, res.Score)
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&repoDir, "dir", ".", "Repository directory that was indexed")
	cmd.Flags().IntVarP(&topK, "k", "k", 5, "Number of results to return")

	return cmd
}

// flagsFor suggests the flags that reproduce the embedder an index was built with.
func flagsFor(meta store.IndexMeta) string {
	switch meta.Provider {
	case embedder.ProviderOllama:
		return fmt.Sprintf("-p ollama --ollama-model %s", meta.Model)
	case embedder.ProviderExternal:
		return fmt.Sprintf("-p external --external-model %s", meta.Model)
	default:
		return "-p " + meta.Provider
	}
}
