//go:build unit

package service

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/synctest"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	coderws "github.com/coder/websocket"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func chatSpeechTestWAV(seconds int) []byte {
	data := make([]byte, 44+32000*seconds)
	copy(data, "RIFF")
	binary.LittleEndian.PutUint32(data[4:8], uint32(len(data)-8))
	copy(data[8:], "WAVEfmt ")
	binary.LittleEndian.PutUint32(data[16:20], 16)
	binary.LittleEndian.PutUint16(data[20:22], 1)
	binary.LittleEndian.PutUint16(data[22:24], 1)
	binary.LittleEndian.PutUint32(data[24:28], 16000)
	binary.LittleEndian.PutUint32(data[28:32], 32000)
	binary.LittleEndian.PutUint16(data[32:34], 2)
	binary.LittleEndian.PutUint16(data[34:36], 16)
	copy(data[36:], "data")
	binary.LittleEndian.PutUint32(data[40:44], uint32(len(data)-44))
	return data
}

func TestChatSpeechValidation(t *testing.T) {
	wav := chatSpeechTestWAV(1)
	format, err := ValidateChatSpeechAudio(wav)
	require.NoError(t, err)
	require.Equal(t, "wav", format)
	for _, invalid := range [][]byte{wav[:len(wav)-1], chatSpeechTestWAV(121), []byte("not audio"), append(append([]byte{}, wav...), 0)} {
		_, err = ValidateChatSpeechAudio(invalid)
		require.Error(t, err)
	}
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("file", "speech.wav")
	require.NoError(t, err)
	_, err = part.Write(wav)
	require.NoError(t, err)
	require.NoError(t, writer.Close())
	request := httptest.NewRequest(http.MethodPost, "/", &body)
	request.Header.Set("Content-Type", writer.FormDataContentType())
	input, err := ParseChatAuxiliaryInput("asr", request)
	require.NoError(t, err)
	require.Equal(t, .026, input.ReserveCost)
	for _, raw := range []string{`{"text":"ok","model":"forged"}`, `{"text":"ok","voice_id":"unknown"}`, `{"text":""}`, `{"text":"ok"} {}`} {
		request := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(raw))
		request.Header.Set("Content-Type", "application/json")
		_, err := ParseChatAuxiliaryInput("tts", request)
		require.Error(t, err)
	}
	_, err = ValidateChatTTS(&ChatTTSInput{Text: strings.Repeat("汉", 5001)})
	require.Error(t, err)
}

func chatSpeechFixture(t *testing.T, kind string) (*OpenAIGatewayService, *APIKey, *Account, context.Context) {
	t.Helper()
	account := newDeepSeekToolsAccount()
	account.Credentials["base_url"] = "https://api.deepseek.com"
	key := &APIKey{ID: 2, UserID: 7, Status: StatusActive, Key: "synthetic", GroupID: i64p(11)}
	gateway := &OpenAIGatewayService{cfg: &config.Config{Chat: config.ChatConfig{AuxiliaryAccountID: account.ID}}, accountRepo: &deepSeekToolsAccountRepo{latest: account}}
	scope := &chatAuxiliaryScope{userID: 7, presetID: uuid.New(), sourceGroupID: 11, configuredAccountID: account.ID, operationID: uuid.New(), kind: kind}
	return gateway, key, account, context.WithValue(t.Context(), chatAuxiliaryContextKey{}, scope)
}

func TestChatSpeechASRUsesAuthoritativeDurationOnly(t *testing.T) {
	for _, raw := range []string{`{"request_id":"provider","text":"你好","duration_ms":3200}`, `{"request_id":"provider","text":"hi"}`, `{"request_id":"provider","duration_ms":120001}`} {
		gateway, key, account, ctx := chatSpeechFixture(t, "asr")
		upstream := &deepSeekToolsUpstream{response: &http.Response{StatusCode: 200, Header: http.Header{}, Body: io.NopCloser(strings.NewReader(raw))}}
		gateway.httpUpstream = upstream
		result, body, err := gateway.ForwardChatASR(ctx, key, account, &ChatAuxiliaryInput{Body: []byte("synthetic"), ContentType: "multipart/form-data; boundary=x"})
		require.NotNil(t, result)
		require.Equal(t, 1, upstream.calls)
		require.True(t, HTTPUpstreamRedirectsDisabled(upstream.request.Context()))
		require.Equal(t, "https://api.deepseek.com/v1/tools/asr", upstream.request.URL.String())
		cost, costErr := result.ChatAuxiliaryUsage.cost()
		require.NoError(t, costErr)
		if strings.Contains(raw, "3200") {
			require.NoError(t, err)
			require.Contains(t, string(body), "你好")
			require.Equal(t, .00086667, cost)
		} else {
			require.Error(t, err)
			require.Zero(t, cost)
			require.Nil(t, result.ChatAuxiliaryUsage.Items[0].Quantity)
		}
	}
}

type chatSpeechFrame struct {
	kind coderws.MessageType
	data []byte
}
type chatSpeechSocketFake struct {
	frames []chatSpeechFrame
	writes [][]byte
	closed bool
	cancel context.CancelFunc
}

func (s *chatSpeechSocketFake) Read(ctx context.Context) (coderws.MessageType, []byte, error) {
	if s.cancel != nil {
		s.cancel()
		return 0, nil, ctx.Err()
	}
	if len(s.frames) == 0 {
		return 0, nil, io.EOF
	}
	next := s.frames[0]
	s.frames = s.frames[1:]
	return next.kind, next.data, nil
}
func (s *chatSpeechSocketFake) Write(_ context.Context, _ coderws.MessageType, data []byte) error {
	s.writes = append(s.writes, append([]byte{}, data...))
	return nil
}
func (s *chatSpeechSocketFake) CloseNow() error { s.closed = true; return nil }

