package asr

import (
	"bytes"
	"context"
	"fmt"
	"strings"

	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
)

// OpenAISDKASRClient represents an ASR client using the official OpenAI Go SDK
type OpenAISDKASRClient struct {
	client openai.Client
}

// NewOpenAISDKASRClient creates a new OpenAI SDK ASR client
func NewOpenAISDKASRClient(apiKey string) *OpenAISDKASRClient {
	client := openai.NewClient(
		option.WithAPIKey(apiKey),
	)

	return &OpenAISDKASRClient{
		client: client,
	}
}

// TranscribeAudio transcribes audio data using OpenAI Whisper API
func (c *OpenAISDKASRClient) TranscribeAudio(ctx context.Context, audioData []byte, options *TranscribeRequest) (*TranscribeResponse, error) {
	// Set default options
	if options == nil {
		options = &TranscribeRequest{
			Model:    "whisper-1",
			Language: "auto",
			Format:   "json",
		}
	}

	// Create audio reader
	audioReader := bytes.NewReader(audioData)

	// Create transcription parameters
	params := openai.AudioTranscriptionNewParams{
		File:  openai.File(audioReader, "audio.wav", "audio/wav"),
		Model: openai.AudioModel(options.Model),
	}

	// Add optional parameters
	if options.Language != "" && options.Language != "auto" {
		params.Language = openai.String(options.Language)
	}
	if options.Prompt != "" {
		params.Prompt = openai.String(options.Prompt)
	}
	if options.Temperature > 0 {
		params.Temperature = openai.Float(float64(options.Temperature))
	}

	// Make the API call
	transcription, err := c.client.Audio.Transcriptions.New(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("transcription failed: %w", err)
	}

	// Convert response to our format
	response := &TranscribeResponse{
		Text:     transcription.Text,
		Language: detectLanguageFromText(transcription.Text),
		Duration: 0,   // OpenAI doesn't provide duration in basic response
		Segments: nil, // Would need detailed response format for segments
	}

	return response, nil
}

// TranscribeFile transcribes an audio file using OpenAI Whisper API
func (c *OpenAISDKASRClient) TranscribeFile(ctx context.Context, filePath string, options *TranscribeRequest) (*TranscribeResponse, error) {
	// Read audio file
	audioData, err := readAudioFileData(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read audio file: %w", err)
	}

	return c.TranscribeAudio(ctx, audioData, options)
}

// GetSupportedLanguages returns supported languages for OpenAI Whisper
func (c *OpenAISDKASRClient) GetSupportedLanguages() []string {
	return []string{
		"auto",
		"en", "zh", "ja", "ko", "es", "fr", "de", "it", "pt", "ru",
		"ar", "hi", "th", "vi", "id", "ms", "tl", "tr", "pl", "nl",
		"sv", "da", "no", "fi", "cs", "sk", "hu", "ro", "bg", "hr",
		"sr", "sl", "et", "lv", "lt", "mt", "ga", "cy", "is", "mk",
		"sq", "az", "kk", "ky", "uz", "mn", "hy", "ka", "am", "ne",
		"si", "my", "km", "lo", "bn", "as", "or", "pa", "gu", "ta",
		"te", "kn", "ml", "ur", "fa", "ps", "sd", "dv", "he", "yi",
		"ug", "bo", "dz", "fo", "gl", "eu", "ca", "ast", "an", "oc",
		"br", "gd", "gv", "kw", "lb", "rm", "fur", "lij", "lmo", "vec",
		"nap", "scn", "co", "nrf", "wa", "li", "vls", "fy", "af", "zu",
		"xh", "st", "nso", "tn", "ss", "ve", "nr", "ny", "ig", "yo",
		"ha", "sw", "rw", "rn", "lg", "ak", "tw", "bm", "wo", "ff",
	}
}

// GetSupportedModels returns supported models for OpenAI Whisper
func (c *OpenAISDKASRClient) GetSupportedModels() []string {
	return []string{
		"whisper-1",
	}
}

// ValidateAPIKey validates the API key by making a simple request
func (c *OpenAISDKASRClient) ValidateAPIKey(ctx context.Context) error {
	// Try to list models to validate the API key
	_, err := c.client.Models.List(ctx)
	if err != nil {
		if strings.Contains(err.Error(), "Invalid API key") ||
			strings.Contains(err.Error(), "Unauthorized") ||
			strings.Contains(err.Error(), "authentication") {
			return fmt.Errorf("invalid API key")
		}
		return fmt.Errorf("API key validation failed: %w", err)
	}
	return nil
}

// Helper functions

// detectLanguageFromText provides a simple language detection based on text content
func detectLanguageFromText(text string) string {
	// Simple heuristic - could be improved with proper language detection
	if len(text) == 0 {
		return "unknown"
	}

	// Check for common Chinese characters
	for _, r := range text {
		if r >= 0x4e00 && r <= 0x9fff {
			return "zh"
		}
	}

	// Check for common Japanese characters
	for _, r := range text {
		if (r >= 0x3040 && r <= 0x309f) || (r >= 0x30a0 && r <= 0x30ff) {
			return "ja"
		}
	}

	// Default to English
	return "en"
}

// readAudioFileData reads an audio file and returns its content
func readAudioFileData(filePath string) ([]byte, error) {
	// This would typically use os.ReadFile
	// For now, return an error indicating this needs to be implemented
	return nil, fmt.Errorf("file reading not implemented in this example")
}

// Interface compatibility check
var _ ASRInterface = (*OpenAISDKASRClient)(nil)
