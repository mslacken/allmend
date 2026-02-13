package provider

import "context"

type ProviderConnection interface {
	GetModells(ctx context.Context) ([]string, error)
}
