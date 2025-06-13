package asr

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"time"
)

// QwenASRClient represents a client for Alibaba Qwen ASR API
type QwenASRClient struct {
	apiKey     string
	baseURL    string
	httpClient *http.Client
}

// QwenASRRequest represents the request parameters for Qwen ASR
type QwenASRRequest struct {
	Model      string            `json:"model"`                // paraformer-realtime-v2, paraformer-v1, etc.
	Input      QwenASRInput      `json:"input"`                // audio input
	Parameters QwenASRParameters `json:"parameters,omitempty"` // optional parameters
}

// QwenASRInput represents the audio input for Qwen ASR
type QwenASRInput struct {
	Audio string `json:"audio"` // base64 encoded audio or URL
}

// QwenASRParameters represents optional parameters for Qwen ASR
type QwenASRParameters struct {
	SampleRate  int    `json:"sample_rate,omitempty"`  // audio sample rate
	Format      string `json:"format,omitempty"`       // audio format (wav, mp3, etc.)
	ChannelNum  int    `json:"channel_num,omitempty"`  // number of audio channels
	EnableWords bool   `json:"enable_words,omitempty"` // enable word-level timestamps
	MaxSentence int    `json:"max_sentence,omitempty"` // max sentence length
	Language    string `json:"language,omitempty"`     // language code (zh, en, etc.)
}

// QwenASRResponse represents the response from Qwen ASR
type QwenASRResponse struct {
	Output struct {
		Text      string `json:"text"`
		Sentences []struct {
			Text      string `json:"text"`
			BeginTime int    `json:"begin_time"`
			EndTime   int    `json:"end_time"`
			Words     []struct {
				Text      string `json:"text"`
				BeginTime int    `json:"begin_time"`
				EndTime   int    `json:"end_time"`
			} `json:"words,omitempty"`
		} `json:"sentences,omitempty"`
	} `json:"output"`
	Usage struct {
		Duration int `json:"duration"` // audio duration in milliseconds
	} `json:"usage"`
	RequestID string `json:"request_id"`
}

// QwenASRErrorResponse represents an error response from Qwen ASR API
type QwenASRErrorResponse struct {
	Code      string `json:"code"`
	Message   string `json:"message"`
	RequestID string `json:"request_id"`
}

// NewQwenASRClient creates a new Qwen ASR client
func NewQwenASRClient(apiKey string) *QwenASRClient {
	return &QwenASRClient{
		apiKey:  apiKey,
		baseURL: "https://dashscope.aliyuncs.com/api/v1",
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

// NewQwenASRClientWithConfig creates a new Qwen ASR client with custom configuration
func NewQwenASRClientWithConfig(apiKey, baseURL string, timeout time.Duration) *QwenASRClient {
	return &QwenASRClient{
		apiKey:  apiKey,
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}
}

// TranscribeAudio transcribes audio using Qwen ASR API
func (c *QwenASRClient) TranscribeAudio(ctx context.Context, audioData []byte, options *TranscribeRequest) (*TranscribeResponse, error) {
	// Convert audio data to multipart form
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	// Add model field
	model := "paraformer-realtime-v2"
	if options != nil && options.Model != "" {
		model = convertModelToQwenASR(options.Model)
	}

	err := writer.WriteField("model", model)
	if err != nil {
		return nil, fmt.Errorf("failed to write model field: %w", err)
	}

	// Add audio file
	part, err := writer.CreateFormFile("audio", "audio.wav")
	if err != nil {
		return nil, fmt.Errorf("failed to create form file: %w", err)
	}

	_, err = part.Write(audioData)
	if err != nil {
		return nil, fmt.Errorf("failed to write audio data: %w", err)
	}

	// Add optional parameters
	if options != nil {
		if options.Language != "" {
			writer.WriteField("language", options.Language)
		}
		if options.Temperature > 0 {
			writer.WriteField("temperature", fmt.Sprintf("%.2f", options.Temperature))
		}
	}

	err = writer.Close()
	if err != nil {
		return nil, fmt.Errorf("failed to close multipart writer: %w", err)
	}

	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/services/aigc/asr/transcription", &buf)
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	// Set headers
	httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)
	httpReq.Header.Set("Content-Type", writer.FormDataContentType())

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
		var errorResp QwenASRErrorResponse
		if err := json.Unmarshal(body, &errorResp); err != nil {
			return nil, fmt.Errorf("transcription failed with status %d: %s", resp.StatusCode, string(body))
		}
		return nil, fmt.Errorf("transcription failed: %s (code: %s, request_id: %s)",
			errorResp.Message, errorResp.Code, errorResp.RequestID)
	}

	// Parse Qwen ASR response
	var qwenResp QwenASRResponse
	if err := json.Unmarshal(body, &qwenResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// Convert to standard response format
	transcribeResp := &TranscribeResponse{
		Text:     qwenResp.Output.Text,
		Language: detectLanguage(qwenResp.Output.Text),
		Duration: float64(qwenResp.Usage.Duration) / 1000.0, // convert ms to seconds
	}

	// Add segments if available
	if len(qwenResp.Output.Sentences) > 0 {
		transcribeResp.Segments = make([]Segment, len(qwenResp.Output.Sentences))
		for i, sentence := range qwenResp.Output.Sentences {
			transcribeResp.Segments[i] = Segment{
				ID:    i,
				Start: float64(sentence.BeginTime) / 1000.0, // convert ms to seconds
				End:   float64(sentence.EndTime) / 1000.0,   // convert ms to seconds
				Text:  sentence.Text,
			}
		}
	}

	return transcribeResp, nil
}

// TranscribeFile transcribes an audio file using Qwen ASR API
func (c *QwenASRClient) TranscribeFile(ctx context.Context, filePath string, options *TranscribeRequest) (*TranscribeResponse, error) {
	// Read audio file
	audioData, err := readAudioFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read audio file: %w", err)
	}

	return c.TranscribeAudio(ctx, audioData, options)
}

