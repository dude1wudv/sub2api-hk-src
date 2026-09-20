package handler

import (
	"bytes"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func newStepFunAudioTestContext(rec *httptest.ResponseRecorder, path, contentType string, body []byte) *gin.Context {
	req := httptest.NewRequest(http.MethodPost, path, bytes.NewReader(body))
	req.Header.Set("Content-Type", contentType)
	ctx, _ := gin.CreateTestContext(rec)
	ctx.Request = req
	return ctx
}

func TestParseStepFunAudioRequestValidatesNativeModelsAndEndpoints(t *testing.T) {
	t.Parallel()

	t.Run("speech", func(t *testing.T) {
		body := []byte(`{"model":"stepaudio-2.5-tts","input":"你好，StepFun"}`)
		ctx := newStepFunAudioTestContext(httptest.NewRecorder(), "/v1/audio/speech", "application/json", body)

		req, err := parseStepFunAudioRequest(ctx, body)
		require.NoError(t, err)
		require.Equal(t, "/v1/audio/speech", req.Endpoint)
		require.Equal(t, "stepaudio-2.5-tts", req.Model)
		require.NotNil(t, req.AudioUsage)
		require.Equal(t, "tts", req.AudioUsage.Mode)
		require.Positive(t, req.AudioUsage.DurationOrUnits)
	})

	t.Run("transcriptions", func(t *testing.T) {
		var body bytes.Buffer
		writer := multipart.NewWriter(&body)
		require.NoError(t, writer.WriteField("model", "stepaudio-2.5-asr"))
		part, err := writer.CreateFormFile("file", "sample.wav")
		require.NoError(t, err)
		_, err = part.Write([]byte("wav bytes"))
		require.NoError(t, err)
		require.NoError(t, writer.Close())

		ctx := newStepFunAudioTestContext(httptest.NewRecorder(), "/v1/audio/transcriptions", writer.FormDataContentType(), body.Bytes())
		req, err := parseStepFunAudioRequest(ctx, body.Bytes())
		require.NoError(t, err)
		require.Equal(t, "/v1/audio/transcriptions", req.Endpoint)
		require.Equal(t, "stepaudio-2.5-asr", req.Model)
		require.Nil(t, req.AudioUsage)
		require.NotEmpty(t, req.BodyHash)
	})

	for _, tc := range []struct {
		name        string
		path        string
		contentType string
		body        string
		wantMessage string
	}{
		{
			name:        "speech model",
			path:        "/v1/audio/speech",
			contentType: "application/json",
			body:        `{"model":"stepaudio-2.5-asr","input":"hello"}`,
			wantMessage: "model must be stepaudio-2.5-tts",
		},
		{
			name:        "transcription model",
			path:        "/v1/audio/transcriptions",
			contentType: "application/json",
			body:        `{"model":"stepaudio-2.5-tts"}`,
			wantMessage: "transcriptions requires multipart/form-data",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx := newStepFunAudioTestContext(httptest.NewRecorder(), tc.path, tc.contentType, []byte(tc.body))
			_, err := parseStepFunAudioRequest(ctx, []byte(tc.body))
			require.EqualError(t, err, tc.wantMessage)
		})
	}
}
