package toolcmd

import (
	"bytes"
	"io"
	"os"
	"testing"

	"github.com/SUSE/allmend/internal/testenv"
	"github.com/SUSE/allmend/pkg/tool"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
)

func TestToolList(t *testing.T) {
	env := testenv.New(t)
	defer env.RemoveAll()

	// Helper to capture stdout
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
		// Create tools map
		servers := map[string]tool.Server{
			"server1": {
				Name: "server1",
				Type: "http",
				URL: "http://example.com/api",
				Tools: []tool.Tool{
					{Name: "tool1", Description: "Tool 1"},
				},
			},
			"server2": {
				Name: "server2",
				Type: "stdio",
				Command: []string{"npx", "server"},
				Tools: []tool.Tool{
					{Name: "tool2", Description: "Tool 2"},
				},
			},
		}

		config := map[string]interface{}{
			"servers": servers,
		}

		toolsBytes, err := yaml.Marshal(config)
		if err != nil {
			t.Fatalf("Failed to marshal tools: %v", err)
		}

		env.WriteFile("config/tools.conf", string(toolsBytes))

		// Run
		output := captureOutput(func() {
			listCmd.Run(listCmd, []string{})
		})

		// Check headers
		assert.Contains(t, output, "NAME")
		assert.Contains(t, output, "SERVER")
		assert.Contains(t, output, "TYPE")
		assert.Contains(t, output, "COMMAND/URL")
		
		// Check tool 1
		assert.Contains(t, output, "tool1")
		assert.Contains(t, output, "server1")
		assert.Contains(t, output, "http")
		assert.Contains(t, output, "http://example.com/api")
		assert.Contains(t, output, "Tool 1")
		
		// Check tool 2
		assert.Contains(t, output, "tool2")
		assert.Contains(t, output, "server2")
		assert.Contains(t, output, "stdio")
		assert.Contains(t, output, "npx server")
		assert.Contains(t, output, "Tool 2")
	})

	t.Run("ExplicitPath", func(t *testing.T) {
		servers := map[string]tool.Server{
			"custom/server": {
				Name: "custom-server",
				URL: "custom/server",
				Tools: []tool.Tool{
					{Name: "customtool", Description: "Custom Path Tool"},
				},
			},
		}
		
		config := map[string]interface{}{
			"servers": servers,
		}
		
		toolsBytes, _ := yaml.Marshal(config)

		// Write to a custom location in temp env
		customPath := "custom/tools.conf"
		env.WriteFile(customPath, string(toolsBytes))

		viper.Set("tools_file", env.GetPath(customPath))
		defer viper.Set("tools_file", "")

		output := captureOutput(func() {
			listCmd.Run(listCmd, []string{})
		})

		assert.Contains(t, output, "customtool")
		assert.Contains(t, output, "Custom Path Tool")
		assert.Contains(t, output, "custom-server")
	})
}
