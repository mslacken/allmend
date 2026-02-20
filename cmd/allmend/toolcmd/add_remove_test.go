package toolcmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/SUSE/allmend/internal/testenv"
	"github.com/SUSE/allmend/pkg/mcp"
	"github.com/SUSE/allmend/pkg/tool"
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

func mockMCPServer() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Handle SSE request
		if r.Method == "GET" && r.Header.Get("Accept") == "text/event-stream" {
			w.Header().Set("Content-Type", "text/event-stream")
			w.Header().Set("Cache-Control", "no-cache")
			w.Header().Set("Connection", "keep-alive")
			w.WriteHeader(http.StatusOK)

			// Send endpoint event
			fmt.Fprintf(w, "event: endpoint\ndata: /rpc\n\n")
			if f, ok := w.(http.Flusher); ok {
				f.Flush()
			}
			
			// Wait a bit to simulate open connection so client can read it
			time.Sleep(100 * time.Millisecond)
			return
		}

		// Handle POST request (JSON-RPC)
		if r.Method != "POST" {
			// For robustness, ignore other methods or return 405
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

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
			// No response needed
			return
		case "tools/list":
			result := mcp.ListToolsResult{
				Tools: []mcp.Tool{
					{
						Name:        "test-tool",
						Description: "A test tool",
					},
				},
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

func TestAddToolServer(t *testing.T) {
	env := testenv.New(t)
	defer env.RemoveAll()

	server := mockMCPServer()
	defer server.Close()

	t.Run("AddServer", func(t *testing.T) {
		output := captureOutput(func() {
			serverAddCmd.Run(serverAddCmd, []string{server.URL})
		})

		assert.Contains(t, output, "Connected to server: mock-server")
		assert.Contains(t, output, "Server '" + server.URL + "' added/updated successfully")

		// Verify file content
		toolsPath := filepath.Join(env.BaseDir, "config", "tools.conf")
		store, err := tool.Load(toolsPath)
		require.NoError(t, err)

		s, ok := store.Servers[server.URL]
		require.True(t, ok)
		assert.Equal(t, server.URL, s.URL)
	})

	t.Run("AddExistingServer", func(t *testing.T) {
		// Try to add it again
		output := captureOutput(func() {
			serverAddCmd.Run(serverAddCmd, []string{server.URL})
		})

		assert.Contains(t, output, "Server '" + server.URL + "' already exists. Updating configuration...")
	})
}

func TestRemoveServer(t *testing.T) {
	env := testenv.New(t)
	defer env.RemoveAll()

	server := mockMCPServer()
	defer server.Close()

	// Add a server to be removed
	serverAddCmd.Run(serverAddCmd, []string{server.URL})

	t.Run("RemoveServer", func(t *testing.T) {
		output := captureOutput(func() {
			removeCmd.Run(removeCmd, []string{server.URL})
		})

		assert.Contains(t, output, "Server '" + server.URL + "' removed successfully.")

		// Verify it's gone
		toolsPath := filepath.Join(env.BaseDir, "config", "tools.conf")
		store, err := tool.Load(toolsPath)
		require.NoError(t, err)

		_, ok := store.Servers[server.URL]
		assert.False(t, ok)
	})

	t.Run("RemoveNonExistentServer", func(t *testing.T) {
		output := captureOutput(func() {
			removeCmd.Run(removeCmd, []string{"http://nonexistent.com"})
		})

		assert.Contains(t, output, "Error: Server 'http://nonexistent.com' not found.")
	})
}
