package state

import (
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"os"
	"sync"
	"time"

	"audio-assistant/internal/audio"
)

type State int

const (
	StateIdle State = iota
	StateListening
	StateProcessing
	StateSpeaking
)

// 缓冲区配置
const (
	// 内存缓冲区大小（约1秒的音频数据）
	memBufferSize = 100
	// 单个音频块的最大大小
	maxChunkSize = 2048
	// 输出速率限制（每秒处理的音频块数）
	outputRateLimit = 100
	// 临时文件目录
	tempDir = "temp"
)

// 音频处理统计
type AudioStats struct {
	TotalInputChunks  int64
	TotalOutputChunks int64
	DroppedChunks     int64
	LastInputTime     time.Time
	LastOutputTime    time.Time
	TotalBytesWritten int64
	TotalBytesRead    int64
}

func (s State) String() string {
	switch s {
	case StateIdle:
		return "Idle"
	case StateListening:
		return "Listening"
	case StateProcessing:
		return "Processing"
	case StateSpeaking:
		return "Speaking"
	default:
		return "Unknown"
	}
}

type Manager struct {
	currentState State
	mu           sync.Mutex
	// 内存缓冲区
	memBuffer [][]float32
	// 临时文件
	tempFile *os.File
	// 音频处理统计
	stats AudioStats
	// 输出速率限制器
	outputTicker *time.Ticker
}

func NewManager() *Manager {
	// 创建临时目录
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		log.Fatalf("Failed to create temp directory: %v", err)
	}

	// 创建临时文件
	tempFile, err := os.CreateTemp(tempDir, "audio_*.raw")
	if err != nil {
		log.Fatalf("Failed to create temp file: %v", err)
	}

	return &Manager{
		currentState: StateIdle,
		memBuffer:    make([][]float32, 0, memBufferSize),
		tempFile:     tempFile,
		stats: AudioStats{
			LastInputTime:  time.Now(),
			LastOutputTime: time.Now(),
		},
		outputTicker: time.NewTicker(time.Second / outputRateLimit),
	}
}

func (m *Manager) Run(ctx context.Context, input *audio.Input, output *audio.AudioOutput) error {
	log.Println("Starting audio assistant...")

	if err := input.Start(); err != nil {
		return err
	}
	// 新的 AudioOutput 不需要显式启动

	// 启动音频处理循环
	go m.processAudio(ctx, input, output)

	// 等待上下文取消
	<-ctx.Done()

	// 清理资源
	m.cleanup()
	return nil
}

// 写入音频数据到临时文件
func (m *Manager) writeToTempFile(data []float32) error {
	// 将 float32 切片转换为字节切片
	bytes := make([]byte, len(data)*4)
	for i, v := range data {
		binary.LittleEndian.PutUint32(bytes[i*4:], uint32(v))
	}

	// 写入文件
	_, err := m.tempFile.Write(bytes)
	if err != nil {
		return fmt.Errorf("failed to write to temp file: %v", err)
	}

	m.stats.TotalBytesWritten += int64(len(bytes))
	return nil
}

// 从临时文件读取音频数据
func (m *Manager) readFromTempFile(size int) ([]float32, error) {
	// 读取字节数据
	bytes := make([]byte, size*4)
	n, err := m.tempFile.Read(bytes)
	if err != nil {
		if err == io.EOF {
			// 如果到达文件末尾，重置文件指针并返回空数据
			if _, err := m.tempFile.Seek(0, 0); err != nil {
				return nil, fmt.Errorf("failed to reset temp file: %v", err)
			}
			return nil, nil
		}
		return nil, fmt.Errorf("failed to read from temp file: %v", err)
	}

	// 如果读取的字节数不足，调整数据大小
	if n < size*4 {
		bytes = bytes[:n]
	}

	// 将字节切片转换为 float32 切片
	data := make([]float32, n/4)
	for i := 0; i < n/4; i++ {
		data[i] = float32(binary.LittleEndian.Uint32(bytes[i*4:]))
	}

	m.stats.TotalBytesRead += int64(n)
	return data, nil
}

// 重置临时文件
func (m *Manager) resetTempFile() error {
	// 截断文件
	if err := m.tempFile.Truncate(0); err != nil {
		return err
	}
	// 重置文件指针
	_, err := m.tempFile.Seek(0, 0)
	return err
}

// 清理资源
func (m *Manager) cleanup() {
	if m.tempFile != nil {
		m.tempFile.Close()
		os.Remove(m.tempFile.Name())
	}
}

