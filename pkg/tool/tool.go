package tool

// Server represents an MCP server configuration.
type Server struct {
	Name    string         `yaml:"name"`
	Type    string         `yaml:"type"`              // "http" or "stdio" (default "http")
	URL     string         `yaml:"url,omitempty"`     // for http
	Command []string       `yaml:"command,omitempty"` // for stdio
	Config  map[string]any `yaml:"config,omitempty"`
}

// Tool defines the structure for a tool available on a server.
type Tool struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description,omitempty"`
	InputSchema any    `yaml:"input_schema,omitempty"`
}
