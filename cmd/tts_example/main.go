package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"audio-assistant/internal/tts"
)

func main() {
	fmt.Println("TTS (Text-to-Speech) Example")
	fmt.Println("============================")

	// Get API key from environment
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		fmt.Println("❌ OPENAI_API_KEY environment variable not set")
		fmt.Println("Please set your OpenAI API key:")
		fmt.Println("export OPENAI_API_KEY='your-api-key-here'")
		return
	}

	// Initialize TTS service
	fmt.Println("1. Initializing TTS service...")
	config := tts.DefaultTTSServiceConfig()
	config.Voice = tts.VoiceAlloy
	config.Speed = 1.0
	config.OutputFormat = tts.FormatMP3

	service, err := tts.NewTTSService(apiKey, config)
	if err != nil {
		log.Fatalf("Failed to create TTS service: %v", err)
	}

	if err := service.Start(); err != nil {
		log.Fatalf("Failed to start TTS service: %v", err)
	}
	defer service.Stop()

	serviceConfig := service.GetConfig()
	fmt.Printf("   ✓ TTS service started successfully\n")
	fmt.Printf("   ✓ Model: %s\n", serviceConfig.Model)
	fmt.Printf("   ✓ Voice: %s\n", serviceConfig.Voice)
	fmt.Printf("   ✓ Speed: %.2f\n", serviceConfig.Speed)
	fmt.Printf("   ✓ Format: %s\n", serviceConfig.OutputFormat)

	// Test basic text synthesis
	fmt.Println("\n2. Testing basic text synthesis...")
	testTexts := []string{
		"你好，我是你的语音助手。",
		"今天天气很好，适合出门散步。",
		"请问有什么可以帮助您的吗？",
		"谢谢使用我们的语音助手服务。",
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	for i, text := range testTexts {
		fmt.Printf("   Text %d: %s\n", i+1, text)

		audioData, err := service.SynthesizeText(ctx, text)
		if err != nil {
			fmt.Printf("   ✗ Synthesis failed: %v\n", err)
			continue
		}

		fmt.Printf("   ✓ Synthesis successful, audio size: %d bytes\n", len(audioData))
	}

	// Test file synthesis
	fmt.Println("\n3. Testing file synthesis...")
	testText := "这是一个测试音频文件，用于验证TTS功能是否正常工作。"
	filename := "test_tts_output.mp3"

	fmt.Printf("   Synthesizing text to file: %s\n", filename)
	if err := service.SynthesizeToFile(ctx, testText, filename); err != nil {
		fmt.Printf("   ✗ File synthesis failed: %v\n", err)
	} else {
		fmt.Printf("   ✓ File synthesis successful: %s\n", filename)
	}

	// Test auto filename generation
	fmt.Println("\n4. Testing auto filename generation...")
	autoText := "自动生成文件名的测试音频。"

	generatedFile, err := service.SynthesizeWithAutoFilename(ctx, autoText, "auto_test")
	if err != nil {
		fmt.Printf("   ✗ Auto filename synthesis failed: %v\n", err)
	} else {
		fmt.Printf("   ✓ Auto filename synthesis successful: %s\n", generatedFile)
	}

	// Test different voices
	fmt.Println("\n5. Testing different voices...")
	voices := service.GetAvailableVoices()
	fmt.Printf("   Available voices: %d\n", len(voices))

	voiceTestText := "Hello, this is a voice test."
	for i, voice := range voices {
		if i >= 3 { // Test only first 3 voices to save time
			fmt.Printf("   ... testing %d more voices\n", len(voices)-3)
			break
		}

		// Update voice
		newConfig := service.GetConfig()
		newConfig.Voice = voice
		if err := service.UpdateConfig(newConfig); err != nil {
			fmt.Printf("   ✗ Failed to update voice to %s: %v\n", voice, err)
			continue
		}

		audioData, err := service.SynthesizeText(ctx, voiceTestText)
		if err != nil {
			fmt.Printf("   ✗ Voice %s failed: %v\n", voice, err)
		} else {
			fmt.Printf("   ✓ Voice %s: %d bytes\n", voice, len(audioData))
		}
	}

	// Test LLM response processing
	fmt.Println("\n6. Testing LLM response processing...")
	llmResponses := []string{
		"**你好！** 我是你的智能助手。\n\n我可以帮助你解答问题、提供信息和进行对话。",
		"今天的天气预报显示：\n- 温度：25°C\n- 湿度：60%\n- 风速：5km/h",
		"这里是一些`代码示例`：\n```go\nfmt.Println(\"Hello World\")\n```",
		"*重要提醒*：请记得保存你的工作！",
	}

	for i, response := range llmResponses {
		preview := response
		if len(response) > 50 {
			preview = response[:50] + "..."
		}
		fmt.Printf("   LLM Response %d: %s\n", i+1, preview)

		audioData, err := service.ProcessLLMResponse(ctx, response)
		if err != nil {
			fmt.Printf("   ✗ LLM response processing failed: %v\n", err)
		} else {
			fmt.Printf("   ✓ LLM response processed: %d bytes\n", len(audioData))
		}
	}

	// Test configuration management
	fmt.Println("\n7. Testing configuration management...")
	originalConfig := service.GetConfig()
	fmt.Printf("   Original config: model=%s, voice=%s, speed=%.2f\n",
		originalConfig.Model, originalConfig.Voice, originalConfig.Speed)

	// Test different models
	models := service.GetAvailableModels()
	fmt.Printf("   ✓ Available models: %d\n", len(models))
	for _, model := range models {
		fmt.Printf("   - %s\n", model)
	}

	// Update configuration
	newConfig := originalConfig
	newConfig.Model = tts.ModelTTS1HD
	newConfig.Voice = tts.VoiceNova
	newConfig.Speed = 1.2

	if err := service.UpdateConfig(newConfig); err != nil {
		fmt.Printf("   ✗ Config update failed: %v\n", err)
	} else {
		updatedConfig := service.GetConfig()
		fmt.Printf("   ✓ Config updated: model=%s, voice=%s, speed=%.2f\n",
			updatedConfig.Model, updatedConfig.Voice, updatedConfig.Speed)
	}

	// Restore original configuration
	if err := service.UpdateConfig(originalConfig); err != nil {
		fmt.Printf("   ✗ Config restore failed: %v\n", err)
	} else {
		fmt.Printf("   ✓ Configuration restored\n")
	}

	// Test cache functionality
	fmt.Println("\n8. Testing cache functionality...")
	cacheStats := service.GetCacheStats()
	fmt.Printf("   Cache enabled: %v\n", cacheStats["enabled"])
	fmt.Printf("   Cache entries: %v\n", cacheStats["entries"])
	fmt.Printf("   Cache size: %v bytes\n", cacheStats["total_bytes"])

	// Test same text multiple times to see caching effect
	cacheTestText := "这是缓存测试文本。"

	// First synthesis (should hit API)
	start := time.Now()
	_, err = service.SynthesizeText(ctx, cacheTestText)
	firstDuration := time.Since(start)
	if err != nil {
		fmt.Printf("   ✗ First synthesis failed: %v\n", err)
	} else {
		fmt.Printf("   ✓ First synthesis: %v\n", firstDuration)
	}

	// Second synthesis (should hit cache)
	start = time.Now()
	_, err = service.SynthesizeText(ctx, cacheTestText)
	secondDuration := time.Since(start)
	if err != nil {
		fmt.Printf("   ✗ Second synthesis failed: %v\n", err)
	} else {
		fmt.Printf("   ✓ Second synthesis (cached): %v\n", secondDuration)
	}

	// Show updated cache stats
	cacheStats = service.GetCacheStats()
	fmt.Printf("   Updated cache entries: %v\n", cacheStats["entries"])
	fmt.Printf("   Updated cache size: %v bytes\n", cacheStats["total_bytes"])

	// Clear cache
	service.ClearCache()
	cacheStats = service.GetCacheStats()
	fmt.Printf("   ✓ Cache cleared, entries: %v\n", cacheStats["entries"])

	// Test available formats
	fmt.Println("\n9. Testing available formats...")
	formats := service.GetAvailableFormats()
	fmt.Printf("   ✓ Available formats: %d\n", len(formats))
	for _, format := range formats {
		fmt.Printf("   - %s\n", format)
	}

	fmt.Println("\n✅ TTS example completed successfully!")
	fmt.Println("\nNext steps:")
	fmt.Println("- Integrate TTS with LLM for complete voice assistant pipeline")
	fmt.Println("- Connect with audio playback for real-time speech")
	fmt.Println("- Implement voice response optimization")
	fmt.Println("- Add multi-language support")
}
