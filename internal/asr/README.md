# ASR (Automatic Speech Recognition) 模块

这个模块提供了基于 OpenAI Whisper API 的语音识别功能，可以将音频转换为文本。

## 功能特性

- **OpenAI Whisper 集成**：使用最先进的语音识别模型
- **多语言支持**：支持 99+ 种语言的语音识别
- **多种输入格式**：支持文件和字节数据输入
- **灵活配置**：可调整模型、语言、温度等参数
- **VAD 集成**：与语音活动检测模块无缝集成
- **详细响应**：提供时间戳和置信度信息

## 主要组件

### 1. ASR 客户端 (`Client`)

基础的 HTTP 客户端，提供与 OpenAI Whisper API 的直接通信。

```go
// 创建客户端
client := asr.NewClient("your-openai-api-key")

// 简单转录
text, err := client.TranscribeSimple(ctx, "audio.wav")

// 详细转录
response, err := client.TranscribeFile(ctx, "audio.wav", &asr.TranscribeRequest{
    Model:       "whisper-1",
    Language:    "zh",
    Temperature: 0.0,
    Format:      "verbose_json",
})
```

### 2. ASR 服务 (`Service`)

高级服务管理器，提供与音频模块和 VAD 的集成。

```go
// 创建服务
config := asr.DefaultConfig()
config.APIKey = "your-openai-api-key"
config.Language = "zh"

service, err := asr.NewService(config)

// 启动服务
err := service.Start(ctx)

// 转录音频数据
text, err := service.TranscribeAudioData(ctx, audioData, sampleRate)

// 转录语音片段
segments, err := service.TranscribeSpeechSegments(ctx, audioData, sampleRate, vadSegments)

// 停止服务
service.Stop()
```

## API 参考

### TranscribeRequest 参数

| 参数 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| `Model` | `string` | "whisper-1" | 使用的模型名称 |
| `Language` | `string` | "" | 语言代码 (ISO-639-1)，空值为自动检测 |
| `Prompt` | `string` | "" | 可选的提示文本，用于引导模型风格 |
| `Temperature` | `float32` | 0.0 | 采样温度 (0.0-1.0) |
| `Format` | `string` | "verbose_json" | 响应格式：json, text, srt, verbose_json, vtt |

### TranscribeResponse 字段

| 字段 | 类型 | 说明 |
|------|------|------|
| `Text` | `string` | 转录的文本内容 |
| `Language` | `string` | 检测到的语言 |
| `Duration` | `float64` | 音频时长（秒） |
| `Segments` | `[]Segment` | 详细的时间戳片段 |

### Segment 字段

| 字段 | 类型 | 说明 |
|------|------|------|
| `ID` | `int` | 片段 ID |
| `Start` | `float64` | 开始时间（秒） |
| `End` | `float64` | 结束时间（秒） |
| `Text` | `string` | 片段文本 |
| `NoSpeechProb` | `float64` | 无语音概率 |

## 使用示例

### 基础使用

```go
package main

import (
    "context"
    "fmt"
    "log"
    "os"
    
    "audio-assistant/internal/asr"
)

func main() {
    // 创建客户端
    client := asr.NewClient(os.Getenv("OPENAI_API_KEY"))
    
    // 简单转录
    ctx := context.Background()
    text, err := client.TranscribeSimple(ctx, "audio.wav")
    if err != nil {
        log.Fatal(err)
    }
    
    fmt.Printf("转录结果: %s\n", text)
}
```

### 服务集成使用

```go
package main

import (
    "context"
    "log"
    "os"
    
    "audio-assistant/internal/asr"
)

func main() {
    // 创建 ASR 服务
    config := asr.DefaultConfig()
    config.APIKey = os.Getenv("OPENAI_API_KEY")
    config.Language = "zh"  // 中文
    
    service, err := asr.NewService(config)
    if err != nil {
        log.Fatal(err)
    }
    
    // 启动服务
    ctx := context.Background()
    if err := service.Start(ctx); err != nil {
        log.Fatal(err)
    }
    defer service.Stop()
    
    // 转录音频文件
    text, err := service.TranscribeFile(ctx, "audio.wav")
    if err != nil {
        log.Fatal(err)
    }
    
    fmt.Printf("转录结果: %s\n", text)
}
```

