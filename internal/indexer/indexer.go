package indexer

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"semcode/internal/chunker"
	"semcode/internal/embedder"
	"semcode/internal/store"
	"semcode/internal/walker"
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

	// 1. Walk the directory to find target files (e.g., just .go files for now)
	files, err := walker.Walk(dir, []string{".go"})
	if err != nil {
		return fmt.Errorf("failed to walk directory: %w", err)
	}
	fmt.Printf("Found %d files to process.\n", len(files))

	var allChunks []chunker.Chunk

	// 2. Extract semantic chunks from each file
	for _, file := range files {
		content, err := os.ReadFile(file)
		if err != nil {
			fmt.Printf("Warning: failed to read %s: %v\n", file, err)
			continue
		}

		chunks, err := i.chunker.Chunk(file, content)
		if err != nil {
			fmt.Printf("Warning: failed to chunk %s: %v\n", file, err)
			continue
		}

		allChunks = append(allChunks, chunks...)
	}

	fmt.Printf("Extracted %d chunks in total. Embedding now...\n", len(allChunks))

	// 3. Batch process embeddings (Batch size of 100 to avoid API limits)
	batchSize := 100
	var finalRecords []store.Record

	for j := 0; j < len(allChunks); j += batchSize {
		end := j + batchSize
		if end > len(allChunks) {
			end = len(allChunks)
		}

		batch := allChunks[j:end]

		// Extract raw text for the embedder
		var texts []string
		for _, c := range batch {
			// We embed the file path with the content to give the model context
			texts = append(texts, fmt.Sprintf("File: %s\n%s", c.FilePath, c.Content))
		}

		// 4. Send to Embedder
		vectors, err := i.embedder.Embed(ctx, texts)
		if err != nil {
			return fmt.Errorf("failed to embed batch: %w", err)
		}

		// 5. Combine chunks and vectors into Store Records
		for k, vec := range vectors {
			finalRecords = append(finalRecords, store.Record{
				Chunk:  batch[k],
				Vector: vec,
			})
		}

		fmt.Printf("Embedded %d/%d chunks...\n", end, len(allChunks))
	}

	// 6. Save to Store
	if err := i.store.Add(finalRecords); err != nil {
		return fmt.Errorf("failed to add records to store: %w", err)
	}

	// Create default path if it doesn't exist
	dbPath := filepath.Join(dir, ".semcode", "index.gob")
	if err := i.store.Save(dbPath); err != nil {
		return fmt.Errorf("failed to save store to disk: %w", err)
	}

	fmt.Printf("Successfully indexed repo to %s\n", dbPath)
	return nil
}
