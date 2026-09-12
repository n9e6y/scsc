package config

// Config represents the root configuration for the CLI.
type Config struct {
	Provider string         `yaml:"provider"`
	Ollama   OllamaConfig   `yaml:"ollama"`
	External ExternalConfig `yaml:"external"`
	Chunking ChunkingConfig `yaml:"chunking"`
	Index    IndexConfig    `yaml:"index"`
}

// OllamaConfig holds settings for the local Ollama provider.
type OllamaConfig struct {
	BaseURL string `yaml:"base_url"` // e.g., http://localhost:11434
	Model   string `yaml:"model"`    // e.g., nomic-embed-text
}

// ExternalConfig holds settings for an OpenAI-compatible HTTP provider.
type ExternalConfig struct {
	BaseURL   string `yaml:"base_url"`    // e.g., https://api.openai.com/v1
	APIKeyEnv string `yaml:"api_key_env"` // e.g., OPENAI_API_KEY
	Model     string `yaml:"model"`       // e.g., text-embedding-3-small
}

// ChunkingConfig holds settings for the AST/Window chunker.
type ChunkingConfig struct {
	MaxLines int `yaml:"max_lines"` // Fallback threshold for large AST nodes
}

// IndexConfig controls where and how the store is persisted.
type IndexConfig struct {
	Path string `yaml:"path"` // e.g., .semcode/index.gob
}
