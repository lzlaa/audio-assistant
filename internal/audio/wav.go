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

	// Read header
	var header WAVHeader
	if err := binary.Read(file, binary.LittleEndian, &header); err != nil {
		return nil, 0, fmt.Errorf("failed to read WAV header: %w", err)
	}

	// Validate header
	if string(header.ChunkID[:]) != "RIFF" || string(header.Format[:]) != "WAVE" {
		return nil, 0, fmt.Errorf("invalid WAV file format")
	}

	if header.AudioFormat != 1 {
		return nil, 0, fmt.Errorf("unsupported audio format: %d (only PCM is supported)", header.AudioFormat)
	}

	if header.BitsPerSample != 16 {
		return nil, 0, fmt.Errorf("unsupported bits per sample: %d (only 16-bit is supported)", header.BitsPerSample)
	}

	// Calculate number of samples
	numSamples := int(header.Subchunk2Size) / (int(header.BitsPerSample) / 8)
	audioData := make([]float32, numSamples)

	// Read audio data
	for i := 0; i < numSamples; i++ {
		var sample int16
		if err := binary.Read(file, binary.LittleEndian, &sample); err != nil {
			return nil, 0, fmt.Errorf("failed to read audio sample %d: %w", i, err)
		}

		// Convert to float32 in range [-1.0, 1.0]
		audioData[i] = float32(sample) / 32767.0
	}

	return audioData, int(header.SampleRate), nil
}
