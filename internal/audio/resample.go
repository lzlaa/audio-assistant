package audio

import (
	"fmt"
)

// Resample performs simple linear interpolation resampling
func Resample(inputSamples []float32, inputSampleRate, outputSampleRate int) ([]float32, error) {
	if inputSampleRate <= 0 || outputSampleRate <= 0 {
		return nil, fmt.Errorf("invalid sample rates: input=%d, output=%d", inputSampleRate, outputSampleRate)
	}

	if len(inputSamples) == 0 {
		return []float32{}, nil
	}

	// 如果采样率相同，直接返回
	if inputSampleRate == outputSampleRate {
		result := make([]float32, len(inputSamples))
		copy(result, inputSamples)
		return result, nil
	}

	// 计算重采样比率
	ratio := float64(inputSampleRate) / float64(outputSampleRate)
	outputLength := int(float64(len(inputSamples)) / ratio)

	if outputLength <= 0 {
		return []float32{}, nil
	}

	outputSamples := make([]float32, outputLength)

	// 线性插值重采样
	for i := 0; i < outputLength; i++ {
		// 计算在输入数组中的位置
		sourcePos := float64(i) * ratio
		sourceIndex := int(sourcePos)
		fraction := sourcePos - float64(sourceIndex)

		// 边界检查
		if sourceIndex >= len(inputSamples)-1 {
			outputSamples[i] = inputSamples[len(inputSamples)-1]
			continue
		}

		// 线性插值
		sample1 := inputSamples[sourceIndex]
		sample2 := inputSamples[sourceIndex+1]
		outputSamples[i] = sample1 + float32(fraction)*(sample2-sample1)
	}

	fmt.Printf("重采样: %d Hz (%d 样本) -> %d Hz (%d 样本)\n",
		inputSampleRate, len(inputSamples), outputSampleRate, len(outputSamples))

	return outputSamples, nil
}

// GetTargetSampleRate returns the target sample rate for the audio system
func GetTargetSampleRate() int {
	return sampleRate // 16000 Hz
}
