package asr

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Client represents an ASR client for OpenAI Whisper API
type Client struct {
	apiKey     string
	baseURL    string
	httpClient *http.Client
}

// TranscribeRequest represents the request parameters for transcription
type TranscribeRequest struct {
	Model       string  `json:"model,omitempty"`           // whisper-1
	Language    string  `json:"language,omitempty"`        // ISO-639-1 language code
	Prompt      string  `json:"prompt,omitempty"`          // Optional text to guide the model's style
	Temperature float32 `json:"temperature,omitempty"`     // Sampling temperature (0-1)
	Format      string  `json:"response_format,omitempty"` // json, text, srt, verbose_json, vtt
}

// TranscribeResponse represents the response from transcription
type TranscribeResponse struct {
	Text     string    `json:"text"`
	Language string    `json:"language,omitempty"`
	Duration float64   `json:"duration,omitempty"`
	Segments []Segment `json:"segments,omitempty"`
}

// Segment represents a transcription segment with timing
type Segment struct {
	ID               int     `json:"id"`
	Seek             int     `json:"seek"`
	Start            float64 `json:"start"`
	End              float64 `json:"end"`
	Text             string  `json:"text"`
	Tokens           []int   `json:"tokens"`
	Temperature      float32 `json:"temperature"`
	AvgLogprob       float64 `json:"avg_logprob"`
	CompressionRatio float64 `json:"compression_ratio"`
	NoSpeechProb     float64 `json:"no_speech_prob"`
}

// ErrorResponse represents an error response from the API
type ErrorResponse struct {
	Error struct {
		Message string `json:"message"`
		Type    string `json:"type"`
		Code    string `json:"code"`
	} `json:"error"`
}

// NewClient creates a new ASR client
func NewClient(apiKey string) *Client {
	return &Client{
		apiKey:  apiKey,
		baseURL: "https://api.openai.com/v1",
		httpClient: &http.Client{
			Timeout: 60 * time.Second, // Longer timeout for audio processing
		},
	}
}

// NewClientWithConfig creates a new ASR client with custom configuration
func NewClientWithConfig(apiKey, baseURL string, timeout time.Duration) *Client {
	return &Client{
		apiKey:  apiKey,
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}
}

// TranscribeFile transcribes an audio file to text
func (c *Client) TranscribeFile(ctx context.Context, audioFilePath string, req *TranscribeRequest) (*TranscribeResponse, error) {
	// Open the audio file
	file, err := os.Open(audioFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open audio file: %w", err)
	}
	defer file.Close()

	// Get file info
	fileInfo, err := file.Stat()
	if err != nil {
		return nil, fmt.Errorf("failed to get file info: %w", err)
	}

	// Check file size (OpenAI limit is 25MB)
	const maxFileSize = 25 * 1024 * 1024 // 25MB
	if fileInfo.Size() > maxFileSize {
		return nil, fmt.Errorf("file size %d bytes exceeds maximum allowed size of %d bytes", fileInfo.Size(), maxFileSize)
	}

	// Validate file extension
	ext := strings.ToLower(filepath.Ext(audioFilePath))
	supportedFormats := []string{".mp3", ".mp4", ".mpeg", ".mpga", ".m4a", ".wav", ".webm"}
	isSupported := false
	for _, format := range supportedFormats {
		if ext == format {
			isSupported = true
			break
		}
	}
	if !isSupported {
		return nil, fmt.Errorf("unsupported audio format: %s. Supported formats: %v", ext, supportedFormats)
	}

	return c.transcribeReader(ctx, file, filepath.Base(audioFilePath), req)
}

// TranscribeBytes transcribes audio data from bytes to text
func (c *Client) TranscribeBytes(ctx context.Context, audioData []byte, filename string, req *TranscribeRequest) (*TranscribeResponse, error) {
	// Check data size
	const maxFileSize = 25 * 1024 * 1024 // 25MB
	if len(audioData) > maxFileSize {
		return nil, fmt.Errorf("data size %d bytes exceeds maximum allowed size of %d bytes", len(audioData), maxFileSize)
	}

	reader := bytes.NewReader(audioData)
	return c.transcribeReader(ctx, reader, filename, req)
}

