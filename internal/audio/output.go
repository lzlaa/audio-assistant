package audio

import (
	"sync"

	"github.com/gordonklaus/portaudio"
)

type Output struct {
	stream *portaudio.Stream
	buffer []float32
	mu     sync.Mutex
	queue  [][]float32
}

func NewOutput() (*Output, error) {
	output := &Output{
		buffer: make([]float32, framesPerBuffer),
		queue:  make([][]float32, 0),
	}

	stream, err := portaudio.OpenDefaultStream(0, channels, float64(sampleRate), framesPerBuffer, &output.buffer)
	if err != nil {
		return nil, err
	}

	output.stream = stream
	return output, nil
}

func (o *Output) Start() error {
	return o.stream.Start()
}

func (o *Output) Write(data []float32) error {
	o.mu.Lock()
	defer o.mu.Unlock()

	// 将数据添加到队列
	o.queue = append(o.queue, data)

	// 如果队列中有数据，尝试写入
	if len(o.queue) > 0 {
		// 复制数据到缓冲区
		copy(o.buffer, o.queue[0])
		// 移除已处理的数据
		o.queue = o.queue[1:]
		return o.stream.Write()
	}

	return nil
}

func (o *Output) Close() error {
	if o.stream != nil {
		return o.stream.Close()
	}
	return nil
}
