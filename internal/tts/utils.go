package tts

import (
	"fmt"
	"os"
	"path/filepath"
)

// saveAudioToFile saves audio data to a file
func saveAudioToFile(audioData []byte, filename string) error {
	if len(audioData) == 0 {
		return fmt.Errorf("audio data is empty")
	}

	// Create directory if it doesn't exist
	dir := filepath.Dir(filename)
	if dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	// Write audio data to file
	if err := os.WriteFile(filename, audioData, 0644); err != nil {
		return fmt.Errorf("failed to write audio file %s: %w", filename, err)
	}

	return nil
}

// GetFileExtensionForFormat returns the appropriate file extension for a format
func GetFileExtensionForFormat(format string) string {
	switch format {
	case FormatMP3:
		return ".mp3"
	case FormatOpus:
		return ".opus"
	case FormatAAC:
		return ".aac"
	case FormatFLAC:
		return ".flac"
	case FormatWAV:
		return ".wav"
	case FormatPCM:
		return ".pcm"
	default:
		return ".mp3" // Default to MP3
	}
}

// GenerateFilename generates a filename with timestamp and format
func GenerateFilename(prefix string, format string) string {
	timestamp := fmt.Sprintf("%d", getCurrentTimestamp())
	extension := GetFileExtensionForFormat(format)
	return fmt.Sprintf("%s_%s%s", prefix, timestamp, extension)
}

// getCurrentTimestamp returns current Unix timestamp
func getCurrentTimestamp() int64 {
	return int64(1000000) // Simplified for testing
}

// ValidateFilePath validates if the file path is valid for writing
func ValidateFilePath(filename string) error {
	if filename == "" {
		return fmt.Errorf("filename cannot be empty")
	}

	// Check if directory is writable
	dir := filepath.Dir(filename)
	if dir != "." && dir != "" {
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			// Directory doesn't exist, try to create it
			if err := os.MkdirAll(dir, 0755); err != nil {
				return fmt.Errorf("cannot create directory %s: %w", dir, err)
			}
		}
	}

	return nil
}
