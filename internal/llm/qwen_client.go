package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// QwenClient represents a client for Alibaba Qwen (通义千问) API
type QwenClient struct {
	apiKey     string
	baseURL    string
	httpClient *http.Client
}

// QwenMessage represents a chat message for Qwen API
type QwenMessage struct {
	Role    string `json:"role"`    // system, user, assistant
	Content string `json:"content"` // message content
}

// QwenChatRequest represents the request parameters for Qwen chat completion
type QwenChatRequest struct {
	Model       string        `json:"model"`                 // qwen-turbo, qwen-plus, qwen-max, etc.
	Messages    []QwenMessage `json:"messages"`              // conversation messages
	MaxTokens   int           `json:"max_tokens,omitempty"`  // maximum tokens to generate
	Temperature float32       `json:"temperature,omitempty"` // sampling temperature (0-2)
	TopP        float32       `json:"top_p,omitempty"`       // nucleus sampling (0-1)
	TopK        int           `json:"top_k,omitempty"`       // top-k sampling
	Stream      bool          `json:"stream,omitempty"`      // whether to stream responses
	Stop        []string      `json:"stop,omitempty"`        // stop sequences
}

// QwenChatResponse represents the response from Qwen chat completion
type QwenChatResponse struct {
	Output struct {
		Text         string `json:"text"`
		FinishReason string `json:"finish_reason"`
	} `json:"output"`
	Usage struct {
		OutputTokens int `json:"output_tokens"`
		InputTokens  int `json:"input_tokens"`
		TotalTokens  int `json:"total_tokens"`
	} `json:"usage"`
	RequestID string `json:"request_id"`
}

// QwenErrorResponse represents an error response from Qwen API
type QwenErrorResponse struct {
	Code      string `json:"code"`
	Message   string `json:"message"`
	RequestID string `json:"request_id"`
}