// 添加音频数据
func (m *Manager) addAudioData(data []float32) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 检查单个音频块大小
	if len(data) > maxChunkSize {
		log.Printf("Warning: Input chunk size %d exceeds limit %d, truncating", len(data), maxChunkSize)
		data = data[:maxChunkSize]
	}

	// 更新输入统计
	m.stats.TotalInputChunks++
	m.stats.LastInputTime = time.Now()

	// 如果内存缓冲区已满，写入临时文件
	if len(m.memBuffer) >= memBufferSize {
		if err := m.writeToTempFile(m.memBuffer[0]); err != nil {
			return err
		}
		m.memBuffer = m.memBuffer[1:]
		m.stats.DroppedChunks++
	}

	// 添加到内存缓冲区
	m.memBuffer = append(m.memBuffer, data)
	return nil
}

// 获取音频数据
func (m *Manager) getAudioData() ([]float32, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 如果内存缓冲区为空，尝试从临时文件读取
	if len(m.memBuffer) == 0 {
		data, err := m.readFromTempFile(maxChunkSize)
		if err != nil {
			return nil, err
		}
		return data, nil
	}

	// 从内存缓冲区获取数据
	data := m.memBuffer[0]
	m.memBuffer = m.memBuffer[1:]

	// 更新输出统计
	m.stats.TotalOutputChunks++
	m.stats.LastOutputTime = time.Now()

	return data, nil
}

// 获取当前缓冲区大小
func (m *Manager) getBufferSize() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.memBuffer)
}

// 打印统计信息
func (m *Manager) printStats() {
	m.mu.Lock()
	defer m.mu.Unlock()

	inputRate := float64(m.stats.TotalInputChunks) / time.Since(m.stats.LastInputTime).Seconds()
	outputRate := float64(m.stats.TotalOutputChunks) / time.Since(m.stats.LastOutputTime).Seconds()

	log.Printf("Stats - Memory Buffer: %d, Input Rate: %.2f/s, Output Rate: %.2f/s, Dropped: %d, Written: %d bytes, Read: %d bytes",
		len(m.memBuffer), inputRate, outputRate, m.stats.DroppedChunks, m.stats.TotalBytesWritten, m.stats.TotalBytesRead)
}

func (m *Manager) processAudio(ctx context.Context, input *audio.Input, output *audio.AudioOutput) {
	// 使用更短的采样间隔
	ticker := time.NewTicker(time.Millisecond * 10) // 100Hz 的采样率
	defer ticker.Stop()

	// 统计信息打印定时器
	statsTicker := time.NewTicker(time.Second)
	defer statsTicker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-statsTicker.C:
			m.printStats()
		case <-ticker.C:
			// 读取音频数据
			data, err := input.Read()
			if err != nil {
				log.Printf("Error reading audio: %v", err)
				continue
			}

			// 根据当前状态处理音频数据
			switch m.getState() {
			case StateIdle:
				if len(data) > 0 {
					if err := m.addAudioData(data); err != nil {
						log.Printf("Error adding audio data: %v", err)
						continue
					}

					if m.getBufferSize() >= memBufferSize/2 {
						log.Printf("Switching to Listening state, buffer size: %d", m.getBufferSize())
						m.setState(StateListening)
					}
				}
			case StateListening:
				if err := m.addAudioData(data); err != nil {
					log.Printf("Error adding audio data: %v", err)
					continue
				}

				// TODO: 实现 VAD 检测
				log.Printf("Switching to Processing state, buffer size: %d", m.getBufferSize())
				m.setState(StateProcessing)
			case StateProcessing:
				// TODO: 实现语音识别和 LLM 处理
				log.Printf("Switching to Speaking state, buffer size: %d", m.getBufferSize())
				m.setState(StateSpeaking)
			case StateSpeaking:
				// 播放音频数据
				log.Printf("Playing audio data")

				// 持续播放直到没有更多数据
				for {
					// 等待输出速率限制
					<-m.outputTicker.C

					audioData, err := m.getAudioData()
					if err != nil {
						log.Printf("Error getting audio data: %v", err)
						continue
					}

					// 如果没有数据，退出循环
					if audioData == nil {
						break
					}

					// TODO: 更新为使用新的 AudioOutput API
					// if err := output.PlaySamples(audioData); err != nil {
					//	log.Printf("Error playing audio chunk: %v", err)
					// }
					log.Printf("Audio chunk ready for playback (length: %d)", len(audioData))
				}

				// 重置临时文件
				if err := m.resetTempFile(); err != nil {
					log.Printf("Error resetting temp file: %v", err)
				}

				log.Printf("Switching back to Idle state")
				m.setState(StateIdle)
			}
		}
	}
}

func (m *Manager) getState() State {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.currentState
}

func (m *Manager) setState(s State) {
	m.mu.Lock()
	defer m.mu.Unlock()
	oldState := m.currentState
	m.currentState = s
	if oldState != s {
		log.Printf("State changed: %s -> %s", oldState, s)
	}
}

// GetState 获取当前状态（公开方法）
func (m *Manager) GetState() State {
	return m.getState()
}

// SetState 设置状态（公开方法）
func (m *Manager) SetState(s State) {
	m.setState(s)
}
