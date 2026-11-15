//go:build unit

package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/ctxkey"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

func suiteTestAccount() *Account {
	account := newDeepSeekToolsAccount()
	account.Extra[DeepSeekSuiteCapabilitiesKey] = []any{"chat", "responses", "messages", "files", "ocr", "asr", "tts", "tts_ws"}
	return account
}

func suiteTestKey() *APIKey {
	return &APIKey{ID: 2, UserID: 7, User: &User{ID: 7}, GroupID: i64p(223), Group: &Group{ID: 223, Platform: PlatformOpenAI, RateMultiplier: .35}}
}

func TestDeepSeekSuiteExplicitCapabilityAndScope(t *testing.T) {
	account := suiteTestAccount()
	require.True(t, account.DeepSeekSuiteAllowed(i64p(223), "files"))
	require.False(t, account.DeepSeekSuiteAllowed(i64p(81), "files"))
	require.False(t, account.DeepSeekSuiteAllowed(i64p(223), "unknown"))
	account.Name = "全家桶"
	delete(account.Extra, DeepSeekSuiteCapabilitiesKey)
	require.False(t, account.DeepSeekSuiteAllowed(i64p(223), "files"), "display names are not authorization")
	require.True(t, account.DeepSeekSuiteAllowed(i64p(223), "search"), "existing explicit search permission stays valid")
	account = suiteTestAccount()
	require.True(t, deepSeekSuiteAccountAllowed(WithDeepSeekSuiteScope(t.Context(), "files", 223, account.ID), account))
	require.False(t, deepSeekSuiteAccountAllowed(WithDeepSeekSuiteScope(t.Context(), "files", 223, 99), account))
	svc := &OpenAIGatewayService{accountRepo: &deepSeekToolsAccountRepo{latest: account}}
	account.GroupIDs = []int64{81}
	_, err := svc.DeepSeekSuiteAccount(t.Context(), suiteTestKey(), account, "files")
	require.Error(t, err, "real group membership is checked after waiting")
}

func TestDeepSeekSuiteURLs(t *testing.T) {
	svc := &OpenAIGatewayService{}
	account := suiteTestAccount()
	for _, tc := range []struct{ base, kind, want string }{
		{"https://kedaya.ai/v1", "files", "https://kedaya.ai/v1/files"},
		{"https://vip.mdkj.lol", "tts", "https://vip.mdkj.lol/v1/tools/tts"},
		{"https://relay.example/prefix/v1?tenant=a", "messages", "https://relay.example/prefix/v1/messages?tenant=a"},
		{"https://api.deepseek.com/v1", "files", "https://api.deepseek.com/files"},
		{"https://api.deepseek.com/v1", "messages", "https://api.deepseek.com/anthropic/v1/messages"},
	} {
		account.Credentials["base_url"] = tc.base
		got, err := svc.deepSeekSuiteURL(account, tc.kind, "")
		require.NoError(t, err)
		require.Equal(t, tc.want, got)
	}
	account.Credentials["base_url"] = "http://127.0.0.1/private"
	_, err := svc.deepSeekSuiteURL(account, "files", "")
	require.Error(t, err)
}

func TestDeepSeekSuiteAudioAndStrictJSON(t *testing.T) {
	for _, tc := range []struct {
		body string
		ok   bool
	}{
		{`{"model":"deepseek-tts","input":"你好a😀","voice":"stella","response_format":"pcm"}`, true},
		{`{"model":"deepseek-tts","input":"hi","voice":"alloy"}`, false},
		{`{"model":"deepseek-tts","input":"hi","speed":1.5}`, false},
		{`{"model":"deepseek-tts","input":"hi","response_format":"mp3"}`, false},
		{`{"model":"deepseek-tts","input":"short","input":"unmetered duplicate"}`, false},
	} {
		req := httptest.NewRequest("POST", "/v1/audio/speech", strings.NewReader(tc.body))
		req.Header.Set("Content-Type", "application/json")
		input, err := ParseDeepSeekSuiteInput(req, "tts", true)
		if !tc.ok {
			require.Error(t, err, tc.body)
			continue
		}
		require.NoError(t, err)
		require.Equal(t, int64(6), input.TTSCharacters)
		require.InDelta(t, .0018, input.ReserveCost, 1e-10)
		require.Equal(t, "你好a😀", gjson.GetBytes(input.Body, "text").String())
		require.False(t, gjson.GetBytes(input.Body, "model").Exists())
	}
	require.False(t, validDeepSeekSuiteJSON([]byte(`{"a":{"file_id":"owned","file_id":"foreign"}}`)))
	require.True(t, validDeepSeekSuiteJSON([]byte(`{"future":{"big":9007199254740993}}`)))
}

