// File: internal/store/gobStore_test.go
package store

import (
	"encoding/gob"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"semcode/internal/chunker"
)

// Helper to check if two floats are "close enough" (handles floating point drift)
func almostEqual(a, b, tolerance float32) bool {
	return float32(math.Abs(float64(a-b))) <= tolerance
}

func TestGobStore_EndToEnd(t *testing.T) {
	// 1. Create a temporary directory for our test index
	tmpDir, err := os.MkdirTemp("", "semcode-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	indexPath := filepath.Join(tmpDir, "index.gob")
	store := NewGobStore()

	// 2. Define some mock records with known vectors.
	// Vector A points straight along the X-axis.
	// Vector B points straight along the Y-axis.
	// Vector C points slightly off the X-axis (very similar to A, totally different from B).
	records := []Record{
		{
			Chunk:  chunker.Chunk{ID: "chunk-a", FilePath: "auth.go", Content: "func Login()"},
			Vector: []float32{1.0, 0.0, 0.0},
		},
		{
			Chunk:  chunker.Chunk{ID: "chunk-b", FilePath: "db.go", Content: "func Connect()"},
			Vector: []float32{0.0, 1.0, 0.0},
		},
		{
			Chunk:  chunker.Chunk{ID: "chunk-c", FilePath: "auth_retry.go", Content: "func RetryLogin()"},
			Vector: []float32{0.9, 0.1, 0.0},
		},
	}

	// 3. Add and Save
	if err := store.Add(records); err != nil {
		t.Fatalf("Failed to add records: %v", err)
	}
	if err := store.Save(indexPath); err != nil {
		t.Fatalf("Failed to save store: %v", err)
	}

	// 4. Create a BRAND NEW store to prove loading works
	loadedStore := NewGobStore()
	if err := loadedStore.Load(indexPath); err != nil {
		t.Fatalf("Failed to load store: %v", err)
	}

	// 5. Query time!
	// Our query vector points straight at the X-axis (identical to chunk-a)
	query := []float32{1.0, 0.0, 0.0}

	results, err := loadedStore.Search(query, 3)
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}

	// 6. Assertions
	if len(results) != 3 {
		t.Fatalf("Expected 3 results, got %d", len(results))
	}

	// Rank 1 should be chunk-a (Identical vector, score ~ 1.0)
	if results[0].Record.Chunk.ID != "chunk-a" {
		t.Errorf("Expected chunk-a to be rank 1, got %s", results[0].Record.Chunk.ID)
	}
	if !almostEqual(results[0].Score, 1.0, 0.001) {
		t.Errorf("Expected score ~ 1.0, got %f", results[0].Score)
	}

	// Rank 2 should be chunk-c (Very similar vector, score ~ 0.9)
	if results[1].Record.Chunk.ID != "chunk-c" {
		t.Errorf("Expected chunk-c to be rank 2, got %s", results[1].Record.Chunk.ID)
	}

	// Rank 3 should be chunk-b (Orthogonal vector, score ~ 0.0)
	if results[2].Record.Chunk.ID != "chunk-b" {
		t.Errorf("Expected chunk-b to be rank 3, got %s", results[2].Record.Chunk.ID)
	}
	if !almostEqual(results[2].Score, 0.0, 0.001) {
		t.Errorf("Expected score ~ 0.0, got %f", results[2].Score)
	}
}

func TestGobStore_MetaRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "index.gob")
	meta := IndexMeta{
		Provider:       "ollama",
		Model:          "nomic-embed-text",
		Dims:           3,
		ChunkerVersion: "ast-v1",
		IndexedAt:      time.Date(2026, 9, 25, 12, 0, 0, 0, time.UTC),
	}

	s := NewGobStore()
	s.SetMeta(meta)
	if err := s.Add([]Record{{Chunk: chunker.Chunk{ID: "a"}, Vector: []float32{1, 0, 0}}}); err != nil {
		t.Fatal(err)
	}
	if err := s.Save(path); err != nil {
		t.Fatal(err)
	}

	loaded := NewGobStore()
	if err := loaded.Load(path); err != nil {
		t.Fatal(err)
	}
	if got := loaded.Meta(); !got.IndexedAt.Equal(meta.IndexedAt) || got.Provider != meta.Provider ||
		got.Model != meta.Model || got.Dims != meta.Dims || got.ChunkerVersion != meta.ChunkerVersion {
		t.Fatalf("meta round-trip: got %+v, want %+v", got, meta)
	}
}

func TestGobStore_DimensionMismatchIsAnError(t *testing.T) {
	s := NewGobStore()
	_ = s.Add([]Record{{Chunk: chunker.Chunk{ID: "a"}, Vector: []float32{1, 0, 0}}})

	if _, err := s.Search([]float32{1, 0}, 5); err == nil {
		t.Fatal("expected an error when the query has different dimensions than the index")
	}
}

func TestGobStore_LoadMissingFileIsEmpty(t *testing.T) {
	s := NewGobStore()
	if err := s.Load(filepath.Join(t.TempDir(), "nope.gob")); err != nil {
		t.Fatalf("missing file should load as empty, got %v", err)
	}
	if s.Meta().Provider != "" {
		t.Fatal("missing file should have empty meta")
	}
}

func TestGobStore_LoadOutdatedFormat(t *testing.T) {
	path := filepath.Join(t.TempDir(), "index.gob")
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	// The pre-versioning format was a bare []Record.
	if err := gob.NewEncoder(f).Encode([]Record{{Chunk: chunker.Chunk{ID: "old"}, Vector: []float32{1}}}); err != nil {
		t.Fatal(err)
	}
	f.Close()

	err = NewGobStore().Load(path)
	if err == nil || !strings.Contains(err.Error(), "re-run 'semcode index'") {
		t.Fatalf("expected an outdated-index error, got %v", err)
	}
}
