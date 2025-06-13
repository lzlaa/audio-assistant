package tts

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// TTSService manages TTS operations and provides high-level functionality
type TTSService struct {
	client       *TTSClient
	config       TTSServiceConfig
	mu           sync.RWMutex
	isRunning    bool
	outputDir    string
	cacheEnabled bool
	cache        map[string][]byte // Simple in-memory cache
}

// TTSServiceConfig represents TTS service configuration
type TTSServiceConfig struct {
	Model          string  `json:"model"`
	Voice          string  `json:"voice"`
	Speed          float64 `json:"speed"`
	OutputFormat   string  `json:"output_format"`
	OutputDir      string  `json:"output_dir"`
	CacheEnabled   bool    `json:"cache_enabled"`
	MaxTextLength  int     `json:"max_text_length"`
	DefaultTimeout int     `json:"default_timeout_seconds"`
}

// DefaultTTSServiceConfig returns default TTS service configuration
func DefaultTTSServiceConfig() TTSServiceConfig {
	return TTSServiceConfig{
		Model:          ModelTTS1,
		Voice:          VoiceAlloy,
		Speed:          1.0,
		OutputFormat:   FormatMP3,
		OutputDir:      "output/tts",
		CacheEnabled:   true,
		MaxTextLength:  4096,
		DefaultTimeout: 60,
	}
}

// NewTTSService creates a new TTS service
func NewTTSService(apiKey string, config TTSServiceConfig) (*TTSService, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("API key is required")
	}

	client := NewTTSClient(apiKey)
	client.SetModel(config.Model)
	client.SetVoice(config.Voice)
	client.SetSpeed(config.Speed)

	service := &TTSService{
		client:       client,
		config:       config,
		outputDir:    config.OutputDir,
		cacheEnabled: config.CacheEnabled,
		cache:        make(map[string][]byte),
	}

	// Create output directory
	if err := os.MkdirAll(config.OutputDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create output directory: %w", err)
	}

	return service, nil
}

// Start starts the TTS service
func (s *TTSService) Start() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.isRunning {
		return fmt.Errorf("TTS service is already running")
	}

	s.isRunning = true
	log.Println("TTS service started successfully")
	return nil
}

// Stop stops the TTS service
func (s *TTSService) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.isRunning {
		return fmt.Errorf("TTS service is not running")
	}

	s.isRunning = false
	s.clearCache()
	log.Println("TTS service stopped")
	return nil
}

// IsRunning returns whether the service is running
func (s *TTSService) IsRunning() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.isRunning
}

// SynthesizeText converts text to speech and returns audio data
func (s *TTSService) SynthesizeText(ctx context.Context, text string) ([]byte, error) {
	if !s.IsRunning() {
		return nil, fmt.Errorf("TTS service is not running")
	}

	// Validate text
	if err := s.validateText(text); err != nil {
		return nil, fmt.Errorf("text validation failed: %w", err)
	}

	// Check cache first
	if s.cacheEnabled {
		if audioData := s.getCachedAudio(text); audioData != nil {
			return audioData, nil
		}
	}

	// Create context with timeout
	if ctx == nil {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(context.Background(),
			time.Duration(s.config.DefaultTimeout)*time.Second)
		defer cancel()
	}

	// Synthesize text
	audioData, err := s.client.SynthesizeText(ctx, text, s.config.OutputFormat)
	if err != nil {
		return nil, fmt.Errorf("synthesis failed: %w", err)
	}

	// Cache the result
	if s.cacheEnabled {
		s.cacheAudio(text, audioData)
	}

	return audioData, nil
}

// SynthesizeToFile converts text to speech and saves to file
func (s *TTSService) SynthesizeToFile(ctx context.Context, text string, filename string) error {
	if !s.IsRunning() {
		return fmt.Errorf("TTS service is not running")
	}

	// If filename is not absolute, make it relative to output directory
	if !filepath.IsAbs(filename) {
		filename = filepath.Join(s.outputDir, filename)
	}

	// Validate file path
	if err := ValidateFilePath(filename); err != nil {
		return fmt.Errorf("invalid file path: %w", err)
	}

	// Synthesize text
	audioData, err := s.SynthesizeText(ctx, text)
	if err != nil {
		return fmt.Errorf("synthesis failed: %w", err)
	}

	// Save to file
	if err := saveAudioToFile(audioData, filename); err != nil {
		return fmt.Errorf("failed to save audio file: %w", err)
	}

	log.Printf("TTS audio saved to: %s", filename)
	return nil
}

// SynthesizeWithAutoFilename converts text to speech and saves with auto-generated filename
func (s *TTSService) SynthesizeWithAutoFilename(ctx context.Context, text string, prefix string) (string, error) {
	if !s.IsRunning() {
		return "", fmt.Errorf("TTS service is not running")
	}

	// Generate filename
	filename := GenerateFilename(prefix, s.config.OutputFormat)
	fullPath := filepath.Join(s.outputDir, filename)

	// Synthesize and save
	if err := s.SynthesizeToFile(ctx, text, fullPath); err != nil {
		return "", err
	}

	return fullPath, nil
}

