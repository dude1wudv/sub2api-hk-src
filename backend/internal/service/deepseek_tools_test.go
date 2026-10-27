package service

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type deepSeekToolsUpstream struct {
	HTTPUpstream
	calls    int
	request  *http.Request
	body     []byte
	proxyURL string
	response *http.Response
	err      error
}

func (upstream *deepSeekToolsUpstream) Do(request *http.Request, proxyURL string, _ int64, _ int) (*http.Response, error) {
	upstream.calls++
	upstream.request = request
	upstream.body, _ = io.ReadAll(request.Body)
	upstream.proxyURL = proxyURL
	return upstream.response, upstream.err
}

func newDeepSeekToolsAccount() *Account {
	return &Account{
		ID: 1112, Platform: PlatformOpenAI, Type: AccountTypeAPIKey, Status: StatusActive,
		Concurrency: 2, Schedulable: true, GroupIDs: []int64{223, 81},
		Credentials: map[string]any{"base_url": "https://tools.example/v1", "api_key": "synthetic-upstream-key"},
		Extra:       map[string]any{"deepseek_tools_group_ids": []any{float64(223)}},
	}
}

func newDeepSeekToolsContext(body []byte) (*gin.Context, *httptest.ResponseRecorder) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/tools/search", bytes.NewReader(body))
	c.Request.Header.Set("Authorization", "Bearer synthetic-client-key")
	c.Request.Header.Set("X-Client-Request-ID", "reused-client-id")
	groupID := int64(223)
	c.Set("api_key", &APIKey{GroupID: &groupID})
	return c, recorder
}

func TestValidateDeepSeekToolRequest(t *testing.T) {
	for _, test := range []struct {
		name string
		kind DeepSeekToolKind
		body string
		want string
	}{
		{"minimal search", DeepSeekToolSearch, `{"query":"DeepSeek"}`, "DeepSeek"},
		{"all fields and unknown", DeepSeekToolSearch, `{"query":"模型 site:example.com","count":50,"ori_question":"原始问题","localization":"","need_content":true,"future_field":{"keep":true}}`, "模型 site:example.com\n原始问题"},
		{"Unicode query boundary", DeepSeekToolSearch, `{"query":"` + strings.Repeat("界", 128) + `"}`, strings.Repeat("界", 128)},
		{"null original question", DeepSeekToolSearch, `{"query":"q","ori_question":null}`, "q"},
		{"fetch", DeepSeekToolFetch, `{"url":"https://example.com/article?q=中文","future_field":true}`, "https://example.com/article?q=中文"},
		{"fetch HTTP", DeepSeekToolFetch, `{"url":"http://example.com/"}`, "http://example.com/"},
	} {
		t.Run(test.name, func(t *testing.T) {
			text, err := ValidateDeepSeekToolRequest(test.kind, []byte(test.body))
			require.NoError(t, err)
			require.Equal(t, test.want, text)
		})
	}
	for _, test := range []struct {
		name string
		kind DeepSeekToolKind
		body string
	}{
		{"empty body", DeepSeekToolSearch, ``},
		{"malformed JSON", DeepSeekToolSearch, `{"query":`},
		{"array", DeepSeekToolSearch, `[]`},
		{"null", DeepSeekToolSearch, `null`},
		{"missing query", DeepSeekToolSearch, `{}`},
		{"query type", DeepSeekToolSearch, `{"query":true}`},
		{"blank query", DeepSeekToolSearch, `{"query":" \t "}`},
		{"query too long", DeepSeekToolSearch, `{"query":"` + strings.Repeat("界", 129) + `"}`},
		{"zero count", DeepSeekToolSearch, `{"query":"q","count":0}`},
		{"count too large", DeepSeekToolSearch, `{"query":"q","count":51}`},
		{"fraction count", DeepSeekToolSearch, `{"query":"q","count":1.2}`},
		{"count string", DeepSeekToolSearch, `{"query":"q","count":"1"}`},
		{"null count", DeepSeekToolSearch, `{"query":"q","count":null}`},
		{"null bool", DeepSeekToolSearch, `{"query":"q","need_content":null}`},
		{"bool type", DeepSeekToolSearch, `{"query":"q","need_content":1}`},
		{"localization type", DeepSeekToolSearch, `{"query":"q","localization":null}`},
		{"empty original question", DeepSeekToolSearch, `{"query":"q","ori_question":""}`},
		{"original question type", DeepSeekToolSearch, `{"query":"q","ori_question":2}`},
		{"original question too long", DeepSeekToolSearch, `{"query":"q","ori_question":"` + strings.Repeat("界", 129) + `"}`},
		{"missing URL", DeepSeekToolFetch, `{}`},
		{"URL type", DeepSeekToolFetch, `{"url":false}`},
		{"URL scheme", DeepSeekToolFetch, `{"url":"file:///etc/hosts"}`},
		{"URL host", DeepSeekToolFetch, `{"url":"https:/article"}`},
		{"URL user info", DeepSeekToolFetch, `{"url":"https://user:password@example.com/"}`},
		{"URL too long", DeepSeekToolFetch, `{"url":"https://example.com/` + strings.Repeat("a", 8192) + `"}`},
		{"unknown kind", DeepSeekToolKind("other"), `{"query":"q"}`},
	} {
		t.Run(test.name, func(t *testing.T) {
			_, err := ValidateDeepSeekToolRequest(test.kind, []byte(test.body))
			require.Error(t, err)
		})
	}
}

