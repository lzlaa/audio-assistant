package audio

import (
	"fmt"
	"sync"

	"github.com/gordonklaus/portaudio"
)

const (
	sampleRate      = 16000
	channels        = 1
	framesPerBuffer = 10240
)

type Input struct {
	stream *portaudio.Stream
	buffer []float32
	mu     sync.Mutex
	queue  [][]float32
}

func NewInput() (*Input, error) {
	// 使用统一的音频管理器
	manager := GetManager()
	if err := manager.Initialize(); err != nil {
		return nil, fmt.Errorf("failed to initialize audio system: %w", err)
	}

	input := &Input{
		buffer: make([]float32, framesPerBuffer),
		queue:  make([][]float32, 0),
	}

	stream, err := portaudio.OpenDefaultStream(channels, 0, float64(sampleRate), framesPerBuffer, input.buffer)
	if err != nil {
		manager.Terminate() // 清理
		return nil, fmt.Errorf("failed to open input stream: %w", err)
	}

	input.stream = stream
	return input, nil
}

func (i *Input) Start() error {
	return i.stream.Start()
}

func (i *Input) Read() ([]float32, error) {
	i.mu.Lock()
	defer i.mu.Unlock()

	// 如果队列中有数据，直接返回
	if len(i.queue) > 0 {
		data := i.queue[0]
		i.queue = i.queue[1:]
		return data, nil
	}

	// 否则读取新的数据
	err := i.stream.Read()
	if err != nil {
		return nil, err
	}

	// 复制缓冲区数据
	data := make([]float32, len(i.buffer))
	copy(data, i.buffer)
	return data, nil
}

func (i *Input) Close() error {
	var err error
	if i.stream != nil {
		err = i.stream.Close()
	}

	// 使用统一的音频管理器终止
	manager := GetManager()
	if termErr := manager.Terminate(); termErr != nil && err == nil {
		err = termErr
	}

	return err
}
