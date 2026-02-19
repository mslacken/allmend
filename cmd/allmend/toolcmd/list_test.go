package toolcmd

import (
	"bytes"
	"io"
	"os"
	"testing"

	"github.com/SUSE/allmend/internal/testenv"
	"github.com/SUSE/allmend/pkg/tool"
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
			},
			"server2": {
				Name: "server2",
				Type: "stdio",
				Command: []string{"npx", "server"},
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
		
		// Since connections fail, we expect no tools
		assert.NotContains(t, output, "tool1")
		assert.NotContains(t, output, "tool2")
	})

	t.Run("ExplicitPath", func(t *testing.T) {
		// TODO: This test requires mocking MCP server connection
		t.Skip("Skipping test requiring MCP server connection")
	})
}
