// File: internal/embedder/ollama/ollama.go
// Description: HTTP client for interacting with a local Ollama instance for embeddings.

package ollama

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
)

// maxConcurrentRequests bounds how many embedding requests we fire at Ollama
// at once. The legacy /api/embeddings endpoint only accepts one prompt per
// request, so without this, embedding a repo means one HTTP round trip per
// chunk, serially - slow on anything but a tiny codebase. Ollama can service
// several requests in flight, so we parallelize with a small worker pool
// instead of waiting on each request in turn.
const maxConcurrentRequests = 8

type OllamaEmbedder struct {
	baseURL string
	model   string
	client  *http.Client
}

type embedRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
}

type embedResponse struct {
	Embedding []float32 `json:"embedding"`
}

func New(baseURL, model string) *OllamaEmbedder {
	if baseURL == "" {
		baseURL = "http://localhost:11434"
	}
	if model == "" {
		model = "nomic-embed-text"
	}

	return &OllamaEmbedder{
		baseURL: baseURL,
		model:   model,
		client:  &http.Client{},
	}
}

// Model fulfills the Embedder interface.
func (o *OllamaEmbedder) Model() string {
	return o.model
}

// Dimensions provides a default for nomic-embed-text if known, otherwise 0.
func (o *OllamaEmbedder) Dimensions() int {
	return 768 // nomic-embed-text dimension
}

// Embed fans requests out across a small worker pool since the standard
// Ollama /api/embeddings endpoint handles one prompt per request.
func (o *OllamaEmbedder) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	url := fmt.Sprintf("%s/api/embeddings", o.baseURL)
	results := make([][]float32, len(texts))

	sem := make(chan struct{}, maxConcurrentRequests)
	var wg sync.WaitGroup
	errCh := make(chan error, len(texts))

	for i, text := range texts {
		wg.Add(1)
		go func(i int, text string) {
			defer wg.Done()

			sem <- struct{}{}
			defer func() { <-sem }()

			vec, err := o.embedOne(ctx, url, text)
			if err != nil {
				errCh <- err
				return
			}
			results[i] = vec
		}(i, text)
	}

	wg.Wait()
	close(errCh)

	if err := <-errCh; err != nil {
		return nil, err
	}

	return results, nil
}

func (o *OllamaEmbedder) embedOne(ctx context.Context, url, text string) ([]float32, error) {
	reqBody := embedRequest{
		Model:  o.model,
		Prompt: text,
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(bodyBytes))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := o.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("ollama request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ollama returned status: %s", resp.Status)
	}

	var resData embedResponse
	if err := json.NewDecoder(resp.Body).Decode(&resData); err != nil {
		return nil, fmt.Errorf("failed to decode ollama response: %w", err)
	}

	return resData.Embedding, nil
}
