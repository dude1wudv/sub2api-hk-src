//go:build unit

package service

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
	"testing/synctest"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/tlsfingerprint"
	coderws "github.com/coder/websocket"
	"github.com/stretchr/testify/require"
)

type slowTTFTOutputRegressionUpstream struct {
	HTTPUpstream
	body io.ReadCloser
}

func (u *slowTTFTOutputRegressionUpstream) Do(*http.Request, string, int64, int) (*http.Response, error) {
	return &http.Response{StatusCode: http.StatusOK, Body: u.body}, nil
}

func (u *slowTTFTOutputRegressionUpstream) DoWithTLS(r *http.Request, proxy string, id int64, cap int, _ *tlsfingerprint.Profile) (*http.Response, error) {
	return u.Do(r, proxy, id, cap)
}

func TestSlowTTFTOutputRegression(t *testing.T) {
	t.Run("HTTP deltas distinguish output from metadata", func(t *testing.T) {
		cases := []struct {
			name  string
			first string
			slow  bool
		}{
			{"response reasoning summary", `{"type":"response.reasoning_summary_text.delta","delta":"thinking"}`, false},
			{"response reasoning text", `{"type":"response.reasoning_text.delta","delta":"thinking"}`, false},
			{"response custom tool input", `{"type":"response.custom_tool_call_input.delta","delta":"{}"}`, false},
			{"Anthropic thinking", `{"type":"content_block_delta","delta":{"thinking":"thinking"}}`, false},
			{"Chat Completions reasoning_content", `{"choices":[{"delta":{"reasoning_content":"thinking"}}]}`, false},
			{"Chat Completions reasoning", `{"choices":[{"delta":{"reasoning":"thinking"}}]}`, false},
			{"Gemini thought text", `{"candidates":[{"content":{"parts":[{"text":"thinking","thought":true}]}}]}`, false},
			{"reasoning summary done", `{"type":"response.reasoning_summary_text.done","text":"thinking"}`, false},
			{"reasoning text done", `{"type":"response.reasoning_text.done","text":"thinking"}`, false},
			{"custom tool input done", `{"type":"response.custom_tool_call_input.done","input":"{}"}`, false},
			{"custom tool call added", `{"type":"response.output_item.added","item":{"type":"custom_tool_call","name":"lookup"}}`, false},
			{"Anthropic thinking block start", `{"type":"content_block_start","content_block":{"type":"thinking","thinking":"thinking"}}`, false},
			{"empty delta", `{"type":"response.reasoning_summary_text.delta","delta":""}`, true},
			{"empty reasoning summary done", `{"type":"response.reasoning_summary_text.done","text":""}`, true},
			{"empty reasoning text done", `{"type":"response.reasoning_text.done","text":""}`, true},
			{"empty custom tool input done", `{"type":"response.custom_tool_call_input.done","input":""}`, true},
			{"custom tool call added without name", `{"type":"response.output_item.added","item":{"type":"custom_tool_call","name":""}}`, true},
			{"empty Anthropic thinking block start", `{"type":"content_block_start","content_block":{"type":"thinking","thinking":""}}`, true},
			{"created", `{"type":"response.created"}`, true},
			{"in progress", `{"type":"response.in_progress"}`, true},
			{"role only", `{"choices":[{"delta":{"role":"assistant"}}]}`, true},
			{"usage only", `{"choices":[],"usage":{"completion_tokens":1}}`, true},
			{"signature only", `{"type":"content_block_delta","delta":{"signature":"sig"}}`, true},
			{"image delta", `{"type":"response.image_generation_call.partial_image_b64","delta":"aW1hZ2U="}`, true},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				synctest.Test(t, func(t *testing.T) {
					s, repo, cache := slowTTFTTestService(true)
					reader, writer := io.Pipe()
					upstream := &slowTTFTOutputRegressionUpstream{body: reader}
					wrapped := WithSlowTTFTUpstream(upstream, s)
					req, err := http.NewRequest(http.MethodPost, "http://example.com/v1/responses", strings.NewReader(`{"stream":true}`))
					require.NoError(t, err)
					resp, err := wrapped.Do(req, "", repo.account.ID, 1)
					require.NoError(t, err)
					readDone := make(chan error, 1)
					go func() {
						_, readErr := io.Copy(io.Discard, resp.Body)
						if readErr == nil {
							readErr = resp.Body.Close()
						}
						readDone <- readErr
					}()

					time.Sleep(3 * time.Second)
					_, err = io.WriteString(writer, "data: "+tc.first+"\n\n")
					require.NoError(t, err)
					time.Sleep(17 * time.Second)
					_, err = io.WriteString(writer, "data: {\"type\":\"response.output_text.delta\",\"delta\":\"body\"}\n\n")
					require.NoError(t, err)
					require.NoError(t, writer.Close())
					require.NoError(t, <-readDone)

					attempts, slow := cache.snapshot()
					require.Len(t, attempts, 1)
					require.Equal(t, []bool{tc.slow}, slow)
				})
			})
		}
	})

	t.Run("WebSocket reasoning output stops slow timer", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			s, repo, cache := slowTTFTTestService(true)
			base := &slowTTFTFakeFrameConn{messages: [][]byte{
				[]byte(`{"type":"response.reasoning_summary_text.delta","delta":"thinking"}`),
				[]byte(`{"type":"response.output_text.delta","delta":"body"}`),
				[]byte(`{"type":"response.completed"}`),
			}}
			conn := &slowTTFTFrameConn{FrameConn: base, ctx: context.Background(), protection: s, accountID: repo.account.ID}
			require.NoError(t, conn.WriteFrame(context.Background(), coderws.MessageText, []byte(`{"type":"response.create"}`)))

			time.Sleep(3 * time.Second)
			_, _, err := conn.ReadFrame(context.Background())
			require.NoError(t, err)
			time.Sleep(17 * time.Second)
			_, _, err = conn.ReadFrame(context.Background())
			require.NoError(t, err)
			_, _, err = conn.ReadFrame(context.Background())
			require.NoError(t, err)

			attempts, slow := cache.snapshot()
			require.Len(t, attempts, 1)
			require.Equal(t, []bool{false}, slow)
		})
	})
}
