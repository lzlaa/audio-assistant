package asr

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"audio-assistant/internal/audio"
	"audio-assistant/internal/vad"
)

// Service manages ASR operations and integrates with audio and VAD modules
type Service struct {
	client    *Client
	config    *Config
	isRunning bool
	tempDir   string
}

// Config represents ASR service configuration
type Config struct {
	APIKey      string
	BaseURL     string
	Model       string
	Language    string
	Temperature float32
	Timeout     time.Duration
	TempDir     string
}

// DefaultConfig returns default ASR configuration
func DefaultConfig() *Config {
	return &Config{
		BaseURL:     "https://api.openai.com/v1",
		Model:       "whisper-1",
		Language:    "", // Auto-detect
		Temperature: 0.0,
		Timeout:     60 * time.Second,
		TempDir:     "temp",
	}
}

// NewService creates a new ASR service
func NewService(config *Config) (*Service, error) {
	if config == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}

	if config.APIKey == "" {
		return nil, fmt.Errorf("OpenAI API key is required")
	}

	// Ensure temp directory exists
	if err := os.MkdirAll(config.TempDir, 0755); err != nil {
		log.Printf("Warning: failed to create temp directory %s: %v", config.TempDir, err)
		config.TempDir = "."
	}

	// Create client
	var client *Client
	if config.BaseURL != "" && config.BaseURL != "https://api.openai.com/v1" {
		client = NewClientWithConfig(config.APIKey, config.BaseURL, config.Timeout)
	} else {
		client = NewClient(config.APIKey)
		client.httpClient.Timeout = config.Timeout
	}

	return &Service{
		client:  client,
		config:  config,
		tempDir: config.TempDir,
	}, nil
}

// Start starts the ASR service
func (s *Service) Start(ctx context.Context) error {
	if s.isRunning {
		return fmt.Errorf("ASR service is already running")
	}

	// Validate API key
	if err := s.client.ValidateAPIKey(ctx); err != nil {
		return fmt.Errorf("ASR service validation failed: %w", err)
	}

	s.isRunning = true
	log.Println("ASR service started successfully")

	return nil
}

// Stop stops the ASR service
func (s *Service) Stop() {
	if !s.isRunning {
		return
	}

	s.isRunning = false
	log.Println("ASR service stopped")
}

// IsRunning returns whether the service is running
func (s *Service) IsRunning() bool {
	return s.isRunning
}

// TranscribeAudioData transcribes audio data to text
func (s *Service) TranscribeAudioData(ctx context.Context, audioData []float32, sampleRate int) (string, error) {
	if !s.isRunning {
		return "", fmt.Errorf("ASR service is not running")
	}

	// Create temporary WAV file
	tempFile := filepath.Join(s.tempDir, fmt.Sprintf("asr_temp_%d.wav", time.Now().UnixNano()))
	defer os.Remove(tempFile) // Clean up temp file

	// Save audio data to WAV file
	if err := audio.SaveToWAV(tempFile, audioData, sampleRate); err != nil {
		return "", fmt.Errorf("failed to save audio to WAV: %w", err)
	}

	// Transcribe the file
	return s.TranscribeFile(ctx, tempFile)
}

// TranscribeFile transcribes an audio file to text
func (s *Service) TranscribeFile(ctx context.Context, filePath string) (string, error) {
	if !s.isRunning {
		return "", fmt.Errorf("ASR service is not running")
	}

	text, err := s.client.TranscribeWithLanguage(ctx, filePath, s.config.Language)
	if err != nil {
		return "", fmt.Errorf("transcription failed: %w", err)
	}

	return text, nil
}

// TranscribeWithDetails transcribes audio and returns detailed response
func (s *Service) TranscribeWithDetails(ctx context.Context, filePath string) (*TranscribeResponse, error) {
	if !s.isRunning {
		return nil, fmt.Errorf("ASR service is not running")
	}

	req := &TranscribeRequest{
		Model:       s.config.Model,
		Language:    s.config.Language,
		Temperature: s.config.Temperature,
		Format:      "verbose_json",
	}

	response, err := s.client.TranscribeFile(ctx, filePath, req)
	if err != nil {
		return nil, fmt.Errorf("transcription failed: %w", err)
	}

	return response, nil
}

