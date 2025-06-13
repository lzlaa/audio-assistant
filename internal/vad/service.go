package vad

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"audio-assistant/internal/audio"
)

// Service manages VAD operations and integrates with audio module
type Service struct {
	client     *Client
	audioInput *audio.Input
	vadConfig  *DetectRequest
	isRunning  bool
	stopChan   chan struct{}
	resultChan chan *DetectResponse
	tempDir    string
}

// Config represents VAD service configuration
type Config struct {
	ServerURL            string
	Threshold            float64
	MinSpeechDurationMs  int
	MinSilenceDurationMs int
	TempDir              string
}

// DefaultConfig returns default VAD configuration
func DefaultConfig() *Config {
	return &Config{
		ServerURL:            "http://localhost:8000",
		Threshold:            0.5,
		MinSpeechDurationMs:  250,
		MinSilenceDurationMs: 100,
		TempDir:              "temp",
	}
}

// NewService creates a new VAD service
func NewService(config *Config, audioInput *audio.Input) *Service {
	if config == nil {
		config = DefaultConfig()
	}

	// Ensure temp directory exists
	if err := os.MkdirAll(config.TempDir, 0755); err != nil {
		log.Printf("Warning: failed to create temp directory %s: %v", config.TempDir, err)
		config.TempDir = "."
	}

	return &Service{
		client:     NewClient(config.ServerURL),
		audioInput: audioInput,
		vadConfig: &DetectRequest{
			Threshold:            config.Threshold,
			MinSpeechDurationMs:  config.MinSpeechDurationMs,
			MinSilenceDurationMs: config.MinSilenceDurationMs,
		},
		stopChan:   make(chan struct{}),
		resultChan: make(chan *DetectResponse, 10),
		tempDir:    config.TempDir,
	}
}

// Start starts the VAD service
func (s *Service) Start() error {
	if s.isRunning {
		return fmt.Errorf("VAD service is already running")
	}

	// Check if VAD server is healthy
	health, err := s.client.Health()
	if err != nil {
		return fmt.Errorf("VAD server health check failed: %w", err)
	}

	log.Printf("VAD server is healthy: %s", health.Status)

	// Get model info
	info, err := s.client.Info()
	if err != nil {
		log.Printf("Warning: failed to get VAD model info: %v", err)
	} else {
		log.Printf("VAD model: %s, sample rate: %d Hz, window size: %d ms",
			info.ModelName, info.SampleRate, info.WindowSizeMs)
	}

	s.isRunning = true
	log.Println("VAD service started")

	return nil
}

// Stop stops the VAD service
func (s *Service) Stop() {
	if !s.isRunning {
		return
	}

	close(s.stopChan)
	s.isRunning = false
	log.Println("VAD service stopped")
}

// IsRunning returns whether the service is running
func (s *Service) IsRunning() bool {
	return s.isRunning
}

// DetectFromAudioData detects speech activity from audio data
func (s *Service) DetectFromAudioData(audioData []float32, sampleRate int) (*DetectResponse, error) {
	if !s.isRunning {
		return nil, fmt.Errorf("VAD service is not running")
	}

	// Create temporary WAV file
	tempFile := filepath.Join(s.tempDir, fmt.Sprintf("vad_temp_%d.wav", time.Now().UnixNano()))
	defer os.Remove(tempFile) // Clean up temp file

	// Save audio data to WAV file
	if err := audio.SaveToWAV(tempFile, audioData, sampleRate); err != nil {
		return nil, fmt.Errorf("failed to save audio to WAV: %w", err)
	}

	// Detect speech activity
	response, err := s.client.DetectFromFile(tempFile, s.vadConfig)
	if err != nil {
		return nil, fmt.Errorf("VAD detection failed: %w", err)
	}

	return response, nil
}

// DetectFromFile detects speech activity from an audio file
func (s *Service) DetectFromFile(filePath string) (*DetectResponse, error) {
	if !s.isRunning {
		return nil, fmt.Errorf("VAD service is not running")
	}

	response, err := s.client.DetectFromFile(filePath, s.vadConfig)
	if err != nil {
		return nil, fmt.Errorf("VAD detection failed: %w", err)
	}

	return response, nil
}

// HasSpeechInAudioData checks if audio data contains speech
func (s *Service) HasSpeechInAudioData(audioData []float32, sampleRate int) (bool, error) {
	response, err := s.DetectFromAudioData(audioData, sampleRate)
	if err != nil {
		return false, err
	}

	return len(response.SpeechSegments) > 0, nil
}

// HasSpeechInFile checks if audio file contains speech
func (s *Service) HasSpeechInFile(filePath string) (bool, error) {
	response, err := s.DetectFromFile(filePath)
	if err != nil {
		return false, err
	}

	return len(response.SpeechSegments) > 0, nil
}

// GetSpeechSegments returns speech segments from audio data
func (s *Service) GetSpeechSegments(audioData []float32, sampleRate int) ([]SpeechSegment, error) {
	response, err := s.DetectFromAudioData(audioData, sampleRate)
	if err != nil {
		return nil, err
	}

	return response.SpeechSegments, nil
}

// GetSpeechSegmentsFromFile returns speech segments from audio file
func (s *Service) GetSpeechSegmentsFromFile(filePath string) ([]SpeechSegment, error) {
	response, err := s.DetectFromFile(filePath)
	if err != nil {
		return nil, err
	}

	return response.SpeechSegments, nil
}

// UpdateConfig updates VAD configuration
func (s *Service) UpdateConfig(threshold float64, minSpeechMs, minSilenceMs int) {
	s.vadConfig.Threshold = threshold
	s.vadConfig.MinSpeechDurationMs = minSpeechMs
	s.vadConfig.MinSilenceDurationMs = minSilenceMs

	log.Printf("VAD config updated: threshold=%.2f, min_speech=%dms, min_silence=%dms",
		threshold, minSpeechMs, minSilenceMs)
}

// GetConfig returns current VAD configuration
func (s *Service) GetConfig() *DetectRequest {
	return &DetectRequest{
		Threshold:            s.vadConfig.Threshold,
		MinSpeechDurationMs:  s.vadConfig.MinSpeechDurationMs,
		MinSilenceDurationMs: s.vadConfig.MinSilenceDurationMs,
	}
}
