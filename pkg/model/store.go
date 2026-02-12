package model

import (
	"fmt"
	"os"
	"sort"

	"gopkg.in/yaml.v3"
)

// Load reads models from the specified YAML file.
func Load(path string) (*Store, error) {
	store := &Store{
		Items: make(map[string]Model),
		Path:  path,
	}

	if _, err := os.Stat(path); os.IsNotExist(err) {
		return store, nil
	}

	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open models file %s: %w", path, err)
	}
	defer f.Close()

	if err := yaml.NewDecoder(f).Decode(&store.Items); err != nil {
		return nil, fmt.Errorf("failed to decode models from %s: %w", path, err)
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

// Save writes the collection of models to the specified YAML file.
func (s *Store) Save() error {
	if s.Path == "" {
		return fmt.Errorf("no path specified for model store")
	}

	f, err := os.Create(s.Path)
	if err != nil {
		return fmt.Errorf("failed to create models file %s: %w", s.Path, err)
	}
	defer f.Close()

	enc := yaml.NewEncoder(f)
	enc.SetIndent(2)
	if err := enc.Encode(s.Items); err != nil {
		return fmt.Errorf("failed to encode models to %s: %w", s.Path, err)
	}
	return nil
}

// List returns a sorted slice of models.
func (s *Store) List() []Model {
	keys := make([]string, 0, len(s.Items))
	for k := range s.Items {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	models := make([]Model, 0, len(keys))
	for _, k := range keys {
		models = append(models, s.Items[k])
	}
	return models
}
