// File: internal/store/gob_store.go
// Description: In-memory store that persists to disk via encoding/gob and performs brute-force vector search.

package store

import (
	"encoding/gob"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"sync"
)

// GobStore implements the Store interface using an in-memory slice backed by a gob file.
type GobStore struct {
	mu      sync.RWMutex
	records []Record
}

// NewGobStore initializes an empty GobStore.
func NewGobStore() *GobStore {
	return &GobStore{
		records: make([]Record, 0),
	}
}

// Add appends new records to the store.
// In a fully robust version, we would check for chunk.Hash to update existing records,
// but for v1, we will simply append.
func (s *GobStore) Add(records []Record) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.records = append(s.records, records...)
	return nil
}

// Search performs brute-force cosine similarity across all stored records.
func (s *GobStore) Search(queryVec []float32, topK int) ([]ScoredRecord, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var results []ScoredRecord

	for _, rec := range s.records {
		score, err := cosineSimilarity(queryVec, rec.Vector)
		if err != nil {
			continue // skip mismatched dimensions
		}
		results = append(results, ScoredRecord{
			Record: rec,
			Score:  score,
		})
	}

	// Sort results descending by score
	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})

	if len(results) > topK {
		results = results[:topK]
	}

	return results, nil
}

// Save writes the entire in-memory store to a file using encoding/gob.
func (s *GobStore) Save(path string) error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Ensure the directory exists (.semcode/)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("failed to create store file: %w", err)
	}
	defer file.Close()

	encoder := gob.NewEncoder(file)
	if err := encoder.Encode(s.records); err != nil {
		return fmt.Errorf("failed to encode records: %w", err)
	}

	return nil
}

// Load reads the store from a gob file on disk.
func (s *GobStore) Load(path string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // Fresh start, no file exists yet
		}
		return fmt.Errorf("failed to open store file: %w", err)
	}
	defer file.Close()

	decoder := gob.NewDecoder(file)
	if err := decoder.Decode(&s.records); err != nil {
		return fmt.Errorf("failed to decode records: %w", err)
	}

	return nil
}

// cosineSimilarity calculates the angle distance between two vectors.
func cosineSimilarity(a, b []float32) (float32, error) {
	if len(a) != len(b) {
		return 0, fmt.Errorf("vector dimensions mismatch: %d != %d", len(a), len(b))
	}

	var dotProduct, normA, normB float32
	for i := range a {
		dotProduct += a[i] * b[i]
		normA += a[i] * a[i]
		normB += b[i] * b[i]
	}

	if normA == 0 || normB == 0 {
		return 0, nil
	}

	return dotProduct / (float32(math.Sqrt(float64(normA))) * float32(math.Sqrt(float64(normB)))), nil
}
