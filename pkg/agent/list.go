package agent

import (
	"fmt"
	"path/filepath"
)

// List scans the provided directories for agent files (.agt, .json, .yaml, .yml)
// and returns a list of successfully loaded agents.
// Errors encountered during scanning or parsing individual files are returned in the second return value.
func List(paths []string) ([]*Agent, []error) {
	var agents []*Agent
	var errs []error
	loaded := make(map[string]bool)

	patterns := []string{"*.agt", "*.json", "*.yaml", "*.yml"}

	for _, p := range paths {
		for _, pat := range patterns {
			fullPattern := filepath.Join(p, pat)
			matches, err := filepath.Glob(fullPattern)
			if err != nil {
				errs = append(errs, fmt.Errorf("glob failed for %s: %w", fullPattern, err))
				continue
			}

			for _, file := range matches {
				if loaded[file] {
					continue
				}
				loaded[file] = true

				agent, err := Load(file)
				if err != nil {
					errs = append(errs, fmt.Errorf("failed to load agent %s: %w", file, err))
					continue
				}
				agents = append(agents, agent)
			}
		}
	}

	return agents, errs
}

// ListNames returns a slice of all agent names found in the provided paths.
// It ignores errors and only returns successfully parsed names.
func ListNames(paths []string) []string {
	agents, _ := List(paths)
	names := make([]string, 0, len(agents))
	for _, a := range agents {
		if a.Name != "" {
			names = append(names, a.Name)
		}
	}
	return names
}