type suiteMemoryRepo struct {
	*openAIRecordUsageLogRepoStub
	DeepSeekSuiteRepository
	records map[string]*DeepSeekSuiteOperation
}

func newSuiteMemoryRepo() *suiteMemoryRepo {
	return &suiteMemoryRepo{openAIRecordUsageLogRepoStub: &openAIRecordUsageLogRepoStub{inserted: true}, records: map[string]*DeepSeekSuiteOperation{}}
}

func (r *suiteMemoryRepo) CreateDeepSeekOperation(_ context.Context, op *DeepSeekSuiteOperation) error {
	clone := *op
	r.records[op.ID] = &clone
	return nil
}
func (r *suiteMemoryRepo) SaveDeepSeekOperation(ctx context.Context, op *DeepSeekSuiteOperation) error {
	return r.CreateDeepSeekOperation(ctx, op)
}
func (r *suiteMemoryRepo) GetDeepSeekFile(_ context.Context, userID, groupID int64, id string) (*DeepSeekSuiteOperation, error) {
	op := r.records[id]
	if op == nil || op.UserID != userID || op.GroupID != groupID || !DeepSeekSuiteFileVisible(op) {
		return nil, ErrChatNotFound
	}
	return op, nil
}

func TestDeepSeekSuiteFilesOwnershipAffinityAndOpaqueFields(t *testing.T) {
	repo := newSuiteMemoryRepo()
	id := uuid.NewString()
	repo.records[id] = &DeepSeekSuiteOperation{ID: id, UserID: 7, GroupID: 223, AccountID: 1112, RemoteID: "file-provider-owned", Status: "awaiting_meter"}
	svc := &OpenAIGatewayService{usageLogRepo: repo}
	body := []byte(`{"model":"deepseek-flash","input":[{"type":"input_file","file_id":"file-ds-` + id + `"}],"future":{"large":9007199254740993,"signature":"keep-me"}}`)
	got, accountID, err := svc.ResolveDeepSeekFiles(t.Context(), suiteTestKey(), body)
	require.NoError(t, err)
	require.EqualValues(t, 1112, accountID)
	require.Equal(t, "file-provider-owned", gjson.GetBytes(got, "input.0.file_id").String())
	require.Contains(t, string(got), "9007199254740993")
	require.Contains(t, string(got), "keep-me")
	key := suiteTestKey()
	key.UserID = 8
	_, _, err = svc.ResolveDeepSeekFiles(t.Context(), key, body)
	require.Error(t, err)
	key.UserID, *key.GroupID = 7, 81
	_, _, err = svc.ResolveDeepSeekFiles(t.Context(), key, body)
	require.Error(t, err)
	_, _, err = svc.ResolveDeepSeekFiles(t.Context(), suiteTestKey(), []byte(`{"input":[{"file_id":"file-provider-owned"}]}`))
	require.Error(t, err)
	second := uuid.NewString()
	repo.records[second] = &DeepSeekSuiteOperation{ID: second, UserID: 7, GroupID: 223, AccountID: 1182, RemoteID: "file-other-account"}
	both := []byte(`{"input":[{"file_id":"file-ds-` + id + `"},{"file_id":"file-ds-` + second + `"}]}`)
	_, _, err = svc.ResolveDeepSeekFiles(t.Context(), suiteTestKey(), both)
	require.ErrorContains(t, err, "different upstream accounts")
	repo.records[id].Metadata = json.RawMessage(`{"deleted":true}`)
	_, _, err = svc.ResolveDeepSeekFiles(t.Context(), suiteTestKey(), body)
	require.Error(t, err)
}

