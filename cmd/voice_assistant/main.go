package main

import (
	"context"
	"encoding/binary"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"sync"
	"syscall"
	"time"

	"audio-assistant/internal/asr"
	"audio-assistant/internal/audio"
	"audio-assistant/internal/llm"
	"audio-assistant/internal/state"
	"audio-assistant/internal/tts"
	"audio-assistant/internal/vad"
)

// VoiceAssistant 语音助手结构体
type VoiceAssistant struct {
	// 音频模块
	audioInput   *audio.Input
	audioOutput  *audio.AudioOutput
	stateManager *state.Manager

	// API 客户端
	vadClient *vad.Client
	asrClient *asr.Client
	llmClient *llm.OpenAISDKClient
	ttsClient *tts.TTSClient

	// 控制
	ctx          context.Context
	cancel       context.CancelFunc
	shutdownChan chan bool

	// 状态变量
	isListening         bool
	conversationHistory []llm.Message

	// 打断检测状态
	interruptDetectionStart time.Time
	isDetectingInterrupt    bool

	// 播放控制
	playbackCtx    context.Context
	playbackCancel context.CancelFunc

	// 同步控制
	mu sync.RWMutex

	// 配置参数
	config *Config
}

// Config 配置结构体
type Config struct {
	// API 配置
	OpenAIAPIKey string
	VADServerURL string

	// 音频配置
	VADThreshold            float64
	MinSpeechDurationMs     int
	MinSilenceDurationMs    int
	MaxRecordingDurationSec int

	// 打断控制配置
	AllowInterrupt         bool    // 是否允许打断播放
	InterruptThreshold     float64 // 打断检测阈值（更高=更难打断）
	InterruptMinDurationMs int     // 打断最小持续时间

	// LLM 配置
	LLMModel       string
	LLMTemperature float32
	SystemPrompt   string

	// TTS 配置
	TTSModel string
	TTSVoice string
	TTSSpeed float64

	// 调试配置
	SaveAudioFiles bool
	AudioOutputDir string
}

// getDefaultConfig 获取默认配置
func getDefaultConfig() *Config {
	return &Config{
		VADServerURL:            "http://localhost:8080",
		VADThreshold:            0.5,
		MinSpeechDurationMs:     500,
		MinSilenceDurationMs:    1000,
		MaxRecordingDurationSec: 30,
		// 打断控制配置
		AllowInterrupt:         true, // 默认允许打断
		InterruptThreshold:     0.7,  // 较高的阈值，避免误触发
		InterruptMinDurationMs: 200,  // 需要持续200ms的语音才能打断
		LLMModel:               "gpt-4o-mini",
		LLMTemperature:         0.7,
		SystemPrompt:           "你是一个有帮助的AI助手。请用简洁、友好的方式回答问题。",
		TTSModel:               "tts-1",
		TTSVoice:               "alloy",
		TTSSpeed:               1.0,
		SaveAudioFiles:         false,
		AudioOutputDir:         "temp",
	}
}

// NewVoiceAssistant 创建新的语音助手
func NewVoiceAssistant(config *Config) (*VoiceAssistant, error) {
	// 创建输出目录
	if config.SaveAudioFiles {
		if err := os.MkdirAll(config.AudioOutputDir, 0755); err != nil {
			return nil, fmt.Errorf("创建输出目录失败: %w", err)
		}
	}

	// 创建状态管理器
	stateManager := state.NewManager()

	// 创建音频模块
	audioInput, err := audio.NewInput()
	if err != nil {
		return nil, fmt.Errorf("创建音频输入失败: %w", err)
	}

	audioOutput, err := audio.NewAudioOutput(16000)
	if err != nil {
		audioInput.Close()
		return nil, fmt.Errorf("创建音频输出失败: %w", err)
	}

	// 创建客户端
	vadClient := vad.NewClient(config.VADServerURL)

	asrClient := asr.NewClient(config.OpenAIAPIKey)

	llmConfig := &llm.Config{
		APIKey: config.OpenAIAPIKey,
	}
	llmClient := llm.NewClient(llmConfig)

	ttsClient := tts.NewTTSClient(config.OpenAIAPIKey)
	ttsClient.SetModel(config.TTSModel)
	ttsClient.SetVoice(config.TTSVoice)
	ttsClient.SetSpeed(config.TTSSpeed)

	ctx, cancel := context.WithCancel(context.Background())

	return &VoiceAssistant{
		audioInput:          audioInput,
		audioOutput:         audioOutput,
		stateManager:        stateManager,
		vadClient:           vadClient,
		asrClient:           asrClient,
		llmClient:           llmClient,
		ttsClient:           ttsClient,
		ctx:                 ctx,
		cancel:              cancel,
		shutdownChan:        make(chan bool, 1),
		isListening:         false,
		conversationHistory: make([]llm.Message, 0),
		config:              config,
	}, nil
}

