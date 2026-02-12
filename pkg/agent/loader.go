package agent

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// Load reads an agent file from the given path, detecting format by extension.
func Load(path string) (*Agent, error) {
	ext := strings.ToLower(filepath.Ext(path))
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open file %s: %w", path, err)
	}
	defer f.Close()

	switch ext {
	case ".json":
		var agent Agent
		if err := json.NewDecoder(f).Decode(&agent); err != nil {
			return nil, fmt.Errorf("failed to parse JSON agent %s: %w", path, err)
		}
		agent.SourceFile = path
		return &agent, nil
	case ".yaml", ".yml":
		var agent Agent
		if err := yaml.NewDecoder(f).Decode(&agent); err != nil {
			return nil, fmt.Errorf("failed to parse YAML agent %s: %w", path, err)
		}
		agent.SourceFile = path
		return &agent, nil
	case ".agt":
		agent, err := ParseAgent(f)
		if err == nil && agent != nil {
			agent.SourceFile = path
		}
		return agent, err
	default:
		return nil, fmt.Errorf("unsupported file extension: %s", ext)
	}
}
