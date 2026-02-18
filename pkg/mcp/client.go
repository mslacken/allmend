package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
)

// Client is an MCP client.
type Client struct {
	Transport Transport
	IDCounter int
	mu        sync.Mutex
}

// NewClient creates a new MCP client with the given transport.
func NewClient(transport Transport) *Client {
	return &Client{
		Transport: transport,
		IDCounter: 1,
	}
}

func (c *Client) nextID() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	id := c.IDCounter
	c.IDCounter++
	return id
}

// Call sends a JSON-RPC request.
func (c *Client) Call(ctx context.Context, method string, params any, result any) error {
	reqBody := JSONRPCRequest{
		JSONRPC: "2.0",
		Method:  method,
		Params:  params,
		ID:      c.nextID(),
	}

	resp, err := c.Transport.Send(ctx, reqBody)
	if err != nil {
		return err
	}

	if resp.Error != nil {
		return fmt.Errorf("RPC error %d: %s", resp.Error.Code, resp.Error.Message)
	}

	if result == nil {
		return nil
	}

	// Re-marshal result to unmarshal into specific type
	resultData, err := json.Marshal(resp.Result)
	if err != nil {
		return fmt.Errorf("remarshaling result: %w", err)
	}

	if err := json.Unmarshal(resultData, result); err != nil {
		return fmt.Errorf("unmarshaling result data: %w", err)
	}

	return nil
}

// Initialize performs the handshake.
func (c *Client) Initialize(ctx context.Context) (*InitializeResult, error) {
	// Initialize transport if needed (e.g. start process)
	if err := c.Transport.Initialize(ctx); err != nil {
		return nil, fmt.Errorf("initializing transport: %w", err)
	}

	params := map[string]any{
		"protocolVersion": "2024-11-05",
		"capabilities": map[string]any{
			"roots": map[string]any{
				"listChanged": true,
			},
		},
		"clientInfo": map[string]string{
			"name":    "allmend-cli",
			"version": "0.1.0",
		},
	}
	var res InitializeResult
	if err := c.Call(ctx, "initialize", params, &res); err != nil {
		return nil, err
	}
	// Notify initialized
	if err := c.Call(ctx, "notifications/initialized", nil, nil); err != nil {
		// Just log warning?
	}
	return &res, nil
}

// ListTools fetches the list of tools.
func (c *Client) ListTools(ctx context.Context) ([]Tool, error) {
	var res ListToolsResult
	if err := c.Call(ctx, "tools/list", nil, &res); err != nil {
		return nil, err
	}
	return res.Tools, nil
}

// Close closes the underlying transport.
func (c *Client) Close() error {
	return c.Transport.Close()
}
