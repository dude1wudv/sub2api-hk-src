package service

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	coderws "github.com/coder/websocket"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

type reasoningDefaultNativeConn struct{ *stagedPassthroughConn }

func (c *reasoningDefaultNativeConn) WriteJSON(ctx context.Context, value any) error {
	payload, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return c.WriteFrame(ctx, coderws.MessageText, payload)
}

func TestReasoningEffortDefaultWSMultiTurn(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, mode := range []string{"ingress", "passthrough"} {
		for _, policy := range []struct {
			name, effort string
			mappings     []ReasoningEffortMapping
		}{
			{name: "ceiling-deny-default-downgraded", effort: "medium"},
			{name: "model-scoped-mapping", effort: "low", mappings: []ReasoningEffortMapping{{From: "high", To: "low", MatchType: "exact", Model: "gpt-6.1-sol"}}},
			{name: "model-scoped-deny-default-removed", effort: "", mappings: []ReasoningEffortMapping{{From: "high", To: ReasoningEffortMappingDeny, MatchType: "exact", Model: "gpt-6.1-sol"}}},
		} {
			t.Run(mode+"/"+policy.name, func(t *testing.T) {
				ctx, cancel := context.WithCancelCause(context.Background())
				defer cancel(context.Canceled)
				upstream := newStagedPassthroughConn()
				cfg := passthroughLifecycleConfig()
				cfg.Gateway.OpenAIWS.ReadTimeoutSeconds = 5
				cfg.Gateway.OpenAIWS.IngressInterTurnIdleTimeoutSeconds = 5
				account := passthroughLifecycleAccount()
				svc := newPassthroughLifecycleService(cfg, upstream)
				if mode == "ingress" {
					cfg.Gateway.OpenAIWS.ModeRouterV2Enabled = false
					cfg.Gateway.OpenAIWS.MaxConnsPerAccount = 1
					cfg.Gateway.OpenAIWS.MaxIdlePerAccount = 1
					account.Extra = map[string]any{"responses_websockets_v2_enabled": true}
					pool := newOpenAIWSConnPool(cfg)
					pool.setClientDialerForTest(&stagedPassthroughDialer{conn: &reasoningDefaultNativeConn{upstream}})
					svc.openaiWSPool = pool
				}
				results := make(chan *OpenAIForwardResult, 3)
				hooks := &OpenAIWSIngressHooks{
					ReasoningEffortDefault:      ReasoningEffortDefaultConfig{Enabled: true, Rules: []ReasoningEffortDefaultRule{{Platform: PlatformOpenAI, MatchType: "prefix", Model: "gpt-6.1-sol", Effort: "high"}}},
					ReasoningEffortPlatform:     PlatformOpenAI,
					MaxReasoningEffort:          "medium",
					MaxReasoningEffortOverLimit: ReasoningEffortOverLimitDeny,
					ReasoningEffortMappings:     policy.mappings,
					AfterTurn: func(_ int, result *OpenAIForwardResult, err error) {
						if err == nil {
							results <- result
						}
					},
				}
				server, serverErrors := startPassthroughHookRecordingServer(t, ctx, svc, account, hooks)
				defer server.Close()
				client := dialPassthroughLifecycleClientWithPayload(t, server, `{"type":"response.create","model":"gpt-6.1-sol","stream":false,"input":[]}`)
				defer func() { _ = client.CloseNow() }()
				frames := []string{"", `{"type":"response.create","model":"gpt-6.1-sol","stream":false,"input":[],"reasoning":{"effort":"low"}}`, `{"type":"response.create","stream":false,"input":[]}`}
				for turn, frame := range frames {
					if turn > 0 {
						writeCtx, cancelWrite := context.WithTimeout(ctx, 3*time.Second)
						err := client.Write(writeCtx, coderws.MessageText, []byte(frame))
						cancelWrite()
						require.NoError(t, err)
					}
					payload := requirePassthroughUpstreamWrite(t, upstream, 3*time.Second)
					expected, source := policy.effort, ReasoningEffortSourceDefault
					if turn == 1 {
						expected, source = "low", ReasoningEffortSourceExplicit
					}
					require.Equal(t, expected, gjson.GetBytes(payload, "reasoning.effort").String(), "turn %d upstream JSON: %s", turn+1, payload)
					upstream.Send(fmt.Sprintf(`{"type":"response.completed","response":{"id":"resp_reasoning_%d","model":"gpt-6.1-sol","usage":{"input_tokens":1,"output_tokens":1}}}`, turn+1))
					event, err := readPassthroughLifecycleFrame(t, client, 3*time.Second)
					require.NoError(t, err)
					require.Equal(t, "response.completed", gjson.GetBytes(event, "type").String())
					select {
					case result := <-results:
						require.Equal(t, source, optionalStringValue(result.ReasoningEffortSource))
						require.Equal(t, expected, optionalStringValue(result.ReasoningEffort))
						if source == ReasoningEffortSourceDefault {
							require.Nil(t, result.RequestedReasoningEffort)
							require.Nil(t, coalesceRequestedReasoningEffortWithSource(result.RequestedReasoningEffort, result.ReasoningEffort, result.ReasoningEffortSource))
						} else {
							require.Equal(t, "low", optionalStringValue(result.RequestedReasoningEffort))
						}
					case <-time.After(3 * time.Second):
						t.Fatal("missing turn result")
					}
				}
				require.NoError(t, client.Close(coderws.StatusNormalClosure, "done"))
				select {
				case err := <-serverErrors:
					require.NoError(t, err)
				case <-time.After(3 * time.Second):
					t.Fatal("WS relay did not terminate")
				}
			})
		}
	}
}
