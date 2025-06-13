# LLM (Large Language Model) 模块

这个模块提供了基于 OpenAI GPT API 的大语言模型功能，可以进行智能对话和文本生成。

## 功能特性

- **OpenAI GPT 集成**：支持 GPT-3.5-turbo、GPT-4 等最新模型
- **对话管理**：维护对话历史和上下文
- **语音优化**：专门为语音助手优化的响应生成
- **灵活配置**：可调整模型、温度、最大token等参数
- **多种对话模式**：支持单轮对话、多轮对话、带上下文对话
- **智能截断**：自动管理对话历史长度和token限制

## 主要组件

### 1. LLM 客户端 (`Client`)

基础的 HTTP 客户端，提供与 OpenAI GPT API 的直接通信。

```go
// 创建客户端
client := llm.NewClient("your-openai-api-key")

// 简单对话
response, err := client.SimpleChat(ctx, "你好，请介绍一下你自己")

// 带系统消息的对话
response, err := client.ChatWithSystem(ctx, "你是一个专业的技术助手", "什么是Go语言？")

// 完整对话请求
chatResp, err := client.ChatCompletion(ctx, &llm.ChatRequest{
    Model: "gpt-3.5-turbo",
    Messages: []llm.Message{
        {Role: "system", Content: "你是一个智能助手"},
        {Role: "user", Content: "用户问题"},
    },
    MaxTokens:   150,
    Temperature: 0.7,
})
```

### 2. LLM 服务 (`Service`)

高级服务管理器，提供对话历史管理和语音助手集成。

```go
// 创建服务
config := llm.DefaultConfig()
config.APIKey = "your-openai-api-key"
config.Model = "gpt-3.5-turbo"
config.MaxTokens = 150

service, err := llm.NewService(config)

// 启动服务
err := service.Start(ctx)

// 对话（带历史记录）
response, err := service.Chat(ctx, "我叫张三")
response, err := service.Chat(ctx, "我叫什么名字？") // 会记住之前的对话

// 语音优化响应
response, err := service.GenerateVoiceResponse(ctx, "今天天气怎么样？")

// 处理ASR转录文本
response, err := service.ProcessTranscribedText(ctx, "请告诉我现在几点了")

// 停止服务
service.Stop()
```

## API 参考

### ChatRequest 参数

| 参数 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| `Model` | `string` | "gpt-3.5-turbo" | 使用的模型名称 |
| `Messages` | `[]Message` | - | 对话消息列表 |
| `MaxTokens` | `int` | 1000 | 最大生成token数 |
| `Temperature` | `float32` | 0.7 | 采样温度 (0.0-2.0) |
| `TopP` | `float32` | - | 核采样参数 (0.0-1.0) |
| `PresencePenalty` | `float32` | - | 存在惩罚 (-2.0-2.0) |
| `FrequencyPenalty` | `float32` | - | 频率惩罚 (-2.0-2.0) |

### Message 结构

| 字段 | 类型 | 说明 |
|------|------|------|
| `Role` | `string` | 消息角色：system, user, assistant |
| `Content` | `string` | 消息内容 |

### ChatResponse 字段

| 字段 | 类型 | 说明 |
|------|------|------|
| `ID` | `string` | 响应ID |
| `Model` | `string` | 使用的模型 |
| `Choices` | `[]Choice` | 生成的选择列表 |
| `Usage` | `Usage` | Token使用统计 |

## 使用示例

### 基础使用

```go
package main

import (
    "context"
    "fmt"
    "log"
    "os"
    
    "audio-assistant/internal/llm"
)

func main() {
    // 创建客户端
    client := llm.NewClient(os.Getenv("OPENAI_API_KEY"))
    
    // 简单对话
    ctx := context.Background()
    response, err := client.SimpleChat(ctx, "你好，请介绍一下你自己")
    if err != nil {
        log.Fatal(err)
    }
    
    fmt.Printf("AI回复: %s\n", response)
}
```

### 服务集成使用

```go
package main

import (
    "context"
    "log"
    "os"
    
    "audio-assistant/internal/llm"
)

func main() {
    // 创建 LLM 服务
    config := llm.DefaultConfig()
    config.APIKey = os.Getenv("OPENAI_API_KEY")
    config.Model = "gpt-3.5-turbo"
    config.MaxTokens = 150
    
    service, err := llm.NewService(config)
    if err != nil {
        log.Fatal(err)
    }
    
    // 启动服务
    ctx := context.Background()
    if err := service.Start(ctx); err != nil {
        log.Fatal(err)
    }
    defer service.Stop()
    
    // 对话
    response, err := service.Chat(ctx, "你好，我是用户")
    if err != nil {
        log.Fatal(err)
    }
    
    fmt.Printf("AI回复: %s\n", response)
}
```

### 多轮对话

