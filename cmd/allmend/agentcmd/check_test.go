package agentcmd

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/SUSE/allmend/internal/testenv"
	"github.com/SUSE/allmend/pkg/agent"
	"github.com/SUSE/allmend/pkg/mcp"
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

func mockMCPServer(tools []mcp.Tool) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req mcp.JSONRPCRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}

		res := mcp.JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
		}

		switch req.Method {
		case "initialize":
			result := mcp.InitializeResult{
				ProtocolVersion: "2024-11-05",
				ServerInfo: mcp.Implementation{
					Name:    "mock-server",
					Version: "1.0.0",
				},
			}
			res.Result = result
		case "notifications/initialized":
			return
		case "tools/list":
			result := mcp.ListToolsResult{
				Tools: tools,
			}
			res.Result = result
		default:
			err := mcp.RPCError{
				Code:    -32601,
				Message: "Method not found",
			}
			res.Error = &err
		}

		json.NewEncoder(w).Encode(res)
	}))
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
	env.WriteFile("agents/test.agt", agentContent)

	t.Run("CheckMissingTools", func(t *testing.T) {
		// Mock server with only some tools
		server := mockMCPServer([]mcp.Tool{
			{Name: "req_tool_1"},
			{Name: "rec_tool_2"},
		})
		defer server.Close()

		servers := map[string]tool.Server{
			"server_missing": {
				Name: "server_missing",
				Type: "http",
				URL:  server.URL,
			},
		}
		config := map[string]interface{}{
			"servers": servers,
		}
		toolsBytes, _ := yaml.Marshal(config)
		env.WriteFile("config/tools.conf", string(toolsBytes))

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
		// Mock server with all tools
		server := mockMCPServer([]mcp.Tool{
			{Name: "req_tool_1"},
			{Name: "req_tool_2"},
			{Name: "rec_tool_1"},
			{Name: "rec_tool_2"},
		})
		defer server.Close()
		
		servers := map[string]tool.Server{
			"server_all": {
				Name: "server_all",
				Type: "http",
				URL:  server.URL,
			},
		}
		config := map[string]interface{}{
			"servers": servers,
		}
		toolsBytes, _ := yaml.Marshal(config)
		env.WriteFile("config/tools.conf", string(toolsBytes))
		
		var err error
		output := captureOutput(func() {
			err = checkCmd.RunE(checkCmd, []string{"TestAgent"})
		})

		assert.NoError(t, err)
		assert.Contains(t, output, "All required and recommended tools for agent 'TestAgent' are available.")
	})

	t.Run("CheckSubAgents", func(t *testing.T) {
		// Mock server for sub-agent tools
		server := mockMCPServer([]mcp.Tool{
			{Name: "sub_tool_1"},
		})
		defer server.Close()

		servers := map[string]tool.Server{
			"server_sub": {
				Name: "server_sub",
				Type: "http",
				URL:  server.URL,
			},
		}
		config := map[string]interface{}{
			"servers": servers,
		}
		toolsBytes, _ := yaml.Marshal(config)
		env.WriteFile("config/tools.conf", string(toolsBytes))

		// Manually construct agent with sub-agent
		mainAgent := &agent.Agent{
			Name: "MainAgent",
			SubAgents: []*agent.Agent{
				{
					Name: "SubAgent1",
					Tools: &agent.AgentTools{
						Required: []*agent.MCPTools{
							{Name: "sub_tool_1"},
						},
					},
				},
				{
					Name: "SubAgent2",
					Tools: &agent.AgentTools{
						Required: []*agent.MCPTools{
							{Name: "sub_tool_missing"},
						},
					},
				},
			},
		}

		// Use the internal checkAgentTools function via CheckTools
		// But CheckTools loads file based on viper config.
		// We can overwrite CheckTools to accept agent directly, which it does.
		// But CheckTools loads tools from file.
		
		// Run CheckTools
		err := checkAgentTools(mainAgent, &tool.Store{Servers: servers})
		
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "missing required tools for agent SubAgent2")
	})
}