// Start 启动语音助手
func (va *VoiceAssistant) Start(ctx context.Context) error {
	// 检查 VAD 服务是否可用
	if err := va.checkVADService(); err != nil {
		return fmt.Errorf("VAD服务检查失败: %w", err)
	}

	fmt.Println("=== 语音助手已就绪，您可以开始对话 ===")

	// 启动主处理循环
	go va.processingLoop()

	// 等待关闭信号
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-va.shutdownChan:
		return nil
	}
}

// checkVADService 检查VAD服务是否可用
func (va *VoiceAssistant) checkVADService() error {
	tempAudio := make([]float32, 8000) // 0.5秒的静音
	tempFile, err := va.saveAudioToTempFile(tempAudio)
	if err != nil {
		return err
	}
	defer os.Remove(tempFile)

	vadReq := &vad.DetectRequest{
		Threshold:            va.config.VADThreshold,
		MinSpeechDurationMs:  va.config.MinSpeechDurationMs,
		MinSilenceDurationMs: va.config.MinSilenceDurationMs,
	}

	_, err = va.vadClient.HasSpeech(tempFile, vadReq)
	return err
}

// processingLoop 主处理循环
func (va *VoiceAssistant) processingLoop() {
	audioBuffer := make([][]float32, 0)
	recordingStart := time.Time{}
	silenceStart := time.Time{}

	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-va.ctx.Done():
			va.shutdownChan <- true
			return

		case <-ticker.C:
			// 读取音频数据
			audioData, err := va.audioInput.Read()
			if err != nil {
				log.Printf("读取音频数据失败: %v", err)
				continue
			}

			currentState := va.stateManager.GetState()

			switch currentState {
			case state.StateIdle, state.StateListening:
				// 检测语音活动
				hasSpeech, err := va.detectSpeechActivity(audioData)
				if err != nil {
					log.Printf("语音活动检测失败: %v", err)
					continue
				}

				if hasSpeech {
					// 检测到语音，开始或继续录音
					if !va.isListening {
						va.isListening = true
						recordingStart = time.Now()
						audioBuffer = audioBuffer[:0]
						va.stateManager.SetState(state.StateListening)
						fmt.Println("🎤 开始录音...")
					}

					audioBuffer = append(audioBuffer, audioData)
					silenceStart = time.Time{} // 重置静音开始时间

					// 检查录音时长限制
					if time.Since(recordingStart) > time.Duration(va.config.MaxRecordingDurationSec)*time.Second {
						fmt.Println("⏰ 录音时间超过限制，自动结束录音")
						va.processRecording(audioBuffer)
						va.resetRecording(&audioBuffer, &recordingStart, &silenceStart)
					}
				} else if va.isListening {
					// 在录音中检测到静音
					if silenceStart.IsZero() {
						silenceStart = time.Now()
					}

					audioBuffer = append(audioBuffer, audioData)

					// 检查静音时长
					if time.Since(silenceStart) > time.Duration(va.config.MinSilenceDurationMs)*time.Millisecond {
						fmt.Println("🔇 检测到静音，结束录音")
						va.processRecording(audioBuffer)
						va.resetRecording(&audioBuffer, &recordingStart, &silenceStart)
					}
				}

			case state.StateSpeaking:
				// 播放中，检测打断（使用更严格的条件）
				if va.config.AllowInterrupt {
					hasInterrupt, err := va.detectInterrupt(audioData)
					if err == nil && hasInterrupt {
						// 检测到潜在打断，开始计时验证
						if !va.isDetectingInterrupt {
							va.isDetectingInterrupt = true
							va.interruptDetectionStart = time.Now()
							fmt.Println("🎯 检测到可能的打断...")
						}

						// 检查打断持续时间
						if time.Since(va.interruptDetectionStart) > time.Duration(va.config.InterruptMinDurationMs)*time.Millisecond {
							fmt.Println("🚫 确认用户打断")
							va.handleInterrupt()
							va.isDetectingInterrupt = false
						}
					} else {
						// 没有检测到打断，重置状态
						if va.isDetectingInterrupt {
							va.isDetectingInterrupt = false
							fmt.Println("📢 继续播放...")
						}
					}
				}
			}
		}
	}
}

