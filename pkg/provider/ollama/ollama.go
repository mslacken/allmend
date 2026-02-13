package ollama

import (
	"context"
	"fmt"
	"iter"
	"net/http"
	"net/url"

	"github.com/ollama/ollama/api"
	"google.golang.org/adk/model"
	"google.golang.org/genai"
)

var yieldErr = fmt.Errorf("yield stopped")

// Provider implements the model.LLM interface for Ollama.
type Provider struct {
	client *api.Client
	model  string
}

// New creates a new Ollama provider.
func New(endpoint, modelName string) (*Provider, error) {
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
	}, nil
}

// Name returns the name of the model.
func (p *Provider) Name() string {
	return p.model
}

// GenerateContent generates content from the model.
func (p *Provider) GenerateContent(ctx context.Context, req *model.LLMRequest, stream bool) iter.Seq2[*model.LLMResponse, error] {
	return func(yield func(*model.LLMResponse, error) bool) {
		messages := make([]api.Message, 0, len(req.Contents))
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
		}

		err := p.client.Chat(ctx, chatReq, func(resp api.ChatResponse) error {
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