// transcribeReader handles the actual transcription request
func (c *Client) transcribeReader(ctx context.Context, reader io.Reader, filename string, req *TranscribeRequest) (*TranscribeResponse, error) {
	// Set default values
	if req == nil {
		req = &TranscribeRequest{}
	}
	if req.Model == "" {
		req.Model = "whisper-1"
	}
	if req.Format == "" {
		req.Format = "verbose_json" // Get detailed response with segments
	}

	// Create multipart form
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	// Add audio file
	part, err := writer.CreateFormFile("file", filename)
	if err != nil {
		return nil, fmt.Errorf("failed to create form file: %w", err)
	}

	if _, err := io.Copy(part, reader); err != nil {
		return nil, fmt.Errorf("failed to copy audio data: %w", err)
	}

	// Add model
	if err := writer.WriteField("model", req.Model); err != nil {
		return nil, fmt.Errorf("failed to write model field: %w", err)
	}

	// Add optional parameters
	if req.Language != "" {
		if err := writer.WriteField("language", req.Language); err != nil {
			return nil, fmt.Errorf("failed to write language field: %w", err)
		}
	}

	if req.Prompt != "" {
		if err := writer.WriteField("prompt", req.Prompt); err != nil {
			return nil, fmt.Errorf("failed to write prompt field: %w", err)
		}
	}

	if req.Temperature > 0 {
		if err := writer.WriteField("temperature", fmt.Sprintf("%.2f", req.Temperature)); err != nil {
			return nil, fmt.Errorf("failed to write temperature field: %w", err)
		}
	}

	if req.Format != "" {
		if err := writer.WriteField("response_format", req.Format); err != nil {
			return nil, fmt.Errorf("failed to write response_format field: %w", err)
		}
	}

	writer.Close()

	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/audio/transcriptions", &buf)
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
		var errorResp ErrorResponse
		if err := json.Unmarshal(body, &errorResp); err != nil {
			return nil, fmt.Errorf("transcription failed with status %d: %s", resp.StatusCode, string(body))
		}
		return nil, fmt.Errorf("transcription failed: %s (type: %s, code: %s)",
			errorResp.Error.Message, errorResp.Error.Type, errorResp.Error.Code)
	}

	// Parse response based on format
	if req.Format == "text" {
		return &TranscribeResponse{
			Text: string(body),
		}, nil
	}

	// Parse JSON response
	var transcribeResp TranscribeResponse
	if err := json.Unmarshal(body, &transcribeResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &transcribeResp, nil
}

// TranscribeSimple provides a simple interface for transcription with default settings
func (c *Client) TranscribeSimple(ctx context.Context, audioFilePath string) (string, error) {
	req := &TranscribeRequest{
		Model:  "whisper-1",
		Format: "text",
	}

	resp, err := c.TranscribeFile(ctx, audioFilePath, req)
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(resp.Text), nil
}

// TranscribeSimpleBytes provides a simple interface for transcription from bytes
func (c *Client) TranscribeSimpleBytes(ctx context.Context, audioData []byte, filename string) (string, error) {
	req := &TranscribeRequest{
		Model:  "whisper-1",
		Format: "text",
	}

	resp, err := c.TranscribeBytes(ctx, audioData, filename, req)
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(resp.Text), nil
}

// TranscribeWithLanguage transcribes audio with a specific language hint
func (c *Client) TranscribeWithLanguage(ctx context.Context, audioFilePath, language string) (string, error) {
	req := &TranscribeRequest{
		Model:    "whisper-1",
		Language: language,
		Format:   "text",
	}

	resp, err := c.TranscribeFile(ctx, audioFilePath, req)
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(resp.Text), nil
}

// GetSupportedLanguages returns a list of supported language codes
func (c *Client) GetSupportedLanguages() []string {
	return []string{
		"af", "ar", "hy", "az", "be", "bs", "bg", "ca", "zh", "hr", "cs", "da", "nl", "en", "et", "fi", "fr", "gl", "de", "el", "he", "hi", "hu", "is", "id", "it", "ja", "kn", "kk", "ko", "lv", "lt", "mk", "ms", "ml", "mt", "mi", "mr", "ne", "no", "fa", "pl", "pt", "ro", "ru", "sr", "sk", "sl", "es", "sw", "sv", "tl", "ta", "th", "tr", "uk", "ur", "vi", "cy",
	}
}

// ValidateAPIKey checks if the API key is valid by making a simple request
func (c *Client) ValidateAPIKey(ctx context.Context) error {
	// Create a minimal test request
	req, err := http.NewRequestWithContext(ctx, "GET", c.baseURL+"/models", nil)
	if err != nil {
		return fmt.Errorf("failed to create test request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send test request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		return fmt.Errorf("invalid API key")
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API validation failed with status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}
