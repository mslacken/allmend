package providercmd

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/SUSE/allmend/internal/testenv"
	"github.com/SUSE/allmend/pkg/provider"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func captureOutput(f func()) string {
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	f()

	w.Close()
	os.Stdout = oldStdout
	var buf bytes.Buffer
	io.Copy(&buf, r)
	return buf.String()
}

func TestAddOllama(t *testing.T) {
	env := testenv.New(t)
	defer env.RemoveAll()

	// Reset flags
	addOllamaCmd.Flags().Set("endpoint", "http://localhost:11434")

	t.Run("DefaultEndpoint", func(t *testing.T) {
		output := captureOutput(func() {
			addOllamaCmd.Run(addOllamaCmd, []string{"my-ollama"})
		})

		assert.Contains(t, output, "Provider 'my-ollama' added successfully.")

		// Verify file content
		providersPath := filepath.Join(env.BaseDir, "config", "providers.conf")
		store, err := provider.Load(providersPath)
		require.NoError(t, err)
		
		p, ok := store.Items["my-ollama"]
		require.True(t, ok)
		assert.Equal(t, "ollama", p.Type)
		assert.Equal(t, "http://localhost:11434", p.Config["endpoint"])
	})

	t.Run("CustomEndpoint", func(t *testing.T) {
		addOllamaCmd.Flags().Set("endpoint", "http://remote:11434")
		
		output := captureOutput(func() {
			addOllamaCmd.Run(addOllamaCmd, []string{"remote-ollama"})
		})

		assert.Contains(t, output, "Provider 'remote-ollama' added successfully.")

		providersPath := filepath.Join(env.BaseDir, "config", "providers.conf")
		store, err := provider.Load(providersPath)
		require.NoError(t, err)

		p, ok := store.Items["remote-ollama"]
		require.True(t, ok)
		assert.Equal(t, "http://remote:11434", p.Config["endpoint"])
	})
}

func TestAddGoogle(t *testing.T) {
	env := testenv.New(t)
	defer env.RemoveAll()

	resetFlags := func() {
		addGoogleCmd.Flags().Set("api-key", "")
		addGoogleCmd.Flags().Set("project-id", "")
		addGoogleCmd.Flags().Set("location", "us-central1")
		addGoogleCmd.Flags().Set("backend", "gemini")
		os.Unsetenv("GEMINI_API_KEY")
	}

	t.Run("WithAPIKeyFlag", func(t *testing.T) {
		resetFlags()
		addGoogleCmd.Flags().Set("api-key", "secret-key")
		
		output := captureOutput(func() {
			addGoogleCmd.Run(addGoogleCmd, []string{"gemini-flag"})
		})

		assert.Contains(t, output, "Provider 'gemini-flag' added successfully.")

		providersPath := filepath.Join(env.BaseDir, "config", "providers.conf")
		store, err := provider.Load(providersPath)
		require.NoError(t, err)

		p, ok := store.Items["gemini-flag"]
		require.True(t, ok)
		assert.Equal(t, "secret-key", p.Config["api_key"])
	})

	t.Run("WithEnvVar", func(t *testing.T) {
		resetFlags()
		os.Setenv("GEMINI_API_KEY", "env-key")
		defer os.Unsetenv("GEMINI_API_KEY")

		output := captureOutput(func() {
			addGoogleCmd.Run(addGoogleCmd, []string{"gemini-env"})
		})

		assert.Contains(t, output, "Using API Key from environment or .env file.")
		assert.Contains(t, output, "Provider 'gemini-env' added successfully.")

		providersPath := filepath.Join(env.BaseDir, "config", "providers.conf")
		store, err := provider.Load(providersPath)
		require.NoError(t, err)

		p, ok := store.Items["gemini-env"]
		require.True(t, ok)
		assert.Equal(t, "env-key", p.Config["api_key"])
	})

	t.Run("WithEnvFile", func(t *testing.T) {
		resetFlags()
		
		// Create .env file in the current working directory where the test runs
		// Note: os.ReadFile in config.GetEnvOrFile reads from current dir
		err := os.WriteFile(".env", []byte("GEMINI_API_KEY=file-key\n"), 0644)
		require.NoError(t, err)
		defer os.Remove(".env")

		output := captureOutput(func() {
			addGoogleCmd.Run(addGoogleCmd, []string{"gemini-file"})
		})

		assert.Contains(t, output, "Using API Key from environment or .env file.")
		assert.Contains(t, output, "Provider 'gemini-file' added successfully.")

		providersPath := filepath.Join(env.BaseDir, "config", "providers.conf")
		store, err := provider.Load(providersPath)
		require.NoError(t, err)

		p, ok := store.Items["gemini-file"]
		require.True(t, ok)
		assert.Equal(t, "file-key", p.Config["api_key"])
	})
}
