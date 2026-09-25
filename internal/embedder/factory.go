// File: internal/embedder/factory.go
// Description: Builds an Embedder from a provider name so every command constructs them the same way.

package embedder

import (
	"fmt"

	"semcode/internal/embedder/external"
	"semcode/internal/embedder/ollama"
	"semcode/internal/embedder/placeholder"
)

// Provider names accepted by New.
const (
	ProviderPlaceholder = placeholder.ProviderName
	ProviderOllama      = ollama.ProviderName
	ProviderExternal    = external.ProviderName
)

// Options carries provider-specific settings. Fields for other providers are ignored.
type Options struct {
	OllamaURL   string
	OllamaModel string

	ExternalBaseURL string // OpenAI-compatible base URL, e.g. https://api.openai.com/v1
	ExternalModel   string
	ExternalKeyEnv  string // name of the env var holding the API key
}

// New returns the Embedder for provider. Unknown providers are an error rather than
// a silent fallback, because an index built with the wrong embedder is useless.
func New(provider string, opts Options) (Embedder, error) {
	switch provider {
	case ProviderPlaceholder:
		return placeholder.New(), nil
	case ProviderOllama:
		return ollama.New(opts.OllamaURL, opts.OllamaModel), nil
	case ProviderExternal:
		e, err := external.New(opts.ExternalBaseURL, opts.ExternalKeyEnv, opts.ExternalModel)
		if err != nil {
			return nil, err
		}
		return e, nil
	default:
		return nil, fmt.Errorf("unknown embedding provider %q (want %s, %s or %s)",
			provider, ProviderPlaceholder, ProviderOllama, ProviderExternal)
	}
}
