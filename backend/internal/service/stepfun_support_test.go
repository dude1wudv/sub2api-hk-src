package service

import (
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestStepFunProtocolBaseURLsFollowAccountMode(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name          string
		mode          string
		wantOpenAI    string
		wantAnthropic string
	}{
		{
			name:          "payg",
			mode:          AccountModePayG,
			wantOpenAI:    DefaultStepFunPayGBaseURL,
			wantAnthropic: DefaultStepFunPayGAnthropicBaseURL,
		},
		{
			name:          "step plan",
			mode:          AccountModeStepPlan,
			wantOpenAI:    DefaultStepFunPlanBaseURL,
			wantAnthropic: DefaultStepFunPlanAnthropicBaseURL,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			account := &Account{
				Platform: PlatformStepFun,
				Type:     AccountTypeAPIKey,
				Credentials: map[string]any{
					"api_key":      "sk-stepfun-test",
					"account_mode": tc.mode,
					"api_protocol": APIProtocolAdaptive,
				},
			}

			require.True(t, account.IsStepFun())
			require.Equal(t, tc.wantOpenAI, account.GetOpenAIBaseURL())
			require.Equal(t, tc.wantOpenAI, account.GetCNProtocolBaseURL(APIProtocolChatCompletions))
			require.Equal(t, tc.wantOpenAI, account.GetCNProtocolBaseURL(APIProtocolResponses))
			require.Equal(t, tc.wantOpenAI, account.GetOpenAIFormatBaseURL())
			require.Equal(t, tc.wantAnthropic, account.GetCNProtocolBaseURL(APIProtocolAnthropic))
			require.Equal(t, tc.wantAnthropic, account.GetAnthropicProtocolBaseURL())
			require.NotContains(t, tc.wantAnthropic, "/v1")
		})
	}
}

func TestStepFunSupportsNativeResponsesAndCompositeModelRouting(t *testing.T) {
	t.Parallel()

	account := &Account{
		Platform: PlatformStepFun,
		Type:     AccountTypeAPIKey,
		Credentials: map[string]any{
			"api_key":      "sk-stepfun-test",
			"api_protocol": APIProtocolResponses,
		},
	}
	require.Equal(t, APIProtocolResponses, account.GetAPIProtocol())
	require.True(t, account.SupportsNativeCNResponses())
	require.True(t, account.UsesNativeCNResponses())

	for _, model := range []string{"step-3.5-flash", "step-image-edit-2", "stepaudio-2.5-asr", "stepaudio-2.5-realtime"} {
		platform, ok := DetectModelPlatform(model)
		require.True(t, ok, "model=%s", model)
		require.Equal(t, PlatformStepFun, platform, "model=%s", model)
	}
}

func TestStepFunImagesAcceptStepImageEditModel(t *testing.T) {
	t.Parallel()
	gin.SetMode(gin.TestMode)

	for _, path := range []string{"/v1/images/generations", "/v1/images/edits"} {
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Request = httptest.NewRequest("POST", path, nil)
		body := []byte(`{"model":"step-image-edit-2","prompt":"a cat"}`)
		if path == "/v1/images/edits" {
			body = []byte(`{"model":"step-image-edit-2","prompt":"a cat","images":[{"image_url":"https://example.com/input.png"}]}`)
		}
		req, err := (&OpenAIGatewayService{}).ParseOpenAIImagesRequest(c, body)
		require.NoError(t, err, "path=%s", path)
		require.Equal(t, "step-image-edit-2", req.Model)
	}
}
