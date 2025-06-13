package main

import (
	"audio-assistant/internal/audio"
	"audio-assistant/internal/vad"
	"fmt"
)

func main() {
	fmt.Println("采集 5 秒音频...")
	audioData, err := audio.RecordAudio()
	if err != nil {
		fmt.Println("音频采集失败:", err)
		return
	}
	fmt.Println("音频采集完成，调用 VAD 服务...")

	vadURL := "http://localhost:5001/vad"
	ts, err := vad.CallVadService(audioData, vadURL)
	if err != nil {
		fmt.Println("VAD 检测失败:", err)
		return
	}
	fmt.Println("VAD 检测结果：")
	for i, t := range ts {
		fmt.Printf("片段%d: start=%d, end=%d\n", i+1, t.Start, t.End)
	}
}
