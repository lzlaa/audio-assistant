package main

import (
	"fmt"
	"log"
	"math"
	"os"

	"audio-assistant/internal/audio"
	"audio-assistant/internal/vad"
)

func main() {
	fmt.Println("VAD Client Example")
	fmt.Println("==================")

	// Create VAD client
	client := vad.NewClient("http://localhost:8000")

	// Test health check
	fmt.Println("1. Testing VAD server health...")
	health, err := client.Health()
	if err != nil {
		log.Fatalf("VAD server health check failed: %v", err)
	}
	fmt.Printf("   ✓ VAD server is %s (timestamp: %s)\n", health.Status, health.Timestamp)

	// Get model info
	fmt.Println("\n2. Getting VAD model information...")
	info, err := client.Info()
	if err != nil {
		log.Printf("   Warning: failed to get model info: %v", err)
	} else {
		fmt.Printf("   ✓ Model: %s\n", info.ModelName)
		fmt.Printf("   ✓ Sample Rate: %d Hz\n", info.SampleRate)
		fmt.Printf("   ✓ Window Size: %d ms\n", info.WindowSizeMs)
	}

	// Create test audio data
	fmt.Println("\n3. Creating test audio data...")
	sampleRate := 16000
	duration := 2.0    // seconds
	frequency := 440.0 // Hz (A4 note)

	numSamples := int(float64(sampleRate) * duration)
	audioData := make([]float32, numSamples)

	// Generate sine wave with some silence
	for i := 0; i < numSamples; i++ {
		t := float64(i) / float64(sampleRate)

		// Add speech-like signal for first and last 0.5 seconds, silence in middle
		if t < 0.5 || t > 1.5 {
			// Generate complex waveform that might be detected as speech
			audioData[i] = float32(0.3 * (math.Sin(2*math.Pi*frequency*t) +
				0.5*math.Sin(2*math.Pi*frequency*2*t) +
				0.3*math.Sin(2*math.Pi*frequency*3*t)))
		} else {
			// Silence in the middle
			audioData[i] = 0.0
		}
	}
	fmt.Printf("   ✓ Generated %.1fs of test audio with speech at beginning and end\n", duration)

	// Save to WAV file
	testFile := "test_vad_audio.wav"
	defer os.Remove(testFile)

	if err := audio.SaveToWAV(testFile, audioData, sampleRate); err != nil {
		log.Fatalf("Failed to save test audio: %v", err)
	}
	fmt.Printf("   ✓ Saved test audio to %s\n", testFile)

	// Test VAD detection with different configurations
	fmt.Println("\n4. Testing VAD detection with different configurations...")

	configs := []struct {
		name       string
		threshold  float64
		minSpeech  int
		minSilence int
	}{
		{"Default", 0.5, 250, 100},
		{"Sensitive", 0.3, 100, 50},
		{"Conservative", 0.7, 500, 200},
	}

	for _, config := range configs {
		fmt.Printf("\n   Testing %s configuration (threshold=%.1f, min_speech=%dms, min_silence=%dms):\n",
			config.name, config.threshold, config.minSpeech, config.minSilence)

		req := &vad.DetectRequest{
			Threshold:            config.threshold,
			MinSpeechDurationMs:  config.minSpeech,
			MinSilenceDurationMs: config.minSilence,
		}

		response, err := client.DetectFromFile(testFile, req)
		if err != nil {
			log.Printf("   ✗ Detection failed: %v", err)
			continue
		}

		if response.Status != "success" {
			log.Printf("   ✗ Detection unsuccessful: %s", response.Message)
			continue
		}

		fmt.Printf("   ✓ Total duration: %.2fs\n", response.Statistics.TotalAudioDuration)
		fmt.Printf("   ✓ Speech duration: %.2fs (%.1f%%)\n",
			response.Statistics.TotalSpeechDuration, response.Statistics.SpeechRatio*100)
		fmt.Printf("   ✓ Silence duration: %.2fs\n", response.Statistics.TotalAudioDuration-response.Statistics.TotalSpeechDuration)
		fmt.Printf("   ✓ Found %d speech segments:\n", len(response.SpeechSegments))

		for i, segment := range response.SpeechSegments {
			fmt.Printf("     - Segment %d: %.2fs - %.2fs (duration: %.2fs)\n",
				i+1, segment.Start, segment.End, segment.End-segment.Start)
		}
	}

	// Test service integration
	fmt.Println("\n5. Testing VAD service integration...")

	// Create audio input (might fail if no audio device available)
	audioInput, err := audio.NewInput()
	if err != nil {
		fmt.Printf("   ⚠ Audio input not available: %v\n", err)
		fmt.Println("   Skipping service integration test")
	} else {
		defer audioInput.Close()

		// Create VAD service
		vadConfig := vad.DefaultConfig()
		vadConfig.TempDir = "temp_vad_example"
		defer os.RemoveAll(vadConfig.TempDir)

		service := vad.NewService(vadConfig, audioInput)

		// Start service
		if err := service.Start(); err != nil {
			log.Printf("   ✗ Failed to start VAD service: %v", err)
		} else {
			fmt.Printf("   ✓ VAD service started successfully\n")

			// Test detection with service
			hasSpeech, err := service.HasSpeechInFile(testFile)
			if err != nil {
				log.Printf("   ✗ Service detection failed: %v", err)
			} else {
				fmt.Printf("   ✓ Service detected speech: %v\n", hasSpeech)
			}

			// Get speech segments
			segments, err := service.GetSpeechSegmentsFromFile(testFile)
			if err != nil {
				log.Printf("   ✗ Failed to get speech segments: %v", err)
			} else {
				fmt.Printf("   ✓ Service found %d speech segments\n", len(segments))
			}

			service.Stop()
			fmt.Printf("   ✓ VAD service stopped\n")
		}
	}

	fmt.Println("\n✅ VAD client example completed successfully!")
	fmt.Println("\nNext steps:")
	fmt.Println("- Integrate VAD client with your audio processing pipeline")
	fmt.Println("- Adjust VAD parameters based on your use case")
	fmt.Println("- Use real-time detection with audio input streams")
}
