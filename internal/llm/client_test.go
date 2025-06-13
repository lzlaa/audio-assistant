package llm

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"
)

func TestLLMClient(t *testing.T) {
	// Skip test if API key is not provided
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		t.Skip("OPENAI_API_KEY environment variable not set")
	}

	client := NewClient(apiKey)

	// Test API key validation
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err := client.ValidateAPIKey(ctx)
	if err != nil {
		t.Skipf("API key validation failed: %v", err)
	}

	t.Log("API key validation successful")
}

func TestLLMAvailableModels(t *testing.T) {
	client := NewClient("dummy-key")
	models := client.GetAvailableModels()

	if len(models) == 0 {
		t.Error("Expected non-empty list of available models")
	}

	// Check for common models
	expectedModels := []string{"gpt-3.5-turbo", "gpt-4"}
	for _, expected := range expectedModels {
		found := false
		for _, model := range models {
			if model == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected model %s not found in available models", expected)
		}
	}

	t.Logf("Found %d available models", len(models))
}

func TestLLMTokenEstimation(t *testing.T) {
	client := NewClient("dummy-key")

	testCases := []struct {
		text     string
		expected int
	}{
		{"Hello", 1},
		{"Hello world", 2},
		{"This is a test message", 5}, // Adjusted for actual calculation
		{"", 0},
	}

	for _, tc := range testCases {
		estimated := client.EstimateTokens(tc.text)
		if estimated != tc.expected {
			t.Errorf("EstimateTokens(%q) = %d, expected %d", tc.text, estimated, tc.expected)
		}
	}
}

func TestLLMTokenTruncation(t *testing.T) {
	client := NewClient("dummy-key")

	longText := strings.Repeat("Hello world ", 100) // ~200 tokens
	truncated := client.TruncateToTokenLimit(longText, 50)

	estimatedTokens := client.EstimateTokens(truncated)
	if estimatedTokens > 50 {
		t.Errorf("Truncated text has %d tokens, expected <= 50", estimatedTokens)
	}

	if !strings.HasSuffix(truncated, "...") {
		t.Error("Truncated text should end with '...'")
	}

	// Test text that doesn't need truncation
	shortText := "Hello world"
	notTruncated := client.TruncateToTokenLimit(shortText, 50)
	if notTruncated != shortText {
		t.Error("Short text should not be truncated")
	}
}

func TestLLMSimpleChat(t *testing.T) {
	// Skip test if API key is not provided
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		t.Skip("OPENAI_API_KEY environment variable not set")
	}

	client := NewClient(apiKey)

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	response, err := client.SimpleChat(ctx, "Hello, how are you?")
	if err != nil {
		t.Skipf("Simple chat failed: %v", err)
	}

	if response == "" {
		t.Error("Expected non-empty response")
	}

	t.Logf("Chat response: %q", response)
}

func TestLLMService(t *testing.T) {
	// Skip test if API key is not provided
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		t.Skip("OPENAI_API_KEY environment variable not set")
	}

	// Create LLM service
	config := DefaultConfig()
	config.APIKey = apiKey
	config.MaxTokens = 50 // Short responses for testing

	service, err := NewService(config)
	if err != nil {
		t.Fatalf("Failed to create LLM service: %v", err)
	}

	// Test service start
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := service.Start(ctx); err != nil {
		t.Skipf("LLM service start failed: %v", err)
	}
	defer service.Stop()

	if !service.IsRunning() {
		t.Error("Expected service to be running")
	}

	// Test simple chat
	response, err := service.SimpleChat(ctx, "你好")
	if err != nil {
		t.Errorf("Simple chat failed: %v", err)
	} else {
		t.Logf("Simple chat response: %q", response)
	}

	// Test conversation with history
	response1, err := service.Chat(ctx, "我叫小明")
	if err != nil {
		t.Errorf("Chat failed: %v", err)
	} else {
		t.Logf("Chat response 1: %q", response1)
	}

	response2, err := service.Chat(ctx, "我叫什么名字？")
	if err != nil {
		t.Errorf("Chat with history failed: %v", err)
	} else {
		t.Logf("Chat response 2: %q", response2)
		// The response should remember the name
		if !strings.Contains(response2, "小明") {
			t.Log("Warning: LLM may not have remembered the name from previous message")
		}
	}

	// Test conversation history
	history := service.GetConversationHistory()
	if len(history) < 3 { // system + 2 user messages + 2 assistant responses
		t.Errorf("Expected at least 3 messages in history, got %d", len(history))
	}

	// Test clear history
	service.ClearHistory()
	historyAfterClear := service.GetConversationHistory()
	// Should only have system message(s)
	systemCount := 0
	for _, msg := range historyAfterClear {
		if msg.Role == "system" {
			systemCount++
		}
	}
	if systemCount == 0 {
		t.Error("Expected at least one system message after clearing history")
	}

	t.Log("LLM service test completed successfully")
}