// TranscribeAudioDataWithDetails transcribes audio data and returns detailed response
func (s *Service) TranscribeAudioDataWithDetails(ctx context.Context, audioData []float32, sampleRate int) (*TranscribeResponse, error) {
	if !s.isRunning {
		return nil, fmt.Errorf("ASR service is not running")
	}

	// Create temporary WAV file
	tempFile := filepath.Join(s.tempDir, fmt.Sprintf("asr_temp_%d.wav", time.Now().UnixNano()))
	defer os.Remove(tempFile) // Clean up temp file

	// Save audio data to WAV file
	if err := audio.SaveToWAV(tempFile, audioData, sampleRate); err != nil {
		return nil, fmt.Errorf("failed to save audio to WAV: %w", err)
	}

	// Transcribe the file
	return s.TranscribeWithDetails(ctx, tempFile)
}

// TranscribeSpeechSegments transcribes speech segments detected by VAD
func (s *Service) TranscribeSpeechSegments(ctx context.Context, audioData []float32, sampleRate int, segments []vad.SpeechSegment) ([]SegmentTranscription, error) {
	if !s.isRunning {
		return nil, fmt.Errorf("ASR service is not running")
	}

	if len(segments) == 0 {
		return nil, nil
	}

	var results []SegmentTranscription

	for i, segment := range segments {
		// Extract audio segment
		startSample := int(segment.Start * float64(sampleRate))
		endSample := int(segment.End * float64(sampleRate))

		// Bounds checking
		if startSample < 0 {
			startSample = 0
		}
		if endSample > len(audioData) {
			endSample = len(audioData)
		}
		if startSample >= endSample {
			continue
		}

		segmentAudio := audioData[startSample:endSample]

		// Transcribe segment
		text, err := s.TranscribeAudioData(ctx, segmentAudio, sampleRate)
		if err != nil {
			log.Printf("Failed to transcribe segment %d: %v", i, err)
			continue
		}

		if text != "" {
			results = append(results, SegmentTranscription{
				SegmentIndex: i,
				Start:        segment.Start,
				End:          segment.End,
				Duration:     segment.Duration,
				Text:         text,
			})
		}
	}

	return results, nil
}

// SegmentTranscription represents a transcribed speech segment
type SegmentTranscription struct {
	SegmentIndex int     `json:"segment_index"`
	Start        float64 `json:"start"`
	End          float64 `json:"end"`
	Duration     float64 `json:"duration"`
	Text         string  `json:"text"`
}

// TranscribeWithLanguageHint transcribes audio with a specific language hint
func (s *Service) TranscribeWithLanguageHint(ctx context.Context, audioData []float32, sampleRate int, language string) (string, error) {
	if !s.isRunning {
		return "", fmt.Errorf("ASR service is not running")
	}

	// Create temporary WAV file
	tempFile := filepath.Join(s.tempDir, fmt.Sprintf("asr_temp_%d.wav", time.Now().UnixNano()))
	defer os.Remove(tempFile) // Clean up temp file

	// Save audio data to WAV file
	if err := audio.SaveToWAV(tempFile, audioData, sampleRate); err != nil {
		return "", fmt.Errorf("failed to save audio to WAV: %w", err)
	}

	// Transcribe with language hint
	text, err := s.client.TranscribeWithLanguage(ctx, tempFile, language)
	if err != nil {
		return "", fmt.Errorf("transcription failed: %w", err)
	}

	return text, nil
}

// UpdateConfig updates ASR configuration
func (s *Service) UpdateConfig(model, language string, temperature float32) {
	s.config.Model = model
	s.config.Language = language
	s.config.Temperature = temperature

	log.Printf("ASR config updated: model=%s, language=%s, temperature=%.2f",
		model, language, temperature)
}

// GetConfig returns current ASR configuration
func (s *Service) GetConfig() *Config {
	return &Config{
		APIKey:      s.config.APIKey,
		BaseURL:     s.config.BaseURL,
		Model:       s.config.Model,
		Language:    s.config.Language,
		Temperature: s.config.Temperature,
		Timeout:     s.config.Timeout,
		TempDir:     s.config.TempDir,
	}
}

// GetSupportedLanguages returns supported language codes
func (s *Service) GetSupportedLanguages() []string {
	return s.client.GetSupportedLanguages()
}

// ValidateConfiguration validates the current configuration
func (s *Service) ValidateConfiguration(ctx context.Context) error {
	if s.config.APIKey == "" {
		return fmt.Errorf("API key is required")
	}

	if s.config.Model == "" {
		return fmt.Errorf("model is required")
	}

	if s.config.Temperature < 0 || s.config.Temperature > 1 {
		return fmt.Errorf("temperature must be between 0 and 1")
	}

	// Validate API key
	return s.client.ValidateAPIKey(ctx)
}
