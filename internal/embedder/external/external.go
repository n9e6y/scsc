// File: internal/embedder/external/external.go
// Description: OpenAI-compatible HTTP client for remote embeddings (OpenAI, Voyage, etc).

package external

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"sync"
)

// ProviderName is the name this embedder is selected by.
const ProviderName = "external"

type ExternalEmbedder struct {
	baseURL string
	model   string
	apiKey  string
	client  *http.Client

	mu   sync.Mutex
	dims int // learned from the first successful response; 0 until then
}

// OpenAI-compatible payload structures
type embedRequest struct {
	Model string   `json:"model"`
	Input []string `json:"input"`
}

type embedResponse struct {
	Data []struct {
		Embedding []float32 `json:"embedding"`
		Index     int       `json:"index"`
	} `json:"data"`
}

func New(baseURL, apiKeyEnv, model string) (*ExternalEmbedder, error) {
	if baseURL == "" {
		baseURL = "https://api.openai.com/v1"
	}

	key := os.Getenv(apiKeyEnv)
	if key == "" {
		return nil, fmt.Errorf("missing API key in environment variable: %s", apiKeyEnv)
	}

	return &ExternalEmbedder{
		baseURL: baseURL,
		model:   model,
		apiKey:  key,
		client:  &http.Client{},
	}, nil
}

func (e *ExternalEmbedder) Provider() string {
	return ProviderName
}

func (e *ExternalEmbedder) Model() string {
	return e.model
}

// Dimensions returns the vector size seen in the first successful response.
// It is learned rather than hardcoded because it varies by model
// (1536 for text-embedding-3-small, 3072 for -large, 1024 for voyage-code-3, ...).
func (e *ExternalEmbedder) Dimensions() int {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.dims
}

// Embed sends a batch of texts to the OpenAI-compatible endpoint.
func (e *ExternalEmbedder) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	if len(texts) == 0 {
		return nil, nil
	}

	url := fmt.Sprintf("%s/embeddings", e.baseURL)

	reqBody := embedRequest{
		Model: e.model,
		Input: texts,
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
	req.Header.Set("Authorization", "Bearer "+e.apiKey)

	resp, err := e.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("external api request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("external api returned status: %s", resp.Status)
	}

	var resData embedResponse
	if err := json.NewDecoder(resp.Body).Decode(&resData); err != nil {
		return nil, fmt.Errorf("failed to decode external response: %w", err)
	}

	// OpenAI guarantees order matches input, but let's be safe and place by index
	results := make([][]float32, len(texts))
	for _, item := range resData.Data {
		if item.Index >= 0 && item.Index < len(results) {
			results[item.Index] = item.Embedding
		}
	}
	for i, vec := range results {
		if len(vec) == 0 {
			return nil, fmt.Errorf("external api returned no embedding for input %d", i)
		}
	}

	e.mu.Lock()
	if e.dims == 0 {
		e.dims = len(results[0])
	}
	e.mu.Unlock()

	return results, nil
}
