package service

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/tlsfingerprint"
	coderws "github.com/coder/websocket"
	"github.com/stretchr/testify/require"
)

type qwenAudioHTTPStub struct {
	status int
	body   string
	req    *http.Request
	data   []byte
}

func (s *qwenAudioHTTPStub) Do(req *http.Request, _ string, _ int64, _ int) (*http.Response, error) {
	s.req = req
	s.data, _ = io.ReadAll(req.Body)
	return &http.Response{
		StatusCode: s.status, Header: make(http.Header),
		Body: io.NopCloser(strings.NewReader(s.body)),
	}, nil
}

func (s *qwenAudioHTTPStub) DoWithTLS(req *http.Request, proxy string, id int64, concurrency int, _ *tlsfingerprint.Profile) (*http.Response, error) {
	return s.Do(req, proxy, id, concurrency)
}

type qwenAudioWSDialerStub struct {
	conn    *qwenAudioWSConnStub
	url     string
	headers http.Header
}

func (d *qwenAudioWSDialerStub) Dial(_ context.Context, target string, headers http.Header, _ string) (openAIWSClientConn, int, http.Header, error) {
	d.url = target
	d.headers = cloneHeader(headers)
	return d.conn, 0, nil, nil
}

type qwenAudioWSConnStub struct {
	writes    []map[string]any
	taskID    string
	readIndex int
	chars     int
}

func (c *qwenAudioWSConnStub) WriteJSON(_ context.Context, value any) error {
	raw, _ := json.Marshal(value)
	var decoded map[string]any
	_ = json.Unmarshal(raw, &decoded)
	c.writes = append(c.writes, decoded)
	header, _ := decoded["header"].(map[string]any)
	if c.taskID == "" {
		c.taskID, _ = header["task_id"].(string)
	}
	return nil
}

func (c *qwenAudioWSConnStub) ReadMessage(ctx context.Context) ([]byte, error) {
	_, data, err := c.ReadFrame(ctx)
	return data, err
}

func (c *qwenAudioWSConnStub) ReadFrame(_ context.Context) (coderws.MessageType, []byte, error) {
	c.readIndex++
	switch c.readIndex {
	case 1:
		return coderws.MessageText, []byte(`{"header":{"task_id":"` + c.taskID + `","event":"task-started"},"payload":{}}`), nil
	case 2:
		return coderws.MessageText, []byte(`{"header":{"task_id":"` + c.taskID + `","event":"result-generated"},"payload":{}}`), nil
	case 3:
		return coderws.MessageBinary, []byte{0xff, 0xfb, 0x90, 0x64}, nil
	default:
		return coderws.MessageText, []byte(`{"header":{"task_id":"` + c.taskID + `","event":"task-finished"},"payload":{"usage":{"characters":` + jsonNumber(c.chars) + `}}}`), nil
	}
}

func (c *qwenAudioWSConnStub) Ping(context.Context) error { return nil }
func (c *qwenAudioWSConnStub) Close() error               { return nil }

func jsonNumber(value int) string {
	raw, _ := json.Marshal(value)
	return string(raw)
}

func qwenAudioTestAccount() *Account {
	return &Account{
		ID: 9, Platform: PlatformOpenAI, Type: AccountTypeAPIKey, Concurrency: 1,
		Credentials: map[string]any{
			"api_key": "test-secret",
			"model_mapping": map[string]any{
				"asr-public": "qwen-audio-3.0-asr-flash",
				"tts-public": "qwen-audio-3.0-tts-plus",
			},
		},
		Extra: map[string]any{
			QwenAudioHTTPBaseURLExtraKey: "https://workspace.cn-beijing.maas.aliyuncs.com/api/v1",
			QwenAudioWSBaseURLExtraKey:   "wss://workspace.cn-beijing.maas.aliyuncs.com/api-ws/v1",
		},
	}
}

func makePCM16WAV(sampleRate, channels, seconds int) []byte {
	dataSize := sampleRate * channels * 2 * seconds
	out := make([]byte, 44+dataSize)
	copy(out[0:4], "RIFF")
	binary.LittleEndian.PutUint32(out[4:8], uint32(len(out)-8))
	copy(out[8:12], "WAVE")
	copy(out[12:16], "fmt ")
	binary.LittleEndian.PutUint32(out[16:20], 16)
	binary.LittleEndian.PutUint16(out[20:22], 1)
	binary.LittleEndian.PutUint16(out[22:24], uint16(channels))
	binary.LittleEndian.PutUint32(out[24:28], uint32(sampleRate))
	binary.LittleEndian.PutUint32(out[28:32], uint32(sampleRate*channels*2))
	binary.LittleEndian.PutUint16(out[32:34], uint16(channels*2))
	binary.LittleEndian.PutUint16(out[34:36], 16)
	copy(out[36:40], "data")
	binary.LittleEndian.PutUint32(out[40:44], uint32(dataSize))
	return out
}

