import torch
import torchaudio
import numpy as np
from flask import Flask, request, jsonify
import io
import wave

app = Flask(__name__)

# 加载 Silero VAD 模型
model, utils = torch.hub.load(repo_or_dir='snakers4/silero-vad',
                            model='silero_vad',
                            force_reload=True)

(get_speech_timestamps,
 save_audio,
 read_audio,
 VADIterator,
 collect_chunks) = utils

@app.route('/vad', methods=['POST'])
def process_audio():
    try:
        # 获取音频数据
        audio_data = request.get_data()
        
        # 将音频数据转换为 numpy 数组
        with io.BytesIO(audio_data) as wav_io:
            with wave.open(wav_io, 'rb') as wav:
                audio = np.frombuffer(wav.readframes(wav.getnframes()), dtype=np.int16)
                sample_rate = wav.getframerate()
        
        # 转换为 torch tensor
        audio_tensor = torch.from_numpy(audio).float()
        
        # 获取语音时间戳
        speech_timestamps = get_speech_timestamps(audio_tensor, model, 
                                                sampling_rate=sample_rate,
                                                threshold=0.5,
                                                min_speech_duration_ms=250,
                                                min_silence_duration_ms=100)
        
        # 转换时间戳为毫秒
        timestamps = []
        for ts in speech_timestamps:
            timestamps.append({
                'start': int(ts['start'] * 1000 / sample_rate),
                'end': int(ts['end'] * 1000 / sample_rate)
            })
        
        return jsonify({'timestamps': timestamps})
    
    except Exception as e:
        return jsonify({'error': str(e)}), 500

if __name__ == '__main__':
    app.run(host='0.0.0.0', port=5001) 