package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"audio-assistant/internal/llm"
)

func main() {
	fmt.Println("LLM (Large Language Model) Example")
	fmt.Println("==================================")

	// Check for API key
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		fmt.Println("❌ OPENAI_API_KEY environment variable not set")
		fmt.Println("Please set your OpenAI API key:")
		fmt.Println("export OPENAI_API_KEY=your_api_key_here")
		return
	}

	// Create LLM service
	fmt.Println("1. Initializing LLM service...")
	config := llm.DefaultConfig()
	config.APIKey = apiKey
	config.Model = "gpt-3.5-turbo"
	config.MaxTokens = 100 // Short responses for demo
	config.UserName = "测试用户"

	service, err := llm.NewService(config)
	if err != nil {
		log.Fatalf("Failed to create LLM service: %v", err)
	}

	// Start service
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := service.Start(ctx); err != nil {
		log.Fatalf("Failed to start LLM service: %v", err)
	}
	defer service.Stop()

	fmt.Printf("   ✓ LLM service started successfully\n")
	fmt.Printf("   ✓ Model: %s\n", service.GetConfig().Model)
	fmt.Printf("   ✓ Max tokens: %d\n", service.GetConfig().MaxTokens)
	fmt.Printf("   ✓ Temperature: %.2f\n", service.GetConfig().Temperature)

	// Test simple chat
	fmt.Println("\n2. Testing simple chat (stateless)...")
	testSimpleChat(service)

	// Test conversation with history
	fmt.Println("\n3. Testing conversation with history...")
	testConversationWithHistory(service)

	// Test voice-optimized responses
	fmt.Println("\n4. Testing voice-optimized responses...")
	testVoiceOptimizedResponses(service)

	// Test ASR integration simulation
	fmt.Println("\n5. Testing ASR integration simulation...")
	testASRIntegration(service)

	// Test conversation history management
	fmt.Println("\n7. Testing conversation history management...")
	testHistoryManagement(service)

	fmt.Println("\n✅ LLM example completed successfully!")
	fmt.Println("\nNext steps:")
	fmt.Println("- Integrate LLM with ASR for voice-to-text-to-response pipeline")
	fmt.Println("- Connect with TTS for complete voice assistant functionality")
	fmt.Println("- Implement context-aware conversations")
	fmt.Println("- Add conversation memory and personalization")
}

func testSimpleChat(service *llm.Service) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	testQuestions := []string{
		"你好",
		"今天天气怎么样？",
		"请介绍一下你自己",
	}

	for i, question := range testQuestions {
		fmt.Printf("   Question %d: %s\n", i+1, question)

		response, err := service.Chat(ctx, question)
		if err != nil {
			log.Printf("   ✗ Simple chat failed: %v", err)
			continue
		}

		fmt.Printf("   Response %d: %s\n", i+1, response)
		fmt.Println()
	}
}

func testConversationWithHistory(service *llm.Service) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Clear any existing history
	service.ClearHistory()

	conversationSteps := []string{
		"我叫张三，今年25岁",
		"我喜欢编程和音乐",
		"我叫什么名字？",
		"我有什么爱好？",
		"请总结一下我的信息",
	}

	fmt.Printf("   Starting conversation with %d steps...\n", len(conversationSteps))

	for i, step := range conversationSteps {
		fmt.Printf("   Step %d: %s\n", i+1, step)

		response, err := service.Chat(ctx, step)
		if err != nil {
			log.Printf("   ✗ Chat failed: %v", err)
			continue
		}

		fmt.Printf("   Response %d: %s\n", i+1, response)
		fmt.Println()
	}

	// Show conversation history
	history := service.GetConversationHistory()
	fmt.Printf("   ✓ Conversation history contains %d messages\n", len(history))
}

func testVoiceOptimizedResponses(service *llm.Service) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	voiceQuestions := []string{
		"现在几点了？",
		"明天会下雨吗？",
		"推荐一首好听的歌",
		"怎么做番茄炒蛋？",
	}

	for i, question := range voiceQuestions {
		fmt.Printf("   Voice question %d: %s\n", i+1, question)

		response, err := service.GenerateVoiceResponse(ctx, question)
		if err != nil {
			log.Printf("   ✗ Voice response failed: %v", err)
			continue
		}

		fmt.Printf("   Voice response %d: %s\n", i+1, response)
		fmt.Printf("   Response length: %d characters\n", len(response))
		fmt.Println()
	}
}

func testASRIntegration(service *llm.Service) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Simulate transcribed text from ASR
	transcribedTexts := []string{
		"请帮我设置明天早上八点的闹钟",
		"今天的新闻有什么重要的吗",
		"播放一些轻松的音乐",
		"提醒我下午三点开会",
	}

	for i, text := range transcribedTexts {
		fmt.Printf("   Transcribed text %d: %s\n", i+1, text)

		response, err := service.ProcessTranscribedText(ctx, text)
		if err != nil {
			log.Printf("   ✗ Processing transcribed text failed: %v", err)
			continue
		}

		fmt.Printf("   LLM response %d: %s\n", i+1, response)
		fmt.Println()
	}
}

func testHistoryManagement(service *llm.Service) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Clear history
	service.ClearHistory()
	fmt.Printf("   ✓ History cleared\n")

	// Add some conversation
	testMessages := []string{
		"我是一名软件工程师",
		"我在学习Go语言",
		"我对AI很感兴趣",
	}

	for i, msg := range testMessages {
		_, err := service.Chat(ctx, msg)
		if err != nil {
			log.Printf("   ✗ Chat %d failed: %v", i+1, err)
			continue
		}
	}

	// Check history
	history := service.GetConversationHistory()
	fmt.Printf("   ✓ History contains %d messages after conversation\n", len(history))

	// Estimate tokens
	totalTokens := service.GetHistoryTokenCount()
	fmt.Printf("   ✓ Estimated total tokens in history: %d\n", totalTokens)

	// Test token estimation
	testText := "这是一个测试文本，用来估算token数量。"
	estimatedTokens := service.EstimateTokens(testText)
	fmt.Printf("   ✓ Estimated tokens for test text: %d\n", estimatedTokens)

	// Clear history again
	service.ClearHistory()
	historyAfterClear := service.GetConversationHistory()
	fmt.Printf("   ✓ History contains %d messages after clearing\n", len(historyAfterClear))
}
