package tts

import (
	"context"
	"os"
	"testing"
	"time"
)

func TestTTSClient(t *testing.T) {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		t.Skip("OPENAI_API_KEY not set, skipping TTS client tests")
	}

	client := NewTTSClient(apiKey)

	// Test API key validation
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := client.ValidateAPIKey(ctx); err != nil {
		t.Logf("API key validation failed: %v", err)
		t.Skip("API key validation failed, skipping TTS tests")
	}

	t.Log("✓ API key validation passed")
}

func TestTTSClientConfiguration(t *testing.T) {
	client := NewTTSClient("test-key")

	// Test default configuration
	config := client.GetConfig()
	if config.Model != ModelTTS1 {
		t.Errorf("Expected default model %s, got %s", ModelTTS1, config.Model)
	}
	if config.Voice != VoiceAlloy {
		t.Errorf("Expected default voice %s, got %s", VoiceAlloy, config.Voice)
	}
	if config.Speed != 1.0 {
		t.Errorf("Expected default speed 1.0, got %.2f", config.Speed)
	}

	// Test configuration updates
	client.SetModel(ModelTTS1HD)
	client.SetVoice(VoiceNova)
	client.SetSpeed(1.5)

	config = client.GetConfig()
	if config.Model != ModelTTS1HD {
		t.Errorf("Expected model %s, got %s", ModelTTS1HD, config.Model)
	}
	if config.Voice != VoiceNova {
		t.Errorf("Expected voice %s, got %s", VoiceNova, config.Voice)
	}
	if config.Speed != 1.5 {
		t.Errorf("Expected speed 1.5, got %.2f", config.Speed)
	}

	t.Log("✓ Configuration tests passed")
}

func TestTTSClientValidation(t *testing.T) {
	client := NewTTSClient("test-key")

	// Test text validation
	if err := client.ValidateText(""); err == nil {
		t.Error("Expected error for empty text")
	}

	longText := make([]byte, 5000)
	for i := range longText {
		longText[i] = 'a'
	}
	if err := client.ValidateText(string(longText)); err == nil {
		t.Error("Expected error for text too long")
	}

	if err := client.ValidateText("Hello world"); err != nil {
		t.Errorf("Expected no error for valid text, got: %v", err)
	}

	// Test voice validation
	if err := client.ValidateVoice("invalid-voice"); err == nil {
		t.Error("Expected error for invalid voice")
	}

	if err := client.ValidateVoice(VoiceAlloy); err != nil {
		t.Errorf("Expected no error for valid voice, got: %v", err)
	}

	// Test model validation
	if err := client.ValidateModel("invalid-model"); err == nil {
		t.Error("Expected error for invalid model")
	}

	if err := client.ValidateModel(ModelTTS1); err != nil {
		t.Errorf("Expected no error for valid model, got: %v", err)
	}

	// Test format validation
	if err := client.ValidateFormat("invalid-format"); err == nil {
		t.Error("Expected error for invalid format")
	}

	if err := client.ValidateFormat(FormatMP3); err != nil {
		t.Errorf("Expected no error for valid format, got: %v", err)
	}

	t.Log("✓ Validation tests passed")
}

func TestTTSClientAvailableOptions(t *testing.T) {
	client := NewTTSClient("test-key")

	// Test available voices
	voices := client.GetAvailableVoices()
	expectedVoices := []string{VoiceAlloy, VoiceEcho, VoiceFable, VoiceOnyx, VoiceNova, VoiceShimmer}
	if len(voices) != len(expectedVoices) {
		t.Errorf("Expected %d voices, got %d", len(expectedVoices), len(voices))
	}

	// Test available models
	models := client.GetAvailableModels()
	expectedModels := []string{ModelTTS1, ModelTTS1HD}
	if len(models) != len(expectedModels) {
		t.Errorf("Expected %d models, got %d", len(expectedModels), len(models))
	}

	// Test available formats
	formats := client.GetAvailableFormats()
	expectedFormats := []string{FormatMP3, FormatOpus, FormatAAC, FormatFLAC, FormatWAV, FormatPCM}
	if len(formats) != len(expectedFormats) {
		t.Errorf("Expected %d formats, got %d", len(expectedFormats), len(formats))
	}

	t.Log("✓ Available options tests passed")
}

func TestTTSClientSpeedLimits(t *testing.T) {
	client := NewTTSClient("test-key")

	// Test speed limits
	client.SetSpeed(0.1) // Below minimum
	config := client.GetConfig()
	if config.Speed != 0.25 {
		t.Errorf("Expected speed to be clamped to 0.25, got %.2f", config.Speed)
	}

	client.SetSpeed(5.0) // Above maximum
	config = client.GetConfig()
	if config.Speed != 4.0 {
		t.Errorf("Expected speed to be clamped to 4.0, got %.2f", config.Speed)
	}

	client.SetSpeed(1.5) // Valid speed
	config = client.GetConfig()
	if config.Speed != 1.5 {
		t.Errorf("Expected speed 1.5, got %.2f", config.Speed)
	}

	t.Log("✓ Speed limits tests passed")
}

func TestTTSClientCharacterCount(t *testing.T) {
	client := NewTTSClient("test-key")

	testText := "Hello, world!"
	count := client.EstimateCharacterCount(testText)
	if count != len(testText) {
		t.Errorf("Expected character count %d, got %d", len(testText), count)
	}

	t.Log("✓ Character count tests passed")
}