func TestForwardDeepSeekToolPreservesNativeWireAndProxy(t *testing.T) {
	body := []byte(` {"query":"模型 site:example.com","count":2,"need_content":true,"future_field":{"keep":true}} `)
	responseBody := `{"results":[{"title":"Result","url":"https://example.com/","summary":"Summary","content":"# Page","site_name":null,"authority":3,"rerank_score":2.5}],"future_field":{"keep":true}}`
	upstream := &deepSeekToolsUpstream{response: &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": {"application/json"}, "X-Request-Id": {"upstream-request"}},
		Body:       io.NopCloser(strings.NewReader(responseBody)),
	}}
	svc := &OpenAIGatewayService{cfg: &config.Config{}, httpUpstream: upstream}
	account := newDeepSeekToolsAccount()
	proxyID := int64(7)
	account.ProxyID = &proxyID
	account.Proxy = &Proxy{Protocol: "http", Host: "127.0.0.1", Port: 7890}
	account.Credentials[credKeyHeaderOverrideEnabled] = true
	account.Credentials[credKeyHeaderOverrides] = map[string]any{"x-tool-route": "native", "accept": "text/event-stream"}
	c, recorder := newDeepSeekToolsContext(body)
	result, err := svc.ForwardDeepSeekTool(context.Background(), c, account, 223, DeepSeekToolSearch, body)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, 1, upstream.calls)
	require.Equal(t, "https://tools.example/v1/tools/search", upstream.request.URL.String())
	require.Equal(t, http.MethodPost, upstream.request.Method)
	require.Equal(t, body, upstream.body)
	require.Equal(t, "http://127.0.0.1:7890", upstream.proxyURL)
	require.Equal(t, "Bearer synthetic-upstream-key", upstream.request.Header.Get("Authorization"))
	require.Equal(t, "application/json", upstream.request.Header.Get("Content-Type"))
	require.Equal(t, "application/json", upstream.request.Header.Get("Accept"))
	require.Equal(t, "native", getHeaderRaw(upstream.request.Header, "x-tool-route"))
	acceptHeaders := 0
	for name, values := range upstream.request.Header {
		if strings.EqualFold(name, "Accept") {
			acceptHeaders++
			require.Equal(t, []string{"application/json"}, values)
		}
	}
	require.Equal(t, 1, acceptHeaders, "native JSON calls must not also send the account's SSE Accept override")
	require.Empty(t, upstream.request.Header.Get("Originator"))
	require.Empty(t, upstream.request.Header.Get("OpenAI-Beta"))
	require.Equal(t, responseBody, recorder.Body.String())
	require.Equal(t, "deepseek-search", result.Model)
	require.Empty(t, result.UpstreamModel)
	require.Equal(t, 1, result.BillableToolCalls)
	require.Equal(t, DeepSeekToolSearch, result.DeepSeekToolKind)
	require.Equal(t, DeepSeekToolSearch.Endpoint(), result.UpstreamEndpoint)
	require.Equal(t, DeepSeekToolSearch.Endpoint(), GetActualOpenAIUpstreamEndpoint(c))
	require.True(t, strings.HasPrefix(result.RequestID, "deepseek_tools:"))
	require.NotEqual(t, "reused-client-id", result.RequestID)
}

