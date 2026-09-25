package embedder

import "context"

// Embedder is responsible for converting text into vector embeddings.
type Embedder interface {
	// Embed takes a batch of strings and returns a batch of vectors.
	Embed(ctx context.Context, texts []string) ([][]float32, error)

	// Provider returns the provider name ("placeholder", "ollama", "external").
	Provider() string

	// Model returns the name of the model being used.
	Model() string

	// Dimensions returns the size of the vectors produced by this embedder.
	// Remote embedders only know this after their first successful Embed call
	// and return 0 before then.
	Dimensions() int
}
