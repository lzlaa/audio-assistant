#!/bin/bash

# TTS 音频播放测试脚本

echo "========================================"
echo "TTS 音频播放功能测试"
echo "========================================"

# 检查环境变量
if [ -z "$OPENAI_API_KEY" ]; then
    echo "❌ 错误: 请设置 OPENAI_API_KEY 环境变量"
    echo "使用方法: export OPENAI_API_KEY='your-api-key'"
    exit 1
fi

echo "✓ OpenAI API Key 已设置"

# 检查 Go 环境
if ! command -v go &> /dev/null; then
    echo "❌ 错误: Go 未安装或不在 PATH 中"
    exit 1
fi

echo "✓ Go 环境已就绪"

# 创建临时目录
mkdir -p temp

# 编译并运行测试程序
echo ""
echo "📦 编译测试程序..."
if ! go build -o temp/test_tts_playback cmd/test_tts_playback/main.go; then
    echo "❌ 编译失败"
    exit 1
fi

echo "✓ 编译成功"

echo ""
echo "🎵 开始运行 TTS 音频播放测试..."
echo "注意: 请确保您的音响设备已连接并且音量适中"
echo ""

./temp/test_tts_playback

echo ""
echo "✅ TTS 音频播放测试完成"

# 清理
rm -f temp/test_tts_playback 