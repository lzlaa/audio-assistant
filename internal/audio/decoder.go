package audio

import (
	"bytes"
	"fmt"
	"io"
	"os"

	"github.com/tosone/minimp3"
	"github.com/youpy/go-wav"
)

// AudioDecoder 音频解码器
type AudioDecoder struct{}

// NewAudioDecoder 创建新的音频解码器
func NewAudioDecoder() *AudioDecoder {
	return &AudioDecoder{}
}

// DecodeAudioData 解码音频数据，自动检测格式
func (d *AudioDecoder) DecodeAudioData(audioData []byte) ([]float32, int, error) {
	// 检测音频格式
	format := d.detectFormat(audioData)

	switch format {
	case "wav":
		// 先尝试健壮的 WAV 解析器
		samples, rate, err := d.decodeWAVRobust(audioData)
		if err != nil {
			// 如果健壮解析器失败，尝试 go-wav 库
			fmt.Printf("健壮解析器失败，尝试 go-wav 库: %v\n", err)
			return d.decodeWAV(audioData)
		}
		return samples, rate, err
	case "mp3":
		return d.decodeMP3(audioData)
	default:
		// 默认尝试健壮的 WAV 解析器，然后尝试 go-wav，最后尝试 MP3
		samples, rate, err := d.decodeWAVRobust(audioData)
		if err != nil {
			samples, rate, err = d.decodeWAV(audioData)
			if err != nil {
				return d.decodeMP3(audioData)
			}
		}
		return samples, rate, err
	}
}

// DecodeAudioFile 解码音频文件
func (d *AudioDecoder) DecodeAudioFile(filename string) ([]float32, int, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to read file: %w", err)
	}

	return d.DecodeAudioData(data)
}

// detectFormat 检测音频格式
func (d *AudioDecoder) detectFormat(data []byte) string {
	if len(data) >= 4 {
		// 检查 WAV 文件头
		if bytes.Equal(data[:4], []byte("RIFF")) {
			return "wav"
		}

		// 检查 MP3 文件头
		if len(data) >= 3 && (data[0] == 0xFF && (data[1]&0xE0) == 0xE0) {
			return "mp3"
		}

		// 检查 ID3 标签（MP3）
		if bytes.Equal(data[:3], []byte("ID3")) {
			return "mp3"
		}
	}

	return "unknown"
}

// decodeWAV 使用开源库解码 WAV
func (d *AudioDecoder) decodeWAV(audioData []byte) ([]float32, int, error) {
	reader := bytes.NewReader(audioData)
	wavReader := wav.NewReader(reader)

	format, err := wavReader.Format()
	if err != nil {
		return nil, 0, fmt.Errorf("failed to read WAV format: %w", err)
	}

	fmt.Printf("WAV 格式: 声道=%d, 采样率=%d, 位深=%d\n",
		format.NumChannels, format.SampleRate, format.BitsPerSample)

	// 读取所有样本
	var samples []float32

	for {
		sampleData, err := wavReader.ReadSamples()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, 0, fmt.Errorf("failed to read WAV samples: %w", err)
		}

		// 转换样本到 float32
		for _, sample := range sampleData {
			// 使用 go-wav 库的正确方法获取样本值
			var floatSample float32

			// 获取第一个声道的值
			channel0Value := wavReader.IntValue(sample, 0)

			switch format.BitsPerSample {
			case 16:
				floatSample = float32(channel0Value) / 32768.0
			case 32:
				floatSample = float32(channel0Value) / 2147483648.0
			case 8:
				floatSample = float32(channel0Value) / 128.0
			case 24:
				floatSample = float32(channel0Value) / 8388608.0
			default:
				floatSample = float32(channel0Value) / 32768.0
			}

			// 限制范围到 [-1.0, 1.0]
			if floatSample > 1.0 {
				floatSample = 1.0
			} else if floatSample < -1.0 {
				floatSample = -1.0
			}

			samples = append(samples, floatSample)

			// 如果是立体声，混合第二个声道
			if format.NumChannels == 2 {
				channel1Value := wavReader.IntValue(sample, 1)
				var floatSample2 float32

				switch format.BitsPerSample {
				case 16:
					floatSample2 = float32(channel1Value) / 32768.0
				case 32:
					floatSample2 = float32(channel1Value) / 2147483648.0
				case 8:
					floatSample2 = float32(channel1Value) / 128.0
				case 24:
					floatSample2 = float32(channel1Value) / 8388608.0
				default:
					floatSample2 = float32(channel1Value) / 32768.0
				}

				// 限制范围
				if floatSample2 > 1.0 {
					floatSample2 = 1.0
				} else if floatSample2 < -1.0 {
					floatSample2 = -1.0
				}

				// 替换为立体声混合
				samples[len(samples)-1] = (floatSample + floatSample2) / 2.0
			}
		}
	}

	return samples, int(format.SampleRate), nil
}

