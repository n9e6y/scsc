// File: internal/embedder/placeholder/placeholder.go
// Description: A deterministic bag-of-words hash embedder for testing the pipeline locally without APIs.

package placeholder

import (
	"context"
	"hash/fnv"
	"math"
	"strings"
)

// PlaceholderEmbedder implements embedder.Embedder.
type PlaceholderEmbedder struct {
	dimensions int
}

// New creates a new PlaceholderEmbedder with standard dimensions.
func New() *PlaceholderEmbedder {
	return &PlaceholderEmbedder{
		dimensions: 256, // Keep it small for fast local testing
	}
}

// Model returns the mock model name.
func (p *PlaceholderEmbedder) Model() string {
	return "placeholder-bag-of-words-v1"
}

// Dimensions returns the configured vector size.
func (p *PlaceholderEmbedder) Dimensions() int {
	return p.dimensions
}

// Embed takes a batch of strings, tokenizes them by whitespace, hashes the tokens,
// and creates an L2-normalized frequency vector.
func (p *PlaceholderEmbedder) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	results := make([][]float32, len(texts))

	for i, text := range texts {
		// Create an empty vector
		vector := make([]float32, p.dimensions)

		// Very basic tokenization
		tokens := strings.Fields(strings.ToLower(text))

		// Hash tokens into buckets
		for _, token := range tokens {
			h := fnv.New32a()
			h.Write([]byte(token))
			bucket := h.Sum32() % uint32(p.dimensions)
			vector[bucket] += 1.0 // Accumulate frequency
		}

		// L2 Normalize the vector so cosine similarity works correctly
		results[i] = normalize(vector)
	}

	return results, nil
}

// normalize scales the vector so its length (magnitude) is 1.
func normalize(v []float32) []float32 {
	var sumSquares float32
	for _, val := range v {
		sumSquares += val * val
	}

	if sumSquares == 0 {
		return v
	}

	magnitude := float32(math.Sqrt(float64(sumSquares)))
	for i := range v {
		v[i] = v[i] / magnitude
	}

	return v
}