func TestForwardDeepSeekToolBillableSuccessAndEmptyResponses(t *testing.T) {
	for _, test := range []struct {
		name  string
		kind  DeepSeekToolKind
		body  string
		calls int
	}{
		{"search", DeepSeekToolSearch, `{"results":[{"title":"Result","url":"https://example.com/"}]}`, 1},
		{"search empty", DeepSeekToolSearch, `{"results":[]}`, 0},
		{"fetch", DeepSeekToolFetch, `{"title":"Article","content":"# Article"}`, 1},
		{"fetch empty", DeepSeekToolFetch, `{"title":"","content":" \n "}`, 0},
	} {
		t.Run(test.name, func(t *testing.T) {
			upstream := &deepSeekToolsUpstream{response: &http.Response{StatusCode: 200, Header: http.Header{}, Body: io.NopCloser(strings.NewReader(test.body))}}
			svc := &OpenAIGatewayService{cfg: &config.Config{}, httpUpstream: upstream}
			requestBody := []byte(`{"query":"q"}`)
			if test.kind == DeepSeekToolFetch {
				requestBody = []byte(`{"url":"https://example.com/"}`)
			}
			c, recorder := newDeepSeekToolsContext(requestBody)
			result, err := svc.ForwardDeepSeekTool(context.Background(), c, newDeepSeekToolsAccount(), 223, test.kind, requestBody)
			require.NoError(t, err)
			require.NotNil(t, result)
			require.Equal(t, test.calls, result.BillableToolCalls)
			require.Equal(t, test.kind, result.DeepSeekToolKind)
			require.Equal(t, http.StatusOK, recorder.Code)
			require.Equal(t, test.body, recorder.Body.String())
		})
	}
}

func TestForwardDeepSeekToolErrorsDoNotReplayOrMutateAccount(t *testing.T) {
	for _, status := range []int{201, 301, 400, 401, 402, 403, 404, 408, 429, 500, 504} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			body := "upstream non-JSON error"
			upstream := &deepSeekToolsUpstream{response: &http.Response{
				StatusCode: status,
				Header:     http.Header{"Content-Type": {"text/plain"}, "Retry-After": {"3"}},
				Body:       io.NopCloser(strings.NewReader(body)),
			}}
			// Nil account/rate-limit repositories make any chat-health side effect a failure.
			svc := &OpenAIGatewayService{cfg: &config.Config{}, httpUpstream: upstream}
			account := newDeepSeekToolsAccount()
			requestBody := []byte(`{"query":"q"}`)
			c, recorder := newDeepSeekToolsContext(requestBody)
			result, err := svc.ForwardDeepSeekTool(context.Background(), c, account, 223, DeepSeekToolSearch, requestBody)
			require.NoError(t, err)
			require.Nil(t, result)
			require.Equal(t, status, recorder.Code)
			require.Equal(t, body, recorder.Body.String())
			require.Equal(t, 1, upstream.calls)
			require.Equal(t, StatusActive, account.Status)
		})
	}
}

