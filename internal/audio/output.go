package audio

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/gordonklaus/portaudio"
)

// AudioOutput 音频输出结构
type AudioOutput struct {
	stream      *portaudio.Stream
	samples     []float32
	position    int
	finished    bool
	interrupted bool
	mu          sync.Mutex
	sampleRate  int
}

// NewAudioOutput 创建音频输出
func NewAudioOutput(sampleRate int) (*AudioOutput, error) {
	if err := GetManager().Initialize(); err != nil {
		return nil, fmt.Errorf("failed to initialize audio manager: %w", err)
	}

	output := &AudioOutput{
		samples:     make([]float32, 0),
		position:    0,
		finished:    false,
		interrupted: false,
		sampleRate:  sampleRate,
	}

	// 使用回调创建流
	stream, err := portaudio.OpenDefaultStream(0, 1, float64(sampleRate), 1024, output.audioCallback)
	if err != nil {
		return nil, fmt.Errorf("failed to open output stream: %w", err)
	}

	output.stream = stream
	return output, nil
}

// audioCallback 音频回调函数
func (ao *AudioOutput) audioCallback(out []float32) {
	ao.mu.Lock()
	defer ao.mu.Unlock()

	// 如果被打断，立即填充静音
	if ao.interrupted {
		for i := range out {
			out[i] = 0.0
		}
		ao.finished = true
		return
	}

	for i := range out {
		if ao.position < len(ao.samples) {
			out[i] = ao.samples[ao.position]
			ao.position++
		} else {
			out[i] = 0.0
			ao.finished = true
		}
	}
}

// PlayAudioData 播放音频数据，支持多种格式和自动重采样
func (ao *AudioOutput) PlayAudioData(ctx context.Context, audioData []byte, targetSampleRate int) error {
	// 使用解码器解码音频数据
	decoder := NewAudioDecoder()
	samples, sourceSampleRate, err := decoder.DecodeAudioData(audioData)
	if err != nil {
		return fmt.Errorf("failed to decode audio: %w", err)
	}

	fmt.Printf("解码成功: %d 样本, 源采样率: %d Hz\n", len(samples), sourceSampleRate)

	// 重采样到目标采样率
	if sourceSampleRate != targetSampleRate {
		samples, err = decoder.ResampleAudio(samples, sourceSampleRate, targetSampleRate)
		if err != nil {
			return fmt.Errorf("failed to resample audio: %w", err)
		}
	}

	return ao.PlaySamples(ctx, samples)
}

// PlaySamples 播放已解码的音频样本（支持上下文取消）
func (ao *AudioOutput) PlaySamples(ctx context.Context, samples []float32) error {
	if len(samples) == 0 {
		return fmt.Errorf("no audio samples to play")
	}

	ao.mu.Lock()
	ao.samples = make([]float32, len(samples))
	copy(ao.samples, samples)
	ao.position = 0
	ao.finished = false
	ao.interrupted = false
	ao.mu.Unlock()

	// 开始播放
	if err := ao.stream.Start(); err != nil {
		return fmt.Errorf("failed to start audio stream: %w", err)
	}

	// 等待播放完成或被取消
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			// 上下文被取消，停止播放
			ao.Stop()
			return ctx.Err()
		case <-ticker.C:
			ao.mu.Lock()
			finished := ao.finished || ao.interrupted
			ao.mu.Unlock()

			if finished {
				break
			}
		}
	}

	// 停止播放
	if err := ao.stream.Stop(); err != nil {
		return fmt.Errorf("failed to stop audio stream: %w", err)
	}

	return nil
}

// Stop 停止当前播放
func (ao *AudioOutput) Stop() {
	ao.mu.Lock()
	defer ao.mu.Unlock()
	ao.interrupted = true
	ao.finished = true
}

// IsPlaying 检查是否正在播放
func (ao *AudioOutput) IsPlaying() bool {
	ao.mu.Lock()
	defer ao.mu.Unlock()
	return !ao.finished && !ao.interrupted && ao.position < len(ao.samples)
}

// Close 关闭音频输出
func (ao *AudioOutput) Close() error {
	if ao.stream != nil {
		if err := ao.stream.Close(); err != nil {
			return fmt.Errorf("failed to close audio stream: %w", err)
		}
	}

	return GetManager().Terminate()
}
