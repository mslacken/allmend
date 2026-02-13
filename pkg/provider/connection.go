package provider

import (
	"context"
	"fmt"

	"google.golang.org/genai"

	"github.com/SUSE/allmend/pkg/provider/gemini"
	"github.com/SUSE/allmend/pkg/provider/ollama"
)

// GetConnection creates a connection to the provider for management tasks.
func (p Provider) GetConnection(ctx context.Context) (ProviderConnection, error) {
	switch p.Type {
	case "ollama":
		endpoint := "http://localhost:11434"
		if v, ok := p.Config["endpoint"].(string); ok {
			endpoint = v
		}
		// We use a dummy model name because New requires it, but for listing models it might be ignored or we can use empty.
		// However, ollama.New returns *Provider which has the client.
		return ollama.New(endpoint, "")
	case "google", "gemini":
		cfg := &genai.ClientConfig{}

		if v, ok := p.Config["api_key"].(string); ok {
			cfg.APIKey = v
		}
		if v, ok := p.Config["project_id"].(string); ok {
			cfg.Project = v
		}
		if v, ok := p.Config["location"].(string); ok {
			cfg.Location = v
		}
		if v, ok := p.Config["backend"].(string); ok {
			if v == "vertex" {
				cfg.Backend = genai.BackendVertexAI
			} else {
				cfg.Backend = genai.BackendGeminiAPI
			}
		}

		return gemini.New(ctx, cfg)
	default:
		return nil, fmt.Errorf("unsupported provider type for connection: %s", p.Type)
	}
}
