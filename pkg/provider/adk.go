package provider

import (
	"context"
	"fmt"

	"google.golang.org/adk/model"
	"google.golang.org/adk/model/gemini"
	"google.golang.org/genai"

	"github.com/SUSE/allmend/pkg/provider/ollama"
)

// CreateLLM creates an ADK LLM provider from the configuration.
func (p Provider) CreateLLM(ctx context.Context, modelName string) (model.LLM, error) {
	switch p.Type {
	case "ollama":
		endpoint := "http://localhost:11434"
		if v, ok := p.Config["endpoint"].(string); ok {
			endpoint = v
		}
		return ollama.New(endpoint, modelName)
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

		return gemini.NewModel(ctx, modelName, cfg)
	default:
		return nil, fmt.Errorf("unsupported provider type for ADK: %s", p.Type)
	}
}
