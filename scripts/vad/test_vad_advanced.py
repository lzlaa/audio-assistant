import requests
import wave
import numpy as np
import sounddevice as sd
import time
import logging
import json

# 配置日志
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

BASE_URL = "http://127.0.0.1:8000"

def test_service_status():
    """测试服务状态"""
    logger.info("=== 测试服务状态 ===")
    try:
        response = requests.get(f"{BASE_URL}/")
        if response.status_code == 200:
            data = response.json()
            logger.info(f"服务状态: {json.dumps(data, indent=2, ensure_ascii=False)}")
            return True
        else:
            logger.error(f"状态检查失败: {response.status_code}")
            return False
    except Exception as e:
        logger.error(f"连接服务失败: {e}")
        return False

def test_health_check():
    """测试健康检查"""
    logger.info("=== 测试健康检查 ===")
    try:
        response = requests.get(f"{BASE_URL}/health")
        if response.status_code == 200:
            data = response.json()
            logger.info(f"健康状态: {json.dumps(data, indent=2, ensure_ascii=False)}")
            return True
        else:
            logger.error(f"健康检查失败: {response.status_code}")
            return False
    except Exception as e:
        logger.error(f"健康检查异常: {e}")
        return False

def test_model_info():
    """测试模型信息"""
    logger.info("=== 测试模型信息 ===")
    try:
        response = requests.get(f"{BASE_URL}/info")
        if response.status_code == 200:
            data = response.json()
            logger.info(f"模型信息: {json.dumps(data, indent=2, ensure_ascii=False)}")
            return True
        else:
            logger.error(f"获取模型信息失败: {response.status_code}")
            return False
    except Exception as e:
        logger.error(f"获取模型信息异常: {e}")
        return False

def record_audio(duration=5, sample_rate=16000):
    """录制音频"""
    logger.info(f"开始录音 {duration} 秒...")
    audio = sd.rec(int(duration * sample_rate), 
                  samplerate=sample_rate, 
                  channels=1, 
                  dtype='float32')
    sd.wait()
    logger.info("录音完成")
    return audio

def save_wav(audio, filename, sample_rate=16000):
    """保存为 WAV 文件"""
    # 将 float32 转换为 int16
    audio = (audio * 32767).astype(np.int16)
    
    with wave.open(filename, 'wb') as wf:
        wf.setnchannels(1)
        wf.setsampwidth(2)  # 2 bytes for int16
        wf.setframerate(sample_rate)
        wf.writeframes(audio.tobytes())
    
    logger.info(f"音频已保存到 {filename}")

def test_vad_detection(audio_file, threshold=0.5):
    """测试 VAD 检测"""
    logger.info(f"=== 测试 VAD 检测 (threshold={threshold}) ===")
    
    try:
        with open(audio_file, 'rb') as f:
            files = {'audio_file': f}
            params = {'threshold': threshold}
            response = requests.post(f"{BASE_URL}/detect", files=files, params=params)
            
        if response.status_code == 200:
            result = response.json()
            logger.info("VAD 检测结果:")
            logger.info(f"状态: {result['status']}")
            
            # 打印语音片段
            segments = result['speech_segments']
            for i, segment in enumerate(segments):
                logger.info(f"语音片段 {i+1}: {segment['start']:.2f}s - {segment['end']:.2f}s (时长: {segment['duration']:.2f}s)")
            
            # 打印统计信息
            stats = result['statistics']
            logger.info("统计信息:")
            logger.info(f"  总片段数: {stats['total_segments']}")
            logger.info(f"  总语音时长: {stats['total_speech_duration']:.2f}s")
            logger.info(f"  总音频时长: {stats['total_audio_duration']:.2f}s")
            logger.info(f"  语音占比: {stats['speech_ratio']*100:.1f}%")
            logger.info(f"  采样率: {stats['sample_rate']} Hz")
            logger.info(f"  使用阈值: {stats['threshold_used']}")
            
            return result
        else:
            logger.error(f"VAD 检测失败: {response.status_code}")
            logger.error(response.text)
            return None
            
    except Exception as e:
        logger.error(f"VAD 检测异常: {e}")
        return None

def test_different_thresholds(audio_file):
    """测试不同阈值的效果"""
    logger.info("=== 测试不同阈值效果 ===")
    thresholds = [0.3, 0.5, 0.7]
    
    for threshold in thresholds:
        result = test_vad_detection(audio_file, threshold)
        if result:
            stats = result['statistics']
            logger.info(f"阈值 {threshold}: 检测到 {stats['total_segments']} 个片段，语音占比 {stats['speech_ratio']*100:.1f}%")

def main():
    logger.info("开始 VAD 服务高级测试...")
    
    # 1. 测试服务基本功能
    if not test_service_status():
        logger.error("服务状态检查失败，退出测试")
        return
    
    if not test_health_check():
        logger.error("健康检查失败，退出测试")
        return
    
    if not test_model_info():
        logger.error("模型信息获取失败，退出测试")
        return
    
    # 2. 录制测试音频
    audio = record_audio(duration=5)
    test_file = "test_audio_advanced.wav"
    save_wav(audio, test_file)
    
    # 3. 测试 VAD 检测
    result = test_vad_detection(test_file)
    if not result:
        logger.error("VAD 检测失败，退出测试")
        return
    
    # 4. 测试不同阈值
    test_different_thresholds(test_file)
    
    logger.info("所有测试完成！")

if __name__ == "__main__":
    main() 