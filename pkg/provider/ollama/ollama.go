package ollama

import (
	"context"
	"encoding/json"
	"fmt"
	"iter"
	"net/http"
	"net/url"
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
	ctxLen int
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

			if role == "assistant" {
				var textContent string
				var toolCalls []api.ToolCall

				for _, part := range content.Parts {
					if part.Text != "" {
						textContent += part.Text
					}
					if part.FunctionCall != nil {
						args := api.NewToolCallFunctionArguments()
						for k, v := range part.FunctionCall.Args {
							args.Set(k, v)
						}
						toolCalls = append(toolCalls, api.ToolCall{
							Function: api.ToolCallFunction{
								Name:      part.FunctionCall.Name,
								Arguments: args,
							},
						})
					}
				}
				messages = append(messages, api.Message{
					Role:      role,
					Content:   textContent,
					ToolCalls: toolCalls,
				})
			} else {
				// User or Function/Tool role
				// ADK sends FunctionResponse with Role="user"
				for _, part := range content.Parts {
					if part.FunctionResponse != nil {
						// Convert FunctionResponse to Tool message
						var toolResult string
						res := part.FunctionResponse.Response
						if c, ok := res["content"]; ok {
							toolResult = fmt.Sprintf("%v", c)
						} else {
							resBytes, _ := json.Marshal(res)
							toolResult = string(resBytes)
						}

						messages = append(messages, api.Message{
							Role:    "tool",
							Content: toolResult,
						})
					} else if part.Text != "" {
						messages = append(messages, api.Message{
							Role:    "user",
							Content: part.Text,
						})
					}
				}
			}
		}

		// Configure Options once
		options := make(map[string]interface{})
		var format json.RawMessage
		var keepAlive *api.Duration
		var think *api.ThinkValue
		var truncate, shift, logprobs *bool
		var topLogprobs int

		for k, v := range p.config {
			switch k {
			case "format":
				if s, ok := v.(string); ok {
					format = json.RawMessage(fmt.Sprintf("%q", s))
				}
			case "keep_alive":
				if s, ok := v.(string); ok {
					if d, err := time.ParseDuration(s); err == nil {
						keepAlive = &api.Duration{Duration: d}
					}
				} else if f, ok := v.(float64); ok {
					keepAlive = &api.Duration{Duration: time.Duration(f * float64(time.Second))}
				}
			case "think":
				if b, ok := v.(bool); ok {
					think = &api.ThinkValue{Value: b}
				} else if s, ok := v.(string); ok {
					think = &api.ThinkValue{Value: s}
				}
			case "truncate":
				if b, ok := v.(bool); ok {
					truncate = &b
				}
			case "shift":
				if b, ok := v.(bool); ok {
					shift = &b
				}
			case "logprobs":
				if b, ok := v.(bool); ok {
					logprobs = &b
				}
			case "top_logprobs":
				if i, ok := v.(int); ok {
					topLogprobs = i
				} else if f, ok := v.(float64); ok {
					topLogprobs = int(f)
				}
			default:
				options[k] = v
			}
		}

		var tools []api.Tool
		if req.Config != nil && len(req.Config.Tools) > 0 {
			for _, t := range req.Config.Tools {
				for _, fd := range t.FunctionDeclarations {
					schemaMap := schemaToMap(fd.Parameters)
					data, err := json.Marshal(schemaMap)
					if err != nil {
						yield(nil, fmt.Errorf("failed to marshal tool parameters: %w", err))
						return
					}
					var params api.ToolFunctionParameters
					if err := json.Unmarshal(data, &params); err != nil {
						yield(nil, fmt.Errorf("failed to unmarshal tool parameters to ollama struct: %w", err))
						return
					}

					tools = append(tools, api.Tool{
						Type: "function",
						Function: api.ToolFunction{
							Name:        fd.Name,
							Description: fd.Description,
							Parameters:  params,
						},
					})
				}
			}
		}

		chatReq := &api.ChatRequest{
			Model:       p.model,
			Messages:    messages,
			Stream:      &stream,
			Format:      format,
			KeepAlive:   keepAlive,
			Think:       think,
			Truncate:    truncate,
			Shift:       shift,
			Logprobs:    logprobs != nil && *logprobs,
			TopLogprobs: topLogprobs,
			Options:     options,
			Tools:       tools,
		}

		err := p.client.Chat(ctx, chatReq, func(resp api.ChatResponse) error {
			llmResp := &model.LLMResponse{
				Content: &genai.Content{
					Role: "model",
					Parts: []*genai.Part{},
				},
				TurnComplete: resp.Done,
			}

			// Handle Thinking
			if resp.Message.Thinking != "" {
				llmResp.Content.Parts = append(llmResp.Content.Parts, &genai.Part{Text: fmt.Sprintf("<thinking>%s</thinking>", resp.Message.Thinking)})
			}

			if resp.Message.Content != "" {
				llmResp.Content.Parts = append(llmResp.Content.Parts, &genai.Part{Text: resp.Message.Content})
			}

			// Handle Tool Calls
			for _, tc := range resp.Message.ToolCalls {
				var argsMap map[string]any
				data, err := json.Marshal(tc.Function.Arguments)
				if err == nil {
					_ = json.Unmarshal(data, &argsMap)
				}

				llmResp.Content.Parts = append(llmResp.Content.Parts, &genai.Part{
					FunctionCall: &genai.FunctionCall{
						Name: tc.Function.Name,
						Args: argsMap,
					},
				})
			}

			if len(llmResp.Content.Parts) > 0 {
				if !yield(llmResp, nil) {
					return yieldErr
				}
			}

			if resp.DoneReason == "length" {
				if !yield(nil, fmt.Errorf("context overrun")) {
					return yieldErr
				}
				return nil
			}

			return nil
		})

		if err != nil {
			if err == yieldErr {
				return
			}
			yield(nil, err)
			return
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

	supportedKeys, allValidKeys := getSupportedOptions()
	warnings, info := validateConfig(config, supportedKeys, allValidKeys)
	return warnings, info, nil
}

