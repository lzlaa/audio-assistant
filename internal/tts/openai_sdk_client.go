package tts

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"strings"

	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
)

// OpenAISDKTTSClient represents a TTS client using the official OpenAI Go SDK
type OpenAISDKTTSClient struct {
	client openai.Client
}

// NewOpenAISDKTTSClient creates a new OpenAI SDK TTS client
func NewOpenAISDKTTSClient(apiKey string) *OpenAISDKTTSClient {
	client := openai.NewClient(
		option.WithAPIKey(apiKey),
	)

	return &OpenAISDKTTSClient{
		client: client,
	}
}

// SynthesizeText synthesizes text to speech using OpenAI TTS API
func (c *OpenAISDKTTSClient) SynthesizeText(ctx context.Context, text string, options *SynthesizeOptions) (*SynthesizeResponse, error) {
	// Set default options
	if options == nil {
		options = &SynthesizeOptions{
			Voice:  "alloy",
			Format: "mp3",
			Speed:  1.0,
		}
	}

	// Convert voice name to OpenAI format
	voice := convertVoiceToOpenAI(options.Voice)

	// Create speech parameters
	params := openai.AudioSpeechNewParams{
		Model: openai.SpeechModel("tts-1"),
		Input: text,
		Voice: openai.AudioSpeechNewParamsVoice(voice),
	}

	// Add optional parameters
	if options.Speed > 0 && options.Speed != 1.0 {
		params.Speed = openai.Float(float64(options.Speed))
	}

	// Set response format
	switch options.Format {
	case "mp3":
		params.ResponseFormat = openai.AudioSpeechNewParamsResponseFormat("mp3")
	case "opus":
		params.ResponseFormat = openai.AudioSpeechNewParamsResponseFormat("opus")
	case "aac":
		params.ResponseFormat = openai.AudioSpeechNewParamsResponseFormat("aac")
	case "flac":
		params.ResponseFormat = openai.AudioSpeechNewParamsResponseFormat("flac")
	case "wav":
		params.ResponseFormat = openai.AudioSpeechNewParamsResponseFormat("wav")
	case "pcm":
		params.ResponseFormat = openai.AudioSpeechNewParamsResponseFormat("pcm")
	default:
		params.ResponseFormat = openai.AudioSpeechNewParamsResponseFormat("mp3")
	}

	// Make the API call
	audioResponse, err := c.client.Audio.Speech.New(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("speech synthesis failed: %w", err)
	}
	defer audioResponse.Body.Close()

	// Read audio data
	audioData, err := io.ReadAll(audioResponse.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read audio data: %w", err)
	}

	// Convert response to our format (base64 encode the audio data)
	response := &SynthesizeResponse{
		AudioData: base64.StdEncoding.EncodeToString(audioData),
		Format:    options.Format,
		RequestID: "", // OpenAI doesn't provide request ID in response
	}

	return response, nil
}

// SynthesizeToFile synthesizes text to speech and saves to file
func (c *OpenAISDKTTSClient) SynthesizeToFile(ctx context.Context, text, outputPath string, options *SynthesizeOptions) error {
	resp, err := c.SynthesizeText(ctx, text, options)
	if err != nil {
		return err
	}

	return saveOpenAIAudioToFile(resp.AudioData, outputPath)
}

// GetAvailableVoices returns available voices for OpenAI TTS
func (c *OpenAISDKTTSClient) GetAvailableVoices() []Voice {
	return []Voice{
		{ID: "alloy", Name: "Alloy", Gender: "neutral", Language: "en"},
		{ID: "echo", Name: "Echo", Gender: "male", Language: "en"},
		{ID: "fable", Name: "Fable", Gender: "neutral", Language: "en"},
		{ID: "onyx", Name: "Onyx", Gender: "male", Language: "en"},
		{ID: "nova", Name: "Nova", Gender: "female", Language: "en"},
		{ID: "shimmer", Name: "Shimmer", Gender: "female", Language: "en"},
	}
}

// GetSupportedFormats returns supported audio formats
func (c *OpenAISDKTTSClient) GetSupportedFormats() []string {
	return []string{
		"mp3",
		"opus",
		"aac",
		"flac",
		"wav",
		"pcm",
	}
}

// GetSupportedModels returns supported TTS models
func (c *OpenAISDKTTSClient) GetSupportedModels() []string {
	return []string{
		"tts-1",
		"tts-1-hd",
	}
}

// ValidateAPIKey validates the API key by making a simple request
func (c *OpenAISDKTTSClient) ValidateAPIKey(ctx context.Context) error {
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

// convertVoiceToOpenAI converts voice names to OpenAI format
func convertVoiceToOpenAI(voice string) string {
	// Map common voice names to OpenAI voices
	voiceMap := map[string]string{
		"alloy":   "alloy",
		"echo":    "echo",
		"fable":   "fable",
		"onyx":    "onyx",
		"nova":    "nova",
		"shimmer": "shimmer",
		// Add mappings for other voice names
		"female":  "nova",
		"male":    "onyx",
		"neutral": "alloy",
	}

	if openaiVoice, exists := voiceMap[voice]; exists {
		return openaiVoice
	}

	// Default to alloy if voice not found
	return "alloy"
}

// saveOpenAIAudioToFile saves base64 encoded audio data to file
func saveOpenAIAudioToFile(base64Audio, outputPath string) error {
	// This would typically decode base64 and save to file
	// For now, return an error indicating this needs to be implemented
	return fmt.Errorf("file saving not implemented in this example")
}

// Interface compatibility check
var _ TTSInterface = (*OpenAISDKTTSClient)(nil)
