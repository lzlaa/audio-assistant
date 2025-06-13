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

// SynthesizeOptions represents options for text synthesis
type SynthesizeOptions struct {
	Model  string  `json:"model,omitempty"`
	Voice  string  `json:"voice,omitempty"`
	Speed  float32 `json:"speed,omitempty"`
	Format string  `json:"format,omitempty"`
}

// SynthesizeResponse represents the response from synthesis
type SynthesizeResponse struct {
	AudioData string `json:"audio_data"` // base64 encoded audio
	Format    string `json:"format"`
	RequestID string `json:"request_id,omitempty"`
}

// Voice represents a TTS voice
type Voice struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Language string `json:"language"`
	Gender   string `json:"gender"`
}

// QwenTTSClient represents a client for Alibaba Qwen TTS API
type QwenTTSClient struct {
	apiKey     string
	baseURL    string
	httpClient *http.Client
}

// QwenTTSRequest represents the request parameters for Qwen TTS
type QwenTTSRequest struct {
	Model      string            `json:"model"`                // cosyvoice-v1, qwen-tts-v1, etc.
	Input      QwenTTSInput      `json:"input"`                // text input
	Parameters QwenTTSParameters `json:"parameters,omitempty"` // optional parameters
}

// QwenTTSInput represents the text input for Qwen TTS
type QwenTTSInput struct {
	Text string `json:"text"` // text to synthesize
}

// QwenTTSParameters represents optional parameters for Qwen TTS
type QwenTTSParameters struct {
	Voice  string  `json:"voice,omitempty"`  // voice name
	Rate   float32 `json:"rate,omitempty"`   // speech rate (0.5-2.0)
	Volume float32 `json:"volume,omitempty"` // volume (0.0-1.0)
	Pitch  float32 `json:"pitch,omitempty"`  // pitch (-20.0 to 20.0)
	Format string  `json:"format,omitempty"` // audio format (mp3, wav, etc.)
}

// QwenTTSResponse represents the response from Qwen TTS
type QwenTTSResponse struct {
	Output struct {
		Audio string `json:"audio"` // base64 encoded audio data
	} `json:"output"`
	Usage struct {
		Characters int `json:"characters"` // number of characters processed
	} `json:"usage"`
	RequestID string `json:"request_id"`
}

// QwenTTSErrorResponse represents an error response from Qwen TTS API
type QwenTTSErrorResponse struct {
	Code      string `json:"code"`
	Message   string `json:"message"`
	RequestID string `json:"request_id"`
}