func TestParsePCM16WAVComputesExactDurationAndRejectsCorruption(t *testing.T) {
	wav := makePCM16WAV(16000, 1, 2)
	meta, err := ParsePCM16WAV(wav)
	require.NoError(t, err)
	require.Equal(t, 16000, meta.SampleRate)
	require.Equal(t, 2*time.Second, meta.Duration)

	bad := append([]byte(nil), wav...)
	binary.LittleEndian.PutUint32(bad[28:32], 1)
	_, err = ParsePCM16WAV(bad)
	require.ErrorContains(t, err, "inconsistent")
}

func TestQwenAudioEndpointRequiresAliyunTLSAndJoinsAPIRoot(t *testing.T) {
	account := qwenAudioTestAccount()
	httpURL, err := qwenAudioEndpoint(account, QwenAudioHTTPBaseURLExtraKey, "https", qwenASRPath)
	require.NoError(t, err)
	require.Equal(t, "https://workspace.cn-beijing.maas.aliyuncs.com"+qwenASRPath, httpURL)
	wsURL, err := qwenAudioEndpoint(account, QwenAudioWSBaseURLExtraKey, "wss", qwenTTSPath)
	require.NoError(t, err)
	require.Equal(t, "wss://workspace.cn-beijing.maas.aliyuncs.com"+qwenTTSPath, wsURL)

	account.Extra[QwenAudioHTTPBaseURLExtraKey] = "https://127.0.0.1/api/v1"
	_, err = qwenAudioEndpoint(account, QwenAudioHTTPBaseURLExtraKey, "https", qwenASRPath)
	require.ErrorContains(t, err, "workspace Maas")

	for _, disallowed := range []string{
		"https://bucket.oss-cn-beijing.aliyuncs.com/api/v1",
		"https://dashscope.aliyuncs.com/api/v1",
		"https://workspace.us-west-1.maas.aliyuncs.com/api/v1",
	} {
		account.Extra[QwenAudioHTTPBaseURLExtraKey] = disallowed
		_, err = qwenAudioEndpoint(account, QwenAudioHTTPBaseURLExtraKey, "https", qwenASRPath)
		require.ErrorContains(t, err, "workspace Maas", "url=%s", disallowed)
	}
}

func TestForwardQwenASRBuildsDataURLAndParsesSupportedResponseShapes(t *testing.T) {
	for name, response := range map[string]string{
		"official": `{"request_id":"req-1","output":{"text":"hello"}}`,
		"nested":   `{"request_id":"req-2","output":{"output":{"sentence":{"text":"nested"}}}}`,
	} {
		t.Run(name, func(t *testing.T) {
			upstream := &qwenAudioHTTPStub{status: http.StatusOK, body: response}
			svc := &OpenAIGatewayService{httpUpstream: upstream}
			wav := makePCM16WAV(16000, 1, 1)
			result, err := svc.ForwardQwenASR(context.Background(), qwenAudioTestAccount(), "asr-public", wav, WAVMetadata{})
			require.NoError(t, err)
			require.NotEmpty(t, result.Text)
			require.Equal(t, "Bearer test-secret", upstream.req.Header.Get("Authorization"))
			require.Equal(t, "disable", upstream.req.Header.Get("X-DashScope-SSE"))
			var request map[string]any
			require.NoError(t, json.Unmarshal(upstream.data, &request))
			require.Equal(t, "qwen-audio-3.0-asr-flash", request["model"])
			require.Contains(t, string(upstream.data), `data:audio/wav;base64,`)
			require.Contains(t, string(upstream.data), `"sample_rate":"16000"`)
		})
	}
}

