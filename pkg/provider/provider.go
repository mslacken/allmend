package provider

// Interface defines the interface for AI model providers.
type Interface interface {
	// Name returns the unique name of the provider
	Name() string
}
