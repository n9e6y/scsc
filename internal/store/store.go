package store

import (
	"time"

	"semcode/internal/chunker"
)

// Record combines a code Chunk with its vector representation.
type Record struct {
	Chunk  chunker.Chunk
	Vector []float32
}

// ScoredRecord represents a search result with its similarity score.
type ScoredRecord struct {
	Record Record
	Score  float32 // Cosine similarity score (closer to 1.0 is better)
}

// IndexMeta records how an index was built. Query vectors are only comparable to
// document vectors from the same embedding model, so search checks this first.
type IndexMeta struct {
	Provider       string // embedder provider, e.g. "ollama"
	Model          string // embedding model, e.g. "nomic-embed-text"
	Dims           int    // vector length actually stored
	ChunkerVersion string // chunker.Version at index time
	IndexedAt      time.Time
}

// Store handles the persistence and retrieval of embedded chunks.
type Store interface {
	// Add inserts or updates records in the store.
	Add(records []Record) error

	// Search finds the topK most similar records to the query vector.
	Search(queryVec []float32, topK int) ([]ScoredRecord, error)

	// SetMeta records how the index was built.
	SetMeta(meta IndexMeta)

	// Meta returns how the index was built (zero value if unknown).
	Meta() IndexMeta

	// Save persists the store to disk (e.g., as a .gob file).
	Save(path string) error

	// Load retrieves the store from disk.
	Load(path string) error
}
