package service

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestForwardStepFunASRGeneratesSTTUsageFromSuccessfulResponse(t *testing.T) {
	gin.SetMode(gin.TestMode)
	upstream := &qwenAudioHTTPStub{
		status: http.StatusOK,
		body:   `{"text":"hello","duration":2}`,
	}
	svc := &OpenAIGatewayService{httpUpstream: upstream}
	account := &Account{
		ID: 42, Platform: PlatformStepFun, Type: AccountTypeAPIKey, Concurrency: 1,
		Credentials: map[string]any{
			"api_key":  "sk-stepfun-test",
			"base_url": "https://api.stepfun.test/v1",
		},
	}
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	body := []byte(strings.Repeat("audio", 100))
	result, err := svc.ForwardStepFunAudio(context.Background(), c, account, StepFunAudioRequest{
		Model:       "stepaudio-2.5-asr",
		Endpoint:    "/v1/audio/transcriptions",
		ContentType: "multipart/form-data; boundary=test",
		Body:        body,
	})

	require.NoError(t, err)
	require.Equal(t, `{"text":"hello","duration":2}`, recorder.Body.String())
	require.NotNil(t, result)
	require.NotNil(t, result.AudioUsage)
	require.Equal(t, "stt", result.AudioUsage.Mode)
	require.Equal(t, 2.0/3600.0, result.AudioUsage.DurationOrUnits)
	require.Equal(t, "/v1/audio/transcriptions", upstream.req.URL.Path)
}
