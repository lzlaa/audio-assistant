# Audio Assistant

一个基于 Go 语言开发的语音助手项目。

## 功能特性

- 实时语音采集
- 语音活动检测 (VAD)
- 语音识别 (ASR)
- 大语言模型对话 (LLM)
- 文本转语音 (TTS)
- 实时语音播放
- 打断机制

## 环境要求

- Go 1.18 或更高版本
- PortAudio
- Python 3.8+ (用于 VAD 服务)
- OpenAI API Key

## 安装依赖

1. 安装 PortAudio:

```bash
# macOS
brew install portaudio

# Windows
# 下载并安装 PortAudio: https://www.portaudio.com/download.html
```

2. 安装 Go 依赖:

```bash
go mod download
```

## 快速开始

### 运行完整语音助手

1. 设置环境变量：
```bash
export OPENAI_API_KEY="your-openai-api-key"
```

2. 一键启动：
```bash
./scripts/start_voice_assistant.sh
```

### 手动启动

1. 启动 VAD 服务：
```bash
cd scripts
python vad_service.py
```

2. 启动语音助手：
```bash
go run cmd/voice_assistant/main.go
```

### 测试单个组件

- LLM 测试：`go run cmd/llm_example/main.go`
- ASR 测试：`go run cmd/asr_example/main.go`  
- TTS 测试：`go run cmd/tts_example/main.go`

## 项目结构

```
.
├── cmd/            # 主程序入口
├── internal/       # 内部包
│   ├── audio/     # 音频处理
│   ├── vad/       # 语音活动检测
│   ├── asr/       # 语音识别
│   ├── llm/       # 大语言模型
│   ├── tts/       # 文本转语音
│   ├── interrupt/ # 打断控制
│   └── state/     # 状态管理
├── pkg/           # 公共包
└── scripts/       # 脚本文件
```

## 运行示例
```
~: export OPENAI_API_KEY="your-openai-api-key"
~: go run cmd/voice_assistant/main.go

2025/06/13 23:41:56 启动语音助手...
2025/06/13 23:41:56 VAD 服务连接正常
2025/06/13 23:41:56 语音助手已启动，正在监听...
=== 语音助手已就绪，您可以开始对话 ===
2025/06/13 23:41:58 State changed: Idle -> Listening
🎤 开始录音...
🔇 检测到静音，结束录音
2025/06/13 23:42:01 State changed: Listening -> Processing
2025/06/13 23:42:01 State changed: Processing -> Idle
🔄 正在处理音频...
👤 用户: 請你給我講個故事
🤖 助手: 当然！这是一个关于勇敢的小猫咪的故事。小猫咪名叫小花，它住在一个美丽的小村庄里。有一天，小花听说森林里有一只被困的小鸟，于是它决定去救援。小花跋山涉水，终于来到了森林，找到了小鸟。小花用它的爪子和牙齿打开了困住小鸟的陷阱，小鸟获得自由后，非常感激地对小花说：“谢谢你，小花，你是一只勇敢又善良的小猫咪！”从此以后，小花和小鸟成为了最好的朋友，它们一起在森林里探险，分享快乐。故事告诉我们，勇敢和善良是最珍贵的品质，也让我们明白了友谊的力量。希望你喜欢这个故事！
2025/06/13 23:42:06 State changed: Idle -> Speaking
WAV file analysis for /var/folders/63/l52f96md6pd54wmg1mr257br0000gn/T/audio_decode_578213357.wav:
  RIFF chunk size: 4294967295
  File size: 2330444
  Found chunk: fmt , size: 16
  Found chunk: data, size: 4294967295
  Audio format: 1 (PCM=1)
  Channels: 1
  Sample rate: 24000
  Bits per sample: 16
  Data offset: 44
  Data size: 4294967295
  Warning: Data size in header (4294967295) exceeds file bounds. Using actual size: 2330400
  Calculated samples: 1165200
  Successfully loaded 1165200 samples
2025/06/13 23:42:13 准备播放音频: 样本数=1165200, 采样率=24000 Hz, 时长=48.55秒
2025/06/13 23:42:13 重采样音频: 24000 Hz -> 16000 Hz
重采样: 24000 Hz (1165200 样本) -> 16000 Hz (776800 样本)
🚫 检测到用户打断
```