func TestForwardDeepSeekToolRejectsMalformedSuccess(t *testing.T) {
	for _, test := range []struct {
		kind DeepSeekToolKind
		body string
	}{
		{DeepSeekToolSearch, `not JSON`},
		{DeepSeekToolSearch, `{}`},
		{DeepSeekToolSearch, `{"results":null}`},
		{DeepSeekToolSearch, `{"results":{}}`},
		{DeepSeekToolSearch, `{"results":[null]}`},
		{DeepSeekToolSearch, `{"results":[{"title":5}]}`},
		{DeepSeekToolSearch, `{"results":[{"site_name":false}]}`},
		{DeepSeekToolSearch, `{"results":[{"authority":"3"}]}`},
		{DeepSeekToolFetch, `{}`},
		{DeepSeekToolFetch, `{"content":null}`},
		{DeepSeekToolFetch, `{"content":3}`},
		{DeepSeekToolFetch, `{"content":"text","title":5}`},
	} {
		t.Run(string(test.kind)+test.body, func(t *testing.T) {
			upstream := &deepSeekToolsUpstream{response: &http.Response{StatusCode: 200, Header: http.Header{}, Body: io.NopCloser(strings.NewReader(test.body))}}
			svc := &OpenAIGatewayService{cfg: &config.Config{}, httpUpstream: upstream}
			requestBody := []byte(`{"query":"q"}`)
			if test.kind == DeepSeekToolFetch {
				requestBody = []byte(`{"url":"https://example.com/"}`)
			}
			c, recorder := newDeepSeekToolsContext(requestBody)
			result, err := svc.ForwardDeepSeekTool(context.Background(), c, newDeepSeekToolsAccount(), 223, test.kind, requestBody)
			require.Error(t, err)
			require.Nil(t, result)
			require.Equal(t, http.StatusBadGateway, recorder.Code)
			require.Contains(t, recorder.Body.String(), "Malformed upstream response")
			require.Equal(t, 1, upstream.calls)
		})
	}
}

func TestForwardDeepSeekToolScopeAndRequestValidationBeforeUpstream(t *testing.T) {
	for _, test := range []struct {
		name    string
		groupID int64
		body    string
		status  int
	}{
		{"shared group denied", 81, `{"query":"q"}`, http.StatusForbidden},
		{"invalid group denied", 0, `{"query":"q"}`, http.StatusForbidden},
		{"invalid request", 223, `{"query":""}`, http.StatusBadRequest},
	} {
		t.Run(test.name, func(t *testing.T) {
			upstream := &deepSeekToolsUpstream{}
			svc := &OpenAIGatewayService{cfg: &config.Config{}, httpUpstream: upstream}
			c, recorder := newDeepSeekToolsContext([]byte(test.body))
			c.Set("api_key", &APIKey{GroupID: &test.groupID})
			result, err := svc.ForwardDeepSeekTool(context.Background(), c, newDeepSeekToolsAccount(), test.groupID, DeepSeekToolSearch, []byte(test.body))
			require.Error(t, err)
			require.Nil(t, result)
			require.Equal(t, test.status, recorder.Code)
			require.Zero(t, upstream.calls)
		})
	}
}

func TestForwardDeepSeekToolTransportErrorsAndResponseLimit(t *testing.T) {
	for _, test := range []struct {
		name   string
		err    error
		status int
	}{
		{"deadline", context.DeadlineExceeded, http.StatusGatewayTimeout},
		{"transport", errors.New("synthetic transport failure"), http.StatusBadGateway},
	} {
		t.Run(test.name, func(t *testing.T) {
			upstream := &deepSeekToolsUpstream{err: test.err}
			svc := &OpenAIGatewayService{cfg: &config.Config{}, httpUpstream: upstream}
			body := []byte(`{"query":"q"}`)
			c, recorder := newDeepSeekToolsContext(body)
			result, err := svc.ForwardDeepSeekTool(context.Background(), c, newDeepSeekToolsAccount(), 223, DeepSeekToolSearch, body)
			require.ErrorIs(t, err, test.err)
			require.Nil(t, result)
			require.Equal(t, test.status, recorder.Code)
			require.Equal(t, 1, upstream.calls)
		})
	}
	t.Run("response limit", func(t *testing.T) {
		upstream := &deepSeekToolsUpstream{response: &http.Response{StatusCode: 200, Header: http.Header{}, Body: io.NopCloser(strings.NewReader(strings.Repeat("x", 33)))}}
		svc := &OpenAIGatewayService{cfg: &config.Config{Gateway: config.GatewayConfig{UpstreamResponseReadMaxBytes: 32}}, httpUpstream: upstream}
		body := []byte(`{"query":"q"}`)
		c, recorder := newDeepSeekToolsContext(body)
		result, err := svc.ForwardDeepSeekTool(context.Background(), c, newDeepSeekToolsAccount(), 223, DeepSeekToolSearch, body)
		require.ErrorIs(t, err, ErrUpstreamResponseBodyTooLarge)
		require.Nil(t, result)
		require.Equal(t, http.StatusBadGateway, recorder.Code)
	})
}

