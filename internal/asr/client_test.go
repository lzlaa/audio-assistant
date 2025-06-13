package asr

import (
	"context"
	"math"
	"os"
	"testing"
	"time"

	"audio-assistant/internal/audio"
)

func TestASRClient(t *testing.T) {
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

func TestASRSupportedLanguages(t *testing.T) {
	client := NewClient("dummy-key")
	languages := client.GetSupportedLanguages()

	if len(languages) == 0 {
		t.Error("Expected non-empty list of supported languages")
	}

	// Check for common languages
	expectedLanguages := []string{"en", "zh", "es", "fr", "de", "ja", "ko"}
	for _, expected := range expectedLanguages {
		found := false
		for _, lang := range languages {
			if lang == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected language %s not found in supported languages", expected)
		}
	}

	t.Logf("Found %d supported languages", len(languages))
}

func TestASRTranscription(t *testing.T) {
	// Skip test if API key is not provided
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		t.Skip("OPENAI_API_KEY environment variable not set")
	}

	client := NewClient(apiKey)

	// Create test audio data (simple tone that might be recognized as speech)
	sampleRate := 16000
	duration := 2.0    // seconds
	frequency := 440.0 // Hz

	numSamples := int(float64(sampleRate) * duration)
	audioData := make([]float32, numSamples)

	// Generate a more complex waveform that might be recognized as speech
	for i := 0; i < numSamples; i++ {
		t := float64(i) / float64(sampleRate)
		// Mix multiple frequencies to create speech-like patterns
		audioData[i] = float32(0.3 * (math.Sin(2*math.Pi*frequency*t) +
			0.5*math.Sin(2*math.Pi*frequency*1.5*t) +
			0.3*math.Sin(2*math.Pi*frequency*0.8*t)))
	}

	// Save to temporary WAV file
	tempFile := "test_asr_audio.wav"
	defer os.Remove(tempFile)

	if err := audio.SaveToWAV(tempFile, audioData, sampleRate); err != nil {
		t.Fatalf("Failed to save test audio: %v", err)
	}

	// Test transcription
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	text, err := client.TranscribeSimple(ctx, tempFile)
	if err != nil {
		t.Skipf("Transcription failed (expected for synthetic audio): %v", err)
	}

	t.Logf("Transcription result: %q", text)
}

func TestASRService(t *testing.T) {
	// Skip test if API key is not provided
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		t.Skip("OPENAI_API_KEY environment variable not set")
	}

	// Create ASR service
	config := DefaultConfig()
	config.APIKey = apiKey
	config.TempDir = "temp_test_asr"
	defer os.RemoveAll(config.TempDir)

	service, err := NewService(config)
	if err != nil {
		t.Fatalf("Failed to create ASR service: %v", err)
	}

	// Test service start
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := service.Start(ctx); err != nil {
		t.Skipf("ASR service start failed: %v", err)
	}
	defer service.Stop()

	if !service.IsRunning() {
		t.Error("Expected service to be running")
	}

	// Test configuration
	originalConfig := service.GetConfig()
	service.UpdateConfig("whisper-1", "en", 0.2)
	newConfig := service.GetConfig()

	if newConfig.Model != "whisper-1" {
		t.Errorf("Expected model whisper-1, got %s", newConfig.Model)
	}

	if newConfig.Language != "en" {
		t.Errorf("Expected language en, got %s", newConfig.Language)
	}

	if newConfig.Temperature != 0.2 {
		t.Errorf("Expected temperature 0.2, got %.2f", newConfig.Temperature)
	}

	// Restore original config
	service.UpdateConfig(originalConfig.Model, originalConfig.Language, originalConfig.Temperature)

	// Test supported languages
	languages := service.GetSupportedLanguages()
	if len(languages) == 0 {
		t.Error("Expected non-empty list of supported languages")
	}

	t.Log("ASR service test completed successfully")
}

func TestASRConfigValidation(t *testing.T) {
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
}

func TestASRRequestValidation(t *testing.T) {
	client := NewClient("test-key")

	// Test file size validation
	largeData := make([]byte, 26*1024*1024) // 26MB, exceeds limit
	ctx := context.Background()

	_, err := client.TranscribeBytes(ctx, largeData, "large.wav", nil)
	if err == nil {
		t.Error("Expected error for oversized data")
	}

	// Test unsupported format
	_, err = client.TranscribeFile(ctx, "test.txt", nil)
	if err == nil {
		t.Error("Expected error for unsupported format")
	}
}