func TestForwardQwenTTSUsesOneTaskAndFinalProviderCharacterUsage(t *testing.T) {
	conn := &qwenAudioWSConnStub{chars: 37}
	dialer := &qwenAudioWSDialerStub{conn: conn}
	svc := &OpenAIGatewayService{openaiWSPassthroughDialer: dialer}
	instruction := strings.Repeat("情", 150)
	result, err := svc.ForwardQwenTTS(context.Background(), qwenAudioTestAccount(), QwenTTSRequest{
		Model: "tts-public", Input: "中文a", Voice: "longanhuan_v3.6", Speed: 1.25, Instructions: instruction,
	})
	require.NoError(t, err)
	require.Equal(t, 37, result.BilledCharacters, "billing must use task-finished usage, not input rune count")
	require.True(t, bytes.HasPrefix(result.Audio, []byte{0xff, 0xfb}))
	require.Len(t, conn.writes, 3)

	var taskID string
	for index, write := range conn.writes {
		header, ok := write["header"].(map[string]any)
		require.True(t, ok)
		if index == 0 {
			taskID, ok = header["task_id"].(string)
			require.True(t, ok)
		}
		require.Equal(t, taskID, header["task_id"])
	}
	header0, ok := conn.writes[0]["header"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "run-task", header0["action"])
	header1, ok := conn.writes[1]["header"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "continue-task", header1["action"])
	header2, ok := conn.writes[2]["header"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "finish-task", header2["action"])
	payload0, ok := conn.writes[0]["payload"].(map[string]any)
	require.True(t, ok)
	parameters, ok := payload0["parameters"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, instruction, parameters["instruction"], "Qwen instructions must not inherit CosyVoice's old 100-character limit")
	require.Equal(t, 1.25, parameters["rate"])
	require.Equal(t, "Bearer test-secret", dialer.headers.Get("Authorization"))
}

func TestQwenAudioStatusClassificationAndMappingGate(t *testing.T) {
	account := qwenAudioTestAccount()
	model, err := ResolveQwenAudioModel(account, "tts-public")
	require.NoError(t, err)
	require.Equal(t, "qwen-audio-3.0-tts-plus", model)
	_, err = ResolveQwenAudioModel(account, "unknown")
	require.Error(t, err)

	var failover *UpstreamFailoverError
	require.ErrorAs(t, qwenAudioStatusError(http.StatusBadRequest, nil), &failover)
	require.False(t, failover.ShouldRetryNextAccount())
	require.NotContains(t, failover.Error(), "test-secret")
	require.ErrorAs(t, qwenAudioStatusError(http.StatusTooManyRequests, nil), &failover)
	require.True(t, failover.ShouldRetryNextAccount())
}

func TestForwardQwenAudioFailureClassificationAndCancellation(t *testing.T) {
	wav := makePCM16WAV(16000, 1, 1)
	for _, tc := range []struct {
		status    int
		retryable bool
	}{
		{status: http.StatusBadRequest, retryable: false},
		{status: http.StatusTooManyRequests, retryable: true},
		{status: http.StatusServiceUnavailable, retryable: true},
	} {
		upstream := &qwenAudioHTTPStub{status: tc.status, body: `{"message":"must not escape"}`}
		svc := &OpenAIGatewayService{httpUpstream: upstream}
		_, err := svc.ForwardQwenASR(context.Background(), qwenAudioTestAccount(), "asr-public", wav, WAVMetadata{})
		var failover *UpstreamFailoverError
		require.ErrorAs(t, err, &failover)
		require.Equal(t, tc.retryable, failover.ShouldRetryNextAccount())
		require.Empty(t, failover.ResponseBody, "upstream body may contain request content and must stay out of downstream errors/logs")
	}

	canceled, cancel := context.WithCancel(context.Background())
	cancel()
	svc := &OpenAIGatewayService{httpUpstream: &qwenAudioHTTPStub{status: http.StatusOK}}
	_, err := svc.ForwardQwenASR(canceled, qwenAudioTestAccount(), "asr-public", wav, WAVMetadata{})
	require.ErrorIs(t, err, context.Canceled)
}

func TestForwardQwenTTSRejectsMissingFinalUsageAndCapabilityFailsClosed(t *testing.T) {
	conn := &qwenAudioWSConnStub{chars: 0}
	dialer := &qwenAudioWSDialerStub{conn: conn}
	svc := &OpenAIGatewayService{openaiWSPassthroughDialer: dialer}
	_, err := svc.ForwardQwenTTS(context.Background(), qwenAudioTestAccount(), QwenTTSRequest{
		Model: "tts-public", Input: "hello", Voice: "voice_1", Speed: 1,
	})
	var failover *UpstreamFailoverError
	require.ErrorAs(t, err, &failover)
	require.True(t, failover.ShouldRetryNextAccount())

	account := qwenAudioTestAccount()
	require.True(t, account.SupportsOpenAIEndpointCapability(OpenAIEndpointCapabilityQwenAudio))
	account.Extra[QwenAudioWSBaseURLExtraKey] = "ws://127.0.0.1/private"
	require.False(t, account.SupportsOpenAIEndpointCapability(OpenAIEndpointCapabilityQwenAudio))
}

