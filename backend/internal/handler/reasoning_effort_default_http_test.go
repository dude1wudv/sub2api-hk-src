//go:build unit

package handler

import (
	"bytes"
	"io"
	"net/http"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/tlsfingerprint"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

type reasoningDefaultCapturedUpstream struct{ *astraProCapturedUpstream }

func (u *reasoningDefaultCapturedUpstream) DoWithTLS(req *http.Request, proxy string, accountID int64, concurrency int, _ *tlsfingerprint.Profile) (*http.Response, error) {
	return u.Do(req, proxy, accountID, concurrency)
}

func TestReasoningEffortDefaultHTTPUpstreamJSON(t *testing.T) {
	gin.SetMode(gin.TestMode)
	const configJSON = `{"enabled":true,"rules":[{"platform":"openai","match_type":"exact","model":"gpt-6.1-sol","effort":"high"},{"platform":"anthropic","match_type":"exact","model":"claude-opus-4-5","effort":"high"}]}`
	for _, tt := range []struct {
		name, protocol, body, effort, source string
		openAI                               bool
		maxEffort                            string
	}{
		{"OpenAI Responses", "responses", `{"model":"gpt-6.1-sol","input":"hello","stream":false}`, "high", service.ReasoningEffortSourceDefault, true, ""},
		{"OpenAI Chat", "chat", `{"model":"gpt-6.1-sol","messages":[{"role":"user","content":"hello"}],"stream":false}`, "high", service.ReasoningEffortSourceDefault, true, ""},
		{"OpenAI Messages matched", "messages", `{"model":"gpt-6.1-sol","max_tokens":100,"messages":[{"role":"user","content":"hello"}],"stream":false}`, "high", service.ReasoningEffortSourceDefault, true, ""},
		{"OpenAI Messages unmatched keeps medium", "messages", `{"model":"gpt-5.4","max_tokens":100,"messages":[{"role":"user","content":"hello"}],"stream":false}`, "medium", "", true, ""},
		{"OpenAI Chat explicit nested", "chat", `{"model":"gpt-6.1-sol","messages":[{"role":"user","content":"hello"}],"stream":false,"reasoning":{"effort":"low","summary":"detailed"}}`, "low", service.ReasoningEffortSourceExplicit, true, ""},
		{"Anthropic Messages", "messages", `{"model":"claude-opus-4-5","max_tokens":100,"messages":[{"role":"user","content":"hello"}],"stream":false}`, "high", service.ReasoningEffortSourceDefault, false, ""},
		{"Anthropic Chat", "chat", `{"model":"claude-opus-4-5","messages":[{"role":"user","content":"hello"}],"stream":false}`, "high", service.ReasoningEffortSourceDefault, false, ""},
		{"Anthropic Responses", "responses", `{"model":"claude-opus-4-5","input":"hello","stream":false}`, "high", service.ReasoningEffortSourceDefault, false, ""},
		{"OpenAI Responses default capped despite deny", "responses", `{"model":"gpt-6.1-sol","input":"hello","stream":false}`, "medium", service.ReasoningEffortSourceDefault, true, "medium"},
		{"OpenAI Messages default capped despite deny", "messages", `{"model":"gpt-6.1-sol","max_tokens":100,"messages":[{"role":"user","content":"hello"}],"stream":false}`, "medium", service.ReasoningEffortSourceDefault, true, "medium"},
		{"Anthropic Messages default capped despite deny", "messages", `{"model":"claude-opus-4-5","max_tokens":100,"messages":[{"role":"user","content":"hello"}],"stream":false}`, "medium", service.ReasoningEffortSourceDefault, false, "medium"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			settings := service.NewSettingService(&contentModerationHandlerSettingRepo{values: map[string]string{service.SettingKeyGatewayReasoningEffortDefault: configJSON}}, &config.Config{})
			responseBody := `{"id":"msg_default","type":"message","role":"assistant","model":"claude-opus-4-5","content":[{"type":"text","text":"ok"}],"stop_reason":"end_turn","usage":{"input_tokens":1,"output_tokens":1}}`
			contentType := "application/json"
			if tt.openAI {
				responseBody = `{"id":"resp_default","object":"response","model":"gpt-6.1-sol","status":"completed","output":[{"type":"message","role":"assistant","content":[{"type":"output_text","text":"ok"}]}],"usage":{"input_tokens":1,"output_tokens":1}}`
				if tt.protocol != "responses" {
					responseBody = "data: {\"type\":\"response.completed\",\"response\":" + responseBody + "}\n\n"
					contentType = "text/event-stream"
				}
			}
			if !tt.openAI && tt.protocol != "messages" {
				responseBody = smartRoutingToolSSE
				contentType = "text/event-stream"
			}
			upstream := &reasoningDefaultCapturedUpstream{newAstraProCapturedUpstream(&http.Response{StatusCode: http.StatusOK, Header: http.Header{"Content-Type": []string{contentType}}, Body: io.NopCloser(bytes.NewBufferString(responseBody))})}
			c, rec := newAstraProFailoverContext(t, tt.body)
			apiKey, ok := middleware2.GetAPIKeyFromContext(c)
			require.True(t, ok)
			apiKey.Group.AllowMessagesDispatch = true
			apiKey.Group.Hydrated = true
			apiKey.Group.Status = service.StatusActive
			apiKey.Group.MaxReasoningEffort = tt.maxEffort
			apiKey.Group.MaxReasoningEffortOverLimit = service.ReasoningEffortOverLimitDeny
			if tt.openAI {
				h := newOpenAIResponsesFailoverTestHandler(t, upstream)
				h.settingService = settings
				switch tt.protocol {
				case "responses":
					h.Responses(c)
				case "chat":
					h.ChatCompletions(c)
				case "messages":
					h.Messages(c)
				}
			} else {
				apiKey.Group.Platform = service.PlatformAnthropic
				cfg := &config.Config{RunMode: config.RunModeSimple}
				account := &service.Account{ID: 41, Platform: service.PlatformAnthropic, Type: service.AccountTypeAPIKey, Status: service.StatusActive, Schedulable: true, GroupIDs: []int64{apiKey.Group.ID}, Credentials: map[string]any{"api_key": "synthetic-key", "base_url": "https://api.anthropic.com"}}
				snapshot := service.NewSchedulerSnapshotService(&fakeSchedulerCache{accounts: []*service.Account{account}}, nil, nil, nil, nil)
				gateway := service.NewGatewayService(
					nil, &fakeGroupRepo{group: apiKey.Group}, nil, nil, nil, nil, nil, nil, cfg,
					snapshot, nil, nil, nil, nil, nil, upstream, nil, nil, nil, nil, nil, settings, nil, nil, nil, nil, nil, nil,
				)
				billing := service.NewBillingCacheService(nil, nil, nil, nil, nil, nil, cfg, nil)
				t.Cleanup(billing.Stop)
				h := NewGatewayHandler(gateway, nil, nil, nil, nil, service.NewConcurrencyService(&fakeConcurrencyCache{}), billing, nil, service.NewAPIKeyService(nil, nil, nil, nil, nil, nil, cfg), nil, nil, nil, nil, cfg, settings)
				switch tt.protocol {
				case "responses":
					h.Responses(c)
				case "chat":
					h.ChatCompletions(c)
				case "messages":
					h.Messages(c)
				}
			}
			require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
			_, _, bodies := upstream.snapshot()
			require.Len(t, bodies, 1)
			path := "output_config.effort"
			if tt.openAI {
				path = "reasoning.effort"
			}
			require.Equal(t, tt.effort, gjson.GetBytes(bodies[0], path).String(), "upstream JSON: %s", bodies[0])
			require.Equal(t, tt.source, service.ReasoningEffortSourceFromContext(c.Request.Context()))
			if tt.source == service.ReasoningEffortSourceDefault {
				require.Nil(t, service.RequestedReasoningEffortFromContext(c.Request.Context()))
			}
			if !tt.openAI && tt.protocol == "messages" {
				require.False(t, gjson.GetBytes(bodies[0], "thinking").Exists(), "must not invent thinking: %s", bodies[0])
			}
			if tt.name == "OpenAI Chat explicit nested" {
				require.Equal(t, "detailed", gjson.GetBytes(bodies[0], "reasoning.summary").String())
			}
		})
	}
}