func TestChatSpeechTTSRequiresSuccessAndAttestsBytes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, finish := range []string{`{"event":"finish","code":"success"}`, `{"event":"finish","code":"service_error"}`, "", `{"event":"finish","code":"success","unexpected":true}`} {
		gateway, key, account, ctx := chatSpeechFixture(t, "tts")
		pcm := []byte{0, 1, 2, 3, 4, 5}
		socket := &chatSpeechSocketFake{frames: []chatSpeechFrame{{coderws.MessageBinary, pcm[:2]}, {coderws.MessageBinary, pcm[2:]}}}
		if finish != "" {
			socket.frames = append(socket.frames, chatSpeechFrame{coderws.MessageText, []byte(finish)})
		}
		calls := 0
		gateway.chatTTSDial = func(_ context.Context, target string, headers http.Header, proxy string) (chatSpeechSocket, error) {
			calls++
			require.Contains(t, target, "wss://api.deepseek.com/v1/tools/tts?")
			require.Contains(t, target, "format=pcm")
			require.Equal(t, "Bearer "+account.GetCredential("api_key"), headers.Get("Authorization"))
			return socket, nil
		}
		recorder := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(recorder)
		c.Request = httptest.NewRequest(http.MethodPost, "/", nil)
		result, err := gateway.ForwardChatTTS(ctx, c, key, account, &ChatAuxiliaryInput{TTS: &ChatTTSInput{Text: "你好A"}})
		require.NotNil(t, result)
		require.Equal(t, 1, calls)
		require.True(t, socket.closed)
		require.Len(t, socket.writes, 2)
		cost, costErr := result.ChatAuxiliaryUsage.cost()
		require.NoError(t, costErr)
		if strings.Contains(finish, `"success"`) {
			require.NoError(t, err)
			require.Equal(t, .0015, cost)
			digest := sha256.Sum256(pcm)
			require.Contains(t, recorder.Body.String(), hex.EncodeToString(digest[:]))
			require.Contains(t, recorder.Body.String(), "event: audio_complete")
			require.Contains(t, recorder.Body.String(), `"bytes":6`)
		} else {
			require.Error(t, err)
			require.Zero(t, cost)
			require.NotContains(t, recorder.Body.String(), "event: audio_complete")
			require.Contains(t, recorder.Body.String(), "event: error")
		}
	}
}

func TestChatSpeechTTSCancellationClosesWithoutReplay(t *testing.T) {
	gateway, key, account, base := chatSpeechFixture(t, "tts")
	ctx, cancel := context.WithCancel(base)
	defer cancel()
	socket := &chatSpeechSocketFake{cancel: cancel}
	calls := 0
	gateway.chatTTSDial = func(context.Context, string, http.Header, string) (chatSpeechSocket, error) {
		calls++
		return socket, nil
	}
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodPost, "/", nil)
	result, err := gateway.ForwardChatTTS(ctx, c, key, account, &ChatAuxiliaryInput{TTS: &ChatTTSInput{Text: "hello"}})
	require.Error(t, err)
	require.True(t, socket.closed)
	require.Equal(t, 1, calls)
	cost, err := result.ChatAuxiliaryUsage.cost()
	require.NoError(t, err)
	require.Zero(t, cost)
}

type chatSpeechReservationCache struct {
	*memInflightCache
	scopes map[string]time.Time
	lost   bool
}

func (c *chatSpeechReservationCache) ReserveInflightScope(_ context.Context, scope, id string, _ float64, ttl time.Duration) (float64, error) {
	c.scopes[scope+id] = time.Now().Add(ttl)
	return 0, nil
}
func (c *chatSpeechReservationCache) ReleaseInflightScope(_ context.Context, scope, id string) error {
	delete(c.scopes, scope+id)
	return nil
}
func (c *chatSpeechReservationCache) RenewInflightScope(_ context.Context, scope, id string, ttl time.Duration) (bool, error) {
	if c.lost {
		return false, errors.New("lost")
	}
	if !c.scopes[scope+id].After(time.Now()) {
		return false, nil
	}
	c.scopes[scope+id] = time.Now().Add(ttl)
	return true, nil
}
func TestChatSpeechReservationsSurviveThirtyMinutesAndFailClosedOnLoss(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		cache := &chatSpeechReservationCache{memInflightCache: newMemInflightCache(100), scopes: map[string]time.Time{}}
		svc := newInflightSvc(t, cache, 900)
		ctx, res, err := svc.ReserveChatAuxiliary(t.Context(), &User{ID: 7}, &APIKey{ID: 2, Quota: 100}, nil, nil, 3)
		require.NoError(t, err)
		require.NotNil(t, res)
		synctest.Wait()
		time.Sleep(30 * time.Minute)
		synctest.Wait()
		require.NoError(t, ctx.Err())
		require.Equal(t, 1, cache.count())
		for _, until := range cache.scopes {
			require.True(t, until.After(time.Now()))
		}
		cache.lost = true
		time.Sleep(5 * time.Minute)
		synctest.Wait()
		require.ErrorIs(t, context.Cause(ctx), ErrBillingServiceUnavailable)
		res.HandlerDone()
		require.Empty(t, cache.scopes)
		require.Zero(t, cache.count())
	})
}
