#!/bin/bash

# VAD 服务启动脚本

set -e

# 获取脚本所在目录
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

echo "启动 Silero VAD 服务..."

# 检查虚拟环境
if [ ! -d "venv" ]; then
    echo "虚拟环境不存在，正在创建..."
    python3 -m venv venv
fi

# 激活虚拟环境
echo "激活虚拟环境..."
source venv/bin/activate

# 安装依赖
echo "安装依赖..."
pip install -r requirements.txt

# 检查端口是否被占用
if lsof -i :8000 >/dev/null 2>&1; then
    echo "端口 8000 已被占用，请先停止现有服务"
    echo "可以运行: pkill -f vad_server.py"
    exit 1
fi

# 启动服务
echo "启动 VAD 服务 (http://127.0.0.1:8000)..."
python vad_server.py 