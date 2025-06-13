import requests
import wave
import numpy as np
import sounddevice as sd
import time
import logging

# 配置日志
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

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

def test_vad_service(audio_file):
    """测试 VAD 服务"""
    url = "http://127.0.0.1:8000/detect"
    
    try:
        with open(audio_file, 'rb') as f:
            files = {'audio_file': f}
            response = requests.post(url, files=files)
            
        if response.status_code == 200:
            result = response.json()
            logger.info("VAD 检测结果:")
            for segment in result['speech_segments']:
                logger.info(f"语音片段: {segment['start']:.2f}s - {segment['end']:.2f}s")
            return result
        else:
            logger.error(f"请求失败: {response.status_code}")
            logger.error(response.text)
            return None
            
    except Exception as e:
        logger.error(f"测试过程中出错: {e}")
        return None

def main():
    # 1. 录制测试音频
    audio = record_audio(duration=5)
    test_file = "test_audio.wav"
    save_wav(audio, test_file)
    
    # 2. 测试 VAD 服务
    logger.info("开始测试 VAD 服务...")
    result = test_vad_service(test_file)
    
    if result:
        logger.info("VAD 服务测试完成")
    else:
        logger.error("VAD 服务测试失败")

if __name__ == "__main__":
    main() 