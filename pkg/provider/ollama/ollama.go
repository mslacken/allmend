package ollama

import (
	"context"
	"fmt"
	"iter"
	"net/http"
	"net/url"
	"os"
	"reflect"
	"sort"
	"strings"
	"time"

	"github.com/ollama/ollama/api"
	"google.golang.org/adk/model"
	"google.golang.org/genai"
)

var yieldErr = fmt.Errorf("yield stopped")

// Provider implements the model.LLM interface for Ollama.
type Provider struct {
	client *api.Client
	model  string
	config map[string]interface{}
}

// New creates a new Ollama provider.
func New(endpoint, modelName string, config map[string]interface{}) (*Provider, error) {
	url, err := url.Parse(endpoint)
	if err != nil {
		return nil, fmt.Errorf("invalid ollama endpoint: %w", err)
	}
	// api.NewClient requires an http.Client. If nil is passed, it might cause panic in some versions.
	// Safe to pass http.DefaultClient.
	client := api.NewClient(url, http.DefaultClient)
	return &Provider{
		client: client,
		model:  modelName,
		config: config,
	}, nil
}

// Name returns the name of the model.
func (p *Provider) Name() string {
	return p.model
}

// GenerateContent generates content from the model.
func (p *Provider) GenerateContent(ctx context.Context, req *model.LLMRequest, stream bool) iter.Seq2[*model.LLMResponse, error] {
	return func(yield func(*model.LLMResponse, error) bool) {
		messages := make([]api.Message, 0, len(req.Contents)+1)

		if req.Config != nil && req.Config.SystemInstruction != nil {
			var systemText string
			for _, part := range req.Config.SystemInstruction.Parts {
				if part.Text != "" {
					systemText += part.Text
				}
			}
			if systemText != "" {
				messages = append(messages, api.Message{
					Role:    "system",
					Content: systemText,
				})
			}
		}

		for _, content := range req.Contents {
			role := content.Role
			// Map genai roles to ollama roles
			if role == "model" {
				role = "assistant"
			}

			var textContent string
			for _, part := range content.Parts {
				if part.Text != "" {
					textContent += part.Text
				}
				// TODO: Handle other part types like images if needed
			}

			messages = append(messages, api.Message{
				Role:    role,
				Content: textContent,
			})
		}

		chatReq := &api.ChatRequest{
			Model:    p.model,
			Messages: messages,
			Stream:   &stream,
			Options:  p.config,
		}

		var debugSession string
		var newInput string
		if os.Getenv("ALLMEND_DEBUG_OLLAMA") != "" {
			if len(req.Contents) > 0 {
				last := req.Contents[len(req.Contents)-1]
				for _, part := range last.Parts {
					if part.Text != "" {
						newInput += part.Text
					}
				}
			}

			debugReq := *chatReq
			debugReq.DebugRenderOnly = true
			streamFalse := false
			debugReq.Stream = &streamFalse

			_ = p.client.Chat(ctx, &debugReq, func(resp api.ChatResponse) error {
				if resp.DebugInfo != nil {
					debugSession = resp.DebugInfo.RenderedTemplate
				}
				return nil
			})
		}

		var fullOutput string
		err := p.client.Chat(ctx, chatReq, func(resp api.ChatResponse) error {
			fullOutput += resp.Message.Content
			llmResp := &model.LLMResponse{
				Content: &genai.Content{
					Role: "model",
					Parts: []*genai.Part{
						{Text: resp.Message.Content},
					},
				},
				// Map other fields as best as possible
				TurnComplete: resp.Done,
			}

			if resp.Done {
				llmResp.FinishReason = genai.FinishReasonStop
				llmResp.UsageMetadata = &genai.GenerateContentResponseUsageMetadata{
					PromptTokenCount:     int32(resp.PromptEvalCount),
					CandidatesTokenCount: int32(resp.EvalCount),
					TotalTokenCount:      int32(resp.PromptEvalCount + resp.EvalCount),
				}

				if os.Getenv("ALLMEND_DEBUG_OLLAMA") != "" {
					f, err := os.OpenFile("ollama_debug.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
					if err == nil {
						defer f.Close()
						fmt.Fprintf(f, "--- [Debug %s] ---\nInput: %s\n\nSession:\n%s\n\nOutput: %s\n\nTokens: Prompt=%d, Eval=%d\n\n",
							time.Now().Format(time.RFC3339),
							newInput,
							debugSession,
							fullOutput,
							resp.PromptEvalCount,
							resp.EvalCount,
						)
					}
				}
			}

			if !yield(llmResp, nil) {
				return yieldErr
			}
			return nil
		})

		if err != nil && err != yieldErr {
			yield(nil, err)
		}
	}
}

// GetModells returns a list of available models.
func (p *Provider) GetModells(ctx context.Context) ([]string, error) {
	resp, err := p.client.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list ollama models: %w", err)
	}

	var models []string
	for _, m := range resp.Models {
		models = append(models, m.Name)
	}
	return models, nil
}

// CheckModel checks if the model supports the given configuration.
func (p *Provider) CheckModel(ctx context.Context, name string, config map[string]interface{}) ([]string, []string, error) {
	_, err := p.client.Show(ctx, &api.ShowRequest{Name: name})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to show ollama model '%s': %w", name, err)
	}

	var warnings []string
	var info []string
	
	// Dynamically build supported keys from api.Options struct tags
	supportedKeys := make(map[string]bool)
	var allValidKeys []string
	val := reflect.TypeOf(api.Options{})
	for i := 0; i < val.NumField(); i++ {
		field := val.Field(i)
		tag := field.Tag.Get("json")
		if tag != "" {
			// Extract the key name (before comma if any)
			key := tag
			if idx := strings.Index(tag, ","); idx != -1 {
				key = tag[:idx]
			}
			supportedKeys[key] = true
			allValidKeys = append(allValidKeys, key)
		}
	}
	sort.Strings(allValidKeys)

	for k := range config {
		if !supportedKeys[k] {
			warnings = append(warnings, fmt.Sprintf("Parameter '%s' is not a valid Ollama API parameter (must match exactly, e.g. 'temperature', 'top_k').", k))
		}
	}
	
	// Collect valid parameters that are NOT set in the config
	var unsetParams []string
	for _, key := range allValidKeys {
		if _, set := config[key]; !set {
			unsetParams = append(unsetParams, key)
		}
	}
	
	if len(unsetParams) > 0 {
		info = append(info, "Available valid parameters (unset):")
		// Format them nicely, maybe comma separated or just list them?
		// Given there are many, a comma separated list might be better for "info" return
		// or passing them as individual strings.
		// Let's pass them individually for the caller to format.
		info = append(info, unsetParams...)
	}

	return warnings, info, nil
}
