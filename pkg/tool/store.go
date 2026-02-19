package tool

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Store manages the collection of tool servers.
type Store struct {
	Servers map[string]Server `yaml:"servers"`
	Path    string
}

// Load reads the tool definitions from a YAML file.
func Load(path string) (*Store, error) {
	store := &Store{
		Servers: make(map[string]Server),
		Path:    path,
	}

	if _, err := os.Stat(path); os.IsNotExist(err) {
		return store, nil
	}

	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open tool file %s: %w", path, err)
	}
	defer f.Close()

	var temp struct {
		Servers map[string]Server `yaml:"servers"`
	}

	if err := yaml.NewDecoder(f).Decode(&temp); err != nil {
		return nil, fmt.Errorf("failed to decode tools from %s: %w", path, err)
	}
	store.Servers = temp.Servers

	return store, nil
}

// Save writes the tool definitions back to the YAML file.
func (s *Store) Save() error {
	if s.Path == "" {
		return fmt.Errorf("no path specified for tool store")
	}

	dir := filepath.Dir(s.Path)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("creating tool store directory: %w", err)
		}
	}

	f, err := os.Create(s.Path)
	if err != nil {
		return fmt.Errorf("failed to create tool file %s: %w", s.Path, err)
	}
	defer f.Close()

	enc := yaml.NewEncoder(f)
	enc.SetIndent(2)
	
	data := map[string]interface{}{
		"servers": s.Servers,
	}

	if err := enc.Encode(data); err != nil {
		return fmt.Errorf("failed to encode tools to %s: %w", s.Path, err)
	}
	return nil
}
