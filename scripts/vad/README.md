# Silero VAD 语音活动检测服务

这是一个基于 Silero VAD 的语音活动检测服务，使用 FastAPI 提供 HTTP 接口。该服务可以检测音频文件中的语音片段，并返回详细的时间戳和统计信息。

## 功能特性

- 🎯 **高精度语音检测**：基于 Silero VAD 深度学习模型
- 🔧 **可调节参数**：支持自定义检测阈值和时长参数
- 📊 **详细统计信息**：提供语音占比、片段数量等统计数据
- 🎵 **多格式支持**：支持 WAV、MP3、FLAC 格式
- 🚀 **高性能**：延迟加载模型，优化内存使用
- 📋 **完整API文档**：自动生成的 OpenAPI 文档

## 环境要求

- Python 3.8+
- PyTorch 2.0+
- 其他依赖见 `requirements.txt`

## 快速开始

### 1. 自动安装与启动

```bash
# 使用启动脚本（推荐）
./start_vad.sh
```

### 2. 手动安装

```bash
# 创建虚拟环境
python3 -m venv venv
source venv/bin/activate  # Linux/Mac
# 或
.\venv\Scripts\activate  # Windows

# 安装依赖
pip install -r requirements.txt

# 启动服务
python vad_server.py
```

服务将在 http://127.0.0.1:8000 启动。

## API 接口

### 1. 服务状态检查

```bash
GET /
```

返回服务状态和版本信息。

### 2. 健康检查

```bash
GET /health
```

检查服务和模型加载状态。

### 3. 模型信息

```bash
GET /info
```

获取模型详细信息，包括支持的格式和参数。

### 4. 语音活动检测

```bash
POST /detect
```

**请求参数**：
- `audio_file`：音频文件（WAV/MP3/FLAC 格式）
- `threshold`：检测阈值 (0.0-1.0)，默认 0.5
- `min_speech_duration_ms`：最小语音持续时间(毫秒)，默认 250
- `min_silence_duration_ms`：最小静音持续时间(毫秒)，默认 100

**响应示例**：
```json
{
    "status": "success",
    "speech_segments": [
        {
            "start": 0.5,
            "end": 2.3,
            "duration": 1.8
        },
        {
            "start": 3.1,
            "end": 4.8,
            "duration": 1.7
        }
    ],
    "statistics": {
        "total_segments": 2,
        "total_speech_duration": 3.5,
        "total_audio_duration": 5.0,
        "speech_ratio": 0.7,
        "sample_rate": 16000,
        "threshold_used": 0.5
    }
}
```

## 测试工具

### 基础测试

```bash
python test_vad.py
```

### 高级测试

```bash
python test_vad_advanced.py
```

高级测试包括：
- 服务状态检查
- 健康检查测试
- 模型信息获取
- 不同阈值效果对比
- 详细统计信息展示

## 使用示例

### Python 客户端示例

```python
import requests

# 测试服务状态
response = requests.get("http://127.0.0.1:8000/")
print(response.json())

# 上传音频文件进行检测
with open("audio.wav", "rb") as f:
    files = {"audio_file": f}
    params = {"threshold": 0.5}
    response = requests.post("http://127.0.0.1:8000/detect", 
                           files=files, params=params)
    result = response.json()
    
    for segment in result["speech_segments"]:
        print(f"语音片段: {segment['start']:.2f}s - {segment['end']:.2f}s")
```

### cURL 示例

```bash
# 检查服务状态
curl http://127.0.0.1:8000/

# 上传音频文件
curl -X POST "http://127.0.0.1:8000/detect?threshold=0.5" \
     -F "audio_file=@audio.wav"
```

## 参数调优指南

### 检测阈值 (threshold)

- **0.3-0.4**：敏感检测，可能包含较多噪音
- **0.5**：默认值，平衡准确性和敏感性
- **0.6-0.8**：保守检测，只检测明显的语音

### 时长参数

- **min_speech_duration_ms**：过滤掉过短的语音片段
- **min_silence_duration_ms**：合并过近的语音片段

## 性能优化

- 首次使用时会下载模型文件（~40MB）
- 模型采用延迟加载，只在首次检测时加载
- 支持音频文件临时存储，自动清理
- 内存占用约 200-500MB（取决于音频长度）

## 故障排除

### 1. 服务无法启动

```bash
# 检查端口占用
lsof -i :8000

# 停止现有服务
pkill -f vad_server.py
```

### 2. 模型下载失败

```bash
# 清理 torch hub 缓存
rm -rf ~/.cache/torch/hub/snakers4_silero-vad_master

# 重新启动服务
python vad_server.py
```

### 3. 音频格式不支持

- 确保音频文件是 WAV、MP3 或 FLAC 格式
- 推荐使用 16kHz 采样率的 WAV 文件

## 集成到 Go 项目

VAD 服务设计为独立运行，Go 项目可通过 HTTP 客户端调用：

```go
// 示例：Go 客户端调用 VAD 服务
resp, err := http.Post("http://127.0.0.1:8000/detect", 
                      "multipart/form-data", audioData)
```

详细的 Go 集成代码请参考主项目的 VAD 客户端模块。

## 开发说明

- 基于 FastAPI 框架
- 使用 Silero VAD 预训练模型
- 支持 OpenAPI 自动文档生成
- 包含完整的错误处理和日志记录 