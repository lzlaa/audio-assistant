package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"audio-assistant/internal/audio"
	"audio-assistant/internal/state"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 创建状态管理器
	stateManager := state.NewManager()

	// 创建音频采集器
	audioInput, err := audio.NewInput()
	if err != nil {
		log.Fatalf("Failed to create audio input: %v", err)
	}
	defer audioInput.Close()

	// 创建音频播放器
	audioOutput, err := audio.NewOutput()
	if err != nil {
		log.Fatalf("Failed to create audio output: %v", err)
	}
	defer audioOutput.Close()

	// 启动主循环
	go func() {
		if err := stateManager.Run(ctx, audioInput, audioOutput); err != nil {
			log.Printf("Error in main loop: %v", err)
		}
	}()

	// 等待中断信号
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	log.Println("Shutting down...")
}
