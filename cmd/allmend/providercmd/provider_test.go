package providercmd

import (
	"bytes"
	"io"
	"os"
	"testing"

	"github.com/SUSE/allmend/internal/testenv"
	"github.com/SUSE/allmend/pkg/provider"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
)

func TestProviderList(t *testing.T) {
	env := testenv.New(t)
	defer env.RemoveAll()

	captureOutput := func(f func()) string {
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

	t.Run("DefaultLocation", func(t *testing.T) {
		providers := map[string]provider.Provider{
			"openai":    {Type: "openai"},
			"anthropic": {Type: "anthropic"},
		}
		
		providersBytes, err := yaml.Marshal(providers)
		if err != nil {
			t.Fatalf("Failed to marshal providers: %v", err)
		}
		
		// Write to config/providers.conf in temp env
		env.WriteFile("config/providers.conf", string(providersBytes))

		output := captureOutput(func() {
			listProvidersCmd.Run(listProvidersCmd, []string{})
		})

		assert.Contains(t, output, "- openai: openai")
		assert.Contains(t, output, "- anthropic: anthropic")
	})

	t.Run("CustomFormat", func(t *testing.T) {
		providers := map[string]provider.Provider{
			"custom": {Type: "custom_type"},
		}
		providersBytes, _ := yaml.Marshal(providers)
		env.WriteFile("config/providers.conf", string(providersBytes))

		// Set format flag
		oldFormat, _ := listProvidersCmd.Flags().GetString("format")
		listProvidersCmd.Flags().Set("format", "Provider: {{.Name}} ({{.Type}})\n")
		defer listProvidersCmd.Flags().Set("format", oldFormat)

		output := captureOutput(func() {
			listProvidersCmd.Run(listProvidersCmd, []string{})
		})

		assert.Contains(t, output, "Provider: custom (custom_type)")
	})
}
