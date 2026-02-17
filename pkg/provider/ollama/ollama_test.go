package ollama

import (
	"testing"
)

// TestValidateConfig tests the validateConfig function which validates configuration parameters against Ollama API structs.
func TestValidateConfig(t *testing.T) {
	tests := []struct {
		name    string
		config  map[string]interface{}
		wantErr bool // in this case, wantWarning
	}{
		{
			name: "Valid config (Options)",
			config: map[string]interface{}{
				"temperature": 0.7,
				"top_k":       40,
			},
			wantErr: false,
		},
		{
			name: "Valid config (ChatRequest)",
			config: map[string]interface{}{
				"truncate":   true,
				"keep_alive": "5m",
				"format":     "json",
				"think":      "high",
			},
			wantErr: false,
		},
		{
			name: "Mixed config",
			config: map[string]interface{}{
				"temperature": 0.5,
				"shift":       true,
			},
			wantErr: false,
		},
		{
			name: "Invalid config (CamelCase)",
			config: map[string]interface{}{
				"Temperature": 0.7,
			},
			wantErr: true,
		},
		{
			name: "Invalid config (Unknown)",
			config: map[string]interface{}{
				"unknown_param": 123,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			supportedKeys, allValidKeys := getSupportedOptions()
			warnings, _ := validateConfig(tt.config, supportedKeys, allValidKeys)

			if tt.wantErr && len(warnings) == 0 {
				t.Errorf("Expected warnings for %v, got none", tt.config)
			}
			if !tt.wantErr && len(warnings) > 0 {
				t.Errorf("Expected no warnings for %v, got %v", tt.config, warnings)
			}
		})
	}
}
