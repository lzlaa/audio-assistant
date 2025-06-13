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

## 运行

1. 启动 VAD 服务:

```bash
python scripts/vad_service.py
```

2. 运行主程序:

```bash
go run cmd/main.go
```

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

## 开发状态

当前版本为 MVP，实现了基本的音频采集和播放功能。其他功能模块正在开发中。

## 许可证

MIT