// detectSpeechActivity 检测语音活动
func (va *VoiceAssistant) detectSpeechActivity(audioData []float32) (bool, error) {
	// 将 float32 音频数据保存为临时 WAV 文件
	tempFile, err := va.saveAudioToTempFile(audioData)
	if err != nil {
		return false, err
	}
	defer os.Remove(tempFile)

	// 调用 VAD 服务
	vadReq := &vad.DetectRequest{
		Threshold:            va.config.VADThreshold,
		MinSpeechDurationMs:  va.config.MinSpeechDurationMs,
		MinSilenceDurationMs: va.config.MinSilenceDurationMs,
	}

	hasSpeech, err := va.vadClient.HasSpeech(tempFile, vadReq)
	if err != nil {
		return false, err
	}

	return hasSpeech, nil
}

// detectInterrupt 检测打断（使用更严格的阈值）
func (va *VoiceAssistant) detectInterrupt(audioData []float32) (bool, error) {
	if !va.config.AllowInterrupt {
		return false, nil
	}

	// 将 float32 音频数据保存为临时 WAV 文件
	tempFile, err := va.saveAudioToTempFile(audioData)
	if err != nil {
		return false, err
	}
	defer os.Remove(tempFile)

	// 使用更严格的打断检测参数
	vadReq := &vad.DetectRequest{
		Threshold:            va.config.InterruptThreshold,     // 更高的阈值
		MinSpeechDurationMs:  va.config.InterruptMinDurationMs, // 更长的最小持续时间
		MinSilenceDurationMs: va.config.MinSilenceDurationMs,
	}

	hasSpeech, err := va.vadClient.HasSpeech(tempFile, vadReq)
	if err != nil {
		return false, err
	}

	return hasSpeech, nil
}

// saveAudioToTempFile 将音频数据保存为临时文件
func (va *VoiceAssistant) saveAudioToTempFile(audioData []float32) (string, error) {
	// 创建临时文件
	tempFile, err := os.CreateTemp("temp", "audio_*.wav")
	if err != nil {
		return "", err
	}
	defer tempFile.Close()

	// 写入简单的 WAV 头
	sampleRate := 16000
	numChannels := 1
	bitsPerSample := 32

	dataSize := len(audioData) * 4
	fileSize := 36 + dataSize

	// WAV 头部
	header := make([]byte, 44)
	copy(header[0:4], []byte("RIFF"))
	binary.LittleEndian.PutUint32(header[4:8], uint32(fileSize))
	copy(header[8:12], []byte("WAVE"))
	copy(header[12:16], []byte("fmt "))
	binary.LittleEndian.PutUint32(header[16:20], 16)
	binary.LittleEndian.PutUint16(header[20:22], 3) // IEEE float
	binary.LittleEndian.PutUint16(header[22:24], uint16(numChannels))
	binary.LittleEndian.PutUint32(header[24:28], uint32(sampleRate))
	binary.LittleEndian.PutUint32(header[28:32], uint32(sampleRate*numChannels*bitsPerSample/8))
	binary.LittleEndian.PutUint16(header[32:34], uint16(numChannels*bitsPerSample/8))
	binary.LittleEndian.PutUint16(header[34:36], uint16(bitsPerSample))
	copy(header[36:40], []byte("data"))
	binary.LittleEndian.PutUint32(header[40:44], uint32(dataSize))

	// 写入头部
	if _, err := tempFile.Write(header); err != nil {
		return "", err
	}

	// 写入音频数据
	for _, sample := range audioData {
		if err := binary.Write(tempFile, binary.LittleEndian, sample); err != nil {
			return "", err
		}
	}

	return tempFile.Name(), nil
}

