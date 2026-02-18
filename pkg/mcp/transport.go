package mcp

import (
	"context"
)

// Transport defines the interface for communicating with an MCP server.
type Transport interface {
	// Initialize performs any necessary setup (e.g., starting a process).
	Initialize(ctx context.Context) error
	// Send sends a JSON-RPC request and returns the response.
	// For stdio, this might block until a response with the same ID is received.
	Send(ctx context.Context, request JSONRPCRequest) (JSONRPCResponse, error)
	// Close closes the transport.
	Close() error
}