### 与 VAD 集成

```go
func transcribeWithVAD(asrService *asr.Service, vadClient *vad.Client, audioFile string) {
    ctx := context.Background()
    
    // 1. 使用 VAD 检测语音片段
    vadResponse, err := vadClient.DetectFromFile(audioFile, &vad.DetectRequest{
        Threshold: 0.5,
        MinSpeechDurationMs: 250,
    })
    if err != nil {
        log.Fatal(err)
    }
    
    // 2. 加载音频数据
    audioData, sampleRate, err := loadAudioFile(audioFile)
    if err != nil {
        log.Fatal(err)
    }
    
    // 3. 转录每个语音片段
    transcriptions, err := asrService.TranscribeSpeechSegments(
        ctx, audioData, sampleRate, vadResponse.SpeechSegments)
    if err != nil {
        log.Fatal(err)
    }
    
    // 4. 处理转录结果
    for _, t := range transcriptions {
        fmt.Printf("片段 %d (%.2fs-%.2fs): %s\n", 
            t.SegmentIndex, t.Start, t.End, t.Text)
    }
}
```

### 详细转录

```go
func detailedTranscription(service *asr.Service, audioFile string) {
    ctx := context.Background()
    
    response, err := service.TranscribeWithDetails(ctx, audioFile)
    if err != nil {
        log.Fatal(err)
    }
    
    fmt.Printf("完整文本: %s\n", response.Text)
    fmt.Printf("检测语言: %s\n", response.Language)
    fmt.Printf("音频时长: %.2f秒\n", response.Duration)
    fmt.Printf("片段数量: %d\n", len(response.Segments))
    
    for _, segment := range response.Segments {
        fmt.Printf("  %.2fs-%.2fs: %s\n", 
            segment.Start, segment.End, segment.Text)
    }
}
```

## 配置说明

### 默认配置

```go
config := asr.DefaultConfig()
// config.BaseURL = "https://api.openai.com/v1"
// config.Model = "whisper-1"
// config.Language = ""  // 自动检测
// config.Temperature = 0.0
// config.Timeout = 60 * time.Second
// config.TempDir = "temp"
```

### 参数调优指南

#### 语言设置 (Language)
- **空值**：自动检测语言（推荐）
- **"zh"**：中文
- **"en"**：英文
- **"ja"**：日文
- **"ko"**：韩文
- **其他**：支持 99+ 种语言

#### 温度设置 (Temperature)
- **0.0**：确定性输出，一致性最高
- **0.2-0.4**：轻微随机性，适合大多数场景
- **0.6-0.8**：较高随机性，创造性更强
- **1.0**：最高随机性

#### 响应格式 (Format)
- **"text"**：纯文本，最简单
- **"json"**：JSON 格式，包含基本信息
- **"verbose_json"**：详细 JSON，包含时间戳和置信度
- **"srt"**：字幕格式
- **"vtt"**：WebVTT 字幕格式

## 支持的语言

ASR 模块支持 99+ 种语言，包括但不限于：

| 语言 | 代码 | 语言 | 代码 | 语言 | 代码 |
|------|------|------|------|------|------|
| 中文 | zh | 英文 | en | 日文 | ja |
| 韩文 | ko | 西班牙文 | es | 法文 | fr |
| 德文 | de | 俄文 | ru | 阿拉伯文 | ar |
| 印地文 | hi | 葡萄牙文 | pt | 意大利文 | it |

完整语言列表可通过 `service.GetSupportedLanguages()` 获取。

## 错误处理

```go
text, err := service.TranscribeFile(ctx, "audio.wav")
if err != nil {
    // 检查错误类型
    if strings.Contains(err.Error(), "invalid API key") {
        log.Fatal("API 密钥无效")
    } else if strings.Contains(err.Error(), "file size") {
        log.Fatal("文件大小超过限制 (25MB)")
    } else if strings.Contains(err.Error(), "unsupported format") {
        log.Fatal("不支持的音频格式")
    } else {
        log.Fatalf("转录失败: %v", err)
    }
}

fmt.Printf("转录成功: %s\n", text)
```

