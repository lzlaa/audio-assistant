# TTS (Text-to-Speech) 模块

TTS 模块提供了基于 OpenAI TTS API 的文本转语音功能，支持多种语音、模型和音频格式。

## 功能特性

- **多种语音选择**：支持 6 种不同的语音（alloy, echo, fable, onyx, nova, shimmer）
- **多种模型**：支持 tts-1 和 tts-1-hd 模型
- **多种音频格式**：支持 MP3, Opus, AAC, FLAC, WAV, PCM 格式
- **语音速度控制**：支持 0.25x 到 4.0x 的语音速度调节
- **缓存机制**：内置音频缓存，提高响应速度
- **LLM 集成**：专门优化 LLM 响应文本的语音合成
- **文件管理**：支持音频文件保存和自动文件名生成
- **配置管理**：灵活的配置更新和验证

## 快速开始

### 基本使用

```go
package main

import (
    "context"
    "log"
    "os"
    
    "audio-assistant/internal/tts"
)

func main() {
    // 创建 TTS 客户端
    client := tts.NewTTSClient(os.Getenv("OPENAI_API_KEY"))
    
    // 合成语音
    ctx := context.Background()
    audioData, err := client.SynthesizeText(ctx, "你好，世界！", tts.FormatMP3)
    if err != nil {
        log.Fatal(err)
    }
    
    log.Printf("生成音频大小: %d 字节", len(audioData))
}
```

### 使用 TTS 服务

```go
package main

import (
    "context"
    "log"
    "os"
    
    "audio-assistant/internal/tts"
)

func main() {
    // 创建服务配置
    config := tts.DefaultTTSServiceConfig()
    config.Voice = tts.VoiceNova
    config.Speed = 1.2
    config.OutputFormat = tts.FormatMP3
    
    // 创建 TTS 服务
    service, err := tts.NewTTSService(os.Getenv("OPENAI_API_KEY"), config)
    if err != nil {
        log.Fatal(err)
    }
    
    // 启动服务
    if err := service.Start(); err != nil {
        log.Fatal(err)
    }
    defer service.Stop()
    
    // 合成语音到文件
    ctx := context.Background()
    err = service.SynthesizeToFile(ctx, "这是一个测试", "output.mp3")
    if err != nil {
        log.Fatal(err)
    }
    
    log.Println("音频文件已保存")
}
```

## API 参考

### TTSClient

#### 创建客户端

```go
client := tts.NewTTSClient(apiKey string) *TTSClient
```

#### 主要方法

```go
// 合成语音
SynthesizeText(ctx context.Context, text string, format string) ([]byte, error)

// 合成语音到文件
SynthesizeToFile(ctx context.Context, text string, format string, filename string) error

// 验证 API Key
ValidateAPIKey(ctx context.Context) error

// 配置管理
SetModel(model string)
SetVoice(voice string)
SetSpeed(speed float64)
GetConfig() TTSConfig
```

#### 验证方法

```go
// 验证文本
ValidateText(text string) error

// 验证语音
ValidateVoice(voice string) error

// 验证模型
ValidateModel(model string) error

// 验证格式
ValidateFormat(format string) error
```

### TTSService

#### 创建服务

```go
service, err := tts.NewTTSService(apiKey string, config TTSServiceConfig) (*TTSService, error)
```

#### 服务管理

```go
// 启动/停止服务
Start() error
Stop() error
IsRunning() bool
```

#### 语音合成

```go
// 基本合成
SynthesizeText(ctx context.Context, text string) ([]byte, error)

// 合成到文件
SynthesizeToFile(ctx context.Context, text string, filename string) error

// 自动文件名合成
SynthesizeWithAutoFilename(ctx context.Context, text string, prefix string) (string, error)

// 处理 LLM 响应
ProcessLLMResponse(ctx context.Context, llmResponse string) ([]byte, error)
```

#### 配置管理

```go
// 更新配置
UpdateConfig(config TTSServiceConfig) error

// 获取配置
GetConfig() TTSServiceConfig

// 获取可用选项
GetAvailableVoices() []string
GetAvailableModels() []string
GetAvailableFormats() []string
```

#### 缓存管理

