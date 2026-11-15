//go:build unit

package service

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	coderws "github.com/coder/websocket"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

type suiteSpeechFrame struct {
	kind coderws.MessageType
	data []byte
}
type suiteSpeechSocket struct {
	frames chan suiteSpeechFrame
	done   chan struct{}
	once   sync.Once
}

func (s *suiteSpeechSocket) Read(ctx context.Context) (coderws.MessageType, []byte, error) {
	select {
	case frame := <-s.frames:
		return frame.kind, frame.data, nil
	case <-ctx.Done():
		return 0, nil, ctx.Err()
	case <-s.done:
		return 0, nil, context.Canceled
	}
}
func (s *suiteSpeechSocket) Write(_ context.Context, _ coderws.MessageType, raw []byte) error {
	if gjson.GetBytes(raw, "event").String() == "finish_text" {
		s.frames <- suiteSpeechFrame{coderws.MessageBinary, []byte{1, 2, 3, 4}}
		s.frames <- suiteSpeechFrame{coderws.MessageText, []byte(`{"event":"finish","code":"success","message":"ok"}`)}
	}
	return nil
}
func (s *suiteSpeechSocket) CloseNow() error {
	s.once.Do(func() { close(s.done) })
	return nil
}

func TestDeepSeekSuiteWebSocketNativeFramesAndMeter(t *testing.T) {
	account := suiteTestAccount()
	fake := &suiteSpeechSocket{frames: make(chan suiteSpeechFrame, 3), done: make(chan struct{})}
	type outcome struct {
		result *OpenAIForwardResult
		err    error
	}
	results := make(chan outcome, 1)
	headersSeen := make(chan http.Header, 1)
	svc := &OpenAIGatewayService{accountRepo: &deepSeekToolsAccountRepo{latest: account}}
	svc.chatTTSDial = func(_ context.Context, target string, headers http.Header, _ string) (chatSpeechSocket, error) {
		if target != "wss://tools.example/v1/tools/tts?format=pcm&voice_id=tide" {
			return nil, ErrChatInvalid
		}
		headersSeen <- headers.Clone()
		return fake, nil
	}
	router := gin.New()
	router.GET("/v1/tools/tts", func(c *gin.Context) {
		result, err := svc.ForwardDeepSeekTTSWebSocket(c.Request.Context(), c, suiteTestKey(), account, &DeepSeekSuiteOperation{ID: uuid.NewString()}, func(text string) bool { return text == "你好a😀" })
		results <- outcome{result, err}
	})
	server := httptest.NewServer(router)
	defer server.Close()
	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancel()
	client, _, err := coderws.Dial(ctx, strings.Replace(server.URL, "http:", "ws:", 1)+"/v1/tools/tts?voice_id=tide&format=pcm",
		&coderws.DialOptions{Subprotocols: []string{"tts"}, HTTPHeader: http.Header{"Authorization": {"Bearer synthetic-gateway-key"}}})
	require.NoError(t, err)
	defer func() { _ = client.CloseNow() }()
	require.Equal(t, "tts", client.Subprotocol())
	require.NoError(t, client.Write(ctx, coderws.MessageText, []byte(`{"event":"text","text":"你好a😀"}`)))
	require.NoError(t, client.Write(ctx, coderws.MessageText, []byte(`{"event":"finish_text"}`)))
	kind, audio, err := client.Read(ctx)
	require.NoError(t, err)
	require.Equal(t, coderws.MessageBinary, kind)
	require.Equal(t, []byte{1, 2, 3, 4}, audio)
	kind, finish, err := client.Read(ctx)
	require.NoError(t, err)
	require.Equal(t, coderws.MessageText, kind)
	require.Equal(t, "success", gjson.GetBytes(finish, "code").String())
	_ = client.Close(coderws.StatusNormalClosure, "done")
	select {
	case out := <-results:
		require.NoError(t, out.err)
		require.NotNil(t, out.result)
		require.Equal(t, "confirmed", out.result.ChatAuxiliaryUsage.Items[0].MeteringStatus)
		amount, err := out.result.ChatAuxiliaryUsage.cost()
		require.NoError(t, err)
		require.InDelta(t, .0018, amount, 1e-10)
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	}
	headers := <-headersSeen
	require.Equal(t, "Bearer synthetic-upstream-key", headers.Get("Authorization"))
	require.Empty(t, headers.Get("Sec-WebSocket-Protocol"))
}
