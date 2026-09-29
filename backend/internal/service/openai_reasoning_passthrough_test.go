package service

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

func TestAPIKeyResponsesPreservesReasoningEfforts(t *testing.T) {
	for _, passthrough := range []bool{false, true} {
		for _, model := range []string{"stealth/space-bunny-alpha", "deepseek-v4-pro", "public-alias"} {
			for _, effort := range []string{"none", "minimal", "low", "medium", "high", "xhigh", "max", "ultra"} {
				for _, stream := range []bool{false, true} {
					t.Run(fmt.Sprintf("%s/%s/stream=%t/passthrough=%t", model, effort, stream, passthrough), func(t *testing.T) {
						response := `{"id":"resp_test","status":"completed","output":[],"usage":{"input_tokens":1,"output_tokens":2}}`
						contentType := "application/json"
						if stream {
							response = "data: {\"type\":\"response.completed\",\"response\":" + response + "}\n\ndata: [DONE]\n\n"
							contentType = "text/event-stream"
						}
						upstream := &httpUpstreamRecorder{resp: &http.Response{StatusCode: http.StatusOK,
							Header: http.Header{"Content-Type": []string{contentType}}, Body: io.NopCloser(strings.NewReader(response))}}
						svc := &OpenAIGatewayService{cfg: &config.Config{}, httpUpstream: upstream}
						mapped := model
						if model == "public-alias" {
							mapped = "stealth/space-bunny-alpha"
						}
						account := rawGPT56ResponsesAPIKeyAccount(model, mapped)
						account.Extra["openai_passthrough"] = passthrough
						body, err := json.Marshal(map[string]any{"model": model, "stream": stream, "input": "hello", "reasoning": map[string]string{"effort": effort}})
						require.NoError(t, err)
						c, _ := gin.CreateTestContext(httptest.NewRecorder())
						c.Request = httptest.NewRequest(http.MethodPost, "/v1/responses", strings.NewReader(string(body)))
						SetOpenAIClientTransport(c, OpenAIClientTransportHTTP)
						result, err := svc.Forward(context.Background(), c, account, body)
						require.NoError(t, err)
						require.NotNil(t, result)
						wantModel := mapped
						if passthrough {
							// Raw passthrough keeps the client model as well as its effort.
							wantModel = model
						}
						require.Equal(t, wantModel, gjson.GetBytes(upstream.lastBody, "model").String())
						require.Equal(t, effort, gjson.GetBytes(upstream.lastBody, "reasoning.effort").String())
						require.NotNil(t, result.ReasoningEffort)
						require.Equal(t, effort, *result.ReasoningEffort)
						requested := CanonicalRequestedReasoningEffort(body, model)
						require.NotNil(t, requested)
						require.Equal(t, effort, *requested)
					})
				}
			}
		}
	}
}
