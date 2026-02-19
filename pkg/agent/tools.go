package agent

import (
	"context"
	"fmt"
	"os"
	"sync"

	"github.com/SUSE/allmend/pkg/mcp"
	"github.com/SUSE/allmend/pkg/tool"
	adkmodel "google.golang.org/adk/model"
	adktool "google.golang.org/adk/tool"
	"google.golang.org/genai"
)

// MCPTool implements adktool.Tool for an MCP tool.
type MCPTool struct {
	name        string
	description string
	inputSchema any
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

// Declaration returns the FunctionDeclaration for the tool.
// This allows the tool to be recognized as a FunctionTool by the ADK runner.
func (t *MCPTool) Declaration() *genai.FunctionDeclaration {
	decl := &genai.FunctionDeclaration{
		Name:        t.name,
		Description: t.description,
	}

	if t.inputSchema != nil {
		if schemaMap, ok := t.inputSchema.(map[string]any); ok {
			schema, err := mapToGenaiSchema(schemaMap)
			if err != nil {
				// Log warning? We can't return error here.
				fmt.Fprintf(os.Stderr, "Warning: failed to convert schema for tool %s: %v\n", t.name, err)
			} else {
				decl.Parameters = schema
			}
		}
	}
	return decl
}

// ProcessRequest registers the tool with the LLM request.
// This implements the toolinternal.RequestProcessor interface required by ADK.
func (t *MCPTool) ProcessRequest(ctx adktool.Context, req *adkmodel.LLMRequest) error {
	decl := t.Declaration()

	// Add to request config
	if req.Config == nil {
		req.Config = &genai.GenerateContentConfig{}
	}
	
	// Check if we already have a Tool with FunctionDeclarations
	var funcTool *genai.Tool
	for _, gt := range req.Config.Tools {
		if gt.FunctionDeclarations != nil {
			funcTool = gt
			break
		}
	}
	
	if funcTool == nil {
		funcTool = &genai.Tool{}
		req.Config.Tools = append(req.Config.Tools, funcTool)
	}
	
	funcTool.FunctionDeclarations = append(funcTool.FunctionDeclarations, decl)
	
	// Also register this tool instance in req.Tools so the runner can find it to execute
	if req.Tools == nil {
		req.Tools = make(map[string]any)
	}
	req.Tools[t.name] = t

	return nil
}

func mapToGenaiSchema(m map[string]any) (*genai.Schema, error) {
	s := &genai.Schema{}
	
	if t, ok := m["type"].(string); ok {
		s.Type = genai.Type(t)
	}
	
	if desc, ok := m["description"].(string); ok {
		s.Description = desc
	}
	
	if props, ok := m["properties"].(map[string]any); ok {
		s.Properties = make(map[string]*genai.Schema)
		for k, v := range props {
			if vMap, ok := v.(map[string]any); ok {
				propSchema, err := mapToGenaiSchema(vMap)
				if err != nil {
					return nil, err
				}
				s.Properties[k] = propSchema
			}
		}
	}
	
	if items, ok := m["items"].(map[string]any); ok {
		itemSchema, err := mapToGenaiSchema(items)
		if err != nil {
			return nil, err
		}
		s.Items = itemSchema
	}
	
	if req, ok := m["required"].([]any); ok {
		for _, r := range req {
			if rStr, ok := r.(string); ok {
				s.Required = append(s.Required, rStr)
			}
		}
	}
	
	// Handle enum?
	if enum, ok := m["enum"].([]any); ok {
		for _, e := range enum {
			if eStr, ok := e.(string); ok {
				s.Enum = append(s.Enum, eStr)
			}
		}
	}

	return s, nil
}


// Run executes the tool via MCP.
func (t *MCPTool) Run(ctx adktool.Context, args any) (map[string]any, error) {
	argsMap, ok := args.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("expected map[string]any args, got %T", args)
	}

	// Call the MCP tool
	// The method is usually "tools/call".
	params := map[string]any{
		"name":      t.name,
		"arguments": argsMap,
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
		for _, c := range callResult.Content {
			if c.Type == "text" {
				fmt.Fprintf(os.Stderr, "Tool Error (%s): %s\n", t.name, c.Text)
			}
		}
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
		client := getClient(server)
		
		// Dynamically fetch tools from the server
		tools, err := client.ListTools(context.Background())
		if err != nil {
			fmt.Printf("Warning: failed to list tools from server %s: %v\n", server.Name, err)
			continue
		}

		for _, t := range tools {
			if requested[t.Name] {
				mcpTool := &MCPTool{
					name:        t.Name,
					description: t.Description,
					inputSchema: t.InputSchema,
					client:      client,
				}
				loadedTools = append(loadedTools, mcpTool)
			}
		}
	}

	return loadedTools, nil
}
