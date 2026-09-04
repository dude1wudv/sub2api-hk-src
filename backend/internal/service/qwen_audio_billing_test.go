package service

import (
	"context"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/ctxkey"
	"github.com/stretchr/testify/require"
)

type qwenAudioDedupBillingRepo struct {
	UsageBillingRepository
	mu       sync.Mutex
	commands []*UsageBillingCommand
	applied  map[string]string
}

func (r *qwenAudioDedupBillingRepo) Apply(_ context.Context, cmd *UsageBillingCommand) (*UsageBillingApplyResult, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	copyCmd := *cmd
	r.commands = append(r.commands, &copyCmd)
	if r.applied == nil {
		r.applied = make(map[string]string)
	}
	if fingerprint, exists := r.applied[cmd.RequestID]; exists {
		if fingerprint != cmd.RequestFingerprint {
			return nil, ErrUsageBillingRequestConflict
		}
		return &UsageBillingApplyResult{Applied: false}, nil
	}
	r.applied[cmd.RequestID] = cmd.RequestFingerprint
	return &UsageBillingApplyResult{Applied: true}, nil
}

func (r *qwenAudioDedupBillingRepo) snapshots() ([]*UsageBillingCommand, map[string]string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	commands := append([]*UsageBillingCommand(nil), r.commands...)
	applied := make(map[string]string, len(r.applied))
	for requestID, fingerprint := range r.applied {
		applied[requestID] = fingerprint
	}
	return commands, applied
}

type qwenAudioDedupUsageRepo struct {
	UsageLogRepository
	mu   sync.Mutex
	seen map[string]struct{}
	logs []*UsageLog
}

func (r *qwenAudioDedupUsageRepo) Create(_ context.Context, usage *UsageLog) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.seen == nil {
		r.seen = make(map[string]struct{})
	}
	key := usage.RequestID + ":" + strconv.FormatInt(usage.APIKeyID, 10)
	if _, exists := r.seen[key]; exists {
		return false, nil
	}
	r.seen[key] = struct{}{}
	copyUsage := *usage
	r.logs = append(r.logs, &copyUsage)
	return true, nil
}

func (r *qwenAudioDedupUsageRepo) snapshots() []*UsageLog {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]*UsageLog(nil), r.logs...)
}

func TestQwenAudioUsagePersistsProviderUnitsAndDoesNotCollapseReusedClientRequestID(t *testing.T) {
	groupID := int64(81)
	ttsPrice := 12.5
	usageRepo := &qwenAudioDedupUsageRepo{}
	billingRepo := &qwenAudioDedupBillingRepo{}
	svc := newOpenAIRecordUsageServiceWithBillingRepoForTest(
		usageRepo,
		billingRepo,
		&openAIRecordUsageUserRepoStub{},
		&openAIRecordUsageSubRepoStub{},
		nil,
	)
	apiKey := &APIKey{
		ID:      99,
		GroupID: &groupID,
		Group: &Group{
			ID:                           groupID,
			Platform:                     PlatformOpenAI,
			RateMultiplier:               1,
			AudioTTSPricePerMillionChars: &ttsPrice,
		},
	}
	user := &User{ID: 100}
	account := &Account{ID: 2, Platform: PlatformOpenAI, Type: AccountTypeAPIKey}
	ctx := context.WithValue(context.Background(), ctxkey.ClientRequestID, "reused-client-request-id")

	record := func(taskID string, billedCharacters int) {
		t.Helper()
		err := svc.RecordUsage(ctx, &OpenAIRecordUsageInput{
			Result: &OpenAIForwardResult{
				RequestID:        StableQwenAudioBillingRequestID(taskID),
				Model:            "qwen-tts",
				BillingModel:     "qwen-tts",
				UpstreamModel:    "qwen-audio-3.0-tts-plus",
				UpstreamEndpoint: qwenTTSPath,
				Duration:         250 * time.Millisecond,
				AudioUsage: &AudioUsage{
					Mode:            "tts",
					DurationOrUnits: float64(billedCharacters) / 1_000_000,
				},
			},
			APIKey:             apiKey,
			User:               user,
			Account:            account,
			InboundEndpoint:    "/v1/audio/speech",
			UpstreamEndpoint:   qwenTTSPath,
			RequestPayloadHash: "same-client-payload-hash",
		})
		require.NoError(t, err)
	}

	record("provider-task-1", 250)
	record("provider-task-2", 250)
	record("provider-task-2", 250)

	commands, applied := billingRepo.snapshots()
	require.Len(t, commands, 3, "the duplicate provider task reaches the idempotent repository")
	require.Len(t, applied, 2, "two provider tasks must remain distinct despite one reused client request ID")
	require.Contains(t, applied, "qwen_audio:provider-task-1")
	require.Contains(t, applied, "qwen_audio:provider-task-2")
	require.Equal(t, "qwen_audio:provider-task-1", commands[0].RequestID)
	require.Equal(t, "qwen_audio:provider-task-2", commands[1].RequestID)
	require.Equal(t, commands[1].RequestFingerprint, commands[2].RequestFingerprint, "a repeated provider task must deduplicate cleanly")
	require.Equal(t, int64(100), commands[0].UserID)
	require.Equal(t, int64(2), commands[0].AccountID)
	require.Equal(t, int64(99), commands[0].APIKeyID)
	require.Equal(t, "qwen-tts", commands[0].Model)
	require.Equal(t, "same-client-payload-hash", commands[0].RequestPayloadHash)
	require.InDelta(t, 0.003125, commands[0].BalanceCost, 1e-9, "250 provider-reported characters at $12.50/M must determine billing")

	logs := usageRepo.snapshots()
	require.Len(t, logs, 2, "usage persistence must keep one row per unique provider task")
	require.Equal(t, "qwen_audio:provider-task-1", logs[0].RequestID)
	require.Equal(t, "qwen_audio:provider-task-2", logs[1].RequestID)
	require.Equal(t, "qwen-tts", logs[0].RequestedModel)
	require.NotNil(t, logs[0].UpstreamModel)
	require.Equal(t, "qwen-audio-3.0-tts-plus", *logs[0].UpstreamModel)
	require.NotNil(t, logs[0].InboundEndpoint)
	require.Equal(t, "/v1/audio/speech", *logs[0].InboundEndpoint)
	require.NotNil(t, logs[0].UpstreamEndpoint)
	require.Equal(t, qwenTTSPath, *logs[0].UpstreamEndpoint)
	require.NotNil(t, logs[0].BillingMode)
	require.Equal(t, string(BillingModePerRequest), *logs[0].BillingMode)
	require.InDelta(t, 0.003125, logs[0].ActualCost, 1e-9)
}
