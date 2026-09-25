// File: internal/store/gobStore.go
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

// gobFormatVersion is bumped whenever the on-disk layout of gobFile changes.
const gobFormatVersion = 1

// gobFile is the on-disk envelope. Versioning it lets Load reject stale indexes
// with a clear message instead of a gob decoding error.
type gobFile struct {
	FormatVersion int
	Meta          IndexMeta
	Records       []Record
}

// GobStore implements the Store interface using an in-memory slice backed by a gob file.
type GobStore struct {
	mu      sync.RWMutex
	meta    IndexMeta
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

// SetMeta records how the index was built.
func (s *GobStore) SetMeta(meta IndexMeta) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.meta = meta
}

// Meta returns how the index was built.
func (s *GobStore) Meta() IndexMeta {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.meta
}

// Search performs brute-force cosine similarity across all stored records.
// A dimension mismatch is an error: it means the query was embedded with a different
// model than the index, and silently skipping records would just return nothing.
func (s *GobStore) Search(queryVec []float32, topK int) ([]ScoredRecord, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var results []ScoredRecord

	for _, rec := range s.records {
		score, err := cosineSimilarity(queryVec, rec.Vector)
		if err != nil {
			return nil, fmt.Errorf("record %s: %w", rec.Chunk.ID, err)
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
	payload := gobFile{FormatVersion: gobFormatVersion, Meta: s.meta, Records: s.records}
	if err := encoder.Encode(payload); err != nil {
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

	var payload gobFile
	if err := gob.NewDecoder(file).Decode(&payload); err != nil || payload.FormatVersion != gobFormatVersion {
		return fmt.Errorf("index at %s is outdated or corrupt, re-run 'semcode index'", path)
	}
	s.meta = payload.Meta
	s.records = payload.Records

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
