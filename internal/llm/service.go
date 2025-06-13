package llm

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"
)

// Service manages LLM operations and conversation context
type Service struct {
	client           *Client
	config           *Config
	isRunning        bool
	conversationHist []Message
	maxHistoryLength int
}

// Config represents LLM service configuration
type Config struct {
	APIKey           string
	BaseURL          string
	Model            string
	Temperature      float32
	MaxTokens        int
	MaxHistoryLength int
	SystemMessage    string
	UserName         string
	Timeout          time.Duration
}

// DefaultConfig returns default LLM configuration
func DefaultConfig() *Config {
	return &Config{
		BaseURL:          "https://api.openai.com/v1",
		Model:            "gpt-3.5-turbo",
		Temperature:      0.7,
		MaxTokens:        150, // Shorter responses for voice
		MaxHistoryLength: 10,  // Keep last 10 exchanges
		SystemMessage:    CreateVoiceAssistantSystemMessage().Content,
		UserName:         "用户",
		Timeout:          30 * time.Second,
	}
}

// NewService creates a new LLM service
func NewService(config *Config) (*Service, error) {
	if config == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}

	if config.APIKey == "" {
		return nil, fmt.Errorf("OpenAI API key is required")
	}

	// Create client
	var client *Client
	if config.BaseURL != "" && config.BaseURL != "https://api.openai.com/v1" {
		client = NewClientWithConfig(config.APIKey, config.BaseURL, config.Timeout)
	} else {
		client = NewClient(config.APIKey)
		client.httpClient.Timeout = config.Timeout
	}

	// Initialize conversation history with system message
	conversationHist := []Message{}
	if config.SystemMessage != "" {
		conversationHist = append(conversationHist, Message{
			Role:    "system",
			Content: config.SystemMessage,
		})
	}

	return &Service{
		client:           client,
		config:           config,
		conversationHist: conversationHist,
		maxHistoryLength: config.MaxHistoryLength,
	}, nil
}

// Start starts the LLM service
func (s *Service) Start(ctx context.Context) error {
	if s.isRunning {
		return fmt.Errorf("LLM service is already running")
	}

	// Validate API key
	if err := s.client.ValidateAPIKey(ctx); err != nil {
		return fmt.Errorf("LLM service validation failed: %w", err)
	}

	s.isRunning = true
	log.Println("LLM service started successfully")

	return nil
}

// Stop stops the LLM service
func (s *Service) Stop() {
	if !s.isRunning {
		return
	}

	s.isRunning = false
	log.Println("LLM service stopped")
}

// IsRunning returns whether the service is running
func (s *Service) IsRunning() bool {
	return s.isRunning
}

// Chat processes user input and returns assistant response
func (s *Service) Chat(ctx context.Context, userMessage string) (string, error) {
	if !s.isRunning {
		return "", fmt.Errorf("LLM service is not running")
	}

	if strings.TrimSpace(userMessage) == "" {
		return "", fmt.Errorf("user message cannot be empty")
	}

	// Add user message to history
	s.conversationHist = append(s.conversationHist, Message{
		Role:    "user",
		Content: userMessage,
	})

	// Create chat request
	req := &ChatRequest{
		Model:       s.config.Model,
		Messages:    s.conversationHist,
		MaxTokens:   s.config.MaxTokens,
		Temperature: s.config.Temperature,
	}

	// Get response from LLM
	response, err := s.client.ChatCompletion(ctx, req)
	if err != nil {
		return "", fmt.Errorf("chat completion failed: %w", err)
	}

	if len(response.Choices) == 0 {
		return "", fmt.Errorf("no response choices returned")
	}

	assistantMessage := strings.TrimSpace(response.Choices[0].Message.Content)

	// Add assistant response to history
	s.conversationHist = append(s.conversationHist, Message{
		Role:    "assistant",
		Content: assistantMessage,
	})

	// Trim history if too long
	s.trimHistory()

	log.Printf("LLM response: %q (tokens: %d)", assistantMessage, response.Usage.TotalTokens)

	return assistantMessage, nil
}

// SimpleChat provides a stateless chat without conversation history
func (s *Service) SimpleChat(ctx context.Context, userMessage string) (string, error) {
	if !s.isRunning {
		return "", fmt.Errorf("LLM service is not running")
	}

	systemMsg := CreateVoiceAssistantSystemMessage()
	response, err := s.client.ChatWithSystem(ctx, systemMsg.Content, userMessage)
	if err != nil {
		return "", fmt.Errorf("simple chat failed: %w", err)
	}

	return response, nil
}

// ChatWithContext processes user input with additional context
func (s *Service) ChatWithContext(ctx context.Context, userMessage, contextInfo string) (string, error) {
	if !s.isRunning {
		return "", fmt.Errorf("LLM service is not running")
	}

	// Combine user message with context
	enhancedMessage := fmt.Sprintf("上下文信息：%s\n\n用户问题：%s", contextInfo, userMessage)

	return s.Chat(ctx, enhancedMessage)
}

// GetConversationHistory returns the current conversation history
func (s *Service) GetConversationHistory() []Message {
	// Return a copy to prevent external modification
	history := make([]Message, len(s.conversationHist))
	copy(history, s.conversationHist)
	return history
}

// ClearHistory clears the conversation history (keeps system message)
func (s *Service) ClearHistory() {
	// Keep only system message
	systemMessages := []Message{}
	for _, msg := range s.conversationHist {
		if msg.Role == "system" {
			systemMessages = append(systemMessages, msg)
		}
	}
	s.conversationHist = systemMessages

	log.Println("Conversation history cleared")
}