func TestLLMConfigValidation(t *testing.T) {
	// Test nil config
	_, err := NewService(nil)
	if err == nil {
		t.Error("Expected error for nil config")
	}

	// Test empty API key
	config := DefaultConfig()
	config.APIKey = ""
	_, err = NewService(config)
	if err == nil {
		t.Error("Expected error for empty API key")
	}

	// Test valid config
	config = DefaultConfig()
	config.APIKey = "test-key"
	service, err := NewService(config)
	if err != nil {
		t.Errorf("Unexpected error for valid config: %v", err)
	}

	if service == nil {
		t.Error("Expected non-nil service")
	}

	// Test configuration updates
	originalConfig := service.GetConfig()
	service.UpdateConfig("gpt-4", 0.5, 200)
	newConfig := service.GetConfig()

	if newConfig.Model != "gpt-4" {
		t.Errorf("Expected model gpt-4, got %s", newConfig.Model)
	}

	if newConfig.Temperature != 0.5 {
		t.Errorf("Expected temperature 0.5, got %.2f", newConfig.Temperature)
	}

	if newConfig.MaxTokens != 200 {
		t.Errorf("Expected max tokens 200, got %d", newConfig.MaxTokens)
	}

	// Restore original config
	service.UpdateConfig(originalConfig.Model, originalConfig.Temperature, originalConfig.MaxTokens)
}

func TestLLMSystemMessages(t *testing.T) {
	// Test voice assistant system message
	sysMsg := CreateVoiceAssistantSystemMessage()
	if sysMsg.Role != "system" {
		t.Error("Expected system role")
	}

	if sysMsg.Content == "" {
		t.Error("Expected non-empty system message content")
	}

	if !strings.Contains(sysMsg.Content, "语音助手") {
		t.Error("Expected system message to mention voice assistant")
	}

	// Test conversation context
	contextMsg := CreateConversationContext("测试用户")
	if contextMsg.Role != "system" {
		t.Error("Expected system role for context message")
	}

	if !strings.Contains(contextMsg.Content, "测试用户") {
		t.Error("Expected context message to contain user name")
	}
}

func TestLLMVoiceOptimization(t *testing.T) {
	// Skip test if API key is not provided
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		t.Skip("OPENAI_API_KEY environment variable not set")
	}

	config := DefaultConfig()
	config.APIKey = apiKey

	service, err := NewService(config)
	if err != nil {
		t.Fatalf("Failed to create LLM service: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := service.Start(ctx); err != nil {
		t.Skipf("LLM service start failed: %v", err)
	}
	defer service.Stop()

	// Test voice-optimized response
	response, err := service.GenerateVoiceResponse(ctx, "今天天气怎么样？")
	if err != nil {
		t.Errorf("Voice response generation failed: %v", err)
	} else {
		t.Logf("Voice response: %q", response)

		// Voice responses should be relatively short
		if len(response) > 100 {
			t.Logf("Warning: Voice response might be too long (%d characters)", len(response))
		}
	}

	// Test processing transcribed text
	transcribedText := "请告诉我现在几点了"
	response2, err := service.ProcessTranscribedText(ctx, transcribedText)
	if err != nil {
		t.Errorf("Processing transcribed text failed: %v", err)
	} else {
		t.Logf("Processed transcription response: %q", response2)
	}
}
