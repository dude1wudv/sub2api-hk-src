package handler

import (
	"bytes"
	"encoding/binary"
	"mime/multipart"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func qwenAudioSpeechTestContext(t *testing.T, payload string) *gin.Context {
	t.Helper()
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("POST", "/v1/audio/speech", strings.NewReader(payload))
	c.Request.Header.Set("Content-Type", "application/json")
	return c
}

func handlerPCM16WAV(sampleRate, seconds int) []byte {
	dataSize := sampleRate * 2 * seconds
	out := make([]byte, 44+dataSize)
	copy(out[0:4], "RIFF")
	binary.LittleEndian.PutUint32(out[4:8], uint32(len(out)-8))
	copy(out[8:16], "WAVEfmt ")
	binary.LittleEndian.PutUint32(out[16:20], 16)
	binary.LittleEndian.PutUint16(out[20:22], 1)
	binary.LittleEndian.PutUint16(out[22:24], 1)
	binary.LittleEndian.PutUint32(out[24:28], uint32(sampleRate))
	binary.LittleEndian.PutUint32(out[28:32], uint32(sampleRate*2))
	binary.LittleEndian.PutUint16(out[32:34], 2)
	binary.LittleEndian.PutUint16(out[34:36], 16)
	copy(out[36:40], "data")
	binary.LittleEndian.PutUint32(out[40:44], uint32(dataSize))
	return out
}

func TestParseQwenAudioSpeechAcceptsLongQwenInstructionAndDefaults(t *testing.T) {
	instruction := strings.Repeat("情", 150)
	c := qwenAudioSpeechTestContext(t, `{"model":"qwen-tts","input":"中文回复","voice":"longanhuan_v3.6","instructions":"`+instruction+`"}`)
	req, err := parseQwenAudioSpeechRequest(c)
	require.NoError(t, err)
	require.Equal(t, instruction, req.speech.Instructions)
	require.Equal(t, 1.0, *req.speech.Speed)
	require.Equal(t, "mp3", req.speech.ResponseFormat)
}

func TestParseQwenAudioSpeechRejectsUnsupportedOrUnsafeInputs(t *testing.T) {
	tests := map[string]string{
		"stream":        `{"model":"qwen-tts","input":"hello","voice":"voice_1","stream":true}`,
		"format":        `{"model":"qwen-tts","input":"hello","voice":"voice_1","response_format":"wav"}`,
		"speed":         `{"model":"qwen-tts","input":"hello","voice":"voice_1","speed":2.1}`,
		"invalid voice": `{"model":"qwen-tts","input":"hello","voice":"../voice"}`,
		"unknown field": `{"model":"qwen-tts","input":"hello","voice":"voice_1","api_key":"no"}`,
	}
	for name, payload := range tests {
		t.Run(name, func(t *testing.T) {
			_, err := parseQwenAudioSpeechRequest(qwenAudioSpeechTestContext(t, payload))
			require.Error(t, err)
		})
	}
}

func TestParseQwenAudioSpeechUnicodeCodePointBoundary(t *testing.T) {
	for _, tc := range []struct {
		name       string
		codePoints int
		wantError  bool
	}{
		{name: "4000 accepted", codePoints: 4000},
		{name: "4001 rejected", codePoints: 4001, wantError: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			payload := `{"model":"qwen-tts","input":"` + strings.Repeat("语", tc.codePoints) + `","voice":"voice_1"}`
			_, err := parseQwenAudioSpeechRequest(qwenAudioSpeechTestContext(t, payload))
			if tc.wantError {
				require.ErrorContains(t, err, "4000 Unicode code points")
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestParseQwenAudioSpeechInstructionCodePointBoundary(t *testing.T) {
	for _, tc := range []struct {
		name       string
		codePoints int
		wantError  bool
	}{
		{name: "4000 accepted", codePoints: 4000},
		{name: "4001 rejected", codePoints: 4001, wantError: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			payload := `{"model":"qwen-tts","input":"hello","voice":"voice_1","instructions":"` + strings.Repeat("情", tc.codePoints) + `"}`
			_, err := parseQwenAudioSpeechRequest(qwenAudioSpeechTestContext(t, payload))
			if tc.wantError {
				require.ErrorContains(t, err, "instructions exceed 4000 Unicode code points")
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestParseQwenAudioTranscriptionValidatesWAVAndDuration(t *testing.T) {
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	require.NoError(t, writer.WriteField("model", "qwen-asr"))
	file, err := writer.CreateFormFile("file", "voice.wav")
	require.NoError(t, err)
	_, err = file.Write(handlerPCM16WAV(16000, 2))
	require.NoError(t, err)
	require.NoError(t, writer.Close())

	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("POST", "/v1/audio/transcriptions", &body)
	c.Request.Header.Set("Content-Type", writer.FormDataContentType())
	req, err := parseQwenAudioTranscriptionRequest(c)
	require.NoError(t, err)
	require.Equal(t, 2.0/3600.0, req.usageUnits)
	require.Equal(t, 16000, req.wavMeta.SampleRate)
}
