package llm

import (
	"context"
	"fmt"
)

// Client接口定义了LLM客户端必须实现的方法
type Client interface {
	ChatCompletion(ctx context.Context, req *ChatRequest) (*ChatResponse, error)
	ValidateAPIKey(ctx context.Context) error
	GetAvailableModels() []string
	EstimateTokens(text string) int
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
	EnableThinking   *bool     `json:"enable_thinking,omitempty"`   // for DashScope API compatibility
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
