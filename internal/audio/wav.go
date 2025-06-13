package audio

import (
	"encoding/binary"
	"fmt"
	"os"
)

// WAV file header structure
type WAVHeader struct {
	ChunkID       [4]byte // "RIFF"
	ChunkSize     uint32  // File size - 8
	Format        [4]byte // "WAVE"
	Subchunk1ID   [4]byte // "fmt "
	Subchunk1Size uint32  // 16 for PCM
	AudioFormat   uint16  // 1 for PCM
	NumChannels   uint16  // Number of channels
	SampleRate    uint32  // Sample rate
	ByteRate      uint32  // SampleRate * NumChannels * BitsPerSample/8
	BlockAlign    uint16  // NumChannels * BitsPerSample/8
	BitsPerSample uint16  // Bits per sample
	Subchunk2ID   [4]byte // "data"
	Subchunk2Size uint32  // Data size
}

// SaveToWAV saves float32 audio data to a WAV file
func SaveToWAV(filename string, audioData []float32, sampleRate int) error {
	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create WAV file: %w", err)
	}
	defer file.Close()

	numChannels := uint16(1)
	bitsPerSample := uint16(16)
	byteRate := uint32(sampleRate) * uint32(numChannels) * uint32(bitsPerSample) / 8
	blockAlign := numChannels * bitsPerSample / 8
	dataSize := uint32(len(audioData) * 2) // 16-bit samples = 2 bytes each

	// Create WAV header
	header := WAVHeader{
		ChunkID:       [4]byte{'R', 'I', 'F', 'F'},
		ChunkSize:     36 + dataSize,
		Format:        [4]byte{'W', 'A', 'V', 'E'},
		Subchunk1ID:   [4]byte{'f', 'm', 't', ' '},
		Subchunk1Size: 16,
		AudioFormat:   1, // PCM
		NumChannels:   numChannels,
		SampleRate:    uint32(sampleRate),
		ByteRate:      byteRate,
		BlockAlign:    blockAlign,
		BitsPerSample: bitsPerSample,
		Subchunk2ID:   [4]byte{'d', 'a', 't', 'a'},
		Subchunk2Size: dataSize,
	}

	// Write header
	if err := binary.Write(file, binary.LittleEndian, header); err != nil {
		return fmt.Errorf("failed to write WAV header: %w", err)
	}

	// Convert float32 to int16 and write data
	for _, sample := range audioData {
		// Clamp sample to [-1.0, 1.0] range
		if sample > 1.0 {
			sample = 1.0
		} else if sample < -1.0 {
			sample = -1.0
		}

		// Convert to 16-bit signed integer
		int16Sample := int16(sample * 32767)
		if err := binary.Write(file, binary.LittleEndian, int16Sample); err != nil {
			return fmt.Errorf("failed to write audio data: %w", err)
		}
	}

	return nil
}

// LoadFromWAV loads audio data from a WAV file
func LoadFromWAV(filename string) ([]float32, int, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to open WAV file: %w", err)
	}
	defer file.Close()

	// Get file size for validation
	fileInfo, err := file.Stat()
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get file info: %w", err)
	}
	fileSize := fileInfo.Size()

	// Read header
	var header WAVHeader
	if err := binary.Read(file, binary.LittleEndian, &header); err != nil {
		return nil, 0, fmt.Errorf("failed to read WAV header: %w", err)
	}

	// Validate header
	if string(header.ChunkID[:]) != "RIFF" || string(header.Format[:]) != "WAVE" {
		return nil, 0, fmt.Errorf("invalid WAV file format: ChunkID=%s, Format=%s",
			string(header.ChunkID[:]), string(header.Format[:]))
	}

	if header.AudioFormat != 1 {
		return nil, 0, fmt.Errorf("unsupported audio format: %d (only PCM is supported)", header.AudioFormat)
	}

	if header.BitsPerSample != 16 {
		return nil, 0, fmt.Errorf("unsupported bits per sample: %d (only 16-bit is supported)", header.BitsPerSample)
	}

	// Validate data chunk size
	expectedFileSize := int64(44 + header.Subchunk2Size) // Header + data
	if fileSize < expectedFileSize {
		// Adjust data size based on actual file size
		actualDataSize := fileSize - 44
		header.Subchunk2Size = uint32(actualDataSize)
		fmt.Printf("Warning: WAV file smaller than expected. Expected: %d, Actual: %d, Adjusting data size to: %d\n",
			expectedFileSize, fileSize, actualDataSize)
	}

	// Calculate number of samples based on actual data size
	bytesPerSample := int(header.BitsPerSample) / 8
	numSamples := int(header.Subchunk2Size) / bytesPerSample

	// Validate that we have the expected amount of data
	maxPossibleSamples := int(fileSize-44) / bytesPerSample
	if numSamples > maxPossibleSamples {
		numSamples = maxPossibleSamples
		fmt.Printf("Warning: Adjusting sample count from %d to %d based on file size\n",
			int(header.Subchunk2Size)/bytesPerSample, numSamples)
	}

	fmt.Printf("WAV file info: FileSize=%d, DataSize=%d, Channels=%d, SampleRate=%d, BitsPerSample=%d, NumSamples=%d\n",
		fileSize, header.Subchunk2Size, header.NumChannels, header.SampleRate, header.BitsPerSample, numSamples)

	audioData := make([]float32, numSamples)

	// Read audio data with better error handling
	for i := 0; i < numSamples; i++ {
		var sample int16
		if err := binary.Read(file, binary.LittleEndian, &sample); err != nil {
			if err.Error() == "EOF" {
				// Handle EOF gracefully - truncate to actual samples read
				fmt.Printf("Warning: EOF encountered at sample %d of %d. Truncating audio data.\n", i, numSamples)
				audioData = audioData[:i]
				break
			}
			return nil, 0, fmt.Errorf("failed to read audio sample %d: %w", i, err)
		}

		// Convert to float32 in range [-1.0, 1.0]
		audioData[i] = float32(sample) / 32767.0
	}

	return audioData, int(header.SampleRate), nil
}