// processRecording 处理录音
func (va *VoiceAssistant) processRecording(audioBuffer [][]float32) {
	va.stateManager.SetState(state.StateProcessing)

	go func() {
		defer va.stateManager.SetState(state.StateIdle)

		// 合并音频缓冲区
		var combinedAudio []float32
		for _, chunk := range audioBuffer {
			combinedAudio = append(combinedAudio, chunk...)
		}

		if len(combinedAudio) == 0 {
			log.Println("音频缓冲区为空，跳过处理")
			return
		}

		fmt.Println("🔄 正在处理音频...")

		// 保存音频文件（如果启用）
		var audioFilePath string
		if va.config.SaveAudioFiles {
			audioFilePath = va.saveRecordedAudio(combinedAudio)
		}

		// 1. ASR - 语音转文本
		text, err := va.performASR(combinedAudio)
		if err != nil {
			log.Printf("语音识别失败: %v", err)
			va.playErrorMessage("抱歉，语音识别失败了")
			return
		}

		if text == "" {
			log.Println("识别到空文本，跳过处理")
			return
		}

		fmt.Printf("👤 用户: %s\n", text)

		// 2. LLM - 生成回复
		response, err := va.performLLM(text)
		if err != nil {
			log.Printf("LLM处理失败: %v", err)
			va.playErrorMessage("抱歉，我现在无法处理您的请求")
			return
		}

		fmt.Printf("🤖 助手: %s\n", response)

		// 记录对话日志
		if va.config.SaveAudioFiles {
			va.logConversation(text, response, audioFilePath)
		}

		// 3. TTS - 文本转语音并播放
		if err := va.performTTS(response); err != nil {
			log.Printf("TTS处理失败: %v", err)
			va.playErrorMessage("抱歉，语音合成失败了")
			return
		}
	}()
}

// performASR 执行语音识别
func (va *VoiceAssistant) performASR(audioData []float32) (string, error) {
	// 将音频数据保存为临时文件
	tempFile, err := va.saveAudioToTempFile(audioData)
	if err != nil {
		return "", err
	}
	defer os.Remove(tempFile)

	// 调用 ASR
	req := &asr.TranscribeRequest{
		Language: "zh",
		Model:    "whisper-1",
	}

	result, err := va.asrClient.TranscribeFile(va.ctx, tempFile, req)
	if err != nil {
		return "", err
	}

	return result.Text, nil
}

// performLLM 执行LLM对话
func (va *VoiceAssistant) performLLM(userText string) (string, error) {
	va.mu.Lock()
	defer va.mu.Unlock()

	// 添加用户消息到历史
	va.conversationHistory = append(va.conversationHistory, llm.Message{
		Role:    "user",
		Content: userText,
	})

	// 准备消息列表（包含系统提示）
	messages := []llm.Message{
		{
			Role:    "system",
			Content: va.config.SystemPrompt,
		},
	}
	messages = append(messages, va.conversationHistory...)

	// 调用 LLM
	req := &llm.ChatRequest{
		Model:       va.config.LLMModel,
		Messages:    messages,
		Temperature: va.config.LLMTemperature,
		MaxTokens:   500,
	}

	result, err := va.llmClient.ChatCompletion(va.ctx, req)
	if err != nil {
		return "", err
	}

	response := result.Choices[0].Message.Content

	// 添加助手回复到历史
	va.conversationHistory = append(va.conversationHistory, llm.Message{
		Role:    "assistant",
		Content: response,
	})

	// 限制历史长度
	if len(va.conversationHistory) > 20 {
		va.conversationHistory = va.conversationHistory[2:]
	}

	return response, nil
}

// performTTS 执行文本转语音
func (va *VoiceAssistant) performTTS(text string) error {
	va.stateManager.SetState(state.StateSpeaking)
	defer va.stateManager.SetState(state.StateIdle)

	// 创建播放专用的上下文
	va.mu.Lock()
	va.playbackCtx, va.playbackCancel = context.WithCancel(va.ctx)
	playCtx := va.playbackCtx
	va.mu.Unlock()

	defer func() {
		va.mu.Lock()
		if va.playbackCancel != nil {
			va.playbackCancel()
			va.playbackCancel = nil
			va.playbackCtx = nil
		}
		va.mu.Unlock()
	}()

	// 调用 TTS
	audioData, err := va.ttsClient.SynthesizeText(playCtx, text, tts.FormatWAV)
	if err != nil {
		return err
	}

	// 保存 TTS 音频（如果启用）
	if va.config.SaveAudioFiles {
		timestamp := time.Now().Format("20060102_150405")
		filename := filepath.Join(va.config.AudioOutputDir, fmt.Sprintf("tts_%s.wav", timestamp))
		if err := os.WriteFile(filename, audioData, 0644); err != nil {
			log.Printf("保存 TTS 音频失败: %v", err)
		}
	}

	// 播放音频 - 使用播放专用上下文
	err = va.audioOutput.PlayAudioData(playCtx, audioData, 16000)
	if err != nil && err != context.Canceled {
		return fmt.Errorf("播放音频失败: %w", err)
	}

	if err == context.Canceled {
		fmt.Println("🛑 音频播放被打断")
	} else {
		fmt.Println("🔊 语音播放完成")
	}

	return nil
}

