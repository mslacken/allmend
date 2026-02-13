package providercmd

import (
	"testing"

	"github.com/SUSE/allmend/internal/testenv"
	"github.com/stretchr/testify/assert"
)

func TestModelList(t *testing.T) {
	env := testenv.New(t)
	defer env.RemoveAll()

	t.Run("ProviderNotFound", func(t *testing.T) {
		// Empty providers file (valid YAML)
		env.WriteFile("config/providers.conf", "{}")

		output := captureOutput(func() {
			// listCmd expects 1 arg: provider name
			listCmd.Run(listCmd, []string{"non-existent"})
		})

		assert.Contains(t, output, "provider 'non-existent' not found")
	})

	t.Run("ProviderFoundButConnectionFails", func(t *testing.T) {
		// Valid provider config but connection likely fails (localhost:12345)
		content := "test-ollama:\n  type: ollama\n  config:\n    endpoint: http://localhost:12345\n"
		env.WriteFile("config/providers.conf", content)

		output := captureOutput(func() {
			listCmd.Run(listCmd, []string{"test-ollama"})
		})

		assert.Contains(t, output, "fetching models from provider 'test-ollama'")
	})
}

func TestModelAdd(t *testing.T) {
	env := testenv.New(t)
	defer env.RemoveAll()

	t.Run("ProviderNotFound", func(t *testing.T) {
		env.WriteFile("config/providers.conf", "{}")

		output := captureOutput(func() {
			// addModelCmd with provider only -> sync all
			addModelCmd.Run(addModelCmd, []string{"non-existent"})
		})

		assert.Contains(t, output, "provider 'non-existent' not found")
	})
}

// captureOutput captures stdout/stderr. 
// Assuming it's defined elsewhere or was part of previous file?
// Wait, previous file used `captureOutput`. Where is it defined?
// It's likely in `export_test.go` or similar in same package if not imported.
// Since I don't see it imported, it must be in another file in package providercmd.
