package vad

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
)

type SpeechTimestamp struct {
	Start int `json:"start"`
	End   int `json:"end"`
}

type VadResponse struct {
	SpeechTimestamps []SpeechTimestamp `json:"speech_timestamps"`
}

// CallVadService 发送音频数据到 VAD HTTP 服务，返回语音活动区间
func CallVadService(audioData []byte, vadURL string) ([]SpeechTimestamp, error) {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("audio", "audio.wav")
	if err != nil {
		return nil, err
	}
	_, err = part.Write(audioData)
	if err != nil {
		return nil, err
	}
	writer.Close()

	req, err := http.NewRequest("POST", vadURL, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("VAD 服务错误: %s", string(respBody))
	}

	var vadResp VadResponse
	err = json.Unmarshal(respBody, &vadResp)
	if err != nil {
		return nil, err
	}
	return vadResp.SpeechTimestamps, nil
}
