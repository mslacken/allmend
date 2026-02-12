package model

type Model struct {
	Name        string                 `yaml:"name"`
	Description string                 `yaml:"description,omitempty"`
	Type        string                 `yaml:"type,omitempty"`
	Provider    string                 `yaml:"provider"`
	Config      map[string]interface{} `yaml:"config,omitempty"`
}

// Store represents a collection of models.
type Store struct {
	Items map[string]Model `yaml:",inline"`
	// Path is the file path where the models are stored.
	Path string `yaml:"-"`
}
