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

// Client represents an LLM client for OpenAI GPT API
type Client struct {
	apiKey     string
	baseURL    string
	httpClient *http.Client
}

// Message represents a chat message
type Message struct {
	Role    string `json:"role"`    // system, user, assistant
	Content string `json:"content"` // message content
}

// ChatRequest represents the request parameters for chat completion
type ChatRequest struct {
	Model            string    `json:"model"`                       // gpt-3.5-turbo, gpt-4, etc.
	Messages         []Message `json:"messages"`                    // conversation messages
	MaxTokens        int       `json:"max_tokens,omitempty"`        // maximum tokens to generate
	Temperature      float32   `json:"temperature,omitempty"`       // sampling temperature (0-2)
	TopP             float32   `json:"top_p,omitempty"`             // nucleus sampling (0-1)
	N                int       `json:"n,omitempty"`                 // number of completions
	Stream           bool      `json:"stream,omitempty"`            // whether to stream responses
	Stop             []string  `json:"stop,omitempty"`              // stop sequences
	PresencePenalty  float32   `json:"presence_penalty,omitempty"`  // presence penalty (-2 to 2)
	FrequencyPenalty float32   `json:"frequency_penalty,omitempty"` // frequency penalty (-2 to 2)
	User             string    `json:"user,omitempty"`              // user identifier
}

// ChatResponse represents the response from chat completion
type ChatResponse struct {
	ID      string   `json:"id"`
	Object  string   `json:"object"`
	Created int64    `json:"created"`
	Model   string   `json:"model"`
	Choices []Choice `json:"choices"`
	Usage   Usage    `json:"usage"`
}

// Choice represents a completion choice
type Choice struct {
	Index        int     `json:"index"`
	Message      Message `json:"message"`
	FinishReason string  `json:"finish_reason"`
}

// Usage represents token usage information
type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// ErrorResponse represents an error response from the API
type ErrorResponse struct {
	Error struct {
		Message string `json:"message"`
		Type    string `json:"type"`
		Code    string `json:"code"`
	} `json:"error"`
}

// NewClient creates a new LLM client
func NewClient(apiKey string) *Client {
	return &Client{
		apiKey:  apiKey,
		baseURL: "https://api.openai.com/v1",
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

// NewClientWithConfig creates a new LLM client with custom configuration
func NewClientWithConfig(apiKey, baseURL string, timeout time.Duration) *Client {
	return &Client{
		apiKey:  apiKey,
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}
}

// ChatCompletion creates a chat completion
func (c *Client) ChatCompletion(ctx context.Context, req *ChatRequest) (*ChatResponse, error) {
	// Set default values
	if req.Model == "" {
		req.Model = "gpt-3.5-turbo"
	}
	if req.Temperature == 0 {
		req.Temperature = 0.7
	}
	if req.MaxTokens == 0 {
		req.MaxTokens = 1000
	}

	// Marshal request
	reqBody, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/chat/completions", bytes.NewReader(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	// Set headers
	httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)
	httpReq.Header.Set("Content-Type", "application/json")

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
		var errorResp ErrorResponse
		if err := json.Unmarshal(body, &errorResp); err != nil {
			return nil, fmt.Errorf("chat completion failed with status %d: %s", resp.StatusCode, string(body))
		}
		return nil, fmt.Errorf("chat completion failed: %s (type: %s, code: %s)",
			errorResp.Error.Message, errorResp.Error.Type, errorResp.Error.Code)
	}

	// Parse response
	var chatResp ChatResponse
	if err := json.Unmarshal(body, &chatResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &chatResp, nil
}

// SimpleChat provides a simple interface for single-turn conversations
func (c *Client) SimpleChat(ctx context.Context, userMessage string) (string, error) {
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
func (c *Client) ChatWithSystem(ctx context.Context, systemMessage, userMessage string) (string, error) {
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

// Interface compatibility check
var _ LLMInterface = (*Client)(nil)

// ChatWithHistory provides conversation with message history
func (c *Client) ChatWithHistory(ctx context.Context, messages []Message, userMessage string) (string, error) {
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

// GetAvailableModels returns a list of available models
func (c *Client) GetAvailableModels() []string {
	return []string{
		"gpt-4",
		"gpt-4-turbo-preview",
		"gpt-4-0125-preview",
		"gpt-4-1106-preview",
		"gpt-3.5-turbo",
		"gpt-3.5-turbo-0125",
		"gpt-3.5-turbo-1106",
		"gpt-3.5-turbo-16k",
	}
}

// ValidateAPIKey checks if the API key is valid by making a simple request
func (c *Client) ValidateAPIKey(ctx context.Context) error {
	// Create a minimal test request
	req, err := http.NewRequestWithContext(ctx, "GET", c.baseURL+"/models", nil)
	if err != nil {
		return fmt.Errorf("failed to create test request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send test request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		return fmt.Errorf("invalid API key")
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API validation failed with status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// EstimateTokens provides a rough estimate of token count for text
func (c *Client) EstimateTokens(text string) int {
	// Rough estimation: ~4 characters per token for English
	// This is a simplified estimation, actual tokenization may vary
	return len(text) / 4
}

// TruncateToTokenLimit truncates text to fit within token limit
func (c *Client) TruncateToTokenLimit(text string, maxTokens int) string {
	estimatedTokens := c.EstimateTokens(text)
	if estimatedTokens <= maxTokens {
		return text
	}

	// Rough truncation based on character count
	maxChars := maxTokens * 4
	if len(text) <= maxChars {
		return text
	}

	// Truncate and add ellipsis
	truncated := text[:maxChars-3]
	return truncated + "..."
}

// CreateSystemMessage creates a system message for voice assistant
func CreateVoiceAssistantSystemMessage() Message {
	return Message{
		Role: "system",
		Content: `你是一个智能语音助手。请遵循以下规则：

1. 用简洁、自然的中文回复用户
2. 回复长度控制在50字以内，适合语音播放
3. 语气友好、礼貌，像朋友一样交流
4. 如果用户问题不清楚，礼貌地请求澄清
5. 避免使用过于技术性的词汇
6. 回复应该适合口语表达，避免复杂的标点符号

记住：你的回复将被转换为语音，所以要确保内容适合听觉理解。`,
	}
}

// CreateConversationContext creates context for ongoing conversation
func CreateConversationContext(userName string) Message {
	content := fmt.Sprintf(`当前对话上下文：
- 用户名：%s
- 对话类型：语音交互
- 回复要求：简洁、自然、适合语音播放
- 字数限制：50字以内`, userName)

	return Message{
		Role:    "system",
		Content: content,
	}
}
