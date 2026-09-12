package chunker

// Chunk represents a semantic block of code extracted from a file.
type Chunk struct {
	ID        string // sha256(filepath + startLine + endLine)
	FilePath  string
	StartLine int
	EndLine   int
	Content   string
	Hash      string // sha256(content), used for incremental re-indexing
}

// Chunker is responsible for splitting file content into semantic chunks.
type Chunker interface {
	// Chunk takes a file path and its byte content, and returns a slice of Chunks.
	Chunk(filePath string, content []byte) ([]Chunk, error)
}
