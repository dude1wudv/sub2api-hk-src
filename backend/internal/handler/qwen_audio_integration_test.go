package handler

import (
	"bytes"
	"context"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

type qwenAudioIntegrationAccountRepo struct {
	service.AccountRepository
	accounts []service.Account
}

func (r qwenAudioIntegrationAccountRepo) GetByID(_ context.Context, id int64) (*service.Account, error) {
	for i := range r.accounts {
		if r.accounts[i].ID == id {
			account := r.accounts[i]
			return &account, nil
		}
	}
	return nil, service.ErrNoAvailableAccounts
}

func (r qwenAudioIntegrationAccountRepo) ListSchedulableByGroupIDAndPlatform(_ context.Context, _ int64, platform string) ([]service.Account, error) {
	return r.accountsForPlatform(platform), nil
}

func (r qwenAudioIntegrationAccountRepo) ListSchedulableByPlatform(_ context.Context, platform string) ([]service.Account, error) {
	return r.accountsForPlatform(platform), nil
}

func (r qwenAudioIntegrationAccountRepo) ListSchedulableUngroupedByPlatform(_ context.Context, platform string) ([]service.Account, error) {
	return r.accountsForPlatform(platform), nil
}

func (r qwenAudioIntegrationAccountRepo) accountsForPlatform(platform string) []service.Account {
	accounts := make([]service.Account, 0, len(r.accounts))
	for _, account := range r.accounts {
		if account.Platform == platform {
			accounts = append(accounts, account)
		}
	}
	return accounts
}

type qwenAudioIntegrationUpstream struct {
	service.HTTPUpstream
	mu      sync.Mutex
	calls   []int64
	allFail bool
}

func (u *qwenAudioIntegrationUpstream) Do(_ *http.Request, _ string, accountID int64, _ int) (*http.Response, error) {
	u.mu.Lock()
	u.calls = append(u.calls, accountID)
	u.mu.Unlock()
	if accountID == 1 || u.allFail {
		return &http.Response{
			StatusCode: http.StatusInternalServerError,
			Header:     make(http.Header),
			Body:       io.NopCloser(bytes.NewBufferString(`{"request_id":"failed-provider-request","message":"private upstream detail"}`)),
		}, nil
	}
	return &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(bytes.NewBufferString(`{"request_id":"asr-provider-success-2","output":{"text":"transcribed"}}`)),
	}, nil
}

func (u *qwenAudioIntegrationUpstream) accountIDs() []int64 {
	u.mu.Lock()
	defer u.mu.Unlock()
	return append([]int64(nil), u.calls...)
}

type qwenAudioIntegrationUsageRepo struct {
	service.UsageLogRepository
	mu   sync.Mutex
	logs []*service.UsageLog
}

type qwenAudioIntegrationBillingRepo struct {
	service.UsageBillingRepository
	mu       sync.Mutex
	commands []*service.UsageBillingCommand
}

func (r *qwenAudioIntegrationBillingRepo) Apply(_ context.Context, command *service.UsageBillingCommand) (*service.UsageBillingApplyResult, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	copyCommand := *command
	r.commands = append(r.commands, &copyCommand)
	return &service.UsageBillingApplyResult{Applied: true}, nil
}

func (r *qwenAudioIntegrationBillingRepo) snapshots() []*service.UsageBillingCommand {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]*service.UsageBillingCommand(nil), r.commands...)
}

type qwenAudioIntegrationUserRepo struct {
	service.UserRepository
	user *service.User
}

func (r qwenAudioIntegrationUserRepo) GetByID(_ context.Context, _ int64) (*service.User, error) {
	return r.user, nil
}

func (r *qwenAudioIntegrationUsageRepo) Create(_ context.Context, log *service.UsageLog) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	copyLog := *log
	r.logs = append(r.logs, &copyLog)
	return true, nil
}

func (r *qwenAudioIntegrationUsageRepo) snapshots() []*service.UsageLog {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]*service.UsageLog(nil), r.logs...)
}

func qwenAudioIntegrationAccounts() []service.Account {
	newAccount := func(id int64, priority int) service.Account {
		return service.Account{
			ID:          id,
			Name:        "qwen-audio-account",
			Platform:    service.PlatformOpenAI,
			Type:        service.AccountTypeAPIKey,
			Status:      service.StatusActive,
			Schedulable: true,
			Priority:    priority,
			Credentials: map[string]any{
				"api_key": "test-only-key",
				"model_mapping": map[string]any{
					"qwen-asr": "qwen-audio-3.0-asr-flash",
				},
			},
			Extra: map[string]any{
				service.QwenAudioHTTPBaseURLExtraKey: "https://workspace.cn-beijing.maas.aliyuncs.com/api/v1",
				service.QwenAudioWSBaseURLExtraKey:   "wss://workspace.cn-beijing.maas.aliyuncs.com/api-ws/v1",
			},
		}
	}
	return []service.Account{newAccount(1, 0), newAccount(2, 1)}
}