// NewQwenTTSClient creates a new Qwen TTS client
func NewQwenTTSClient(apiKey string) *QwenTTSClient {
	return &QwenTTSClient{
		apiKey:  apiKey,
		baseURL: "https://dashscope.aliyuncs.com/api/v1",
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

// NewQwenTTSClientWithConfig creates a new Qwen TTS client with custom configuration
func NewQwenTTSClientWithConfig(apiKey, baseURL string, timeout time.Duration) *QwenTTSClient {
	return &QwenTTSClient{
		apiKey:  apiKey,
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}
}

// SynthesizeText synthesizes text to speech using Qwen TTS API
func (c *QwenTTSClient) SynthesizeText(ctx context.Context, text string, options *SynthesizeOptions) (*SynthesizeResponse, error) {
	// Convert options to Qwen format
	qwenReq := &QwenTTSRequest{
		Model: "cosyvoice-v1",
		Input: QwenTTSInput{
			Text: text,
		},
		Parameters: QwenTTSParameters{},
	}

	// Set parameters from options
	if options != nil {
		if options.Voice != "" {
			qwenReq.Parameters.Voice = convertVoiceToQwen(options.Voice)
		}
		if options.Speed > 0 {
			qwenReq.Parameters.Rate = options.Speed
		}
		if options.Format != "" {
			qwenReq.Parameters.Format = options.Format
		}
		if options.Model != "" {
			qwenReq.Model = convertModelToQwenTTS(options.Model)
		}
	}

	// Set default values
	if qwenReq.Parameters.Voice == "" {
		qwenReq.Parameters.Voice = "longxiaochun" // Default voice
	}
	if qwenReq.Parameters.Rate == 0 {
		qwenReq.Parameters.Rate = 1.0
	}
	if qwenReq.Parameters.Format == "" {
		qwenReq.Parameters.Format = "mp3"
	}

	// Marshal request
	reqBody, err := json.Marshal(qwenReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/services/aigc/text2speech/synthesis", bytes.NewReader(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	// Set headers
	httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)
	httpReq.Header.Set("Content-Type", "application/json")

	// Send request
	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// Handle error responses
	if resp.StatusCode != http.StatusOK {
		var errorResp QwenTTSErrorResponse
		if err := json.Unmarshal(body, &errorResp); err != nil {
			return nil, fmt.Errorf("synthesis failed with status %d: %s", resp.StatusCode, string(body))
		}
		return nil, fmt.Errorf("synthesis failed: %s (code: %s, request_id: %s)",
			errorResp.Message, errorResp.Code, errorResp.RequestID)
	}

	// Parse Qwen TTS response
	var qwenResp QwenTTSResponse
	if err := json.Unmarshal(body, &qwenResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// Convert to standard response format
	synthesizeResp := &SynthesizeResponse{
		AudioData: qwenResp.Output.Audio, // base64 encoded
		Format:    qwenReq.Parameters.Format,
		RequestID: qwenResp.RequestID,
	}

	return synthesizeResp, nil
}

// SynthesizeToFile synthesizes text to speech and saves to file
func (c *QwenTTSClient) SynthesizeToFile(ctx context.Context, text, outputPath string, options *SynthesizeOptions) error {
	resp, err := c.SynthesizeText(ctx, text, options)
	if err != nil {
		return err
	}

	return saveQwenAudioToFile(resp.AudioData, outputPath)
}

// GetAvailableVoices returns available voices for Qwen TTS
func (c *QwenTTSClient) GetAvailableVoices() []Voice {
	return []Voice{
		{ID: "longxiaochun", Name: "龙小春", Language: "zh-CN", Gender: "female"},
		{ID: "longyueyue", Name: "龙悦悦", Language: "zh-CN", Gender: "female"},
		{ID: "longxiaobai", Name: "龙小白", Language: "zh-CN", Gender: "male"},
		{ID: "longxiaoxin", Name: "龙小新", Language: "zh-CN", Gender: "male"},
		{ID: "longxiaoyu", Name: "龙小雨", Language: "zh-CN", Gender: "female"},
		{ID: "longtianxiang", Name: "龙天翔", Language: "zh-CN", Gender: "male"},
		{ID: "longxiaoming", Name: "龙小明", Language: "zh-CN", Gender: "male"},
		{ID: "longxiaoli", Name: "龙小丽", Language: "zh-CN", Gender: "female"},
		{ID: "longxiaogang", Name: "龙小刚", Language: "zh-CN", Gender: "male"},
		{ID: "longxiaohong", Name: "龙小红", Language: "zh-CN", Gender: "female"},
	}
}

// GetSupportedFormats returns supported audio formats
func (c *QwenTTSClient) GetSupportedFormats() []string {
	return []string{
		"mp3",
		"wav",
		"pcm",
	}
}

// GetSupportedModels returns supported TTS models
func (c *QwenTTSClient) GetSupportedModels() []string {
	return []string{
		"cosyvoice-v1",
		"qwen-tts-v1",
	}
}

// ValidateAPIKey validates the API key by making a simple request
func (c *QwenTTSClient) ValidateAPIKey(ctx context.Context) error {
	_, err := c.SynthesizeText(ctx, "Hello", &SynthesizeOptions{
		Voice:  "longxiaochun",
		Format: "mp3",
	})

	if err != nil {
		if containsAnyString(err.Error(), []string{"Invalid API key", "Unauthorized", "InvalidApiKey"}) {
			return fmt.Errorf("invalid API key")
		}
		return fmt.Errorf("API key validation failed: %w", err)
	}

	return nil
}

// Helper functions

// convertModelToQwenTTS converts standard model names to Qwen TTS equivalents
func convertModelToQwenTTS(model string) string {
	switch model {
	case "tts-1":
		return "cosyvoice-v1"
	case "tts-1-hd":
		return "qwen-tts-v1"
	default:
		if containsAnyString(model, []string{"cosyvoice", "qwen-tts"}) {
			return model
		}
		return "cosyvoice-v1" // default
	}
}

// convertVoiceToQwen converts standard voice names to Qwen equivalents
func convertVoiceToQwen(voice string) string {
	switch voice {
	case "alloy":
		return "longxiaochun"
	case "echo":
		return "longyueyue"
	case "fable":
		return "longxiaobai"
	case "onyx":
		return "longxiaoxin"
	case "nova":
		return "longxiaoyu"
	case "shimmer":
		return "longtianxiang"
	default:
		if containsAnyString(voice, []string{"long"}) {
			return voice
		}
		return "longxiaochun" // default
	}
}

// containsAnyString checks if the text contains any of the given substrings
func containsAnyString(text string, substrings []string) bool {
	for _, substr := range substrings {
		if len(substr) > 0 && len(text) >= len(substr) {
			for i := 0; i <= len(text)-len(substr); i++ {
				if text[i:i+len(substr)] == substr {
					return true
				}
			}
		}
	}
	return false
}

// saveQwenAudioToFile saves base64 encoded audio data to file
func saveQwenAudioToFile(base64Audio, outputPath string) error {
	// This would typically decode base64 and save to file
	// For now, return an error indicating this needs to be implemented
	return fmt.Errorf("file saving not implemented in this example")
}

// Interface compatibility check
var _ TTSInterface = (*QwenTTSClient)(nil)

// TTSInterface defines the interface that TTS clients should implement
type TTSInterface interface {
	SynthesizeText(ctx context.Context, text string, options *SynthesizeOptions) (*SynthesizeResponse, error)
	SynthesizeToFile(ctx context.Context, text, outputPath string, options *SynthesizeOptions) error
	GetAvailableVoices() []Voice
	GetSupportedFormats() []string
	GetSupportedModels() []string
	ValidateAPIKey(ctx context.Context) error
}
