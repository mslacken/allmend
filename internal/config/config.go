package config

import (
	"os"
	"path"
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
