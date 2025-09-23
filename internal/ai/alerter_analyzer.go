package ai

import (
	"Go2NetSpectra/internal/config"
	"context"
	"errors"
	"fmt"

	"github.com/sashabaranov/go-openai"
)

// AlerterAnalyzer implements the Analyzer interface using OpenAI's API
type AlerterAnalyzer struct {
	cfg    *config.AIConfig
	client *openai.Client
}

// NewAlerterAnalyzer creates a new instance of AlerterAnalyzer.
func NewAlerterAnalyzer(cfg *config.AIConfig) (*AlerterAnalyzer, error) {
	if cfg.APIKey == "" {
		return nil, fmt.Errorf("AI API key is not configured")
	}

	// Create a default OpenAI configuration
	clientConfig := openai.DefaultConfig(cfg.APIKey)

	// If a custom BaseURL is defined, override the default one
	if cfg.BaseURL != "" {
		clientConfig.BaseURL = cfg.BaseURL
	}

	// Create the client using the final configuration
	client := openai.NewClientWithConfig(clientConfig)

	return &AlerterAnalyzer{
		cfg:    cfg,
		client: client,
	}, nil
}

// AnalyzeTraffic analyzes the input text and returns a summary or insights.
func (a *AlerterAnalyzer) AnalyzeTraffic(ctx context.Context, input string) (string, error) {
	// Craft the prompt for the AI model
	prompt := fmt.Sprintf(
		"You are a senior network security analyst. "+
			"Please analyze the following network alert summary from the Go2NetSpectra monitoring system. "+
			"Provide a concise analysis of the potential threat, its severity, and recommended next steps for investigation. "+
			"The output should be clear and actionable.\n\n"+
			"--- Alert Data ---\n%s\n--- End of Alert Data ---", input,
	)

	resp, err := a.client.CreateChatCompletion(
		ctx,
		openai.ChatCompletionRequest{
			Model: a.cfg.Model,
			Messages: []openai.ChatCompletionMessage{
				{
					Role:    openai.ChatMessageRoleUser,
					Content: prompt,
				},
			},
		},
	)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			return "", fmt.Errorf("AI request timeout: %w", err)
		}
		if errors.Is(err, context.Canceled) {
			return "", fmt.Errorf("AI request canceled by client: %w", err)
		}
		return "", fmt.Errorf("OpenAI API error: %w", err)
	}

	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("OpenAI API returned no choices")
	}

	return resp.Choices[0].Message.Content, nil
}
