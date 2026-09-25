package chunker

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
)

// Chunker versions are recorded in the index metadata so eval results can be traced
// back to the chunker that produced them. Bump one whenever that chunker's boundaries
// or chunk content change.
const (
	SymbolChunkerVersion = "ast-v1"
	WindowChunkerVersion = "window-v1"
)

// Symbol kinds assigned to chunks.
const (
	KindFunc   = "func"
	KindMethod = "method"
	KindType   = "type"
	KindConst  = "const"
	KindVar    = "var"
	KindWindow = "window" // fixed-window chunk with no single owning symbol
)

// Chunk represents a semantic block of code extracted from a file.
type Chunk struct {
	ID         string // filepath:startLine-endLine
	FilePath   string
	StartLine  int
	EndLine    int
	Content    string
	Hash       string // sha256(content), used for incremental re-indexing
	SymbolName string // e.g. "Walk", "GobStore.Search"; empty for plain windows
	SymbolKind string // one of the Kind* constants
}

// Chunker is responsible for splitting file content into semantic chunks.
type Chunker interface {
	// Chunk takes a file path and its byte content, and returns a slice of Chunks.
	Chunk(filePath string, content []byte) ([]Chunk, error)

	// Version identifies the chunking strategy (see the *ChunkerVersion constants).
	Version() string
}

// newChunk is the single place chunks are built, so IDs and hashes stay consistent
// across chunking strategies.
func newChunk(filePath string, startLine, endLine int, content, symbolName, symbolKind string) Chunk {
	sum := sha256.Sum256([]byte(content))
	return Chunk{
		ID:         fmt.Sprintf("%s:%d-%d", filePath, startLine, endLine),
		FilePath:   filePath,
		StartLine:  startLine,
		EndLine:    endLine,
		Content:    content,
		Hash:       hex.EncodeToString(sum[:]),
		SymbolName: symbolName,
		SymbolKind: symbolKind,
	}
}
