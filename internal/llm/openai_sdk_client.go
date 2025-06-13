package llm

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
)

// OpenAISDKClient represents a client using the official OpenAI Go SDK
type OpenAISDKClient struct {
	client openai.Client
}

// NewOpenAISDKClient creates a new OpenAI SDK client
func NewOpenAISDKClient(apiKey string) *OpenAISDKClient {
	client := openai.NewClient(
		option.WithAPIKey(apiKey),
	)

	return &OpenAISDKClient{
		client: client,
	}
}

// NewOpenAISDKClientWithConfig creates a new OpenAI SDK client with custom configuration
func NewOpenAISDKClientWithConfig(apiKey, baseURL string, timeout time.Duration) *OpenAISDKClient {
	opts := []option.RequestOption{
		option.WithAPIKey(apiKey),
	}

	if baseURL != "" {
		opts = append(opts, option.WithBaseURL(baseURL))
	}

	if timeout > 0 {
		httpClient := &http.Client{Timeout: timeout}
		opts = append(opts, option.WithHTTPClient(httpClient))
	}

	client := openai.NewClient(opts...)

	return &OpenAISDKClient{
		client: client,
	}
}

// ChatCompletion creates a chat completion using OpenAI SDK
func (c *OpenAISDKClient) ChatCompletion(ctx context.Context, req *ChatRequest) (*ChatResponse, error) {
	// Convert our format to OpenAI SDK format
	messages := make([]openai.ChatCompletionMessageParamUnion, len(req.Messages))
	for i, msg := range req.Messages {
		switch msg.Role {
		case "system":
			messages[i] = openai.SystemMessage(msg.Content)
		case "user":
			messages[i] = openai.UserMessage(msg.Content)
		case "assistant":
			messages[i] = openai.AssistantMessage(msg.Content)
		default:
			messages[i] = openai.UserMessage(msg.Content)
		}
	}

	// Set default values
	model := req.Model
	if model == "" {
		model = "gpt-3.5-turbo"
	}

	temperature := req.Temperature
	if temperature == 0 {
		temperature = 0.7
	}

	maxTokens := req.MaxTokens
	if maxTokens == 0 {
		maxTokens = 1000
	}

	// Create the request parameters
	params := openai.ChatCompletionNewParams{
		Messages: messages,
		Model:    model,
	}

	// Add optional parameters using openai helper functions
	if maxTokens > 0 {
		params.MaxTokens = openai.Int(int64(maxTokens))
	}
	if temperature > 0 {
		params.Temperature = openai.Float(float64(temperature))
	}
	if req.TopP > 0 {
		params.TopP = openai.Float(float64(req.TopP))
	}
	if req.N > 0 {
		params.N = openai.Int(int64(req.N))
	}
	if len(req.Stop) > 0 {
		// For now, skip the stop parameter as it requires complex type conversion
		// params.Stop = ... // TODO: implement proper stop parameter conversion
	}
	if req.PresencePenalty != 0 {
		params.PresencePenalty = openai.Float(float64(req.PresencePenalty))
	}
	if req.FrequencyPenalty != 0 {
		params.FrequencyPenalty = openai.Float(float64(req.FrequencyPenalty))
	}
	if req.User != "" {
		params.User = openai.String(req.User)
	}

	// Make the API call
	completion, err := c.client.Chat.Completions.New(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("chat completion failed: %w", err)
	}

	// Convert response to our format
	choices := make([]Choice, len(completion.Choices))
	for i, choice := range completion.Choices {
		choices[i] = Choice{
			Index: i,
			Message: Message{
				Role:    string(choice.Message.Role),
				Content: choice.Message.Content,
			},
			FinishReason: string(choice.FinishReason),
		}
	}

	response := &ChatResponse{
		ID:      completion.ID,
		Object:  string(completion.Object),
		Created: completion.Created,
		Model:   completion.Model,
		Choices: choices,
		Usage: Usage{
			PromptTokens:     int(completion.Usage.PromptTokens),
			CompletionTokens: int(completion.Usage.CompletionTokens),
			TotalTokens:      int(completion.Usage.TotalTokens),
		},
	}

	return response, nil
}

