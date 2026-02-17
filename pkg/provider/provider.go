package provider

import "context"

type ProviderConnection interface {
	GetModells(ctx context.Context) ([]string, error)
	// CheckModel checks if the model supports the given configuration.
	// It returns a list of warnings (for invalid config) and info messages (e.g. available valid params).
	CheckModel(ctx context.Context, name string, config map[string]interface{}) ([]string, []string, error)
}
