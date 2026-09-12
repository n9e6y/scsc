// internal/store/gob_store_test.go
package store

import (
	"math"
	"os"
	"path/filepath"
	"testing"

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
