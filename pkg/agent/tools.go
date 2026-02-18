package agent

import (
	"context"
	"fmt"
	"sync"

	"github.com/SUSE/allmend/pkg/mcp"
	"github.com/SUSE/allmend/pkg/tool"
	adktool "google.golang.org/adk/tool"
)

// MCPTool implements adktool.Tool for an MCP tool.
type MCPTool struct {
	name        string
	description string
	client      *mcp.Client
}

func (t *MCPTool) Name() string {
	return t.name
}

func (t *MCPTool) Description() string {
	return t.description
}

func (t *MCPTool) IsLongRunning() bool {
	return false
}

// Run executes the tool via MCP.
func (t *MCPTool) Run(ctx adktool.Context, args map[string]any) (map[string]any, error) {
	// Call the MCP tool
	// The method is usually "tools/call".
	params := map[string]any{
		"name":      t.name,
		"arguments": args,
	}
	
	var callResult mcp.CallToolResult
	if err := t.client.Call(context.Background(), "tools/call", params, &callResult); err != nil {
		return nil, fmt.Errorf("failed to call MCP tool %s: %w", t.name, err)
	}
	
	// Convert result to map[string]any
	// MCP returns content list.
	output := make(map[string]any)
	if callResult.IsError {
		output["error"] = true
	}
	
	// Aggregate text content
	var textContent string
	for _, c := range callResult.Content {
		if c.Type == "text" {
			textContent += c.Text
		}
	}
	output["content"] = textContent
	
	return output, nil
}

// clientCache ensures we reuse connections to the same server.
var clientCache = make(map[string]*mcp.Client)
var cacheMu sync.Mutex

func getClient(server tool.Server) *mcp.Client {
	cacheMu.Lock()
	defer cacheMu.Unlock()
	
	// Use Name as key if available, otherwise URL
	key := server.Name
	if key == "" {
		key = server.URL
	}
	
	if c, ok := clientCache[key]; ok {
		return c
	}

	var transport mcp.Transport
	if server.Type == "stdio" {
		// Env should be []string
		var env []string
		if val, ok := server.Config["env"]; ok {
			switch v := val.(type) {
			case []string:
				env = v
			case []interface{}:
				for _, e := range v {
					if s, ok := e.(string); ok {
						env = append(env, s)
					}
				}
			}
		}
		
		transport = mcp.NewStdioTransport(server.Command, env)
	} else {
		// Default to HTTP
		transport = mcp.NewHTTPTransport(server.URL)
	}

	c := mcp.NewClient(transport)
	// Initialize it
	// Warning: This blocks. Stdio process starts here.
	if _, err := c.Initialize(context.Background()); err != nil {
		fmt.Printf("Error initializing client for %s: %v\n", key, err)
		// Return valid client anyway? Or nil?
		// If init fails, future calls will likely fail.
	}
	
	clientCache[key] = c
	return c
}

// LoadTools finds and creates tool wrappers for the agent's required tools.
func LoadTools(agent *Agent, store *tool.Store) ([]adktool.Tool, error) {
	if agent.Tools == nil {
		return nil, nil
	}

	var loadedTools []adktool.Tool
	
	// Collect all requested tool names
	requested := make(map[string]bool)
	for _, t := range agent.Tools.Required {
		requested[t.Name] = true
	}
	for _, t := range agent.Tools.Recommended {
		requested[t.Name] = true
	}

	// Iterate over servers to find matching tools
	for _, server := range store.Servers {
		// Check if any tool in this server matches requested
		for _, t := range server.Tools {
			if requested[t.Name] {
				client := getClient(server)
				
				mcpTool := &MCPTool{
					name:        t.Name,
					description: t.Description,
					client:      client,
				}
				loadedTools = append(loadedTools, mcpTool)
			}
		}
	}

	return loadedTools, nil
}
