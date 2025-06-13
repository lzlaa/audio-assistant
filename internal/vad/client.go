package vad

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

// Client represents a VAD HTTP client
type Client struct {
	baseURL    string
	httpClient *http.Client
}

// DetectRequest represents the request parameters for VAD detection
type DetectRequest struct {
	Threshold            float64 `json:"threshold,omitempty"`
	MinSpeechDurationMs  int     `json:"min_speech_duration_ms,omitempty"`
	MinSilenceDurationMs int     `json:"min_silence_duration_ms,omitempty"`
}

// SpeechSegment represents a detected speech segment
type SpeechSegment struct {
	Start    float64 `json:"start"`
	End      float64 `json:"end"`
	Duration float64 `json:"duration"`
}

// DetectResponse represents the response from VAD detection
type DetectResponse struct {
	Status         string           `json:"status"`
	Message        string           `json:"message,omitempty"`
	SpeechSegments []SpeechSegment  `json:"speech_segments"`
	Statistics     DetectStatistics `json:"statistics"`
}

// DetectStatistics represents the statistics from VAD detection
type DetectStatistics struct {
	TotalSegments       int     `json:"total_segments"`
	TotalSpeechDuration float64 `json:"total_speech_duration"`
	TotalAudioDuration  float64 `json:"total_audio_duration"`
	SpeechRatio         float64 `json:"speech_ratio"`
	SampleRate          int     `json:"sample_rate"`
	ThresholdUsed       float64 `json:"threshold_used"`
}

// HealthResponse represents the health check response
type HealthResponse struct {
	Status    string `json:"status"`
	Timestamp string `json:"timestamp"`
}

// InfoResponse represents the model info response
type InfoResponse struct {
	ModelName    string `json:"model_name"`
	SampleRate   int    `json:"sample_rate"`
	WindowSizeMs int    `json:"window_size_ms"`
}

// NewClient creates a new VAD client
func NewClient(baseURL string) *Client {
	return &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Health checks if the VAD service is healthy
func (c *Client) Health() (*HealthResponse, error) {
	resp, err := c.httpClient.Get(c.baseURL + "/health")
	if err != nil {
		return nil, fmt.Errorf("failed to call health endpoint: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("health check failed with status: %d", resp.StatusCode)
	}

	var healthResp HealthResponse
	if err := json.NewDecoder(resp.Body).Decode(&healthResp); err != nil {
		return nil, fmt.Errorf("failed to decode health response: %w", err)
	}

	return &healthResp, nil
}

// Info gets information about the VAD model
func (c *Client) Info() (*InfoResponse, error) {
	resp, err := c.httpClient.Get(c.baseURL + "/info")
	if err != nil {
		return nil, fmt.Errorf("failed to call info endpoint: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("info request failed with status: %d", resp.StatusCode)
	}

	var infoResp InfoResponse
	if err := json.NewDecoder(resp.Body).Decode(&infoResp); err != nil {
		return nil, fmt.Errorf("failed to decode info response: %w", err)
	}

	return &infoResp, nil
}

// DetectFromFile detects speech activity from an audio file
func (c *Client) DetectFromFile(audioFilePath string, req *DetectRequest) (*DetectResponse, error) {
	// Open the audio file
	file, err := os.Open(audioFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open audio file: %w", err)
	}
	defer file.Close()

	// Create multipart form
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	// Add audio file
	part, err := writer.CreateFormFile("audio_file", filepath.Base(audioFilePath))
	if err != nil {
		return nil, fmt.Errorf("failed to create form file: %w", err)
	}

	if _, err := io.Copy(part, file); err != nil {
		return nil, fmt.Errorf("failed to copy file content: %w", err)
	}

	// Add optional parameters
	if req != nil {
		if req.Threshold > 0 {
			writer.WriteField("threshold", fmt.Sprintf("%.2f", req.Threshold))
		}
		if req.MinSpeechDurationMs > 0 {
			writer.WriteField("min_speech_duration_ms", fmt.Sprintf("%d", req.MinSpeechDurationMs))
		}
		if req.MinSilenceDurationMs > 0 {
			writer.WriteField("min_silence_duration_ms", fmt.Sprintf("%d", req.MinSilenceDurationMs))
		}
	}

	writer.Close()

	// Create HTTP request
	httpReq, err := http.NewRequest("POST", c.baseURL+"/detect", &buf)
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	httpReq.Header.Set("Content-Type", writer.FormDataContentType())

	// Send request
	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Parse response
	var detectResp DetectResponse
	if err := json.NewDecoder(resp.Body).Decode(&detectResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return &detectResp, fmt.Errorf("detection failed with status %d: %s", resp.StatusCode, detectResp.Message)
	}

	if detectResp.Status != "success" {
		return &detectResp, fmt.Errorf("detection unsuccessful: %s", detectResp.Message)
	}

	return &detectResp, nil
}

// DetectFromBytes detects speech activity from audio bytes
func (c *Client) DetectFromBytes(audioData []byte, filename string, req *DetectRequest) (*DetectResponse, error) {
	// Create multipart form
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	// Add audio data
	part, err := writer.CreateFormFile("audio_file", filename)
	if err != nil {
		return nil, fmt.Errorf("failed to create form file: %w", err)
	}

	if _, err := part.Write(audioData); err != nil {
		return nil, fmt.Errorf("failed to write audio data: %w", err)
	}

	// Add optional parameters
	if req != nil {
		if req.Threshold > 0 {
			writer.WriteField("threshold", fmt.Sprintf("%.2f", req.Threshold))
		}
		if req.MinSpeechDurationMs > 0 {
			writer.WriteField("min_speech_duration_ms", fmt.Sprintf("%d", req.MinSpeechDurationMs))
		}
		if req.MinSilenceDurationMs > 0 {
			writer.WriteField("min_silence_duration_ms", fmt.Sprintf("%d", req.MinSilenceDurationMs))
		}
	}

	writer.Close()

	// Create HTTP request
	httpReq, err := http.NewRequest("POST", c.baseURL+"/detect", &buf)
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	httpReq.Header.Set("Content-Type", writer.FormDataContentType())

	// Send request
	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Parse response
	var detectResp DetectResponse
	if err := json.NewDecoder(resp.Body).Decode(&detectResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return &detectResp, fmt.Errorf("detection failed with status %d: %s", resp.StatusCode, detectResp.Message)
	}

	if detectResp.Status != "success" {
		return &detectResp, fmt.Errorf("detection unsuccessful: %s", detectResp.Message)
	}

	return &detectResp, nil
}

// HasSpeech checks if the audio contains any speech
func (c *Client) HasSpeech(audioFilePath string, req *DetectRequest) (bool, error) {
	resp, err := c.DetectFromFile(audioFilePath, req)
	if err != nil {
		return false, err
	}

	return len(resp.SpeechSegments) > 0, nil
}

// HasSpeechFromBytes checks if the audio bytes contain any speech
func (c *Client) HasSpeechFromBytes(audioData []byte, filename string, req *DetectRequest) (bool, error) {
	resp, err := c.DetectFromBytes(audioData, filename, req)
	if err != nil {
		return false, err
	}

	return len(resp.SpeechSegments) > 0, nil
}
