# audio-assistant

# Windows 下 Go 端 portaudio 依赖安装说明

1. 安装 portaudio DLL

- 推荐直接下载 [portaudio.dll](http://files.portaudio.com/download.html) 并放到 Windows 系统 PATH 路径下（如 C:\Windows\System32）。
- 或用 Python 安装：

```bash
pip install pipwin
pipwin install portaudio
```

2. 安装 Go 依赖

```bash
go get github.com/gordonklaus/portaudio
```

如遇找不到 DLL 错误，请确认 portaudio.dll 已在 PATH 路径下。

## 项目目录结构

```
audio-assistant/
├── cmd/                # 主程序入口
│   └── main.go
├── internal/
│   ├── audio/          # 采集与播放
│   ├── vad/            # VAD客户端
│   ├── asr/            # Whisper API
│   ├── llm/            # GPT API
│   ├── tts/            # TTS API
│   └── flow/           # 主控流程/状态机
├── scripts/
│   └── silero_vad.py   # Python VAD服务
├── doc/
│   └── design.md
├── go.mod
└── README.md
```