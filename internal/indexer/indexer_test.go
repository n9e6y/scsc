package indexer

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"semcode/internal/chunker"
	"semcode/internal/embedder/placeholder"
	"semcode/internal/store"
)

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestRun_EndToEnd(t *testing.T) {
	repo := t.TempDir()
	writeFile(t, filepath.Join(repo, "db", "conn.go"), `package db

func OpenConnection(dsn string) error { return nil }

func CloseConnection() {}
`)
	writeFile(t, filepath.Join(repo, "web", "render.go"), `package web

type Page struct{ Title string }
`)
	indexPath := filepath.Join(t.TempDir(), "idx", "index.gob")

	emb := placeholder.New()
	idx := New(chunker.NewSymbolChunker(), emb, store.NewGobStore())
	if err := idx.Run(context.Background(), repo, indexPath); err != nil {
		t.Fatalf("Run: %v", err)
	}

	loaded := store.NewGobStore()
	if err := loaded.Load(indexPath); err != nil {
		t.Fatal(err)
	}

	meta := loaded.Meta()
	if meta.Provider != placeholder.ProviderName || meta.Model != emb.Model() ||
		meta.Dims != emb.Dimensions() || meta.ChunkerVersion != chunker.SymbolChunkerVersion || meta.IndexedAt.IsZero() {
		t.Fatalf("unexpected meta: %+v", meta)
	}

	q, _ := emb.Embed(context.Background(), []string{"OpenConnection dsn"})
	results, err := loaded.Search(q[0], 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 3 {
		t.Fatalf("expected 3 chunks (2 funcs + 1 type), got %d", len(results))
	}
	if top := results[0].Record.Chunk; top.SymbolName != "OpenConnection" {
		t.Errorf("top result = %s %q, want OpenConnection", top.FilePath, top.SymbolName)
	}
	for _, r := range results {
		if r.Record.Chunk.Hash == "" {
			t.Errorf("chunk %s has no hash", r.Record.Chunk.ID)
		}
	}
}

func TestRun_EmptyRepoIsAnError(t *testing.T) {
	repo := t.TempDir()
	writeFile(t, filepath.Join(repo, "README.md"), "# nothing to index\n")
	indexPath := filepath.Join(t.TempDir(), "index.gob")

	err := New(chunker.NewSymbolChunker(), placeholder.New(), store.NewGobStore()).
		Run(context.Background(), repo, indexPath)
	if err == nil || !strings.Contains(err.Error(), "no .go code") {
		t.Fatalf("expected a no-code error, got %v", err)
	}
	if _, statErr := os.Stat(indexPath); !os.IsNotExist(statErr) {
		t.Error("no index file should be written for an empty repo")
	}
}
