package config

import (
	"os"
	"path"
	"strings"
)

// ConfigLocations returns default configuration locations.
func ConfigLocations(names ...string) []string {
	name := path.Join(names...)
	paths := []string{path.Join("./config", name)}
	home, _ := os.UserHomeDir()
	if home != "" {
		paths = append(paths, path.Join(home, ".config", "allmend", name))
	}
	paths = append(paths, path.Join("/etc", name), path.Join("/usr/etc/allmend", name))
	return paths
}

// GetEnvOrFile checks environment variable first, then .env file in the current directory.
func GetEnvOrFile(key string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}

	// Simple .env parser
	content, err := os.ReadFile(".env")
	if err != nil {
		return ""
	}

	lines := strings.Split(string(content), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "#") || line == "" {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 && strings.TrimSpace(parts[0]) == key {
			// Basic cleanup of quotes if present
			val := strings.TrimSpace(parts[1])
			val = strings.Trim(val, `"'`)
			return val
		}
	}
	return ""
}