// SimpleChat provides a simple interface for single-turn conversations
func (c *OpenAISDKClient) SimpleChat(ctx context.Context, userMessage string) (string, error) {
	req := &ChatRequest{
		Model: "gpt-3.5-turbo",
		Messages: []Message{
			{Role: "user", Content: userMessage},
		},
		MaxTokens:   1000,
		Temperature: 0.7,
	}

	resp, err := c.ChatCompletion(ctx, req)
	if err != nil {
		return "", err
	}

	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("no response choices returned")
	}

	return strings.TrimSpace(resp.Choices[0].Message.Content), nil
}

// ChatWithSystem provides a simple interface with system message
func (c *OpenAISDKClient) ChatWithSystem(ctx context.Context, systemMessage, userMessage string) (string, error) {
	req := &ChatRequest{
		Model: "gpt-3.5-turbo",
		Messages: []Message{
			{Role: "system", Content: systemMessage},
			{Role: "user", Content: userMessage},
		},
		MaxTokens:   1000,
		Temperature: 0.7,
	}

	resp, err := c.ChatCompletion(ctx, req)
	if err != nil {
		return "", err
	}

	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("no response choices returned")
	}

	return strings.TrimSpace(resp.Choices[0].Message.Content), nil
}

// ChatWithHistory provides conversation with message history
func (c *OpenAISDKClient) ChatWithHistory(ctx context.Context, messages []Message, userMessage string) (string, error) {
	// Add user message to history
	allMessages := append(messages, Message{Role: "user", Content: userMessage})

	req := &ChatRequest{
		Model:       "gpt-3.5-turbo",
		Messages:    allMessages,
		MaxTokens:   1000,
		Temperature: 0.7,
	}

	resp, err := c.ChatCompletion(ctx, req)
	if err != nil {
		return "", err
	}

	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("no response choices returned")
	}

	return strings.TrimSpace(resp.Choices[0].Message.Content), nil
}

// GetAvailableModels returns available OpenAI models
func (c *OpenAISDKClient) GetAvailableModels() []string {
	return []string{
		"gpt-4o",
		"gpt-4o-mini",
		"gpt-4-turbo",
		"gpt-4",
		"gpt-3.5-turbo",
		"gpt-3.5-turbo-16k",
	}
}

// ValidateAPIKey validates the API key by making a simple request
func (c *OpenAISDKClient) ValidateAPIKey(ctx context.Context) error {
	// Try to list models to validate the API key
	_, err := c.client.Models.List(ctx)
	if err != nil {
		if strings.Contains(err.Error(), "Invalid API key") ||
			strings.Contains(err.Error(), "Unauthorized") ||
			strings.Contains(err.Error(), "authentication") {
			return fmt.Errorf("invalid API key")
		}
		return fmt.Errorf("API key validation failed: %w", err)
	}
	return nil
}

// EstimateTokens provides a rough estimate of token count
func (c *OpenAISDKClient) EstimateTokens(text string) int {
	// Rough estimation: 1 token ≈ 0.75 words for English
	words := len(strings.Fields(text))
	tokens := int(float64(words) * 1.33)
	if tokens < 1 {
		tokens = 1
	}
	return tokens
}

// TruncateToTokenLimit truncates text to fit within token limit
func (c *OpenAISDKClient) TruncateToTokenLimit(text string, maxTokens int) string {
	if c.EstimateTokens(text) <= maxTokens {
		return text
	}

	// Simple truncation - could be improved with proper tokenization
	words := strings.Fields(text)
	estimatedWordsPerToken := 0.75
	maxWords := int(float64(maxTokens) * estimatedWordsPerToken)

	if len(words) <= maxWords {
		return text
	}

	return strings.Join(words[:maxWords], " ") + "..."
}

// Interface compatibility check
var _ LLMInterface = (*OpenAISDKClient)(nil)
