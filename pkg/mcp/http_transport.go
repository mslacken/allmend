package mcp

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"
)

// HTTPTransport implements Transport over HTTP with SSE support.
type HTTPTransport struct {
	BaseURL string
	PostURL string
	Client  *http.Client
	sseResp *http.Response
	cancel  context.CancelFunc

	pendingReqs map[int]chan JSONRPCResponse
	mu          sync.Mutex
}

// NewHTTPTransport creates a new HTTP transport.
func NewHTTPTransport(baseURL string) *HTTPTransport {
	return &HTTPTransport{
		BaseURL:     baseURL,
		Client:      &http.Client{},
		pendingReqs: make(map[int]chan JSONRPCResponse),
	}
}

func debugLog(format string, args ...any) {
	if os.Getenv("ALLMEND_DEBUG") != "" {
		fmt.Printf("[MCP-DEBUG] "+format+"\n", args...)
	}
}

// debugReader wraps an io.Reader and logs read data
type debugReader struct {
	r io.Reader
}

func (d *debugReader) Read(p []byte) (n int, err error) {
	n, err = d.r.Read(p)
	if n > 0 {
		debugLog("RX Chunk (%d bytes): %q", n, p[:n])
	}
	if err != nil && err != io.EOF {
		debugLog("RX Error: %v", err)
	}
	return
}

func (t *HTTPTransport) Initialize(ctx context.Context) error {
	// Create a context for the SSE connection that will be canceled on Close()
	sseCtx, cancel := context.WithCancel(context.Background())
	t.cancel = cancel

	debugLog("Connecting to SSE endpoint: %s", t.BaseURL)

	req, err := http.NewRequestWithContext(sseCtx, "GET", t.BaseURL, nil)
	if err != nil {
		cancel()
		return fmt.Errorf("creating SSE request: %w", err)
	}
	req.Header.Set("Accept", "text/event-stream")
	req.Header.Set("Cache-Control", "no-cache")
	req.Header.Set("User-Agent", "allmend-cli/0.1.0")

	resp, err := t.Client.Do(req)
	if err != nil {
		cancel()
		return fmt.Errorf("connecting to SSE endpoint: %w", err)
	}

	debugLog("SSE Connected. Status: %d. Content-Type: %s Proto: %s", resp.StatusCode, resp.Header.Get("Content-Type"), resp.Proto)

	if resp.StatusCode == http.StatusMethodNotAllowed {
		debugLog("SSE endpoint returned 405 (Method Not Allowed). Falling back to direct POST mode.")
		resp.Body.Close()
		return nil
	}

	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		cancel()
		return fmt.Errorf("SSE endpoint returned status %d", resp.StatusCode)
	}

	t.sseResp = resp

	// Read events to find the endpoint
	// Use a larger buffer for scanning to handle potential large data fields
	// Wrap body in debugReader if debug is on
	var bodyReader io.Reader = resp.Body
	if os.Getenv("ALLMEND_DEBUG") != "" {
		bodyReader = &debugReader{r: resp.Body}
	}

	scanner := bufio.NewScanner(bodyReader)

	// Channel to signal endpoint discovery
	endpointFound := make(chan string, 1)
	errChan := make(chan error, 1)

	go func() {
		// Only close errChan if we exit with an error before finding endpoint
		// or if we exit after finding endpoint (connection closed).
		// endpointFound should be closed once found.
		defer func() {
			// If we exit and endpoint hasn't been found/closed yet, close it
			select {
			case <-endpointFound:
			default:
				close(endpointFound)
			}
			close(errChan)
		}()

		var eventType string
		var eventData strings.Builder

		debugLog("Starting scan loop...")
		for scanner.Scan() {
			line := scanner.Text()
			debugLog("Scan line: %q", line)

			if line == "" {
				debugLog("Processing event block: type=%q data=%q", eventType, eventData.String())

				if eventType == "endpoint" {
					debugLog("Found endpoint event: %s", eventData.String())
					// Non-blocking send in case it's already handled
					select {
					case endpointFound <- eventData.String():
						close(endpointFound) // Signal we are done finding endpoint
					default:
					}
				} else if eventType == "message" {
					debugLog("Found message event: %s", eventData.String())
					// Handle JSON-RPC message
					var rpcResp JSONRPCResponse
					if err := json.Unmarshal([]byte(eventData.String()), &rpcResp); err != nil {
						debugLog("Error unmarshaling SSE message: %v", err)
					} else {
						// Look up pending request
						t.mu.Lock()
						ch, ok := t.pendingReqs[rpcResp.ID]
						if ok {
							delete(t.pendingReqs, rpcResp.ID)
						}
						t.mu.Unlock()

						if ok {
							// Send response to waiting caller
							select {
							case ch <- rpcResp:
							default:
								debugLog("Warning: response channel blocked/closed for ID %d", rpcResp.ID)
							}
							close(ch)
						} else {
							debugLog("Received response for unknown or already handled ID: %d", rpcResp.ID)
						}
					}
				}

				// Reset
				eventType = ""
				eventData.Reset()
				continue
			}

			parts := strings.SplitN(line, ":", 2)
			if len(parts) < 2 {
				continue
			}
			field := strings.TrimSpace(parts[0])
			value := ""
			if len(parts) > 1 {
				value = strings.TrimPrefix(parts[1], " ")
			}

			switch field {
			case "event":
				eventType = value
			case "data":
				eventData.WriteString(value)
			}
		}
		if err := scanner.Err(); err != nil {
			// If context was canceled (e.g. by Close()), this is expected
			select {
			case <-sseCtx.Done():
				return
			default:
				debugLog("Scanner error: %v", err)
				// Only send error if we are still initializing
				select {
				case errChan <- err:
				default:
				}
			}
		} else {
			// Scanner finished (EOF).
			debugLog("SSE stream closed (EOF). Pending event: type=%q data_len=%d", eventType, eventData.Len())
			if eventType == "endpoint" && eventData.Len() > 0 {
				select {
				case endpointFound <- eventData.String():
					close(endpointFound)
				default:
				}
			}
		}
	}()

	// Wait for endpoint or timeout
	select {
	case endpoint := <-endpointFound:
		if endpoint == "" {
			return fmt.Errorf("SSE connection closed without providing an endpoint")
		}

		// Parse the endpoint URL
		u, err := url.Parse(endpoint)
		if err != nil {
			return fmt.Errorf("invalid endpoint URL received: %w", err)
		}

		// Resolve relative URL against BaseURL
		base, err := url.Parse(t.BaseURL)
		if err != nil {
			return fmt.Errorf("invalid base URL: %w", err)
		}
		t.PostURL = base.ResolveReference(u).String()
		debugLog("Resolved POST URL: %s", t.PostURL)

		return nil
	case err := <-errChan:
		return fmt.Errorf("error reading SSE stream: %w", err)
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(30 * time.Second):
		return fmt.Errorf("timeout waiting for endpoint event")
	}
}

