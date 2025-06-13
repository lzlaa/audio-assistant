# VAD (Voice Activity Detection) 客户端

这个模块提供了与 Python VAD 服务通信的 Go 客户端，用于检测音频中的语音活动。

## 功能特性

- **HTTP 客户端**：与 Python VAD 服务进行 HTTP 通信
- **多种检测方式**：支持文件和字节数据检测
- **配置灵活**：可调整检测阈值和时间参数
- **服务管理**：提供完整的服务生命周期管理
- **音频集成**：与音频模块无缝集成

## 主要组件

### 1. VAD 客户端 (`Client`)

基础的 HTTP 客户端，提供与 VAD 服务的直接通信。

```go
// 创建客户端
client := vad.NewClient("http://localhost:8000")

// 健康检查
health, err := client.Health()

// 获取模型信息
info, err := client.Info()

// 检测语音活动
response, err := client.DetectFromFile("audio.wav", &vad.DetectRequest{
    Threshold:            0.5,
    MinSpeechDurationMs:  250,
    MinSilenceDurationMs: 100,
})
```

### 2. VAD 服务 (`Service`)

高级服务管理器，提供与音频模块的集成和更便捷的 API。

```go
// 创建服务
config := vad.DefaultConfig()
service := vad.NewService(config, audioInput)

// 启动服务
err := service.Start()

// 检测语音活动
hasSpeech, err := service.HasSpeechInAudioData(audioData, sampleRate)

// 获取语音片段
segments, err := service.GetSpeechSegments(audioData, sampleRate)

// 停止服务
service.Stop()
```

## API 参考

### DetectRequest 参数

| 参数 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| `Threshold` | `float64` | 0.5 | 语音检测阈值 (0.0-1.0) |
| `MinSpeechDurationMs` | `int` | 250 | 最小语音持续时间 (毫秒) |
| `MinSilenceDurationMs` | `int` | 100 | 最小静音持续时间 (毫秒) |

### DetectResponse 字段

| 字段 | 类型 | 说明 |
|------|------|------|
| `Success` | `bool` | 检测是否成功 |
| `Message` | `string` | 错误消息（如果有） |
| `SpeechSegments` | `[]SpeechSegment` | 检测到的语音片段 |
| `TotalDuration` | `float64` | 音频总时长（秒） |
| `SpeechDuration` | `float64` | 语音总时长（秒） |
| `SilenceDuration` | `float64` | 静音总时长（秒） |
| `SpeechRatio` | `float64` | 语音占比 (0.0-1.0) |

### SpeechSegment 字段

| 字段 | 类型 | 说明 |
|------|------|------|
| `Start` | `float64` | 语音片段开始时间（秒） |
| `End` | `float64` | 语音片段结束时间（秒） |

## 使用示例

### 基础使用

```go
package main

import (
    "fmt"
    "log"
    
    "github.com/lzlaa/audio-assistant/internal/vad"
)

func main() {
    // 创建客户端
    client := vad.NewClient("http://localhost:8000")
    
    // 检查服务健康状态
    health, err := client.Health()
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("VAD server status: %s\n", health.Status)
    
    // 检测音频文件中的语音
    response, err := client.DetectFromFile("audio.wav", nil)
    if err != nil {
        log.Fatal(err)
    }
    
    fmt.Printf("Found %d speech segments\n", len(response.SpeechSegments))
    for i, segment := range response.SpeechSegments {
        fmt.Printf("Segment %d: %.2fs - %.2fs\n", i+1, segment.Start, segment.End)
    }
}
```

### 服务集成使用

```go
package main

import (
    "log"
    
    "github.com/lzlaa/audio-assistant/internal/audio"
    "github.com/lzlaa/audio-assistant/internal/vad"
)

func main() {
    // 创建音频输入
    audioInput, err := audio.NewInput()
    if err != nil {
        log.Fatal(err)
    }
    defer audioInput.Close()
    
    // 创建 VAD 服务
    config := vad.DefaultConfig()
    config.Threshold = 0.3  // 更敏感的检测
    
    service := vad.NewService(config, audioInput)
    
    // 启动服务
    if err := service.Start(); err != nil {
        log.Fatal(err)
    }
    defer service.Stop()
    
    // 使用服务检测语音
    // ... 你的音频处理逻辑
}
```

### 实时语音检测

