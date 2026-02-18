package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// HTTPTransport implements Transport over HTTP.
type HTTPTransport struct {
	BaseURL string
	Client  *http.Client
}

// NewHTTPTransport creates a new HTTP transport.
func NewHTTPTransport(baseURL string) *HTTPTransport {
	return &HTTPTransport{
		BaseURL: baseURL,
		Client:  &http.Client{},
	}
}

func (t *HTTPTransport) Initialize(ctx context.Context) error {
	// HTTP is stateless, no init needed beyond URL check perhaps.
	return nil
}

func (t *HTTPTransport) Send(ctx context.Context, request JSONRPCRequest) (JSONRPCResponse, error) {
	var resp JSONRPCResponse

	data, err := json.Marshal(request)
	if err != nil {
		return resp, fmt.Errorf("marshaling request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", t.BaseURL, bytes.NewReader(data))
	if err != nil {
		return resp, fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	httpResp, err := t.Client.Do(req)
	if err != nil {
		return resp, fmt.Errorf("sending request: %w", err)
	}
	defer httpResp.Body.Close()

	if httpResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(httpResp.Body)
		return resp, fmt.Errorf("server returned error %d: %s", httpResp.StatusCode, string(body))
	}

	respBody, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return resp, fmt.Errorf("reading response body: %w", err)
	}

	if err := json.Unmarshal(respBody, &resp); err != nil {
		return resp, fmt.Errorf("unmarshaling response: %w", err)
	}

	return resp, nil
}

func (t *HTTPTransport) Close() error {
	return nil
}