func TestDeepSeekSuiteFileMultipartOrderAndUnknownMeter(t *testing.T) {
	var wire bytes.Buffer
	writer := multipart.NewWriter(&wire)
	part, err := writer.CreateFormFile("file", "report.pdf")
	require.NoError(t, err)
	_, _ = io.WriteString(part, "%PDF-1.7\nsynthetic")
	require.NoError(t, writer.WriteField("purpose", "text_recognition"))
	require.NoError(t, writer.Close())
	req := httptest.NewRequest("POST", "/v1/files", &wire)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	input, err := ParseDeepSeekSuiteInput(req, "files", false)
	require.NoError(t, err)
	defer func() { _ = req.MultipartForm.RemoveAll() }()
	account := suiteTestAccount()
	repo := newSuiteMemoryRepo()
	upstream := &deepSeekToolsUpstream{response: &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": {"application/json"}},
		Body: io.NopCloser(strings.NewReader(`{"id":"file-provider-1","object":"file","filename":"provider-name","bytes":18}`))}}
	svc := &OpenAIGatewayService{cfg: &config.Config{}, accountRepo: &deepSeekToolsAccountRepo{latest: account}, usageLogRepo: repo, httpUpstream: upstream}
	op, err := svc.NewDeepSeekOperation(t.Context(), suiteTestKey(), account, "files")
	require.NoError(t, err)
	c, _ := newDeepSeekToolsContext(nil)
	result, raw, err := svc.ForwardDeepSeekSuite(t.Context(), c, suiteTestKey(), account, input, op)
	require.NoError(t, err)
	require.Nil(t, result.ChatAuxiliaryUsage.Items[0].Quantity)
	require.Equal(t, "unreported", result.ChatAuxiliaryUsage.Items[0].MeteringStatus)
	require.Equal(t, op.FileID(), gjson.GetBytes(raw, "id").String())
	require.Equal(t, "report.pdf", gjson.GetBytes(raw, "filename").String())
	require.Equal(t, "file-provider-1", op.RemoteID)
	require.Less(t, bytes.Index(upstream.body, []byte(`name="purpose"`)), bytes.Index(upstream.body, []byte(`name="file"`)))
	require.Contains(t, string(upstream.body), op.UpstreamName)
}

func TestDeepSeekSuiteNativeProtocolPreservesToolFieldsAndFinalUsage(t *testing.T) {
	account := suiteTestAccount()
	for _, kind := range []string{"chat", "responses", "messages"} {
		t.Run(kind, func(t *testing.T) {
			raw := `{"model":"deepseek-flash","tools":[{"type":"deepseek_web_search"}],"safety_check":{"level":"high"},"title":{"enabled":true},"future":{"big":9007199254740993},"messages":[{"role":"latest_reminder","content":"keep"}]}`
			upstream := &deepSeekToolsUpstream{response: &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": {"application/json"}},
				Body: io.NopCloser(strings.NewReader(`{"model":"deepseek-flash","kind":"native","signature":"opaque","citations":[{"url":"https://example.com"}],"server_tool_calls":[{"id":"s"}],"usage":{"prompt_tokens":20,"completion_tokens":10,"server_tool_use":{"tool_search":2,"tool_fetch":1,"tool_image_search":0},"safety_tokens":4}}`))}}
			svc := &OpenAIGatewayService{cfg: &config.Config{}, accountRepo: &deepSeekToolsAccountRepo{latest: account}, httpUpstream: upstream}
			c, _ := newDeepSeekToolsContext(nil)
			result, payload, err := svc.ForwardDeepSeekSuite(t.Context(), c, suiteTestKey(), account, &DeepSeekSuiteInput{Kind: kind, Conversation: true, Model: "deepseek-flash", Body: []byte(raw)}, &DeepSeekSuiteOperation{ID: uuid.NewString()})
			require.NoError(t, err)
			require.Contains(t, string(upstream.body), "9007199254740993")
			require.Equal(t, "high", gjson.GetBytes(upstream.body, "safety_check.level").String())
			require.Equal(t, "opaque", gjson.GetBytes(payload, "signature").String())
			require.Equal(t, "Bearer synthetic-upstream-key", upstream.request.Header.Get("Authorization"))
			cost, err := result.ChatAuxiliaryUsage.cost()
			require.NoError(t, err)
			require.InDelta(t, .015, cost, 1e-10)
		})
	}
	result := &OpenAIForwardResult{}
	for _, n := range []string{"1", "2", "2"} {
		observeDeepSeekConversation([]byte(`{"usage":{"prompt_tokens":5,"completion_tokens":8,"safety_tokens":4,"server_tool_use":{"tool_search":`+n+`}}}`), "chat", result, &ClaudeUsage{})
	}
	amount, err := result.ChatAuxiliaryUsage.cost()
	require.NoError(t, err)
	require.Equal(t, .01, amount)
	require.Equal(t, 8, result.Usage.OutputTokens, "safety tokens are not charged twice")
}

