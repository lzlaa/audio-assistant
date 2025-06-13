import io
import os
from flask import Flask, request, jsonify
import torch
import soundfile as sf
import torchaudio
from silero_vad import load_silero_vad, read_audio, get_speech_timestamps

app = Flask(__name__)

# 加载 Silero VAD 模型
model = load_silero_vad()


@app.route('/vad', methods=['POST'])
def vad_endpoint():
    if 'audio' not in request.files:
        return jsonify({'error': 'No audio file provided'}), 400
    file = request.files['audio']
    audio_bytes = file.read()

    # 使用 silero_vad 的 read_audio 函数读取音频
    try:
        # 将字节数据写入临时文件
        temp_path = 'temp_audio.wav'
        with open(temp_path, 'wb') as temp_file:
            temp_file.write(audio_bytes)

        # 读取音频
        wav = read_audio(temp_path, sampling_rate=16000)

        # 获取语音活动区间
        speech_timestamps = get_speech_timestamps(
            wav,
            model,
            return_seconds=True  # 返回秒为单位的时间戳
        )

        # 清理临时文件
        if os.path.exists(temp_path):
            os.remove(temp_path)

        return jsonify({'speech_timestamps': speech_timestamps})

    except Exception as e:
        # 清理临时文件
        if os.path.exists('temp_audio.wav'):
            os.remove('temp_audio.wav')
        return jsonify({'error': f'Processing failed: {str(e)}'}), 500


if __name__ == '__main__':
    app.run(host='0.0.0.0', port=5001, debug=True)