```go
// 获取缓存统计
GetCacheStats() map[string]interface{}

// 清除缓存
ClearCache()
```

## 配置选项

### TTSServiceConfig

```go
type TTSServiceConfig struct {
    Model          string  `json:"model"`            // TTS 模型
    Voice          string  `json:"voice"`            // 语音类型
    Speed          float64 `json:"speed"`            // 语音速度
    OutputFormat   string  `json:"output_format"`    // 输出格式
    OutputDir      string  `json:"output_dir"`       // 输出目录
    CacheEnabled   bool    `json:"cache_enabled"`    // 是否启用缓存
    MaxTextLength  int     `json:"max_text_length"`  // 最大文本长度
    DefaultTimeout int     `json:"default_timeout_seconds"` // 默认超时
}
```

### 默认配置

```go
config := tts.DefaultTTSServiceConfig()
// Model: "tts-1"
// Voice: "alloy"
// Speed: 1.0
// OutputFormat: "mp3"
// OutputDir: "output/tts"
// CacheEnabled: true
// MaxTextLength: 4096
// DefaultTimeout: 60
```

## 支持的选项

### 模型

- `tts-1`: 标准 TTS 模型，速度快
- `tts-1-hd`: 高清 TTS 模型，质量更高

### 语音

- `alloy`: 中性语音
- `echo`: 男性语音
- `fable`: 英式男性语音
- `onyx`: 深沉男性语音
- `nova`: 女性语音
- `shimmer`: 柔和女性语音

### 音频格式

- `mp3`: MP3 格式（默认）
- `opus`: Opus 格式，适合实时应用
- `aac`: AAC 格式
- `flac`: FLAC 无损格式
- `wav`: WAV 格式
- `pcm`: PCM 原始音频

### 语音速度

- 范围：0.25x - 4.0x
- 默认：1.0x
- 推荐：0.8x - 1.5x

## 使用示例

### 1. 基本文本合成

```go
client := tts.NewTTSClient(apiKey)
audioData, err := client.SynthesizeText(ctx, "Hello, world!", tts.FormatMP3)
```

### 2. 不同语音测试

```go
voices := []string{tts.VoiceAlloy, tts.VoiceNova, tts.VoiceEcho}
for _, voice := range voices {
    client.SetVoice(voice)
    audioData, err := client.SynthesizeText(ctx, "测试语音", tts.FormatMP3)
    // 处理音频数据
}
```

### 3. 语音速度调节

```go
client.SetSpeed(0.8)  // 慢速
audioData1, _ := client.SynthesizeText(ctx, text, tts.FormatMP3)

client.SetSpeed(1.5)  // 快速
audioData2, _ := client.SynthesizeText(ctx, text, tts.FormatMP3)
```

### 4. 批量文件生成

```go
texts := []string{"文本1", "文本2", "文本3"}
for i, text := range texts {
    filename := fmt.Sprintf("output_%d.mp3", i+1)
    err := service.SynthesizeToFile(ctx, text, filename)
    if err != nil {
        log.Printf("生成文件 %s 失败: %v", filename, err)
    }
}
```

### 5. LLM 响应处理

```go
llmResponse := "**重要提醒**：请记得保存你的工作！\n\n详细信息请查看文档。"
audioData, err := service.ProcessLLMResponse(ctx, llmResponse)
// 自动优化文本格式，移除 Markdown 标记
```

### 6. 缓存使用

```go
// 第一次合成（调用 API）
audioData1, _ := service.SynthesizeText(ctx, "缓存测试")

// 第二次合成（使用缓存）
audioData2, _ := service.SynthesizeText(ctx, "缓存测试")

// 查看缓存统计
stats := service.GetCacheStats()
fmt.Printf("缓存条目: %v, 总大小: %v 字节", stats["entries"], stats["total_bytes"])
```

## 错误处理

### 常见错误

1. **API Key 无效**
```go
if err := client.ValidateAPIKey(ctx); err != nil {
    log.Fatal("API Key 验证失败:", err)
}
```

2. **文本过长**
```go
if err := client.ValidateText(text); err != nil {
    log.Fatal("文本验证失败:", err)
}
```

3. **网络超时**
```go
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()
audioData, err := client.SynthesizeText(ctx, text, format)
```

