package agentcmd

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

func TestCheckAgent(t *testing.T) {
	env := testenv.New(t)
	defer env.RemoveAll()

	// 1. Setup Agent
	agentContent := `%Meta
Name: TestAgent
%Tools
%Required
req_tool_1
req_tool_2
%Recommended
rec_tool_1
rec_tool_2
`
	// testenv already configures agent_paths to point to agents/ directory
	env.WriteFile("agents/test.agt", agentContent)

	// 2. Setup Tools
	// req_tool_1 exists, req_tool_2 missing
	// rec_tool_1 missing, rec_tool_2 exists
	servers := map[string]tool.Server{
		"server1": {
			Name: "server1",
			Type: "http",
			URL:  "http://example.com",
			Tools: []tool.Tool{
				{Name: "req_tool_1"},
				{Name: "rec_tool_2"},
			},
		},
	}
	config := map[string]interface{}{
		"servers": servers,
	}
	toolsBytes, _ := yaml.Marshal(config)
	// Default location for tools file is config/tools.conf relative to config file
	env.WriteFile("config/tools.conf", string(toolsBytes))

	t.Run("CheckMissingTools", func(t *testing.T) {
		var err error
		output := captureOutput(func() {
			err = checkCmd.RunE(checkCmd, []string{"TestAgent"})
		})

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "missing required tools")

		assert.Contains(t, output, "Error: Missing required tools for agent 'TestAgent':")
		assert.Contains(t, output, "- req_tool_2")
		assert.NotContains(t, output, "- req_tool_1")

		assert.Contains(t, output, "Warning: Missing recommended tools for agent 'TestAgent':")
		assert.Contains(t, output, "- rec_tool_1")
		assert.NotContains(t, output, "- rec_tool_2")
	})

	t.Run("CheckAgentNotFound", func(t *testing.T) {
		var err error
		captureOutput(func() {
			err = checkCmd.RunE(checkCmd, []string{"UnknownAgent"})
		})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "agent 'UnknownAgent' not found")
	})
	
	t.Run("CheckAllToolsAvailable", func(t *testing.T) {
		// Update tools to have everything
		servers["server1"] = tool.Server{
			Name: "server1",
			Type: "http",
			URL:  "http://example.com",
			Tools: []tool.Tool{
				{Name: "req_tool_1"},
				{Name: "req_tool_2"},
				{Name: "rec_tool_1"},
				{Name: "rec_tool_2"},
			},
		}
		config["servers"] = servers
		toolsBytes, _ := yaml.Marshal(config)
		env.WriteFile("config/tools.conf", string(toolsBytes))
		
		var err error
		output := captureOutput(func() {
			err = checkCmd.RunE(checkCmd, []string{"TestAgent"})
		})

		assert.NoError(t, err)
		assert.Contains(t, output, "All required and recommended tools for agent 'TestAgent' are available.")
	})
}
