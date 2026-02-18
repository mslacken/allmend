package tool

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"gopkg.in/yaml.v3"
)

// Store manages the collection of tool servers.
type Store struct {
	Servers  map[string]Server `yaml:"servers"`
	filePath string
}

// Load reads the tool definitions from a YAML file.
func Load(path string) (*Store, error) {
	s := &Store{
		Servers:  make(map[string]Server),
		filePath: path,
	}

	if _, err := os.Stat(path); os.IsNotExist(err) {
		// If the file doesn't exist, create it with an empty map.
		if err := s.Save(); err != nil {
			return nil, fmt.Errorf("failed to create tool store: %w", err)
		}
		return s, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading tool file: %w", err)
	}

	var temp struct {
		Servers map[string]Server `yaml:"servers"`
	}
	
	if err := yaml.Unmarshal(data, &temp); err != nil {
		return nil, fmt.Errorf("unmarshaling tools: %w", err)
	}
    s.Servers = temp.Servers

	return s, nil
}

// Save writes the tool definitions back to the YAML file.
func (s *Store) Save() error {
	dir := filepath.Dir(s.filePath)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("creating tool store directory: %w", err)
		}
	}

	data, err := yaml.Marshal(map[string]interface{}{
		"servers": s.Servers,
	})
	if err != nil {
		return fmt.Errorf("marshaling tools: %w", err)
	}

	return os.WriteFile(s.filePath, data, 0644)
}

// List returns a sorted slice of all tools from all servers.
func (s *Store) List() []Tool {
	var list []Tool
	for _, server := range s.Servers {
		list = append(list, server.Tools...)
	}
	sort.Slice(list, func(i, j int) bool {
		return list[i].Name < list[j].Name
	})
	return list
}