type deepSeekToolsRealUpstream struct {
	HTTPUpstream
	client *http.Client
}

func (upstream *deepSeekToolsRealUpstream) Do(request *http.Request, _ string, _ int64, _ int) (*http.Response, error) {
	return upstream.client.Do(request)
}

func TestForwardDeepSeekToolRealHTTPAndIndependentRequestIDs(t *testing.T) {
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		calls.Add(1)
		require.Equal(t, "/v1/tools/fetch", request.URL.Path)
		require.Equal(t, "Bearer synthetic-upstream-key", request.Header.Get("Authorization"))
		body, err := io.ReadAll(request.Body)
		require.NoError(t, err)
		require.JSONEq(t, `{"url":"https://example.com/","future":true}`, string(body))
		writer.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(writer, `{"title":"Article","content":"# Article","future":true}`)
	}))
	defer server.Close()
	account := newDeepSeekToolsAccount()
	account.Credentials["base_url"] = server.URL
	cfg := &config.Config{}
	cfg.Security.URLAllowlist.AllowInsecureHTTP = true
	svc := &OpenAIGatewayService{cfg: cfg, httpUpstream: &deepSeekToolsRealUpstream{client: server.Client()}}
	body := []byte(`{"url":"https://example.com/","future":true}`)
	var lastRequestID string
	for range 2 {
		c, recorder := newDeepSeekToolsContext(body)
		result, err := svc.ForwardDeepSeekTool(context.Background(), c, account, 223, DeepSeekToolFetch, body)
		require.NoError(t, err)
		require.NotNil(t, result)
		require.Equal(t, 1, result.BillableToolCalls)
		require.NotEqual(t, lastRequestID, result.RequestID)
		require.JSONEq(t, `{"title":"Article","content":"# Article","future":true}`, recorder.Body.String())
		lastRequestID = result.RequestID
	}
	require.Equal(t, int32(2), calls.Load())
}

func TestForwardDeepSeekToolReadTimeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		writer.WriteHeader(http.StatusOK)
		writer.(http.Flusher).Flush()
		<-request.Context().Done()
	}))
	defer server.Close()
	account := newDeepSeekToolsAccount()
	account.Credentials["base_url"] = server.URL
	cfg := &config.Config{}
	cfg.Security.URLAllowlist.AllowInsecureHTTP = true
	svc := &OpenAIGatewayService{cfg: cfg, httpUpstream: &deepSeekToolsRealUpstream{client: server.Client()}}
	body := []byte(`{"query":"q"}`)
	c, recorder := newDeepSeekToolsContext(body)
	ctx, cancel := context.WithTimeout(context.Background(), 40*time.Millisecond)
	defer cancel()
	result, err := svc.ForwardDeepSeekTool(ctx, c, account, 223, DeepSeekToolSearch, body)
	require.Error(t, err)
	require.Nil(t, result)
	require.Equal(t, http.StatusGatewayTimeout, recorder.Code)
}

type deepSeekToolsAccountRepo struct {
	AccountRepository
	latest *Account
	err    error
	calls  int
}

