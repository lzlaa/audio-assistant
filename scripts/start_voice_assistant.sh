#!/bin/bash

# 语音助手启动脚本
# 使用方法: ./scripts/start_voice_assistant.sh

echo "=== 语音助手启动脚本 ==="

# 检查必要的环境变量
if [ -z "$OPENAI_API_KEY" ]; then
    echo "错误: 请设置 OPENAI_API_KEY 环境变量"
    echo "示例: export OPENAI_API_KEY='sk-your-api-key'"
    exit 1
fi

# 创建必要的目录
echo "创建必要目录..."
mkdir -p temp output

# 检查 VAD 服务是否运行
echo "检查 VAD 服务状态..."
VAD_URL=${VAD_SERVER_URL:-"http://localhost:8000"}

if ! curl -s -f "${VAD_URL}/health" > /dev/null 2>&1; then
    echo "警告: VAD 服务未运行，正在启动..."
    
    # 检查 VAD 服务脚本是否存在
    if [ -f "scripts/start_vad.sh" ]; then
        echo "启动 VAD 服务..."
        bash scripts/start_vad.sh &
        VAD_PID=$!
        
        # 等待 VAD 服务启动
        echo "等待 VAD 服务启动..."
        sleep 5
        
        # 再次检查
        retry_count=0
        max_retries=10
        while [ $retry_count -lt $max_retries ]; do
            if curl -s -f "${VAD_URL}/health" > /dev/null 2>&1; then
                echo "VAD 服务启动成功"
                break
            fi
            echo "等待 VAD 服务启动... ($((retry_count+1))/$max_retries)"
            sleep 2
            retry_count=$((retry_count+1))
        done
        
        if [ $retry_count -eq $max_retries ]; then
            echo "错误: VAD 服务启动失败"
            kill $VAD_PID 2>/dev/null
            exit 1
        fi
    else
        echo "错误: 找不到 VAD 服务启动脚本 scripts/start_vad.sh"
        echo "请先启动 VAD 服务，或者手动运行:"
        echo "cd vad_service && python app.py"
        exit 1
    fi
else
    echo "VAD 服务已运行"
fi

# 显示配置信息
echo ""
echo "=== 配置信息 ==="
echo "OpenAI API Key: ${OPENAI_API_KEY:0:12}..."
echo "VAD Server URL: $VAD_URL"
echo "音频输出目录: output"
echo ""

# 检查 Go 环境
if ! command -v go &> /dev/null; then
    echo "错误: 未找到 Go 环境，请先安装 Go"
    exit 1
fi

# 构建并运行语音助手
echo "=== 启动语音助手 ==="
echo "按 Ctrl+C 停止"
echo ""

# 设置清理函数
cleanup() {
    echo ""
    echo "正在清理资源..."
    if [ ! -z "$VAD_PID" ]; then
        echo "停止 VAD 服务..."
        kill $VAD_PID 2>/dev/null
        wait $VAD_PID 2>/dev/null
    fi
    echo "清理完成"
    exit 0
}

# 注册信号处理
trap cleanup SIGINT SIGTERM

# 运行语音助手
export VAD_SERVER_URL="$VAD_URL"
go run cmd/voice_assistant/main.go

# 如果程序正常退出，也要清理
cleanup 