package provider

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
)

func TestLoad(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "allmend-test-provider")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	providers := map[string]Provider{
		"openai":    {Type: "openai"},
		"anthropic": {Type: "anthropic"},
	}
	
	bytes, err := yaml.Marshal(providers)
	if err != nil {
		t.Fatalf("Failed to marshal providers: %v", err)
	}
	
	path := filepath.Join(tmpDir, "providers.conf")
	err = os.WriteFile(path, bytes, 0644)
	if err != nil {
		t.Fatalf("Failed to write providers file: %v", err)
	}
	
	loaded, err := Load(path)
	if err != nil {
		t.Fatalf("Failed to load providers: %v", err)
	}
	
	assert.Equal(t, "openai", loaded.Items["openai"].Type)
	assert.Equal(t, "openai", loaded.Items["openai"].Name) // Name should be set from key
	assert.Equal(t, "anthropic", loaded.Items["anthropic"].Type)
	assert.Equal(t, "anthropic", loaded.Items["anthropic"].Name)
}

func TestLoadEmpty(t *testing.T) {
	loaded, err := Load("nonexistent.conf")
	assert.NoError(t, err)
	assert.Empty(t, loaded.Items)
}
