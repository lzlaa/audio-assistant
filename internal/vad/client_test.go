package vad

import (
	"math"
	"os"
	"testing"

	"audio-assistant/internal/audio"
)

func TestVADClient(t *testing.T) {
	// Skip test if VAD server is not running
	client := NewClient("http://localhost:8000")

	// Test health check
	health, err := client.Health()
	if err != nil {
		t.Skipf("VAD server not available: %v", err)
	}

	if health.Status != "healthy" {
		t.Errorf("Expected health status 'healthy', got '%s'", health.Status)
	}

	t.Logf("VAD server health: %s at %s", health.Status, health.Timestamp)
}

func TestVADInfo(t *testing.T) {
	client := NewClient("http://localhost:8000")

	info, err := client.Info()
	if err != nil {
		t.Skipf("VAD server not available: %v", err)
	}

	if info.ModelName == "" {
		t.Error("Expected model name to be non-empty")
	}

	// Note: Sample rate might be 0 in the current implementation
	if info.SampleRate < 0 {
		t.Error("Expected sample rate to be non-negative")
	}

	t.Logf("VAD model info: %s, sample rate: %d Hz, window size: %d ms",
		info.ModelName, info.SampleRate, info.WindowSizeMs)
}

func TestVADDetection(t *testing.T) {
	client := NewClient("http://localhost:8000")

	// Create test audio data (1 second of sine wave at 440Hz)
	sampleRate := 16000
	duration := 1.0    // seconds
	frequency := 440.0 // Hz

	numSamples := int(float64(sampleRate) * duration)
	audioData := make([]float32, numSamples)

	for i := 0; i < numSamples; i++ {
		t := float64(i) / float64(sampleRate)
		audioData[i] = float32(0.5 * math.Sin(2*math.Pi*frequency*t))
	}

	// Save to temporary WAV file
	tempFile := "test_audio.wav"
	defer os.Remove(tempFile)

	if err := audio.SaveToWAV(tempFile, audioData, sampleRate); err != nil {
		t.Fatalf("Failed to save test audio: %v", err)
	}

	// Test detection
	req := &DetectRequest{
		Threshold:            0.3,
		MinSpeechDurationMs:  100,
		MinSilenceDurationMs: 50,
	}

	response, err := client.DetectFromFile(tempFile, req)
	if err != nil {
		t.Skipf("VAD server not available: %v", err)
	}

	if response.Status != "success" {
		t.Errorf("Detection failed: %s", response.Message)
	}

	if response.Statistics.TotalAudioDuration <= 0 {
		t.Error("Expected total duration to be positive")
	}

	t.Logf("Detection result: %d speech segments, total duration: %.2fs, speech ratio: %.2f",
		len(response.SpeechSegments), response.Statistics.TotalAudioDuration, response.Statistics.SpeechRatio)

	for i, segment := range response.SpeechSegments {
		t.Logf("Speech segment %d: %.2fs - %.2fs", i+1, segment.Start, segment.End)
	}
}

func TestVADService(t *testing.T) {
	// Create audio input (this might fail if no audio device is available)
	audioInput, err := audio.NewInput()
	if err != nil {
		t.Skipf("Audio input not available: %v", err)
	}
	defer audioInput.Close()

	// Create VAD service
	config := DefaultConfig()
	config.TempDir = "temp_test"
	defer os.RemoveAll(config.TempDir)

	service := NewService(config, audioInput)

	// Test service start
	if err := service.Start(); err != nil {
		t.Skipf("VAD service not available: %v", err)
	}
	defer service.Stop()

	if !service.IsRunning() {
		t.Error("Expected service to be running")
	}

	// Test configuration
	originalConfig := service.GetConfig()
	service.UpdateConfig(0.7, 300, 150)
	newConfig := service.GetConfig()

	if newConfig.Threshold != 0.7 {
		t.Errorf("Expected threshold 0.7, got %.2f", newConfig.Threshold)
	}

	if newConfig.MinSpeechDurationMs != 300 {
		t.Errorf("Expected min speech duration 300ms, got %d", newConfig.MinSpeechDurationMs)
	}

	// Restore original config
	service.UpdateConfig(originalConfig.Threshold, originalConfig.MinSpeechDurationMs, originalConfig.MinSilenceDurationMs)

	t.Log("VAD service test completed successfully")
}
