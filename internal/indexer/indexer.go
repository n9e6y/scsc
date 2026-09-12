package indexer

import (
	"context"
	"fmt"

	"semcode/internal/chunker"
	"semcode/internal/embedder"
	"semcode/internal/store"
)

// Indexer coordinates the extraction, embedding, and storing of code.
type Indexer struct {
	chunker  chunker.Chunker
	embedder embedder.Embedder
	store    store.Store
}

// New creates a new Indexer with the provided dependencies.
func New(c chunker.Chunker, e embedder.Embedder, s store.Store) *Indexer {
	return &Indexer{
		chunker:  c,
		embedder: e,
		store:    s,
	}
}

// Run executes the indexing pipeline over a given directory.
func (i *Indexer) Run(ctx context.Context, dir string) error {
	fmt.Printf("Starting indexing pipeline for: %s\n", dir)

	// TODO: 1. Initialize walker to scan directory (respecting .gitignore)
	// TODO: 2. For each file, pass bytes to i.chunker.Chunk()
	// TODO: 3. Compare hashes with existing chunks in i.store to find what changed
	// TODO: 4. Batch changed chunks and send to i.embedder.Embed()
	// TODO: 5. Save resulting Records to i.store.Add()
	// TODO: 6. Call i.store.Save()

	return nil
}