```go
func detectSpeechRealtime(service *vad.Service, audioInput *audio.Input) {
    for {
        // 读取音频数据
        audioData, err := audioInput.Read()
        if err != nil {
            log.Printf("Failed to read audio: %v", err)
            continue
        }
        
        // 检测语音活动
        hasSpeech, err := service.HasSpeechInAudioData(audioData, 16000)
        if err != nil {
            log.Printf("VAD detection failed: %v", err)
            continue
        }
        
        if hasSpeech {
            fmt.Println("Speech detected!")
            // 处理语音数据...
        }
    }
}
```

## 配置说明

### 默认配置

```go
config := vad.DefaultConfig()
// config.ServerURL = "http://localhost:8000"
// config.Threshold = 0.5
// config.MinSpeechDurationMs = 250
// config.MinSilenceDurationMs = 100
// config.TempDir = "temp"
```

### 参数调优指南

#### 阈值 (Threshold)
- **0.1-0.3**：非常敏感，可能误检背景噪音
- **0.4-0.6**：平衡设置，适合大多数场景
- **0.7-0.9**：保守设置，只检测明显的语音

#### 最小语音持续时间 (MinSpeechDurationMs)
- **100-200ms**：检测短促的语音（如"嗯"、"啊"）
- **250-500ms**：检测正常的词语和短句
- **500ms+**：只检测较长的语音片段

#### 最小静音持续时间 (MinSilenceDurationMs)
- **50-100ms**：快速响应，适合实时应用
- **100-200ms**：平衡设置
- **200ms+**：避免频繁的开始/结束检测

## 错误处理

```go
response, err := client.DetectFromFile("audio.wav", nil)
if err != nil {
    // 网络或服务错误
    log.Printf("Detection failed: %v", err)
    return
}

if !response.Success {
    // VAD 处理错误
    log.Printf("VAD processing failed: %s", response.Message)
    return
}

// 处理成功的响应
fmt.Printf("Detected %d speech segments\n", len(response.SpeechSegments))
```

## 性能考虑

1. **临时文件**：服务会创建临时 WAV 文件，确保有足够的磁盘空间
2. **网络延迟**：HTTP 通信会有网络延迟，考虑使用缓存或批处理
3. **内存使用**：大音频文件会占用较多内存，考虑分块处理
4. **并发限制**：VAD 服务可能有并发限制，避免过多并发请求

## 故障排除

### 常见问题

1. **连接失败**
   ```
   Error: VAD server health check failed: connection refused
   ```
   - 确保 VAD 服务正在运行
   - 检查服务地址和端口

2. **检测失败**
   ```
   Error: detection failed with status 400: Invalid audio format
   ```
   - 检查音频文件格式
   - 确保音频文件完整且可读

3. **临时文件错误**
   ```
   Error: failed to create temp directory
   ```
   - 检查磁盘空间
   - 确保有写入权限

### 调试技巧

1. **启用详细日志**
   ```go
   log.SetLevel(log.DebugLevel)
   ```

2. **检查服务状态**
   ```go
   health, err := client.Health()
   if err == nil {
       fmt.Printf("Server status: %s\n", health.Status)
   }
   ```

3. **验证音频数据**
   ```go
   // 保存音频数据到文件进行检查
   audio.SaveToWAV("debug_audio.wav", audioData, sampleRate)
   ```

## 集成指南

### 与状态机集成

```go
// 在状态机中使用 VAD
func (sm *StateMachine) handleListening() {
    audioData := sm.audioBuffer.GetData()
    
    hasSpeech, err := sm.vadService.HasSpeechInAudioData(audioData, 16000)
    if err != nil {
        log.Printf("VAD error: %v", err)
        return
    }
    
    if hasSpeech {
        sm.TransitionTo(StateProcessing)
    }
}
```

### 与音频管道集成

```go
// 音频处理管道
type AudioPipeline struct {
    input      *audio.Input
    vadService *vad.Service
    // ... 其他组件
}

func (p *AudioPipeline) Process() {
    for {
        audioData, err := p.input.Read()
        if err != nil {
            continue
        }
        
        // VAD 检测
        segments, err := p.vadService.GetSpeechSegments(audioData, 16000)
        if err != nil {
            continue
        }
        
        // 处理语音片段
        for _, segment := range segments {
            p.processSpeechSegment(audioData, segment)
        }
    }
}
```

## 下一步

1. **集成 ASR**：将检测到的语音片段发送给 ASR 模块
2. **优化性能**：实现音频缓存和批处理
3. **添加监控**：添加性能监控和错误统计
4. **扩展功能**：支持更多音频格式和实时流处理 