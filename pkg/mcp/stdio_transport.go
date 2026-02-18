package mcp

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"
)

// StdioTransport implements Transport over stdio with a subprocess.
type StdioTransport struct {
	Command []string
	Env     []string

	cmd    *exec.Cmd
	stdin  io.WriteCloser
	stdout io.ReadCloser
	stderr io.ReadCloser

	mu        sync.Mutex
	responses map[int]chan JSONRPCResponse
	done      chan struct{}
}

// NewStdioTransport creates a new Stdio transport.
func NewStdioTransport(command []string, env []string) *StdioTransport {
	return &StdioTransport{
		Command:   command,
		Env:       env,
		responses: make(map[int]chan JSONRPCResponse),
		done:      make(chan struct{}),
	}
}

func (t *StdioTransport) Initialize(ctx context.Context) error {
	if len(t.Command) == 0 {
		return fmt.Errorf("empty command")
	}

	cmd := exec.CommandContext(ctx, t.Command[0], t.Command[1:]...)
	cmd.Env = append(os.Environ(), t.Env...)
	
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("creating stdin pipe: %w", err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("creating stdout pipe: %w", err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("creating stderr pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("starting command: %w", err)
	}

	t.cmd = cmd
	t.stdin = stdin
	t.stdout = stdout
	t.stderr = stderr

	// Start reading stdout
	go t.readLoop()
	// Start reading stderr
	go t.logStderr()

	return nil
}

func (t *StdioTransport) readLoop() {
	scanner := bufio.NewScanner(t.stdout)
	for scanner.Scan() {
		line := scanner.Bytes()
		var resp JSONRPCResponse
		if err := json.Unmarshal(line, &resp); err != nil {
			// Log error or ignore non-JSON lines?
			// For now, assume strict MCP.
			// fmt.Fprintf(os.Stderr, "Error unmarshaling JSON-RPC from stdout: %v | Line: %s\n", err, string(line))
			continue
		}

		t.mu.Lock()
		ch, ok := t.responses[resp.ID]
		if ok {
			delete(t.responses, resp.ID)
		}
		t.mu.Unlock()

		if ok {
			ch <- resp
			close(ch)
		} else {
            // Handle notifications or requests from server if needed?
            // For now, ignore.
        }
	}
	close(t.done)
}

func (t *StdioTransport) logStderr() {
	scanner := bufio.NewScanner(t.stderr)
	for scanner.Scan() {
		// Log stderr to our stderr, maybe prefixed
		// fmt.Fprintf(os.Stderr, "[MCP Stdio]: %s\n", scanner.Text())
	}
}

func (t *StdioTransport) Send(ctx context.Context, request JSONRPCRequest) (JSONRPCResponse, error) {
	t.mu.Lock()
	if t.cmd == nil {
		t.mu.Unlock()
		return JSONRPCResponse{}, fmt.Errorf("transport not initialized")
	}
	ch := make(chan JSONRPCResponse, 1)
	t.responses[request.ID] = ch
	t.mu.Unlock()

	data, err := json.Marshal(request)
	if err != nil {
		return JSONRPCResponse{}, fmt.Errorf("marshaling request: %w", err)
	}

	// Append newline as delimiter
	data = append(data, '\n')

	if _, err := t.stdin.Write(data); err != nil {
		return JSONRPCResponse{}, fmt.Errorf("writing to stdin: %w", err)
	}

	select {
	case resp := <-ch:
		return resp, nil
	case <-ctx.Done():
		return JSONRPCResponse{}, ctx.Err()
	case <-t.done:
		return JSONRPCResponse{}, fmt.Errorf("transport closed unexpectedly")
	}
}

func (t *StdioTransport) Close() error {
	if t.stdin != nil {
		t.stdin.Close()
	}
	if t.cmd != nil && t.cmd.Process != nil {
		return t.cmd.Process.Kill()
	}
	return nil
}
