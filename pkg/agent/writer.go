package agent

import (
	"fmt"
	"io"
	"strings"
)

// WriteAgent writes an Agent struct to an .agt format writer
func WriteAgent(w io.Writer, agent *Agent) error {
	// Write Meta section
	fmt.Fprintln(w, "%Meta")
	if agent.Name != "" {
		fmt.Fprintf(w, "Name: %s\n", agent.Name)
	}
	if agent.Description != "" {
		fmt.Fprintf(w, "Description: %s\n", agent.Description)
	}
	if agent.Meta != nil {
		if agent.Meta.Author != "" {
			fmt.Fprintf(w, "Author: %s\n", agent.Meta.Author)
		}
		if agent.Meta.Version != "" {
			fmt.Fprintf(w, "Version: %s\n", agent.Meta.Version)
		}
	}
	fmt.Fprintln(w)

	// Write Manifest section
	if agent.Manifest != nil && agent.Manifest.Content != "" {
		fmt.Fprintln(w, "%Manifest")
		fmt.Fprintln(w, strings.TrimSpace(agent.Manifest.Content))
		fmt.Fprintln(w)
	}

	// Write Mission section
	if agent.Mission != nil && agent.Mission.Content != "" {
		fmt.Fprintln(w, "%Mission")
		fmt.Fprintln(w, strings.TrimSpace(agent.Mission.Content))
		fmt.Fprintln(w)
	}

	return nil
}
