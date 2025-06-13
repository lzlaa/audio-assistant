package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"audio-assistant/internal/asr"
	"audio-assistant/internal/vad"
)

func main() {
	fmt.Println("ASR (Automatic Speech Recognition) Example")
	fmt.Println("==========================================")

	// Check for API key
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		fmt.Println("❌ OPENAI_API_KEY environment variable not set")
		fmt.Println("Please set your OpenAI API key:")
		fmt.Println("export OPENAI_API_KEY=your_api_key_here")
		return
	}

	// Create ASR service
	fmt.Println("1. Initializing ASR service...")
	config := asr.DefaultConfig()
	config.APIKey = apiKey
	config.Language = "zh" // Chinese
	config.TempDir = "temp_asr_example"
	defer os.RemoveAll(config.TempDir)

	service, err := asr.NewService(config)
	if err != nil {
		log.Fatalf("Failed to create ASR service: %v", err)
	}

	// Start service
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := service.Start(ctx); err != nil {
		log.Fatalf("Failed to start ASR service: %v", err)
	}
	defer service.Stop()

	fmt.Printf("   ✓ ASR service started successfully\n")
	fmt.Printf("   ✓ Model: %s\n", service.GetConfig().Model)
	fmt.Printf("   ✓ Language: %s\n", service.GetConfig().Language)

	// Test with real audio file (if available)
	audioFile := "scripts/vad/test_audio.wav"
	if _, err := os.Stat(audioFile); err == nil {
		fmt.Printf("\n2. Testing with real audio file: %s\n", audioFile)

		// Simple transcription
		fmt.Println("   Testing simple transcription...")
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()

		text, err := service.TranscribeFile(ctx, audioFile)
		if err != nil {
			log.Printf("   ✗ Simple transcription failed: %v", err)
		} else {
			fmt.Printf("   ✓ Transcription result: %q\n", text)
		}

		// Detailed transcription
		fmt.Println("   Testing detailed transcription...")
		ctx2, cancel2 := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel2()

		response, err := service.TranscribeWithDetails(ctx2, audioFile)
		if err != nil {
			log.Printf("   ✗ Detailed transcription failed: %v", err)
		} else {
			fmt.Printf("   ✓ Detailed transcription:\n")
			fmt.Printf("     - Text: %q\n", response.Text)
			fmt.Printf("     - Language: %s\n", response.Language)
			fmt.Printf("     - Duration: %.2fs\n", response.Duration)
			fmt.Printf("     - Segments: %d\n", len(response.Segments))

			for i, segment := range response.Segments {
				if i >= 3 { // Show only first 3 segments
					fmt.Printf("     - ... and %d more segments\n", len(response.Segments)-3)
					break
				}
				fmt.Printf("     - Segment %d: %.2fs-%.2fs: %q\n",
					segment.ID, segment.Start, segment.End, segment.Text)
			}
		}
	} else {
		fmt.Printf("\n2. Audio file %s not found, skipping file transcription test\n", audioFile)
	}

	// Test with VAD integration
	fmt.Println("\n3. Testing ASR + VAD integration...")

	if _, err := os.Stat(audioFile); err == nil {
		// Create VAD client
		vadClient := vad.NewClient("http://localhost:8000")

		// Check if VAD service is available
		health, err := vadClient.Health()
		if err != nil {
			fmt.Printf("   ⚠ VAD service not available: %v\n", err)
			fmt.Println("   Skipping VAD integration test")
		} else {
			fmt.Printf("   ✓ VAD service is %s\n", health.Status)

			// Detect speech segments
			vadResponse, err := vadClient.DetectFromFile(audioFile, &vad.DetectRequest{
				Threshold:            0.5,
				MinSpeechDurationMs:  250,
				MinSilenceDurationMs: 100,
			})

			if err != nil {
				log.Printf("   ✗ VAD detection failed: %v", err)
			} else if vadResponse.Status != "success" {
				log.Printf("   ✗ VAD detection unsuccessful: %s", vadResponse.Message)
			} else {
				fmt.Printf("   ✓ Found %d speech segments\n", len(vadResponse.SpeechSegments))

				// Load audio data for segment transcription
				audioData, sampleRate, err := loadAudioFile(audioFile)
				if err != nil {
					log.Printf("   ✗ Failed to load audio data: %v", err)
				} else {
					fmt.Printf("   ✓ Loaded audio: %.2fs at %d Hz\n",
						float64(len(audioData))/float64(sampleRate), sampleRate)

					// Transcribe each segment
					ctx3, cancel3 := context.WithTimeout(context.Background(), 120*time.Second)
					defer cancel3()

					segmentTranscriptions, err := service.TranscribeSpeechSegments(
						ctx3, audioData, sampleRate, vadResponse.SpeechSegments)

					if err != nil {
						log.Printf("   ✗ Segment transcription failed: %v", err)
					} else {
						fmt.Printf("   ✓ Transcribed %d segments:\n", len(segmentTranscriptions))
						for _, st := range segmentTranscriptions {
							fmt.Printf("     - Segment %d (%.2fs-%.2fs): %q\n",
								st.SegmentIndex, st.Start, st.End, st.Text)
						}
					}
				}
			}
		}
	}

	// Test language support
	fmt.Println("\n4. Testing language support...")
	languages := service.GetSupportedLanguages()
	fmt.Printf("   ✓ Supports %d languages\n", len(languages))
	fmt.Printf("   ✓ Common languages: ")
	commonLangs := []string{"en", "zh", "ja", "ko", "es", "fr", "de", "ru"}
	for i, lang := range commonLangs {
		if i > 0 {
			fmt.Print(", ")
		}
		fmt.Print(lang)
	}
	fmt.Println()

	// Test configuration updates
	fmt.Println("\n5. Testing configuration updates...")
	originalConfig := service.GetConfig()
	fmt.Printf("   Original config: model=%s, language=%s, temperature=%.2f\n",
		originalConfig.Model, originalConfig.Language, originalConfig.Temperature)

	service.UpdateConfig("whisper-1", "en", 0.3)
	newConfig := service.GetConfig()
	fmt.Printf("   Updated config: model=%s, language=%s, temperature=%.2f\n",
		newConfig.Model, newConfig.Language, newConfig.Temperature)

	// Restore original config
	service.UpdateConfig(originalConfig.Model, originalConfig.Language, originalConfig.Temperature)
	fmt.Printf("   ✓ Config restored\n")

	fmt.Println("\n✅ ASR example completed successfully!")
	fmt.Println("\nNext steps:")
	fmt.Println("- Integrate ASR with your audio processing pipeline")
	fmt.Println("- Combine with VAD for automatic speech segment detection")
	fmt.Println("- Experiment with different languages and models")
	fmt.Println("- Use the transcribed text for further processing (LLM, etc.)")
}

// loadAudioFile loads audio data from a WAV file
func loadAudioFile(filePath string) ([]float32, int, error) {
	// This is a simplified version - in practice you might want to use
	// a more robust audio loading library
	return nil, 0, fmt.Errorf("audio loading not implemented in this example")
}
