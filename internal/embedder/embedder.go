package embedder

import "context"

// Embedder is responsible for converting text into vector embeddings.
type Embedder interface {
	// Embed takes a batch of strings and returns a batch of vectors.
	Embed(ctx context.Context, texts []string) ([][]float32, error)

	// Model returns the name of the model being used.
	Model() string

	// Dimensions returns the size of the vectors produced by this embedder.
	Dimensions() int
}
