package provider

import (
	"fmt"
	"os"
	"sort"

	"gopkg.in/yaml.v3"
)

// Provider represents the configuration for a single provider.
type Provider struct {
	Name   string                 `yaml:"name,omitempty"`
	Type   string                 `yaml:"type,omitempty"`
	Config map[string]interface{} `yaml:"config,omitempty"`
	// Add other common fields if necessary
}

// Store represents a collection of provider configurations.
type Store struct {
	Items map[string]Provider `yaml:",inline"`
	// Path is the file path where the providers are stored.
	Path string `yaml:"-"`
}

// Load reads provider configurations from the specified file.
func Load(path string) (*Store, error) {
	store := &Store{
		Items: make(map[string]Provider),
		Path:  path,
	}

	if _, err := os.Stat(path); os.IsNotExist(err) {
		return store, nil
	}

	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open providers file %s: %w", path, err)
	}
	defer f.Close()

	if err := yaml.NewDecoder(f).Decode(&store.Items); err != nil {
		return nil, fmt.Errorf("failed to decode providers from %s: %w", path, err)
	}

	// Ensure the name field is set to the key if it's empty
	for k, v := range store.Items {
		if v.Name == "" {
			v.Name = k
			store.Items[k] = v
		}
	}

	return store, nil
}

// Save writes the collection of provider configurations to the specified file.
func (s *Store) Save() error {
	if s.Path == "" {
		return fmt.Errorf("no path specified for provider store")
	}

	f, err := os.Create(s.Path)
	if err != nil {
		return fmt.Errorf("failed to create providers file %s: %w", s.Path, err)
	}
	defer f.Close()

	enc := yaml.NewEncoder(f)
	enc.SetIndent(2)
	if err := enc.Encode(s.Items); err != nil {
		return fmt.Errorf("failed to encode providers to %s: %w", s.Path, err)
	}
	return nil
}

// List returns a sorted slice of provider configurations.
func (s *Store) List() []Provider {
	keys := make([]string, 0, len(s.Items))
	for k := range s.Items {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	configs := make([]Provider, 0, len(keys))
	for _, k := range keys {
		configs = append(configs, s.Items[k])
	}
	return configs
}