func TestForwardQwenTTSSeparatesProductInputAndProviderUsageLimits(t *testing.T) {
	tooLongConn := &qwenAudioWSConnStub{chars: 1}
	tooLongService := &OpenAIGatewayService{openaiWSPassthroughDialer: &qwenAudioWSDialerStub{conn: tooLongConn}}
	_, err := tooLongService.ForwardQwenTTS(context.Background(), qwenAudioTestAccount(), QwenTTSRequest{
		Model: "tts-public", Input: strings.Repeat("语", QwenAudioMaxTTSCharacters+1), Voice: "voice_1", Speed: 1,
	})
	require.ErrorContains(t, err, "input is invalid")
	require.Empty(t, tooLongConn.writes, "product input limit must reject before opening the provider task")

	for _, tc := range []struct {
		name      string
		usage     int
		wantError bool
	}{
		{name: "provider cumulative maximum accepted", usage: qwenAudioProviderMaxTaskCharacters},
		{name: "provider cumulative maximum exceeded", usage: qwenAudioProviderMaxTaskCharacters + 1, wantError: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			conn := &qwenAudioWSConnStub{chars: tc.usage}
			svc := &OpenAIGatewayService{openaiWSPassthroughDialer: &qwenAudioWSDialerStub{conn: conn}}
			result, err := svc.ForwardQwenTTS(context.Background(), qwenAudioTestAccount(), QwenTTSRequest{
				Model: "tts-public", Input: "short product input", Voice: "voice_1", Speed: 1,
			})
			if tc.wantError {
				var failover *UpstreamFailoverError
				require.ErrorAs(t, err, &failover)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.usage, result.BilledCharacters)
		})
	}
}

func TestQwenAudioCapabilityRequiresMatchedWorkspacePairAndSupportedMapping(t *testing.T) {
	account := qwenAudioTestAccount()
	require.True(t, account.SupportsOpenAIEndpointCapability(OpenAIEndpointCapabilityQwenAudio))

	account.Extra[QwenAudioWSBaseURLExtraKey] = "wss://other-workspace.cn-beijing.maas.aliyuncs.com/api-ws/v1"
	require.False(t, account.SupportsOpenAIEndpointCapability(OpenAIEndpointCapabilityQwenAudio), "cross-workspace pair must fail closed")

	account = qwenAudioTestAccount()
	account.Extra[QwenAudioWSBaseURLExtraKey] = "wss://workspace.ap-southeast-1.maas.aliyuncs.com/api-ws/v1"
	require.False(t, account.SupportsOpenAIEndpointCapability(OpenAIEndpointCapabilityQwenAudio), "cross-region pair must fail closed")

	account = qwenAudioTestAccount()
	account.Credentials["model_mapping"] = map[string]any{"unrelated": "gpt-4.1"}
	require.False(t, account.SupportsOpenAIEndpointCapability(OpenAIEndpointCapabilityQwenAudio), "unrelated non-empty mapping must not enable Qwen audio")
}

func TestForwardQwenTTSInstructionProductBoundary(t *testing.T) {
	for _, tc := range []struct {
		name         string
		instructions string
		wantError    bool
	}{
		{name: "4000 accepted", instructions: strings.Repeat("情", QwenAudioMaxInstructionCharacters)},
		{name: "4001 rejected", instructions: strings.Repeat("情", QwenAudioMaxInstructionCharacters+1), wantError: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			conn := &qwenAudioWSConnStub{chars: 5}
			svc := &OpenAIGatewayService{openaiWSPassthroughDialer: &qwenAudioWSDialerStub{conn: conn}}
			_, err := svc.ForwardQwenTTS(context.Background(), qwenAudioTestAccount(), QwenTTSRequest{
				Model: "tts-public", Input: "hello", Voice: "voice_1", Speed: 1, Instructions: tc.instructions,
			})
			if tc.wantError {
				require.ErrorContains(t, err, "parameters are invalid")
				require.Empty(t, conn.writes)
				return
			}
			require.NoError(t, err)
			require.NotEmpty(t, conn.writes)
		})
	}
}