// NewQwenClient creates a new Qwen client
func NewQwenClient(apiKey string) *QwenClient {
	return &QwenClient{
		apiKey:  apiKey,
		baseURL: "https://dashscope.aliyuncs.com/compatible-mode/v1/chat/completions",
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

// NewQwenClientWithConfig creates a new Qwen client with custom configuration
func NewQwenClientWithConfig(apiKey, baseURL string, timeout time.Duration) *QwenClient {
	return &QwenClient{
		apiKey:  apiKey,
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}
}

// ChatCompletion creates a chat completion using Qwen API
func (c *QwenClient) ChatCompletion(ctx context.Context, req *ChatRequest) (*ChatResponse, error) {
	// Convert OpenAI format to Qwen format
	qwenReq := &QwenChatRequest{
		Model:       convertModelToQwen(req.Model),
		Messages:    convertMessagesToQwen(req.Messages),
		MaxTokens:   req.MaxTokens,
		Temperature: req.Temperature,
		TopP:        req.TopP,
		Stream:      req.Stream,
		Stop:        req.Stop,
	}

	// Set default values
	if qwenReq.Model == "" {
		qwenReq.Model = "qwen-plus"
	}
	if qwenReq.Temperature == 0 {
		qwenReq.Temperature = 0.7
	}
	if qwenReq.MaxTokens == 0 {
		qwenReq.MaxTokens = 1000
	}

	// Marshal request
	reqBody, err := json.Marshal(qwenReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/services/aigc/text-generation/generation", bytes.NewReader(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	// Set headers for Qwen API
	httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("X-DashScope-SSE", "disable") // Disable SSE for non-streaming

	// Send request
	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// Handle error responses
	if resp.StatusCode != http.StatusOK {
		var errorResp QwenErrorResponse
		if err := json.Unmarshal(body, &errorResp); err != nil {
			return nil, fmt.Errorf("chat completion failed with status %d: %s", resp.StatusCode, string(body))
		}
		return nil, fmt.Errorf("chat completion failed: %s (code: %s, request_id: %s)",
			errorResp.Message, errorResp.Code, errorResp.RequestID)
	}

	// Parse Qwen response
	var qwenResp QwenChatResponse
	if err := json.Unmarshal(body, &qwenResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// Convert Qwen response to OpenAI format
	chatResp := &ChatResponse{
		ID:      qwenResp.RequestID,
		Object:  "chat.completion",
		Created: time.Now().Unix(),
		Model:   qwenReq.Model,
		Choices: []Choice{
			{
				Index: 0,
				Message: Message{
					Role:    "assistant",
					Content: qwenResp.Output.Text,
				},
				FinishReason: qwenResp.Output.FinishReason,
			},
		},
		Usage: Usage{
			PromptTokens:     qwenResp.Usage.InputTokens,
			CompletionTokens: qwenResp.Usage.OutputTokens,
			TotalTokens:      qwenResp.Usage.TotalTokens,
		},
	}

	return chatResp, nil
}

// SimpleChat provides a simple interface for single-turn conversations
func (c *QwenClient) SimpleChat(ctx context.Context, userMessage string) (string, error) {
	req := &ChatRequest{
		Model: "qwen-plus",
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
func (c *QwenClient) ChatWithSystem(ctx context.Context, systemMessage, userMessage string) (string, error) {
	req := &ChatRequest{
		Model: "qwen-plus",
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
func (c *QwenClient) ChatWithHistory(ctx context.Context, messages []Message, userMessage string) (string, error) {
	// Add user message to history
	allMessages := append(messages, Message{Role: "user", Content: userMessage})

	req := &ChatRequest{
		Model:       "qwen-plus",
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

// GetAvailableModels returns available Qwen models
func (c *QwenClient) GetAvailableModels() []string {
	return []string{
		"qwen-turbo",
		"qwen-plus",
		"qwen-max",
		"qwen-max-1201",
		"qwen-max-longcontext",
		"qwen2.5-72b-instruct",
		"qwen2.5-32b-instruct",
		"qwen2.5-14b-instruct",
		"qwen2.5-7b-instruct",
		"qwen2.5-3b-instruct",
		"qwen2.5-1.5b-instruct",
		"qwen2.5-0.5b-instruct",
	}
}

// ValidateAPIKey validates the API key by making a simple request
func (c *QwenClient) ValidateAPIKey(ctx context.Context) error {
	_, err := c.SimpleChat(ctx, "Hello")
	if err != nil {
		if strings.Contains(err.Error(), "Invalid API key") ||
			strings.Contains(err.Error(), "Unauthorized") ||
			strings.Contains(err.Error(), "InvalidApiKey") {
			return fmt.Errorf("invalid API key")
		}
		return fmt.Errorf("API key validation failed: %w", err)
	}
	return nil
}

// EstimateTokens provides a rough estimate of token count
func (c *QwenClient) EstimateTokens(text string) int {
	// Rough estimation: 1 token ≈ 0.75 words for English, 1 token ≈ 1.5 characters for Chinese
	chineseChars := 0
	englishWords := 0

	for _, r := range text {
		if r >= 0x4e00 && r <= 0x9fff { // Chinese characters
			chineseChars++
		}
	}

	englishWords = len(strings.Fields(text))

	// Estimate tokens
	tokens := chineseChars/2 + int(float64(englishWords)*1.33)
	if tokens < 1 {
		tokens = 1
	}

	return tokens
}

// TruncateToTokenLimit truncates text to fit within token limit
func (c *QwenClient) TruncateToTokenLimit(text string, maxTokens int) string {
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

// Helper functions

// convertModelToQwen converts OpenAI model names to Qwen equivalents
func convertModelToQwen(model string) string {
	switch model {
	case "gpt-3.5-turbo":
		return "qwen-turbo"
	case "gpt-4":
		return "qwen-plus"
	case "gpt-4-turbo":
		return "qwen-max"
	default:
		if strings.HasPrefix(model, "qwen") {
			return model
		}
		return "qwen-plus" // default
	}
}

// convertMessagesToQwen converts OpenAI messages to Qwen format
func convertMessagesToQwen(messages []Message) []QwenMessage {
	qwenMessages := make([]QwenMessage, len(messages))
	for i, msg := range messages {
		qwenMessages[i] = QwenMessage{
			Role:    msg.Role,
			Content: msg.Content,
		}
	}
	return qwenMessages
}

// Interface compatibility check
var _ LLMInterface = (*QwenClient)(nil)

// LLMInterface defines the interface that both OpenAI and Qwen clients should implement
type LLMInterface interface {
	ChatCompletion(ctx context.Context, req *ChatRequest) (*ChatResponse, error)
	SimpleChat(ctx context.Context, userMessage string) (string, error)
	ChatWithSystem(ctx context.Context, systemMessage, userMessage string) (string, error)
	ChatWithHistory(ctx context.Context, messages []Message, userMessage string) (string, error)
	GetAvailableModels() []string
	ValidateAPIKey(ctx context.Context) error
	EstimateTokens(text string) int
	TruncateToTokenLimit(text string, maxTokens int) string
}
