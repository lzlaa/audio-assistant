import torch
import torchaudio
import numpy as np
from fastapi import FastAPI, UploadFile, File, HTTPException, Query
from fastapi.responses import JSONResponse
import uvicorn
import io
import logging
import tempfile
import os
from typing import Optional

# 配置日志
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

app = FastAPI(
    title="Silero VAD Service",
    description="语音活动检测服务，基于 Silero VAD 模型",
    version="1.0.0"
)

# 全局变量存储模型
model = None
utils = None
model_loaded = False

def load_model():
    """延迟加载 Silero VAD 模型"""
    global model, utils, model_loaded
    if model_loaded:
        return True
    
    try:
        logger.info("开始加载 Silero VAD 模型...")
        model, utils = torch.hub.load(repo_or_dir='snakers4/silero-vad',
                                    model='silero_vad',
                                    force_reload=False)
        model_loaded = True
        logger.info("VAD 模型加载成功")
        return True
    except Exception as e:
        logger.error(f"VAD 模型加载失败: {e}")
        return False

@app.get("/")
async def root():
    """根路径 - 服务状态检查"""
    return {
        "message": "Silero VAD Service",
        "status": "running",
        "model_loaded": model_loaded,
        "version": "1.0.0"
    }

@app.get("/health")
async def health_check():
    """健康检查端点"""
    if not model_loaded:
        # 尝试加载模型
        if not load_model():
            raise HTTPException(status_code=503, detail="模型未加载")
    
    return {
        "status": "healthy",
        "model_loaded": model_loaded,
        "timestamp": __import__("datetime").datetime.now().isoformat()
    }

@app.post("/detect")
async def detect_speech(
    audio_file: UploadFile = File(...),
    threshold: Optional[float] = Query(0.5, ge=0.0, le=1.0, description="语音检测阈值 (0.0-1.0)"),
    min_speech_duration_ms: Optional[int] = Query(250, ge=0, description="最小语音持续时间(毫秒)"),
    min_silence_duration_ms: Optional[int] = Query(100, ge=0, description="最小静音持续时间(毫秒)")
):
    """
    检测音频中的语音活动
    
    Args:
        audio_file: 上传的音频文件（WAV格式）
        threshold: 语音检测阈值，默认0.5
        min_speech_duration_ms: 最小语音持续时间(毫秒)，默认250ms
        min_silence_duration_ms: 最小静音持续时间(毫秒)，默认100ms
    
    Returns:
        JSON 包含语音活动的时间戳列表
    """
    temp_path = None
    try:
        # 延迟加载模型
        if not load_model():
            raise HTTPException(status_code=503, detail="模型加载失败")
        
        # 验证文件类型
        if not audio_file.filename.lower().endswith(('.wav', '.mp3', '.flac')):
            raise HTTPException(status_code=400, detail="只支持 WAV, MP3, FLAC 格式的音频文件")
        
        logger.info(f"接收到音频文件: {audio_file.filename}")
        
        # 读取音频文件
        contents = await audio_file.read()
        logger.info(f"音频文件大小: {len(contents)} bytes")
        
        if len(contents) == 0:
            raise HTTPException(status_code=400, detail="音频文件为空")
        
        # 将音频内容保存到临时文件
        suffix = os.path.splitext(audio_file.filename)[1] or '.wav'
        with tempfile.NamedTemporaryFile(delete=False, suffix=suffix) as temp_file:
            temp_file.write(contents)
            temp_path = temp_file.name
        
        logger.info("开始处理音频...")
        
        # 用 torchaudio 读取音频文件
        waveform, sample_rate = torchaudio.load(temp_path)
        logger.info(f"音频参数: sample_rate={sample_rate}, shape={waveform.shape}, duration={waveform.shape[1]/sample_rate:.2f}s")
        
        # 获取语音活动时间戳
        speech_timestamps = utils[0](
            waveform, 
            model, 
            threshold=threshold,
            sampling_rate=sample_rate,
            min_speech_duration_ms=min_speech_duration_ms,
            min_silence_duration_ms=min_silence_duration_ms
        )
        
        logger.info(f"检测到 {len(speech_timestamps)} 个语音片段")
        
        # 转换时间戳为列表格式
        result = []
        total_speech_duration = 0
        for ts in speech_timestamps:
            duration = ts['end'] - ts['start']
            total_speech_duration += duration
            result.append({
                "start": float(ts['start']),
                "end": float(ts['end']),
                "duration": float(duration)
            })
        
        # 计算统计信息
        audio_duration = waveform.shape[1] / sample_rate
        speech_ratio = total_speech_duration / audio_duration if audio_duration > 0 else 0
        
        return JSONResponse(content={
            "status": "success",
            "speech_segments": result,
            "statistics": {
                "total_segments": len(result),
                "total_speech_duration": round(total_speech_duration, 3),
                "total_audio_duration": round(audio_duration, 3),
                "speech_ratio": round(speech_ratio, 3),
                "sample_rate": sample_rate,
                "threshold_used": threshold
            }
        })
        
    except HTTPException:
        raise
    except Exception as e:
        logger.error(f"处理音频时出错: {e}")
        raise HTTPException(status_code=500, detail=f"音频处理失败: {str(e)}")
    finally:
        # 清理临时文件
        if temp_path and os.path.exists(temp_path):
            try:
                os.unlink(temp_path)
                logger.info("临时文件已清理")
            except Exception as e:
                logger.warning(f"清理临时文件失败: {e}")

@app.get("/info")
async def get_model_info():
    """获取模型信息"""
    if not model_loaded:
        raise HTTPException(status_code=503, detail="模型未加载")
    
    return {
        "model_name": "Silero VAD",
        "model_loaded": model_loaded,
        "supported_sample_rates": [8000, 16000],
        "supported_formats": ["wav", "mp3", "flac"],
        "default_threshold": 0.5,
        "description": "基于深度学习的语音活动检测模型"
    }

if __name__ == "__main__":
    logger.info("启动 VAD 服务...")
    uvicorn.run(app, host="127.0.0.1", port=8000) 