from fastapi import FastAPI, UploadFile, File
from fastapi.responses import JSONResponse
import uvicorn
import logging

# 配置日志
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

app = FastAPI(title="Simple VAD Test Service")

@app.get("/")
async def root():
    """根路径测试"""
    return {"message": "VAD service is running"}

@app.post("/detect")
async def detect_speech(audio_file: UploadFile = File(...)):
    """
    简化的语音检测测试
    """
    try:
        # 读取音频文件
        contents = await audio_file.read()
        logger.info(f"Received audio file: {audio_file.filename}, size: {len(contents)} bytes")
        
        # 返回模拟结果
        result = [
            {"start": 0.5, "end": 2.0},
            {"start": 3.0, "end": 4.5}
        ]
        
        return JSONResponse(content={
            "status": "success",
            "speech_segments": result,
            "message": f"Processed {len(contents)} bytes"
        })
        
    except Exception as e:
        logger.error(f"Error processing audio: {e}")
        return JSONResponse(
            status_code=500,
            content={"status": "error", "message": str(e)}
        )

if __name__ == "__main__":
    logger.info("Starting simple VAD test service...")
    uvicorn.run(app, host="127.0.0.1", port=8000) 