// GetSupportedOptions returns a list of supported configuration options for the model.
func (p *Provider) GetSupportedOptions(ctx context.Context, name string) ([]string, error) {
	_, allValidKeys := getSupportedOptions()
	return allValidKeys, nil
}

func getSupportedOptions() (map[string]bool, []string) {
	supportedKeys := make(map[string]bool)
	var allValidKeys []string

	// Helper to extract JSON tags
	extractTags := func(t reflect.Type) {
		for i := 0; i < t.NumField(); i++ {
			field := t.Field(i)
			tag := field.Tag.Get("json")
			if tag != "" {
				// Extract the key name (before comma if any)
				key := tag
				if idx := strings.Index(tag, ","); idx != -1 {
					key = tag[:idx]
				}
				// Skip internal or non-configurable keys
				if key == "-" || key == "model" || key == "messages" || key == "stream" || key == "tools" || key == "options" || strings.HasPrefix(key, "_") {
					continue
				}
				if !supportedKeys[key] {
					supportedKeys[key] = true
					allValidKeys = append(allValidKeys, key)
				}
			}
		}
	}

	extractTags(reflect.TypeOf(api.Options{}))
	extractTags(reflect.TypeOf(api.ChatRequest{}))

	sort.Strings(allValidKeys)
	return supportedKeys, allValidKeys
}

// validateConfig validates the configuration against valid Ollama API parameters.
// It returns a list of warnings (for invalid parameters) and info (for valid but unset parameters).
func validateConfig(config map[string]interface{}, supportedKeys map[string]bool, allValidKeys []string) ([]string, []string) {
	var warnings []string
	var info []string

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
		info = append(info, unsetParams...)
	}

	return warnings, info
}

func schemaToMap(s *genai.Schema) map[string]interface{} {
	if s == nil {
		return map[string]interface{}{
			"type":       "object",
			"properties": map[string]interface{}{},
		}
	}

	m := schemaToMapRecursive(s)
	// Ensure root has type object if missing (though usually it is)
	if _, ok := m["type"]; !ok {
		m["type"] = "object"
	}
	return m
}

func schemaToMapRecursive(s *genai.Schema) map[string]interface{} {
	if s == nil {
		return nil
	}
	m := make(map[string]interface{})
	if s.Type != "" {
		m["type"] = string(s.Type)
	}
	if s.Description != "" {
		m["description"] = s.Description
	}
	if len(s.Enum) > 0 {
		m["enum"] = s.Enum
	}
	if s.Items != nil {
		m["items"] = schemaToMapRecursive(s.Items)
	}
	if len(s.Properties) > 0 {
		props := make(map[string]interface{})
		for k, v := range s.Properties {
			props[k] = schemaToMapRecursive(v)
		}
		m["properties"] = props
	}
	if len(s.Required) > 0 {
		m["required"] = s.Required
	}
	return m
}
