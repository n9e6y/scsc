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
)

type ExternalEmbedder struct {
	baseURL string
	model   string
	apiKey  string
	client  *http.Client
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

func (e *ExternalEmbedder) Model() string {
	return e.model
}

func (e *ExternalEmbedder) Dimensions() int {
	return 1536 // default text-embedding-3-small dimension
}

// Embed sends a batch of texts to the OpenAI-compatible endpoint.
func (e *ExternalEmbedder) Embed(ctx context.Context, texts []string) ([][]float32, error) {
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

	// OpenAI guarantees order matches input, but let's be safe and pre-allocate
	results := make([][]float32, len(texts))
	for _, item := range resData.Data {
		if item.Index < len(results) {
			results[item.Index] = item.Embedding
		}
	}

	return results, nil
}