// GetSupportedLanguages returns supported languages for Qwen ASR
func (c *QwenASRClient) GetSupportedLanguages() []string {
	return []string{
		"zh",   // Chinese
		"en",   // English
		"ja",   // Japanese
		"ko",   // Korean
		"es",   // Spanish
		"fr",   // French
		"de",   // German
		"it",   // Italian
		"pt",   // Portuguese
		"ru",   // Russian
		"ar",   // Arabic
		"hi",   // Hindi
		"th",   // Thai
		"vi",   // Vietnamese
		"id",   // Indonesian
		"ms",   // Malay
		"auto", // Auto-detect
	}
}

// GetSupportedModels returns supported models for Qwen ASR
func (c *QwenASRClient) GetSupportedModels() []string {
	return []string{
		"paraformer-realtime-v2",
		"paraformer-v1",
		"paraformer-8k-v1",
		"paraformer-mtl-v1",
	}
}

// ValidateAPIKey validates the API key by making a simple request
func (c *QwenASRClient) ValidateAPIKey(ctx context.Context) error {
	// Create a small test audio (silence)
	testAudio := make([]byte, 1024)

	_, err := c.TranscribeAudio(ctx, testAudio, &TranscribeRequest{
		Model:    "paraformer-realtime-v2",
		Language: "zh",
	})

	if err != nil {
		if containsAnyString(err.Error(), []string{"Invalid API key", "Unauthorized", "InvalidApiKey"}) {
			return fmt.Errorf("invalid API key")
		}
		// For test audio, we might get other errors, but if it's not auth-related, the key is probably valid
		return nil
	}

	return nil
}

// Helper functions

// convertModelToQwenASR converts standard model names to Qwen ASR equivalents
func convertModelToQwenASR(model string) string {
	switch model {
	case "whisper-1":
		return "paraformer-realtime-v2"
	case "whisper-large":
		return "paraformer-v1"
	default:
		if containsAnyString(model, []string{"paraformer"}) {
			return model
		}
		return "paraformer-realtime-v2" // default
	}
}

// detectLanguage attempts to detect language from text
func detectLanguage(text string) string {
	if text == "" {
		return "unknown"
	}

	// Simple heuristic: check for Chinese characters
	for _, r := range text {
		if r >= 0x4e00 && r <= 0x9fff {
			return "zh"
		}
	}

	// Default to English for other cases
	return "en"
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

// readAudioFile reads an audio file and returns its content
func readAudioFile(filePath string) ([]byte, error) {
	// This would typically use os.ReadFile or similar
	// For now, return an error indicating this needs to be implemented
	return nil, fmt.Errorf("file reading not implemented in this example")
}

// Interface compatibility check
var _ ASRInterface = (*QwenASRClient)(nil)

// ASRInterface defines the interface that ASR clients should implement
type ASRInterface interface {
	TranscribeAudio(ctx context.Context, audioData []byte, options *TranscribeRequest) (*TranscribeResponse, error)
	TranscribeFile(ctx context.Context, filePath string, options *TranscribeRequest) (*TranscribeResponse, error)
	GetSupportedLanguages() []string
	GetSupportedModels() []string
	ValidateAPIKey(ctx context.Context) error
}
