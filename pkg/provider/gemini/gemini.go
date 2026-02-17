package gemini

import (
	"context"
	"fmt"
	"strings"

	"google.golang.org/genai"
)

// Provider implements the provider.ProviderConnection interface for Gemini.
type Provider struct {
	client *genai.Client
}

// New creates a new Gemini provider connection.
func New(ctx context.Context, config *genai.ClientConfig) (*Provider, error) {
	client, err := genai.NewClient(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("failed to create gemini client: %w", err)
	}
	return &Provider{
		client: client,
	}, nil
}

// GetModells returns a list of available models.
func (p *Provider) GetModells(ctx context.Context) ([]string, error) {
	var models []string
	for m, err := range p.client.Models.All(ctx) {
		if err != nil {
			return nil, fmt.Errorf("failed to iterate gemini models: %w", err)
		}
		
		name := m.Name
		if strings.HasPrefix(name, "models/") {
			name = strings.TrimPrefix(name, "models/")
		}
		models = append(models, name)
	}
	return models, nil
}

// CheckModel checks if the model supports the given configuration.
func (p *Provider) CheckModel(ctx context.Context, name string, config map[string]interface{}) ([]string, []string, error) {
	// For Gemini, we could fetch model details and check supported actions/features.
	// For now, return empty warnings.
	return nil, nil, nil
}

// GetSupportedOptions returns a list of supported configuration options for the model.
func (p *Provider) GetSupportedOptions(ctx context.Context, name string) ([]string, error) {
	// TODO: Implement for Gemini if possible
	return nil, nil
}