4. **权限不足**
```
Error: You have insufficient permissions for this operation. Missing scopes: model.request
```

### 错误处理最佳实践

```go
func synthesizeWithRetry(service *tts.TTSService, text string, maxRetries int) ([]byte, error) {
    var lastErr error
    
    for i := 0; i < maxRetries; i++ {
        ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
        audioData, err := service.SynthesizeText(ctx, text)
        cancel()
        
        if err == nil {
            return audioData, nil
        }
        
        lastErr = err
        log.Printf("合成失败 (尝试 %d/%d): %v", i+1, maxRetries, err)
        
        // 等待后重试
        time.Sleep(time.Duration(i+1) * time.Second)
    }
    
    return nil, fmt.Errorf("合成失败，已重试 %d 次: %w", maxRetries, lastErr)
}
```

## 性能优化

### 1. 缓存策略

```go
// 启用缓存
config.CacheEnabled = true

// 对于重复文本，使用缓存可以显著提高响应速度
```

### 2. 并发处理

```go
func synthesizeMultiple(service *tts.TTSService, texts []string) [][]byte {
    results := make([][]byte, len(texts))
    var wg sync.WaitGroup
    
    for i, text := range texts {
        wg.Add(1)
        go func(index int, content string) {
            defer wg.Done()
            audioData, err := service.SynthesizeText(context.Background(), content)
            if err == nil {
                results[index] = audioData
            }
        }(i, text)
    }
    
    wg.Wait()
    return results
}
```

### 3. 文本优化

```go
// 使用 ProcessLLMResponse 自动优化文本
audioData, err := service.ProcessLLMResponse(ctx, rawLLMResponse)

// 手动优化文本
optimizedText := strings.ReplaceAll(text, "\n\n", ". ")
optimizedText = strings.ReplaceAll(optimizedText, "**", "")
```

## 集成指南

### 与 LLM 模块集成

```go
// 获取 LLM 响应
llmResponse, err := llmService.SimpleChat(ctx, userInput)
if err != nil {
    return err
}

// 转换为语音
audioData, err := ttsService.ProcessLLMResponse(ctx, llmResponse)
if err != nil {
    return err
}

// 播放音频
return audioPlayer.Play(audioData)
```

### 与音频播放模块集成

```go
// 合成语音
audioData, err := ttsService.SynthesizeText(ctx, text)
if err != nil {
    return err
}

// 播放音频
player := audio.NewPlayer()
return player.PlayBytes(audioData)
```

## 故障排除

### 1. API 连接问题

```bash
# 测试网络连接
curl -I https://api.openai.com/v1/audio/speech

# 检查 API Key
echo $OPENAI_API_KEY
```

### 2. 权限问题

确保 API Key 具有以下权限：
- `model.request`: 模型请求权限
- TTS API 访问权限

### 3. 音频文件问题

```go
// 验证音频数据
if len(audioData) == 0 {
    return fmt.Errorf("音频数据为空")
}

// 检查文件权限
if err := tts.ValidateFilePath(filename); err != nil {
    return fmt.Errorf("文件路径无效: %w", err)
}
```

### 4. 性能问题

```go
// 监控合成时间
start := time.Now()
audioData, err := service.SynthesizeText(ctx, text)
duration := time.Since(start)
log.Printf("合成耗时: %v", duration)

// 检查缓存效果
stats := service.GetCacheStats()
log.Printf("缓存命中率: %.2f%%", float64(stats["hits"])/float64(stats["requests"])*100)
```

## 最佳实践

1. **合理使用缓存**：对于重复文本启用缓存
2. **控制文本长度**：保持在 4096 字符以内
3. **选择合适的模型**：tts-1 用于快速响应，tts-1-hd 用于高质量
4. **优化文本格式**：使用 ProcessLLMResponse 处理 LLM 输出
5. **错误处理**：实现重试机制和优雅降级
6. **监控性能**：跟踪合成时间和缓存效果
7. **资源管理**：及时清理临时文件和缓存

## 更新日志

- **v1.0.0**: 初始版本，支持基本 TTS 功能
- 支持多种语音、模型和格式
- 内置缓存机制
- LLM 响应优化
- 完整的错误处理和验证 