func (repo *deepSeekToolsAccountRepo) GetByID(context.Context, int64) (*Account, error) {
	repo.calls++
	return repo.latest, repo.err
}

func TestForwardDeepSeekToolRechecksQueuedAccountPermissions(t *testing.T) {
	for _, test := range []struct {
		name   string
		mutate func(*deepSeekToolsAccountRepo)
		status int
	}{
		{"scope revoked", func(repo *deepSeekToolsAccountRepo) { repo.latest.Extra = nil }, http.StatusForbidden},
		{"group detached", func(repo *deepSeekToolsAccountRepo) { repo.latest.GroupIDs = []int64{81} }, http.StatusForbidden},
		{"account inactive", func(repo *deepSeekToolsAccountRepo) { repo.latest.Status = StatusDisabled }, http.StatusForbidden},
		{"account paused", func(repo *deepSeekToolsAccountRepo) { repo.latest.Schedulable = false }, http.StatusForbidden},
		{"lookup failed", func(repo *deepSeekToolsAccountRepo) { repo.err = errors.New("synthetic lookup failure") }, http.StatusServiceUnavailable},
		{"account missing", func(repo *deepSeekToolsAccountRepo) { repo.latest = nil }, http.StatusServiceUnavailable},
	} {
		t.Run(test.name, func(t *testing.T) {
			admitted := newDeepSeekToolsAccount()
			repo := &deepSeekToolsAccountRepo{latest: newDeepSeekToolsAccount()}
			test.mutate(repo)
			upstream := &deepSeekToolsUpstream{}
			svc := &OpenAIGatewayService{cfg: &config.Config{}, accountRepo: repo, httpUpstream: upstream}
			body := []byte(`{"query":"q"}`)
			c, recorder := newDeepSeekToolsContext(body)
			result, err := svc.ForwardDeepSeekTool(context.Background(), c, admitted, 223, DeepSeekToolSearch, body)
			require.Error(t, err)
			require.Nil(t, result)
			require.Equal(t, test.status, recorder.Code)
			require.Equal(t, 1, repo.calls)
			require.Zero(t, upstream.calls)
			require.Equal(t, StatusActive, admitted.Status)
		})
	}
}

func TestForwardDeepSeekToolRequiresAuthenticatedMatchingGroup(t *testing.T) {
	for _, test := range []struct {
		name  string
		value any
	}{
		{"missing identity", nil},
		{"invalid identity", "invalid"},
		{"key without group", &APIKey{}},
		{"wrong group", &APIKey{GroupID: func() *int64 { groupID := int64(81); return &groupID }()}},
	} {
		t.Run(test.name, func(t *testing.T) {
			upstream := &deepSeekToolsUpstream{}
			svc := &OpenAIGatewayService{cfg: &config.Config{}, httpUpstream: upstream}
			body := []byte(`{"query":"q"}`)
			c, recorder := newDeepSeekToolsContext(body)
			c.Set("api_key", test.value)
			result, err := svc.ForwardDeepSeekTool(context.Background(), c, newDeepSeekToolsAccount(), 223, DeepSeekToolSearch, body)
			require.Error(t, err)
			require.Nil(t, result)
			require.Equal(t, http.StatusForbidden, recorder.Code)
			require.Zero(t, upstream.calls)
		})
	}
}

func TestForwardDeepSeekToolCancellationDoesNotWriteOrBill(t *testing.T) {
	upstream := &deepSeekToolsUpstream{err: context.Canceled}
	svc := &OpenAIGatewayService{cfg: &config.Config{}, httpUpstream: upstream}
	body := []byte(`{"query":"q"}`)
	c, _ := newDeepSeekToolsContext(body)
	result, err := svc.ForwardDeepSeekTool(context.Background(), c, newDeepSeekToolsAccount(), 223, DeepSeekToolSearch, body)
	require.ErrorIs(t, err, context.Canceled)
	require.Nil(t, result)
	require.False(t, c.Writer.Written())
	require.Equal(t, 1, upstream.calls)
}