// playErrorMessage 播放错误消息
func (va *VoiceAssistant) playErrorMessage(message string) {
	if err := va.performTTS(message); err != nil {
		log.Printf("播放错误消息失败: %v", err)
	}
}

// resetRecording 重置录音状态
func (va *VoiceAssistant) resetRecording(audioBuffer *[][]float32, recordingStart, silenceStart *time.Time) {
	va.isListening = false
	*audioBuffer = (*audioBuffer)[:0]
	*recordingStart = time.Time{}
	*silenceStart = time.Time{}
	va.stateManager.SetState(state.StateIdle)
}

// handleInterrupt 处理打断
func (va *VoiceAssistant) handleInterrupt() {
	va.mu.Lock()
	defer va.mu.Unlock()

	// 取消播放上下文（这会停止音频播放）
	if va.playbackCancel != nil {
		va.playbackCancel()
		va.playbackCancel = nil
		va.playbackCtx = nil
	}

	// 同时调用音频输出的停止方法（双重保险）
	va.audioOutput.Stop()

	// 重置所有状态
	va.isDetectingInterrupt = false
	va.interruptDetectionStart = time.Time{}
	va.stateManager.SetState(state.StateIdle)

	fmt.Println("🛑 播放已停止，可以开始新的对话")
}

// saveRecordedAudio 保存录音
func (va *VoiceAssistant) saveRecordedAudio(audioData []float32) string {
	timestamp := time.Now().Format("20060102_150405")
	filename := filepath.Join(va.config.AudioOutputDir, fmt.Sprintf("recording_%s.wav", timestamp))

	tempFile, err := va.saveAudioToTempFile(audioData)
	if err != nil {
		log.Printf("保存录音失败: %v", err)
		return ""
	}

	// 复制临时文件到输出目录
	data, err := os.ReadFile(tempFile)
	if err != nil {
		log.Printf("读取临时文件失败: %v", err)
		os.Remove(tempFile)
		return ""
	}

	if err := os.WriteFile(filename, data, 0644); err != nil {
		log.Printf("保存录音文件失败: %v", err)
		os.Remove(tempFile)
		return ""
	}

	os.Remove(tempFile)
	return filename
}

// logConversation 记录对话
func (va *VoiceAssistant) logConversation(userText, assistantText, audioFile string) {
	logFile := filepath.Join(va.config.AudioOutputDir, "conversation.log")

	timestamp := time.Now().Format("2006-01-02 15:04:05")
	logEntry := fmt.Sprintf("[%s] User: %s\n[%s] Assistant: %s\n",
		timestamp, userText, timestamp, assistantText)

	if audioFile != "" {
		logEntry += fmt.Sprintf("[%s] Audio: %s\n", timestamp, audioFile)
	}
	logEntry += "---\n"

	file, err := os.OpenFile(logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Printf("记录对话日志失败: %v", err)
		return
	}
	defer file.Close()

	if _, err := file.WriteString(logEntry); err != nil {
		log.Printf("写入对话日志失败: %v", err)
	}
}

// Stop 停止语音助手
func (va *VoiceAssistant) Stop() error {
	log.Println("正在停止语音助手...")

	// 取消上下文
	va.cancel()

	// 关闭音频模块
	if va.audioInput != nil {
		va.audioInput.Close()
	}

	if va.audioOutput != nil {
		va.audioOutput.Close()
	}

	log.Println("语音助手已停止")
	return nil
}

func main() {
	// 从环境变量读取配置
	config := getDefaultConfig()

	if apiKey := os.Getenv("OPENAI_API_KEY"); apiKey != "" {
		config.OpenAIAPIKey = apiKey
	}

	if vadURL := os.Getenv("VAD_SERVER_URL"); vadURL != "" {
		config.VADServerURL = vadURL
	}

	// 启用音频文件保存（用于调试）
	config.SaveAudioFiles = false

	// 检查环境变量是否禁用打断功能
	if disableInterrupt := os.Getenv("DISABLE_INTERRUPT"); disableInterrupt == "true" {
		config.AllowInterrupt = false
		fmt.Println("🔒 打断功能已禁用")
	}

	// 创建语音助手
	assistant, err := NewVoiceAssistant(config)
	if err != nil {
		log.Fatalf("创建语音助手失败: %v", err)
	}

	// 设置信号处理
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		fmt.Println("\n收到停止信号，正在关闭...")
		cancel()
		assistant.Stop()
	}()

	// 启动语音助手
	if err := assistant.Start(ctx); err != nil {
		log.Fatalf("启动语音助手失败: %v", err)
	}
}