```go
func multiTurnConversation(service *llm.Service) {
    ctx := context.Background()
    
    // 清除历史记录
    service.ClearHistory()
    
    // 第一轮对话
    response1, err := service.Chat(ctx, "我叫张三，今年25岁")
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("AI: %s\n", response1)
    
    // 第二轮对话（AI会记住之前的信息）
    response2, err := service.Chat(ctx, "我叫什么名字？")
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("AI: %s\n", response2) // 应该回答"张三"
    
    // 查看对话历史
    history := service.GetConversationHistory()
    fmt.Printf("对话历史包含 %d 条消息\n", len(history))
}
```

### 语音助手集成

```go
func voiceAssistantIntegration(service *llm.Service) {
    ctx := context.Background()
    
    // 处理来自ASR的转录文本
    transcribedText := "请帮我设置明天早上八点的闹钟"
    response, err := service.ProcessTranscribedText(ctx, transcribedText)
    if err != nil {
        log.Fatal(err)
    }
    
    fmt.Printf("用户语音: %s\n", transcribedText)
    fmt.Printf("AI回复: %s\n", response)
    
    // 生成语音优化的回复
    voiceResponse, err := service.GenerateVoiceResponse(ctx, "今天天气怎么样？")
    if err != nil {
        log.Fatal(err)
    }
    
    fmt.Printf("语音优化回复: %s\n", voiceResponse)
}
```

### 自定义系统消息

```go
func customSystemMessage(service *llm.Service) {
    // 设置专业技术助手的系统消息
    systemMsg := `你是一个专业的Go语言技术助手。请遵循以下规则：
1. 专注于Go语言相关问题
2. 提供准确的技术信息
3. 回复要简洁明了
4. 包含代码示例（如果适用）`
    
    service.SetSystemMessage(systemMsg)
    
    ctx := context.Background()
    response, err := service.Chat(ctx, "如何在Go中处理错误？")
    if err != nil {
        log.Fatal(err)
    }
    
    fmt.Printf("技术助手回复: %s\n", response)
}
```

## 配置说明

### 默认配置

```go
config := llm.DefaultConfig()
// config.BaseURL = "https://api.openai.com/v1"
// config.Model = "gpt-3.5-turbo"
// config.Temperature = 0.7
// config.MaxTokens = 150
// config.MaxHistoryLength = 10
// config.UserName = "用户"
// config.Timeout = 30 * time.Second
```

### 参数调优指南

#### 模型选择 (Model)
- **gpt-3.5-turbo**：快速、经济，适合大多数场景
- **gpt-4**：更智能，理解能力更强，但速度较慢
- **gpt-4-turbo-preview**：GPT-4的优化版本

#### 温度设置 (Temperature)
- **0.0-0.3**：确定性输出，适合事实性问答
- **0.4-0.7**：平衡创造性和一致性，适合对话
- **0.8-1.0**：高创造性，适合创意写作
- **1.0-2.0**：极高随机性，实验性用途

#### Token限制 (MaxTokens)
- **50-100**：简短回复，适合语音助手
- **150-300**：中等长度，适合一般对话
- **500-1000**：详细回复，适合复杂问题
- **1000+**：长文本生成

#### 历史长度 (MaxHistoryLength)
- **5-10**：短期记忆，适合简单对话
- **10-20**：中期记忆，适合复杂对话
- **20+**：长期记忆，但会增加token消耗

## 支持的模型

| 模型 | 特点 | 适用场景 |
|------|------|----------|
| gpt-3.5-turbo | 快速、经济 | 日常对话、简单问答 |
| gpt-3.5-turbo-16k | 长上下文版本 | 长文档处理 |
| gpt-4 | 高智能、准确 | 复杂推理、专业问答 |
| gpt-4-turbo-preview | GPT-4优化版 | 平衡性能和成本 |

## 错误处理

```go
response, err := service.Chat(ctx, "用户消息")
if err != nil {
    // 检查错误类型
    if strings.Contains(err.Error(), "invalid API key") {
        log.Fatal("API 密钥无效")
    } else if strings.Contains(err.Error(), "rate limit") {
        log.Println("API 速率限制，稍后重试")
        time.Sleep(time.Minute)
        // 重试逻辑
    } else if strings.Contains(err.Error(), "context deadline exceeded") {
        log.Println("请求超时")
    } else {
        log.Printf("对话失败: %v", err)
    }
    return
}

fmt.Printf("AI回复: %s\n", response)
```

## 性能考虑

### 优势
- **智能对话**：使用最先进的语言模型
- **上下文理解**：维护对话历史和上下文
- **灵活配置**：支持多种模型和参数
- **语音优化**：专门为语音助手优化

### 限制
- **API限制**：受OpenAI API速率和配额限制
- **网络依赖**：需要稳定的网络连接
- **成本考虑**：按token使用量计费
- **延迟**：网络请求会有一定延迟

### 优化建议
1. **合理设置MaxTokens**：避免生成过长的回复
2. **管理对话历史**：定期清理或截断历史记录
3. **选择合适模型**：根据需求平衡性能和成本
4. **实现重试机制**：处理网络错误和速率限制
5. **缓存常见回复**：减少API调用次数

## 故障排除

### 常见问题

1. **API 密钥错误**
   ```
   Error: invalid API key
   ```
   - 检查 `OPENAI_API_KEY` 环境变量
   - 确认API密钥有效且有足够余额

