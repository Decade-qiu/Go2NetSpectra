package ai

import (
	"Go2NetSpectra/internal/config"
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/sashabaranov/go-openai"
)

// CommonAnalyzer is a general-purpose AI analyzer that supports streaming.
type CommonAnalyzer struct {
	cfg    *config.AIConfig
	client *openai.Client
}

// NewCommonAnalyzer creates a new instance of CommonAnalyzer.
func NewCommonAnalyzer(cfg *config.AIConfig) (*CommonAnalyzer, error) {
	if cfg.APIKey == "" {
		return nil, fmt.Errorf("AI API key is not configured")
	}
	clientConfig := openai.DefaultConfig(cfg.APIKey)
	if cfg.BaseURL != "" {
		clientConfig.BaseURL = cfg.BaseURL
	}
	client := openai.NewClientWithConfig(clientConfig)
	return &CommonAnalyzer{cfg: cfg, client: client}, nil
}

// AnalyzeStream uses the OpenAI API to process a general-purpose prompt in a stream.
func (a *CommonAnalyzer) AnalyzeStream(ctx context.Context, prompt string, sendChunk func(string) error) error {
	req := openai.ChatCompletionRequest{
		Model:     a.cfg.Model,
		MaxTokens: 2048,
		Messages: []openai.ChatCompletionMessage{
			{
				Role:    openai.ChatMessageRoleUser,
				Content: prompt,
			},
		},
		Stream: true,
	}

	stream, err := a.client.CreateChatCompletionStream(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to create chat completion stream: %w", err)
	}
	defer stream.Close()

	for {
		response, err := stream.Recv()
		if errors.Is(err, io.EOF) {
			return nil // Stream finished successfully
		}
		if err != nil {
			return fmt.Errorf("stream error: %w", err)
		}

		// Get the text chunk from the stream response and send it
		chunk := response.Choices[0].Delta.Content
		if err := sendChunk(chunk); err != nil {
			// This error might occur if the client disconnects
			return fmt.Errorf("failed to send chunk to client: %w", err)
		}
	}
}