// ProcessLLMResponse processes LLM response text for TTS
// This method optimizes text for voice synthesis
func (s *TTSService) ProcessLLMResponse(ctx context.Context, llmResponse string) ([]byte, error) {
	if !s.IsRunning() {
		return nil, fmt.Errorf("TTS service is not running")
	}

	// Optimize text for voice synthesis
	optimizedText := s.optimizeTextForVoice(llmResponse)

	// Synthesize optimized text
	return s.SynthesizeText(ctx, optimizedText)
}

// UpdateConfig updates TTS service configuration
func (s *TTSService) UpdateConfig(config TTSServiceConfig) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Validate configuration
	if err := s.validateConfig(config); err != nil {
		return fmt.Errorf("invalid configuration: %w", err)
	}

	// Update client configuration
	s.client.SetModel(config.Model)
	s.client.SetVoice(config.Voice)
	s.client.SetSpeed(config.Speed)

	// Update service configuration
	s.config = config
	s.outputDir = config.OutputDir
	s.cacheEnabled = config.CacheEnabled

	// Clear cache if caching is disabled
	if !config.CacheEnabled {
		s.clearCache()
	}

	// Create output directory if changed
	if err := os.MkdirAll(config.OutputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	log.Printf("TTS config updated: model=%s, voice=%s, speed=%.2f, format=%s",
		config.Model, config.Voice, config.Speed, config.OutputFormat)
	return nil
}

// GetConfig returns current TTS service configuration
func (s *TTSService) GetConfig() TTSServiceConfig {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.config
}

// GetAvailableVoices returns list of available voices
func (s *TTSService) GetAvailableVoices() []string {
	return s.client.GetAvailableVoices()
}

// GetAvailableModels returns list of available models
func (s *TTSService) GetAvailableModels() []string {
	return s.client.GetAvailableModels()
}

// GetAvailableFormats returns list of available formats
func (s *TTSService) GetAvailableFormats() []string {
	return s.client.GetAvailableFormats()
}

// GetCacheStats returns cache statistics
func (s *TTSService) GetCacheStats() map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()

	totalSize := 0
	for _, data := range s.cache {
		totalSize += len(data)
	}

	return map[string]interface{}{
		"enabled":     s.cacheEnabled,
		"entries":     len(s.cache),
		"total_bytes": totalSize,
	}
}

// ClearCache clears the audio cache
func (s *TTSService) ClearCache() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.clearCache()
	log.Println("TTS cache cleared")
}

// ValidateAPIKey validates the API key
func (s *TTSService) ValidateAPIKey(ctx context.Context) error {
	return s.client.ValidateAPIKey(ctx)
}

// Private methods

func (s *TTSService) validateText(text string) error {
	if text == "" {
		return fmt.Errorf("text cannot be empty")
	}

	if len(text) > s.config.MaxTextLength {
		return fmt.Errorf("text too long: %d characters (max %d)",
			len(text), s.config.MaxTextLength)
	}

	return nil
}

func (s *TTSService) validateConfig(config TTSServiceConfig) error {
	if err := s.client.ValidateModel(config.Model); err != nil {
		return err
	}

	if err := s.client.ValidateVoice(config.Voice); err != nil {
		return err
	}

	if err := s.client.ValidateFormat(config.OutputFormat); err != nil {
		return err
	}

	if config.Speed < 0.25 || config.Speed > 4.0 {
		return fmt.Errorf("invalid speed: %.2f (must be between 0.25 and 4.0)", config.Speed)
	}

	if config.MaxTextLength <= 0 || config.MaxTextLength > 4096 {
		return fmt.Errorf("invalid max text length: %d (must be between 1 and 4096)",
			config.MaxTextLength)
	}

	return nil
}

func (s *TTSService) getCachedAudio(text string) []byte {
	s.mu.RLock()
	defer s.mu.RUnlock()

	cacheKey := s.generateCacheKey(text)
	return s.cache[cacheKey]
}

func (s *TTSService) cacheAudio(text string, audioData []byte) {
	s.mu.Lock()
	defer s.mu.Unlock()

	cacheKey := s.generateCacheKey(text)
	s.cache[cacheKey] = audioData
}

func (s *TTSService) generateCacheKey(text string) string {
	return fmt.Sprintf("%s_%s_%.2f_%s",
		s.config.Model, s.config.Voice, s.config.Speed, text)
}

func (s *TTSService) clearCache() {
	s.cache = make(map[string][]byte)
}

func (s *TTSService) optimizeTextForVoice(text string) string {
	// Remove excessive whitespace
	text = strings.TrimSpace(text)

	// Replace multiple spaces with single space
	for strings.Contains(text, "  ") {
		text = strings.ReplaceAll(text, "  ", " ")
	}

	// Replace multiple newlines with single newline
	for strings.Contains(text, "\n\n") {
		text = strings.ReplaceAll(text, "\n\n", "\n")
	}

	// Convert newlines to periods for better speech flow
	text = strings.ReplaceAll(text, "\n", ". ")

	// Remove markdown formatting that doesn't work well with TTS
	text = strings.ReplaceAll(text, "**", "")
	text = strings.ReplaceAll(text, "*", "")
	text = strings.ReplaceAll(text, "_", "")
	text = strings.ReplaceAll(text, "`", "")

	// Ensure text ends with proper punctuation
	text = strings.TrimSpace(text)
	if !strings.HasSuffix(text, ".") && !strings.HasSuffix(text, "!") &&
		!strings.HasSuffix(text, "?") {
		text += "."
	}

	return text
}