func TestDeepSeekSuitePendingAndReceiptReplay(t *testing.T) {
	logs := newSuiteMemoryRepo()
	bills := &openAIRecordUsageBillingRepoStub{result: &UsageBillingApplyResult{Applied: true}}
	svc := newOpenAIRecordUsageServiceWithBillingRepoForTest(logs.openAIRecordUsageLogRepoStub, bills, &openAIRecordUsageUserRepoStub{}, &openAIRecordUsageSubRepoStub{}, nil)
	svc.usageLogRepo = logs
	op := &DeepSeekSuiteOperation{ID: uuid.NewString(), UserID: 7, KeyID: 2, GroupID: 223, AccountID: 1112, Kind: "ocr"}
	result := suiteResult(op, &DeepSeekSuiteInput{Kind: "ocr"})
	record := &OpenAIRecordUsageInput{Result: result, APIKey: suiteTestKey(), User: &User{ID: 7}, Account: suiteTestAccount()}
	ctx := context.WithValue(t.Context(), ctxkey.ClientRequestID, "same-client-id")
	require.NoError(t, svc.CommitDeepSeekSuite(ctx, op, record))
	require.Equal(t, "awaiting_meter", op.Status)
	require.Zero(t, bills.calls, "unknown pages must not finalize zero billing")
	count := int64(2)
	result.ChatAuxiliaryUsage, _ = chatAuxiliaryUsage("ocr", &count, "confirmed")
	bills.err = errors.New("synthetic billing outage")
	require.Error(t, svc.CommitDeepSeekSuite(t.Context(), op, record))
	require.Equal(t, "settling", logs.records[op.ID].Status)
	snapshot := append([]byte{}, op.Receipt...)
	require.Contains(t, string(snapshot), "deepseek_suite:"+op.ID)
	bills.err = nil
	record.APIKey.Group.RateMultiplier = 999
	require.NoError(t, svc.ApplyChatBillingReceipt(t.Context(), snapshot, nil, func(context.Context) error { return nil }))
	require.InDelta(t, .003, bills.lastCmd.BalanceCost, 1e-10)
	require.NotContains(t, string(snapshot), "synthetic-upstream-key")
	require.Equal(t, float64(1), logs.lastLog.RateMultiplier)
}

func TestDeepSeekSuiteAudioCompletionAndMeter(t *testing.T) {
	for _, tt := range []struct {
		name, kind, body, contentType string
		contentLength                 int64
		chunked, wantError, confirmed bool
		quantity                      int64
	}{
		{"complete audio", "tts", "pcm", "audio/pcm", 3, false, false, true, 6},
		{"complete chunked audio", "tts", "pcm", "audio/pcm", -1, true, false, true, 6},
		{"empty audio", "tts", "", "audio/pcm", 0, false, true, false, 0},
		{"truncated audio", "tts", "pcm", "audio/pcm", 4, false, true, false, 0},
		{"unframed audio", "tts", "pcm", "audio/pcm", -1, false, true, false, 0},
		{"rounded ASR", "asr", `{"duration_ms":1001}`, "application/json", -1, false, false, true, 2},
		{"missing ASR meter", "asr", `{"text":"ok"}`, "application/json", -1, false, false, false, 0},
		{"string ASR meter", "asr", `{"duration_ms":"1001"}`, "application/json", -1, false, true, false, 0},
		{"fractional ASR meter", "asr", `{"duration_ms":1001.5}`, "application/json", -1, false, true, false, 0},
		{"null ASR meter", "asr", `{"duration_ms":null}`, "application/json", -1, false, true, false, 0},
		{"negative ASR meter", "asr", `{"duration_ms":-1}`, "application/json", -1, false, true, false, 0},
	} {
		t.Run(tt.name, func(t *testing.T) {
			account := suiteTestAccount()
			response := &http.Response{StatusCode: 200, ProtoMajor: 1, ContentLength: tt.contentLength,
				Header: http.Header{"Content-Type": {tt.contentType}}, Body: io.NopCloser(strings.NewReader(tt.body))}
			if tt.chunked {
				response.TransferEncoding = []string{"chunked"}
			}
			upstream := &deepSeekToolsUpstream{response: response}
			svc := &OpenAIGatewayService{cfg: &config.Config{}, accountRepo: &deepSeekToolsAccountRepo{latest: account}, httpUpstream: upstream}
			c, _ := newDeepSeekToolsContext(nil)
			result, _, err := svc.ForwardDeepSeekSuite(t.Context(), c, suiteTestKey(), account,
				&DeepSeekSuiteInput{Kind: tt.kind, Body: []byte(`{}`), TTSCharacters: 6}, &DeepSeekSuiteOperation{ID: uuid.NewString()})
			if tt.wantError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			require.NotNil(t, result)
			item := result.ChatAuxiliaryUsage.Items[0]
			if tt.confirmed {
				require.Equal(t, "confirmed", item.MeteringStatus)
				require.Equal(t, tt.quantity, *item.Quantity)
			} else {
				require.NotEqual(t, "confirmed", item.MeteringStatus)
				amount, costErr := result.ChatAuxiliaryUsage.cost()
				require.NoError(t, costErr)
				require.Zero(t, amount)
			}
		})
	}
}