2. **速率限制**
   ```
   Error: rate limit exceeded
   ```
   - 降低请求频率
   - 实现指数退避重试
   - 考虑升级API计划

3. **上下文长度超限**
   ```
   Error: maximum context length exceeded
   ```
   - 减少MaxTokens设置
   - 清理对话历史
   - 使用支持更长上下文的模型

4. **网络超时**
   ```
   Error: context deadline exceeded
   ```
   - 增加超时时间
   - 检查网络连接
   - 实现重试机制

### 调试技巧

1. **启用详细日志**
   ```go
   log.SetLevel(log.DebugLevel)
   ```

2. **验证API密钥**
   ```go
   err := client.ValidateAPIKey(ctx)
   if err != nil {
       log.Printf("API key validation failed: %v", err)
   }
   ```

3. **监控Token使用**
   ```go
   totalTokens := service.GetHistoryTokenCount()
   log.Printf("Current history tokens: %d", totalTokens)
   ```

4. **检查对话历史**
   ```go
   history := service.GetConversationHistory()
   for i, msg := range history {
       log.Printf("Message %d [%s]: %s", i, msg.Role, msg.Content)
   }
   ```

## 集成指南

### 与ASR集成

```go
func processVoiceInput(asrService *asr.Service, llmService *llm.Service, audioData []float32) {
    ctx := context.Background()
    
    // 1. ASR转录
    transcribedText, err := asrService.TranscribeAudioData(ctx, audioData, 16000)
    if err != nil {
        log.Printf("ASR error: %v", err)
        return
    }
    
    // 2. LLM处理
    response, err := llmService.ProcessTranscribedText(ctx, transcribedText)
    if err != nil {
        log.Printf("LLM error: %v", err)
        return
    }
    
    log.Printf("用户: %s", transcribedText)
    log.Printf("AI: %s", response)
    
    // 3. 发送给TTS模块...
}
```

### 与状态机集成

```go
func (sm *StateMachine) handleLLMProcessing() {
    if sm.recognizedText == "" {
        sm.TransitionTo(StateListening)
        return
    }
    
    response, err := sm.llmService.Chat(context.Background(), sm.recognizedText)
    if err != nil {
        log.Printf("LLM error: %v", err)
        sm.TransitionTo(StateIdle)
        return
    }
    
    sm.llmResponse = response
    sm.TransitionTo(StateTTSProcessing)
}
```

### 智能对话管理

```go
type ConversationManager struct {
    llmService *llm.Service
    userProfiles map[string]*UserProfile
}

type UserProfile struct {
    Name        string
    Preferences map[string]string
    History     []llm.Message
}

func (cm *ConversationManager) ProcessUserInput(userID, input string) (string, error) {
    profile := cm.getUserProfile(userID)
    
    // 添加用户上下文
    contextualInput := fmt.Sprintf("用户%s说：%s", profile.Name, input)
    
    // 生成回复
    response, err := cm.llmService.Chat(context.Background(), contextualInput)
    if err != nil {
        return "", err
    }
    
    // 更新用户档案
    cm.updateUserProfile(userID, input, response)
    
    return response, nil
}
```

## 高级功能

### 1. 对话模板

```go
// 创建对话模板
func createCustomerServiceTemplate() llm.Message {
    return llm.Message{
        Role: "system",
        Content: `你是一个专业的客服助手。请遵循以下规则：
1. 始终保持礼貌和专业
2. 优先解决用户问题
3. 如果无法解决，引导用户联系人工客服
4. 回复要简洁明了，不超过50字`,
    }
}

// 使用模板
service.SetSystemMessage(createCustomerServiceTemplate().Content)
```

### 2. 情感分析集成

```go
func analyzeAndRespond(service *llm.Service, userInput string) (string, error) {
    // 分析用户情感
    emotionPrompt := fmt.Sprintf("分析以下文本的情感倾向（积极/消极/中性）：%s", userInput)
    emotion, err := service.SimpleChat(context.Background(), emotionPrompt)
    if err != nil {
        return "", err
    }
    
    // 根据情感调整回复风格
    var systemMsg string
    if strings.Contains(emotion, "消极") {
        systemMsg = "用户情绪不佳，请用温暖、理解的语气回复，提供安慰和帮助。"
    } else {
        systemMsg = "用正常的友好语气回复用户。"
    }
    
    return service.ChatWithContext(context.Background(), userInput, systemMsg)
}
```

### 3. 多语言支持

```go
func multiLanguageChat(service *llm.Service, userInput, language string) (string, error) {
    systemMsg := fmt.Sprintf("请用%s回复用户，保持自然流畅的表达。", language)
    return service.ChatWithContext(context.Background(), userInput, systemMsg)
}

// 使用示例
response, err := multiLanguageChat(service, "Hello, how are you?", "中文")
```

## 下一步

1. **集成TTS**：将LLM生成的文本转换为语音
2. **优化性能**：实现响应缓存和批处理
3. **增强功能**：添加知识库检索和工具调用
4. **个性化**：实现用户画像和个性化回复 