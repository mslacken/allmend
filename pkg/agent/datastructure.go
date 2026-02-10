package agent

type Agent struct {
	// name of the agent
	Name string `json:"name" yaml:"name"`
	// description of what the agent does
	Description string `json:"description,omitempty" yaml:"description,omitempty"`
	// The principel manifest of how the agent acts
	Manifest *AgentManifest `json:"manifest" yaml:"manifest" spec:"manifest"`
	// concrete mission of the agent
	Mission *AgentMission `json:"mission" yaml:"mission"`
	// needed and recommended tools
	Tools *AgentTools `json:"tools,omitempty" yaml:"tools,omitempty"`
	// metdata of the agent
	Meta *AgentMeta `json:"meta,omitempty" yaml:"meta,omitempty"`
}

type AgentManifest struct {
	// string desribing general behvior
	Content string `json:"content" yaml:"content"`
}

type AgentTools struct {
	// required tools
	Required []*MCPTools `json:"required" yaml:"required"`
	// recommended tools
	Recommended []*MCPTools `json:"recommended,omitempty" yaml:"recommended,omitempty"`
}

type MCPTools struct {
	// name of the tool
	Name string `json:"name" yaml:"name"`
	// semantic version
	Version string `json:"version" yaml:"version"`
	// is the tool read only, can it be called without asking the user
	ReadOnly bool `json:"read_only,omitempty" yaml:"read_only,omitempty"`
	// trusted keys, which are allowed to sign the tool
	Keys []string `json:"keys,omitempty" yaml:"keys,omitempty"`
}

type AgentMission struct {
	// Exact description of what the agent should do
	Content string `json:"content" yaml:"content"`
}

type AgentMeta struct {
	// author of the agent
	Author string `json:"author,omitempty" yaml:"author,omitempty"`
	// semantic version
	Version string `json:"version,omitempty" yaml:"version,omitempty"`
}
