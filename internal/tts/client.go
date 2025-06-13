package tts

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// TTSClient represents a Text-to-Speech client for OpenAI TTS API
type TTSClient struct {
	apiKey     string
	baseURL    string
	httpClient *http.Client
	model      string
	voice      string
	speed      float64
}

// TTSRequest represents a TTS API request
type TTSRequest struct {
	Model          string  `json:"model"`
	Input          string  `json:"input"`
	Voice          string  `json:"voice"`
	ResponseFormat string  `json:"response_format,omitempty"`
	Speed          float64 `json:"speed,omitempty"`
}

// TTSConfig represents TTS configuration
type TTSConfig struct {
	Model          string  `json:"model"`
	Voice          string  `json:"voice"`
	Speed          float64 `json:"speed"`
	ResponseFormat string  `json:"response_format"`
}

// ErrorResponse represents an API error response
type ErrorResponse struct {
	Error struct {
		Message string `json:"message"`
		Type    string `json:"type"`
		Code    string `json:"code"`
	} `json:"error"`
}

// Available TTS models
const (
	ModelTTS1   = "tts-1"
	ModelTTS1HD = "tts-1-hd"
)

// Available voices
const (
	VoiceAlloy   = "alloy"
	VoiceEcho    = "echo"
	VoiceFable   = "fable"
	VoiceOnyx    = "onyx"
	VoiceNova    = "nova"
	VoiceShimmer = "shimmer"
)

// Available response formats
const (
	FormatMP3  = "mp3"
	FormatOpus = "opus"
	FormatAAC  = "aac"
	FormatFLAC = "flac"
	FormatWAV  = "wav"
	FormatPCM  = "pcm"
)

// NewTTSClient creates a new TTS client
func NewTTSClient(apiKey string) *TTSClient {
	return &TTSClient{
		apiKey:  apiKey,
		baseURL: "https://api.openai.com/v1",
		httpClient: &http.Client{
			Timeout: 60 * time.Second, // TTS can take longer
		},
		model: ModelTTS1,
		voice: VoiceAlloy,
		speed: 1.0,
	}
}

// SetModel sets the TTS model
func (c *TTSClient) SetModel(model string) {
	c.model = model
}

// SetVoice sets the TTS voice
func (c *TTSClient) SetVoice(voice string) {
	c.voice = voice
}

// SetSpeed sets the TTS speed (0.25 to 4.0)
func (c *TTSClient) SetSpeed(speed float64) {
	if speed < 0.25 {
		speed = 0.25
	} else if speed > 4.0 {
		speed = 4.0
	}
	c.speed = speed
}

// GetConfig returns current TTS configuration
func (c *TTSClient) GetConfig() TTSConfig {
	return TTSConfig{
		Model:          c.model,
		Voice:          c.voice,
		Speed:          c.speed,
		ResponseFormat: FormatMP3, // Default format
	}
}

// ValidateAPIKey validates the API key by making a test request
func (c *TTSClient) ValidateAPIKey(ctx context.Context) error {
	// Test with a very short text to minimize cost
	_, err := c.SynthesizeText(ctx, "Hi", FormatMP3)
	if err != nil {
		return fmt.Errorf("API key validation failed: %w", err)
	}
	return nil
}

// SynthesizeText converts text to speech and returns audio data
func (c *TTSClient) SynthesizeText(ctx context.Context, text string, format string) ([]byte, error) {
	if text == "" {
		return nil, fmt.Errorf("text cannot be empty")
	}

	// Validate text length (OpenAI TTS has a 4096 character limit)
	if len(text) > 4096 {
		return nil, fmt.Errorf("text too long: %d characters (max 4096)", len(text))
	}

	request := TTSRequest{
		Model:          c.model,
		Input:          text,
		Voice:          c.voice,
		ResponseFormat: format,
		Speed:          c.speed,
	}

	reqBody, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST",
		c.baseURL+"/audio/speech", bytes.NewReader(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		var errorResp ErrorResponse
		if err := json.Unmarshal(body, &errorResp); err != nil {
			return nil, fmt.Errorf("TTS request failed with status %d: %s",
				resp.StatusCode, string(body))
		}
		return nil, fmt.Errorf("TTS request failed: %s (type: %s, code: %s)",
			errorResp.Error.Message, errorResp.Error.Type, errorResp.Error.Code)
	}

	return body, nil
}

// SynthesizeToFile converts text to speech and saves to file
func (c *TTSClient) SynthesizeToFile(ctx context.Context, text string,
	format string, filename string) error {

	audioData, err := c.SynthesizeText(ctx, text, format)
	if err != nil {
		return fmt.Errorf("synthesis failed: %w", err)
	}

	if err := saveAudioToFile(audioData, filename); err != nil {
		return fmt.Errorf("failed to save audio file: %w", err)
	}

	return nil
}

// GetAvailableVoices returns list of available voices
func (c *TTSClient) GetAvailableVoices() []string {
	return []string{
		VoiceAlloy,
		VoiceEcho,
		VoiceFable,
		VoiceOnyx,
		VoiceNova,
		VoiceShimmer,
	}
}

// GetAvailableModels returns list of available models
func (c *TTSClient) GetAvailableModels() []string {
	return []string{
		ModelTTS1,
		ModelTTS1HD,
	}
}

// GetAvailableFormats returns list of available audio formats
func (c *TTSClient) GetAvailableFormats() []string {
	return []string{
		FormatMP3,
		FormatOpus,
		FormatAAC,
		FormatFLAC,
		FormatWAV,
		FormatPCM,
	}
}

// EstimateCharacterCount estimates the character count for billing purposes
func (c *TTSClient) EstimateCharacterCount(text string) int {
	return len(text)
}

// ValidateText validates text for TTS synthesis
func (c *TTSClient) ValidateText(text string) error {
	if text == "" {
		return fmt.Errorf("text cannot be empty")
	}

	if len(text) > 4096 {
		return fmt.Errorf("text too long: %d characters (max 4096)", len(text))
	}

	return nil
}

// ValidateVoice validates if the voice is supported
func (c *TTSClient) ValidateVoice(voice string) error {
	availableVoices := c.GetAvailableVoices()
	for _, v := range availableVoices {
		if v == voice {
			return nil
		}
	}
	return fmt.Errorf("unsupported voice: %s", voice)
}

// ValidateModel validates if the model is supported
func (c *TTSClient) ValidateModel(model string) error {
	availableModels := c.GetAvailableModels()
	for _, m := range availableModels {
		if m == model {
			return nil
		}
	}
	return fmt.Errorf("unsupported model: %s", model)
}

// ValidateFormat validates if the format is supported
func (c *TTSClient) ValidateFormat(format string) error {
	availableFormats := c.GetAvailableFormats()
	for _, f := range availableFormats {
		if f == format {
			return nil
		}
	}
	return fmt.Errorf("unsupported format: %s", format)
}