func newQwenAudioIntegrationHandler(
	t *testing.T,
	upstream service.HTTPUpstream,
	usageRepo service.UsageLogRepository,
	billingRepo service.UsageBillingRepository,
) *OpenAIGatewayHandler {
	t.Helper()
	cfg := &config.Config{}
	userRepo := qwenAudioIntegrationUserRepo{user: &service.User{ID: 100, Balance: 100}}
	billingCache := service.NewBillingCacheService(nil, userRepo, nil, nil, nil, nil, cfg, nil)
	t.Cleanup(billingCache.Stop)
	gatewayService := service.NewOpenAIGatewayService(
		qwenAudioIntegrationAccountRepo{accounts: qwenAudioIntegrationAccounts()},
		usageRepo,
		billingRepo,
		userRepo,
		nil,
		nil,
		nil,
		cfg,
		nil,
		nil,
		service.NewBillingService(cfg, nil),
		nil,
		billingCache,
		upstream,
		&service.DeferredService{},
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
	)
	return NewOpenAIGatewayHandler(
		gatewayService,
		service.NewConcurrencyService(nil),
		billingCache,
		service.NewAPIKeyService(nil, nil, nil, nil, nil, nil, cfg),
		nil,
		nil,
		nil,
		nil,
		cfg,
	)
}

func newQwenAudioIntegrationASRContext(t *testing.T) (*gin.Context, *httptest.ResponseRecorder) {
	t.Helper()
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	require.NoError(t, writer.WriteField("model", "qwen-asr"))
	file, err := writer.CreateFormFile("file", "voice.wav")
	require.NoError(t, err)
	_, err = file.Write(handlerPCM16WAV(16_000, 2))
	require.NoError(t, err)
	require.NoError(t, writer.Close())

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/audio/transcriptions", &body)
	c.Request.Header.Set("Content-Type", writer.FormDataContentType())
	groupID := int64(81)
	user := &service.User{ID: 100}
	c.Set(string(middleware2.ContextKeyAPIKey), &service.APIKey{
		ID:      99,
		GroupID: &groupID,
		Group: &service.Group{
			ID:             groupID,
			Platform:       service.PlatformOpenAI,
			RateMultiplier: 1,
		},
		User: user,
	})
	c.Set(string(middleware2.ContextKeyUser), middleware2.AuthSubject{UserID: user.ID, Concurrency: 0})
	return c, recorder
}

func TestQwenAudioASRRealSchedulerFailsOverAndPersistsOnlySuccessfulAccountUsage(t *testing.T) {
	gin.SetMode(gin.TestMode)
	upstream := &qwenAudioIntegrationUpstream{}
	usageRepo := &qwenAudioIntegrationUsageRepo{}
	billingRepo := &qwenAudioIntegrationBillingRepo{}
	h := newQwenAudioIntegrationHandler(t, upstream, usageRepo, billingRepo)
	c, recorder := newQwenAudioIntegrationASRContext(t)

	h.QwenAudioTranscriptions(c)

	require.Equal(t, http.StatusOK, recorder.Code)
	require.Equal(t, "transcribed", gjson.Get(recorder.Body.String(), "text").String())
	require.Equal(t, []int64{1, 2}, upstream.accountIDs())
	logs := usageRepo.snapshots()
	require.Len(t, logs, 1, "the failed account attempt must not create usage")
	require.Equal(t, int64(2), logs[0].AccountID)
	require.Equal(t, "qwen_audio:asr-provider-success-2", logs[0].RequestID)
	require.Equal(t, "qwen-asr", logs[0].Model)
	require.Equal(t, "qwen-asr", logs[0].RequestedModel)
	require.NotNil(t, logs[0].UpstreamModel)
	require.Equal(t, "qwen-audio-3.0-asr-flash", *logs[0].UpstreamModel)
	require.NotNil(t, logs[0].InboundEndpoint)
	require.Equal(t, "/v1/audio/transcriptions", *logs[0].InboundEndpoint)
	require.NotNil(t, logs[0].UpstreamEndpoint)
	require.Equal(t, qwenAudioASRUpstreamPath, *logs[0].UpstreamEndpoint)
	require.Positive(t, logs[0].ActualCost)
	commands := billingRepo.snapshots()
	require.Len(t, commands, 1, "only the successful account attempt may be billed")
	require.Equal(t, int64(2), commands[0].AccountID)
	require.Equal(t, "qwen_audio:asr-provider-success-2", commands[0].RequestID)
	require.Positive(t, commands[0].BalanceCost)
}

func TestQwenAudioASRAllAccountsFailWithoutUsageOrBillingRecord(t *testing.T) {
	gin.SetMode(gin.TestMode)
	upstream := &qwenAudioIntegrationUpstream{allFail: true}
	usageRepo := &qwenAudioIntegrationUsageRepo{}
	billingRepo := &qwenAudioIntegrationBillingRepo{}
	h := newQwenAudioIntegrationHandler(t, upstream, usageRepo, billingRepo)
	c, recorder := newQwenAudioIntegrationASRContext(t)

	h.QwenAudioTranscriptions(c)

	require.Equal(t, http.StatusBadGateway, recorder.Code)
	require.Equal(t, []int64{1, 2}, upstream.accountIDs())
	require.Empty(t, usageRepo.snapshots(), "failed requests must not enter usage persistence")
	require.Empty(t, billingRepo.snapshots(), "failed requests must not enter the atomic billing repository")
	require.NotContains(t, recorder.Body.String(), "private upstream detail")
}
