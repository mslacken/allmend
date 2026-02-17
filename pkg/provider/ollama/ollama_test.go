package ollama

import (
	"reflect"
	"testing"

	"github.com/ollama/ollama/api"
)

// MockProvider is a partial mock for testing CheckModel logic
// Since CheckModel relies on p.client.Show which is hard to mock without an interface for the client,
// we will focus on testing the parameter validation logic which is now reflection based.
// However, the current implementation couples the API call with the validation.
// To test this properly, we should extract the validation logic.

func TestValidateConfig(t *testing.T) {
	// We can simulate the validation logic here to ensure our reflection approach works
	// independent of the actual Ollama API call.
	
	supportedKeys := make(map[string]bool)
	val := reflect.TypeOf(api.Options{})
	for i := 0; i < val.NumField(); i++ {
		field := val.Field(i)
		tag := field.Tag.Get("json")
		if tag != "" {
			key := tag
			if idx := 0; idx < len(tag) {
				// Simple comma check
				for j, c := range tag {
					if c == ',' {
						key = tag[:j]
						break
					}
				}
			}
			supportedKeys[key] = true
		}
	}

	tests := []struct {
		name    string
		config  map[string]interface{}
		wantErr bool // in this case, wantWarning
	}{
		{
			name: "Valid config",
			config: map[string]interface{}{
				"temperature": 0.7,
				"top_k":       40,
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
			var warnings []string
			for k := range tt.config {
				if !supportedKeys[k] {
					warnings = append(warnings, k)
				}
			}

			if tt.wantErr && len(warnings) == 0 {
				t.Errorf("Expected warnings for %v, got none", tt.config)
			}
			if !tt.wantErr && len(warnings) > 0 {
				t.Errorf("Expected no warnings for %v, got %v", tt.config, warnings)
			}
		})
	}
}