// SetSystemMessage updates the system message
func (s *Service) SetSystemMessage(systemMessage string) {
	// Remove old system messages
	nonSystemMessages := []Message{}
	for _, msg := range s.conversationHist {
		if msg.Role != "system" {
			nonSystemMessages = append(nonSystemMessages, msg)
		}
	}

	// Add new system message at the beginning
	s.conversationHist = []Message{
		{Role: "system", Content: systemMessage},
	}
	s.conversationHist = append(s.conversationHist, nonSystemMessages...)

	s.config.SystemMessage = systemMessage
	log.Printf("System message updated: %q", systemMessage)
}

// UpdateConfig updates LLM configuration
func (s *Service) UpdateConfig(model string, temperature float32, maxTokens int) {
	s.config.Model = model
	s.config.Temperature = temperature
	s.config.MaxTokens = maxTokens

	log.Printf("LLM config updated: model=%s, temperature=%.2f, maxTokens=%d",
		model, temperature, maxTokens)
}

// GetConfig returns current LLM configuration
func (s *Service) GetConfig() *Config {
	return &Config{
		APIKey:           s.config.APIKey,
		BaseURL:          s.config.BaseURL,
		Model:            s.config.Model,
		Temperature:      s.config.Temperature,
		MaxTokens:        s.config.MaxTokens,
		MaxHistoryLength: s.config.MaxHistoryLength,
		SystemMessage:    s.config.SystemMessage,
		UserName:         s.config.UserName,
		Timeout:          s.config.Timeout,
	}
}

// GetAvailableModels returns supported model names
func (s *Service) GetAvailableModels() []string {
	return s.client.GetAvailableModels()
}

// EstimateTokens estimates token count for text
func (s *Service) EstimateTokens(text string) int {
	return s.client.EstimateTokens(text)
}

// GetHistoryTokenCount estimates total tokens in conversation history
func (s *Service) GetHistoryTokenCount() int {
	totalTokens := 0
	for _, msg := range s.conversationHist {
		totalTokens += s.EstimateTokens(msg.Content)
	}
	return totalTokens
}

// trimHistory trims conversation history to stay within limits
func (s *Service) trimHistory() {
	if len(s.conversationHist) <= s.maxHistoryLength {
		return
	}

	// Keep system messages and trim user/assistant pairs
	systemMessages := []Message{}
	conversationMessages := []Message{}

	for _, msg := range s.conversationHist {
		if msg.Role == "system" {
			systemMessages = append(systemMessages, msg)
		} else {
			conversationMessages = append(conversationMessages, msg)
		}
	}

	// Keep only the most recent conversation messages
	maxConversationMessages := s.maxHistoryLength - len(systemMessages)
	if maxConversationMessages > 0 && len(conversationMessages) > maxConversationMessages {
		// Keep the most recent messages
		startIndex := len(conversationMessages) - maxConversationMessages
		conversationMessages = conversationMessages[startIndex:]
	}

	// Rebuild history
	s.conversationHist = append(systemMessages, conversationMessages...)

	log.Printf("Conversation history trimmed to %d messages", len(s.conversationHist))
}

// ValidateConfiguration validates the current configuration
func (s *Service) ValidateConfiguration(ctx context.Context) error {
	if s.config.APIKey == "" {
		return fmt.Errorf("API key is required")
	}

	if s.config.Model == "" {
		return fmt.Errorf("model is required")
	}

	if s.config.Temperature < 0 || s.config.Temperature > 2 {
		return fmt.Errorf("temperature must be between 0 and 2")
	}

	if s.config.MaxTokens <= 0 {
		return fmt.Errorf("max tokens must be positive")
	}

	// Validate API key
	return s.client.ValidateAPIKey(ctx)
}

// ProcessTranscribedText processes text from ASR and returns response
func (s *Service) ProcessTranscribedText(ctx context.Context, transcribedText string) (string, error) {
	if !s.isRunning {
		return "", fmt.Errorf("LLM service is not running")
	}

	// Clean up transcribed text
	cleanText := strings.TrimSpace(transcribedText)
	if cleanText == "" {
		return "", fmt.Errorf("transcribed text is empty")
	}

	// Add context about voice input
	contextualMessage := fmt.Sprintf("用户通过语音说：%s", cleanText)

	return s.Chat(ctx, contextualMessage)
}

// GenerateVoiceResponse generates a response optimized for voice output
func (s *Service) GenerateVoiceResponse(ctx context.Context, userInput string) (string, error) {
	if !s.isRunning {
		return "", fmt.Errorf("LLM service is not running")
	}

	// Use voice-optimized system message
	voiceSystemMsg := Message{
		Role: "system",
		Content: `你是一个智能语音助手。请遵循以下规则：

1. 回复必须简洁明了，控制在30字以内
2. 使用自然的口语化表达，避免书面语
3. 语气要友好亲切，像朋友聊天一样
4. 避免使用复杂的标点符号和特殊字符
5. 如果需要列举，用简单的语言描述，不要用编号
6. 回复要适合语音播放，听起来自然流畅

记住：你的回复将直接转换为语音播放给用户。`,
	}

	req := &ChatRequest{
		Model: s.config.Model,
		Messages: []Message{
			voiceSystemMsg,
			{Role: "user", Content: userInput},
		},
		MaxTokens:   100, // Even shorter for voice
		Temperature: 0.7,
	}

	response, err := s.client.ChatCompletion(ctx, req)
	if err != nil {
		return "", fmt.Errorf("voice response generation failed: %w", err)
	}

	if len(response.Choices) == 0 {
		return "", fmt.Errorf("no response choices returned")
	}

	return strings.TrimSpace(response.Choices[0].Message.Content), nil
}