## 性能考虑

### 优势
- **高精度**：使用最先进的 Whisper 模型
- **多语言**：支持 99+ 种语言
- **实时处理**：支持流式音频处理
- **灵活配置**：可调整多种参数

### 限制
- **文件大小**：单个文件最大 25MB
- **API 限制**：受 OpenAI API 速率限制
- **网络依赖**：需要稳定的网络连接
- **成本**：按使用量计费

### 优化建议
1. **音频预处理**：使用 VAD 去除静音片段
2. **格式优化**：使用压缩格式减少传输时间
3. **批处理**：合并短片段减少 API 调用
4. **缓存**：缓存常用音频的转录结果

## 故障排除

### 常见问题

1. **API 密钥错误**
   ```
   Error: invalid API key
   ```
   - 检查 `OPENAI_API_KEY` 环境变量
   - 确认 API 密钥有效且有足够余额

2. **文件格式不支持**
   ```
   Error: unsupported audio format
   ```
   - 支持格式：MP3, MP4, MPEG, MPGA, M4A, WAV, WEBM
   - 转换音频格式或使用 WAV 格式

3. **文件过大**
   ```
   Error: file size exceeds maximum allowed size
   ```
   - 压缩音频文件
   - 分割长音频为多个片段

4. **网络超时**
   ```
   Error: context deadline exceeded
   ```
   - 增加超时时间
   - 检查网络连接
   - 减少音频文件大小

### 调试技巧

1. **启用详细日志**
   ```go
   log.SetLevel(log.DebugLevel)
   ```

2. **验证 API 密钥**
   ```go
   err := client.ValidateAPIKey(ctx)
   if err != nil {
       log.Printf("API key validation failed: %v", err)
   }
   ```

3. **检查音频文件**
   ```go
   fileInfo, err := os.Stat(audioFile)
   if err != nil {
       log.Printf("File error: %v", err)
   } else {
       log.Printf("File size: %d bytes", fileInfo.Size())
   }
   ```

## 集成指南

### 与状态机集成

```go
func (sm *StateMachine) handleProcessing() {
    audioData := sm.audioBuffer.GetData()
    
    text, err := sm.asrService.TranscribeAudioData(context.Background(), audioData, 16000)
    if err != nil {
        log.Printf("ASR error: %v", err)
        sm.TransitionTo(StateIdle)
        return
    }
    
    if text != "" {
        sm.recognizedText = text
        sm.TransitionTo(StateLLMProcessing)
    } else {
        sm.TransitionTo(StateListening)
    }
}
```

### 与音频管道集成

```go
type AudioPipeline struct {
    vadService *vad.Service
    asrService *asr.Service
    // ... 其他组件
}

func (p *AudioPipeline) ProcessAudio(audioData []float32, sampleRate int) (string, error) {
    // 1. VAD 检测
    segments, err := p.vadService.GetSpeechSegments(audioData, sampleRate)
    if err != nil {
        return "", err
    }
    
    if len(segments) == 0 {
        return "", nil // 无语音
    }
    
    // 2. ASR 转录
    ctx := context.Background()
    transcriptions, err := p.asrService.TranscribeSpeechSegments(
        ctx, audioData, sampleRate, segments)
    if err != nil {
        return "", err
    }
    
    // 3. 合并转录结果
    var fullText strings.Builder
    for _, t := range transcriptions {
        if fullText.Len() > 0 {
            fullText.WriteString(" ")
        }
        fullText.WriteString(t.Text)
    }
    
    return fullText.String(), nil
}
```

## 下一步

1. **集成 LLM**：将转录文本发送给大语言模型处理
2. **优化性能**：实现音频缓存和批处理
3. **添加监控**：添加转录质量和性能监控
4. **扩展功能**：支持实时流式转录 