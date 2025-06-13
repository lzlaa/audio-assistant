package audio

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"os"
)

// WAVChunk represents a generic WAV chunk
type WAVChunk struct {
	ID   [4]byte
	Size uint32
}

// RobustLoadFromWAV loads audio data from a WAV file with better error handling
func RobustLoadFromWAV(filename string) ([]float32, int, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to open WAV file: %w", err)
	}
	defer file.Close()

	// Get file size
	fileInfo, err := file.Stat()
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get file info: %w", err)
	}
	fileSize := fileInfo.Size()

	// Read entire file content for analysis
	fileContent := make([]byte, fileSize)
	if _, err := io.ReadFull(file, fileContent); err != nil {
		return nil, 0, fmt.Errorf("failed to read file content: %w", err)
	}

	return parseWAVContent(fileContent, filename)
}

// parseWAVContent parses WAV content from bytes
func parseWAVContent(content []byte, filename string) ([]float32, int, error) {
	if len(content) < 12 {
		return nil, 0, fmt.Errorf("file too small to be a valid WAV file")
	}

	reader := bytes.NewReader(content)

	// Read RIFF header
	var riffHeader struct {
		ChunkID   [4]byte
		ChunkSize uint32
		Format    [4]byte
	}

	if err := binary.Read(reader, binary.LittleEndian, &riffHeader); err != nil {
		return nil, 0, fmt.Errorf("failed to read RIFF header: %w", err)
	}

	if string(riffHeader.ChunkID[:]) != "RIFF" {
		return nil, 0, fmt.Errorf("not a RIFF file: %s", string(riffHeader.ChunkID[:]))
	}

	if string(riffHeader.Format[:]) != "WAVE" {
		return nil, 0, fmt.Errorf("not a WAVE file: %s", string(riffHeader.Format[:]))
	}

	fmt.Printf("WAV file analysis for %s:\n", filename)
	fmt.Printf("  RIFF chunk size: %d\n", riffHeader.ChunkSize)
	fmt.Printf("  File size: %d\n", len(content))

	// Parse chunks
	var fmtChunk *FmtChunk
	var dataOffset int64
	var dataSize uint32

	for {
		var chunk WAVChunk
		if err := binary.Read(reader, binary.LittleEndian, &chunk); err != nil {
			if err == io.EOF {
				break
			}
			return nil, 0, fmt.Errorf("failed to read chunk header: %w", err)
		}

		chunkID := string(chunk.ID[:])
		fmt.Printf("  Found chunk: %s, size: %d\n", chunkID, chunk.Size)

		switch chunkID {
		case "fmt ":
			fmtData := make([]byte, chunk.Size)
			if _, err := io.ReadFull(reader, fmtData); err != nil {
				return nil, 0, fmt.Errorf("failed to read fmt chunk: %w", err)
			}

			var parseErr error
			fmtChunk, parseErr = parseFmtChunk(fmtData)
			if parseErr != nil {
				return nil, 0, fmt.Errorf("failed to parse fmt chunk: %w", parseErr)
			}

		case "data":
			dataOffset, _ = reader.Seek(0, io.SeekCurrent)
			dataSize = chunk.Size

			// Skip the data chunk for now
			if _, err := reader.Seek(int64(chunk.Size), io.SeekCurrent); err != nil {
				return nil, 0, fmt.Errorf("failed to skip data chunk: %w", err)
			}

		default:
			// Skip unknown chunks
			if _, err := reader.Seek(int64(chunk.Size), io.SeekCurrent); err != nil {
				return nil, 0, fmt.Errorf("failed to skip chunk %s: %w", chunkID, err)
			}
		}
	}

	if fmtChunk == nil {
		return nil, 0, fmt.Errorf("fmt chunk not found")
	}

	if dataOffset == 0 {
		return nil, 0, fmt.Errorf("data chunk not found")
	}

	fmt.Printf("  Audio format: %d (PCM=%d)\n", fmtChunk.AudioFormat, 1)
	fmt.Printf("  Channels: %d\n", fmtChunk.NumChannels)
	fmt.Printf("  Sample rate: %d\n", fmtChunk.SampleRate)
	fmt.Printf("  Bits per sample: %d\n", fmtChunk.BitsPerSample)
	fmt.Printf("  Data offset: %d\n", dataOffset)
	fmt.Printf("  Data size: %d\n", dataSize)

	// Validate format
	if fmtChunk.AudioFormat != 1 {
		return nil, 0, fmt.Errorf("unsupported audio format: %d (only PCM is supported)", fmtChunk.AudioFormat)
	}

	if fmtChunk.BitsPerSample != 16 {
		return nil, 0, fmt.Errorf("unsupported bits per sample: %d (only 16-bit is supported)", fmtChunk.BitsPerSample)
	}

	// Extract audio data
	return extractAudioData(content, dataOffset, dataSize, fmtChunk)
}

// FmtChunk represents the format chunk
type FmtChunk struct {
	AudioFormat   uint16
	NumChannels   uint16
	SampleRate    uint32
	ByteRate      uint32
	BlockAlign    uint16
	BitsPerSample uint16
}

// parseFmtChunk parses the fmt chunk
func parseFmtChunk(data []byte) (*FmtChunk, error) {
	if len(data) < 16 {
		return nil, fmt.Errorf("fmt chunk too small: %d bytes", len(data))
	}

	reader := bytes.NewReader(data)
	var fmtData FmtChunk

	if err := binary.Read(reader, binary.LittleEndian, &fmtData); err != nil {
		return nil, fmt.Errorf("failed to read fmt chunk: %w", err)
	}

	return &fmtData, nil
}

// extractAudioData extracts audio samples from the data chunk
func extractAudioData(content []byte, dataOffset int64, dataSize uint32, fmtChunk *FmtChunk) ([]float32, int, error) {
	// Validate data bounds
	if dataOffset+int64(dataSize) > int64(len(content)) {
		actualDataSize := int64(len(content)) - dataOffset
		fmt.Printf("  Warning: Data size in header (%d) exceeds file bounds. Using actual size: %d\n",
			dataSize, actualDataSize)
		dataSize = uint32(actualDataSize)
	}

	// Calculate number of samples
	bytesPerSample := int(fmtChunk.BitsPerSample) / 8
	numSamples := int(dataSize) / bytesPerSample

	fmt.Printf("  Calculated samples: %d\n", numSamples)

	if numSamples <= 0 {
		return nil, 0, fmt.Errorf("no audio samples found")
	}

	// Extract samples
	audioData := make([]float32, numSamples)
	dataBytes := content[dataOffset : dataOffset+int64(dataSize)]
	reader := bytes.NewReader(dataBytes)

	for i := 0; i < numSamples; i++ {
		var sample int16
		if err := binary.Read(reader, binary.LittleEndian, &sample); err != nil {
			if err == io.EOF {
				fmt.Printf("  Warning: EOF at sample %d of %d. Truncating.\n", i, numSamples)
				audioData = audioData[:i]
				break
			}
			return nil, 0, fmt.Errorf("failed to read sample %d: %w", i, err)
		}

		// Convert to float32 in range [-1.0, 1.0]
		audioData[i] = float32(sample) / 32767.0
	}

	fmt.Printf("  Successfully loaded %d samples\n", len(audioData))
	return audioData, int(fmtChunk.SampleRate), nil
}