func (t *HTTPTransport) Send(ctx context.Context, request JSONRPCRequest) (JSONRPCResponse, error) {
	var resp JSONRPCResponse

	targetURL := t.PostURL
	if targetURL == "" {
		// Fallback for non-SSE or before init
		targetURL = t.BaseURL
	}

	debugLog("Sending JSON-RPC request to %s: method=%s id=%d", targetURL, request.Method, request.ID)

	// Register pending request
	respChan := make(chan JSONRPCResponse, 1)
	t.mu.Lock()
	t.pendingReqs[request.ID] = respChan
	t.mu.Unlock()

	// Ensure we clean up if something goes wrong or we return early
	// Note: If we receive the response via SSE, we delete from map there.
	// If we receive via POST (200), we delete here.
	defer func() {
		t.mu.Lock()
		if _, ok := t.pendingReqs[request.ID]; ok {
			delete(t.pendingReqs, request.ID)
		}
		t.mu.Unlock()
	}()

	data, err := json.Marshal(request)
	if err != nil {
		return resp, fmt.Errorf("marshaling request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", targetURL, bytes.NewReader(data))
	if err != nil {
		return resp, fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")

	httpResp, err := t.Client.Do(req)
	if err != nil {
		return resp, fmt.Errorf("sending request: %w", err)
	}
	defer httpResp.Body.Close()

	if httpResp.StatusCode == http.StatusAccepted {
		debugLog("Server returned 202 Accepted. Waiting for response via SSE...")
		// Wait for response on channel
		select {
		case r := <-respChan:
			return r, nil
		case <-ctx.Done():
			return resp, ctx.Err()
		case <-time.After(30 * time.Second):
			return resp, fmt.Errorf("timeout waiting for SSE response")
		}
	}

	if httpResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(httpResp.Body)
		return resp, fmt.Errorf("server returned error %d: %s", httpResp.StatusCode, string(body))
	}

	respBody, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return resp, fmt.Errorf("reading response body: %w", err)
	}

	debugLog("Raw response body: %q", string(respBody))

	if err := json.Unmarshal(respBody, &resp); err != nil {
		return resp, fmt.Errorf("unmarshaling response: %w. Body: %q", err, string(respBody))
	}

	debugLog("Received JSON-RPC response: id=%d error=%v", resp.ID, resp.Error)

	return resp, nil
}

func (t *HTTPTransport) Close() error {
	if t.cancel != nil {
		t.cancel()
	}
	if t.sseResp != nil && t.sseResp.Body != nil {
		return t.sseResp.Body.Close()
	}
	return nil
}
