package agent

import (
	"fmt"
	"log/slog"
	"path/filepath"
)

// List scans the provided directories for agent files (.agt, .json, .yaml, .yml)
// and returns a list of successfully loaded agents.
// Errors encountered during scanning or parsing individual files are returned in the second return value.
func Get(paths []string) (map[string]*Agent, error) {
	agents := make(map[string]*Agent)
	patterns := []string{"*.agt", "*.json", "*.yaml", "*.yml"}
	for _, p := range paths {
		for _, pat := range patterns {
			fullPattern := filepath.Join(p, pat)
			matches, err := filepath.Glob(fullPattern)
			if err != nil {
				slog.Warn(fmt.Sprintf("glob failed for %s", fullPattern), "error", err)
				continue
			}
			for _, file := range matches {

				agent, err := Load(file)
				if err != nil {
					slog.Warn("couldn't parse agent definition", "error", err)
					continue
				}
				if _, found := agents[agent.Name]; found {
					return nil, fmt.Errorf("multiple agents with same name found: %s", agent.Name)
				}
				agents[agent.Name] = agent
			}
		}
	}

	return agents, nil
}

// ListNames returns a slice of all agent names found in the provided paths.
// It ignores errors and only returns successfully parsed names.
func ListNames(paths []string) []string {
	agents, _ := Get(paths)
	names := make([]string, 0, len(agents))
	for _, a := range agents {
		if a.Name != "" {
			names = append(names, a.Name)
		}
	}
	return names
}
