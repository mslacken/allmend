package gemini

import (
	"context"
	"fmt"
	"strings"

	"google.golang.org/api/iterator"
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
	it, err := p.client.Models.List(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to list gemini models: %w", err)
	}
	
	var models []string
	for {
		// New genai client iterators take ctx in Next()
		m, err := it.Next(ctx)
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to iterate gemini models: %w", err)
		}
		// Typically model names are like "models/gemini-pro"
		name := m.Name
		if strings.HasPrefix(name, "models/") {
			name = strings.TrimPrefix(name, "models/")
		}
		models = append(models, name)
	}
	return models, nil
}