// decodeMP3 使用开源库解码 MP3
func (d *AudioDecoder) decodeMP3(audioData []byte) ([]float32, int, error) {
	// 使用 DecodeFull 一次性解码整个 MP3 文件
	decoder, pcmData, err := minimp3.DecodeFull(audioData)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to decode MP3: %w", err)
	}
	defer decoder.Close()

	fmt.Printf("MP3 格式: 声道=%d, 采样率=%d, 位深=%d\n",
		decoder.Channels, decoder.SampleRate, 16)

	// 转换 int16 PCM 数据到 float32
	var samples []float32
	pcmSamples := len(pcmData) / 2 // int16 = 2 bytes

	for i := 0; i < pcmSamples; i += decoder.Channels {
		// 从 []byte 转换为 int16
		var sample float32

		if decoder.Channels == 1 {
			// 单声道
			rawSample := int16(pcmData[i*2]) | int16(pcmData[i*2+1])<<8
			sample = float32(rawSample) / 32768.0
		} else {
			// 立体声混合到单声道
			leftRaw := int16(pcmData[i*2]) | int16(pcmData[i*2+1])<<8
			rightRaw := int16(pcmData[(i+1)*2]) | int16(pcmData[(i+1)*2+1])<<8

			left := float32(leftRaw) / 32768.0
			right := float32(rightRaw) / 32768.0
			sample = (left + right) / 2.0
		}

		// 限制范围
		if sample > 1.0 {
			sample = 1.0
		} else if sample < -1.0 {
			sample = -1.0
		}

		samples = append(samples, sample)
	}

	return samples, decoder.SampleRate, nil
}

// decodeWAVRobust 使用健壮的 WAV 解析器（处理 OpenAI TTS 的损坏头）
func (d *AudioDecoder) decodeWAVRobust(audioData []byte) ([]float32, int, error) {
	// 创建临时文件
	tempFile, err := os.CreateTemp("", "audio_decode_*.wav")
	if err != nil {
		return nil, 0, fmt.Errorf("failed to create temp file: %w", err)
	}
	defer func() {
		tempFile.Close()
		os.Remove(tempFile.Name())
	}()

	// 写入数据
	if err := os.WriteFile(tempFile.Name(), audioData, 0644); err != nil {
		return nil, 0, fmt.Errorf("failed to write temp file: %w", err)
	}

	// 使用我们之前的健壮解析器
	return RobustLoadFromWAV(tempFile.Name())
}

// ResampleAudio 使用简单线性插值重采样
func (d *AudioDecoder) ResampleAudio(inputSamples []float32, inputRate, outputRate int) ([]float32, error) {
	if inputRate == outputRate {
		// 不需要重采样
		result := make([]float32, len(inputSamples))
		copy(result, inputSamples)
		return result, nil
	}

	if len(inputSamples) == 0 {
		return []float32{}, nil
	}

	ratio := float64(inputRate) / float64(outputRate)
	outputLength := int(float64(len(inputSamples)) / ratio)

	if outputLength <= 0 {
		return []float32{}, nil
	}

	outputSamples := make([]float32, outputLength)

	for i := 0; i < outputLength; i++ {
		srcIndex := float64(i) * ratio
		srcIndexInt := int(srcIndex)
		fraction := srcIndex - float64(srcIndexInt)

		if srcIndexInt >= len(inputSamples)-1 {
			outputSamples[i] = inputSamples[len(inputSamples)-1]
		} else {
			// 线性插值
			sample1 := inputSamples[srcIndexInt]
			sample2 := inputSamples[srcIndexInt+1]
			outputSamples[i] = sample1 + float32(fraction)*(sample2-sample1)
		}
	}

	fmt.Printf("重采样: %d Hz (%d 样本) -> %d Hz (%d 样本)\n",
		inputRate, len(inputSamples), outputRate, len(outputSamples))

	return outputSamples, nil
}
