//go:build unit

package service

import (
	"context"
	"encoding/json"
	"math"
	"strconv"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
)

func deepSeekToolsScopeAccount(id int64, groups any) Account {
	return Account{
		ID: id, Platform: PlatformOpenAI, Type: AccountTypeAPIKey,
		Status: StatusActive, Schedulable: true, Concurrency: 1,
		GroupIDs: []int64{223, 81},
		Extra:    map[string]any{deepSeekToolsGroupIDsExtraKey: groups},
	}
}

func TestDeepSeekToolsAllowedForGroup(t *testing.T) {
	allowed, other := int64(223), int64(81)
	for _, value := range []any{[]int64{223}, []int{223}, []any{float64(223)}, []any{int64(223)}, []any{json.Number("223")}} {
		account := deepSeekToolsScopeAccount(1112, value)
		require.True(t, account.SupportsOpenAIEndpointCapability(OpenAIEndpointCapabilityDeepSeekTools))
		require.True(t, account.DeepSeekToolsAllowedForGroup(&allowed))
		require.False(t, account.DeepSeekToolsAllowedForGroup(&other), "chat membership does not authorize tools")
		require.False(t, account.DeepSeekToolsAllowedForGroup(nil))
		require.True(t, account.SupportsOpenAIEndpointCapability(OpenAIEndpointCapabilityChatCompletions))
	}
	for _, value := range []any{
		nil, []int64{}, []any{}, []int64{223, 0}, []int64{-223}, "223", []string{"223"},
		[]any{223, "81"}, []any{223, 81.5}, []any{math.Inf(1)}, []any{math.NaN()},
		[]any{math.Exp2(63)}, []any{json.Number("223.0")},
	} {
		account := deepSeekToolsScopeAccount(1112, value)
		require.False(t, account.SupportsOpenAIEndpointCapability(OpenAIEndpointCapabilityDeepSeekTools), "invalid permission must fail closed: %#v", value)
		require.False(t, account.DeepSeekToolsAllowedForGroup(&allowed), "a malformed list cannot partially grant permission")
		require.True(t, account.SupportsOpenAIEndpointCapability(OpenAIEndpointCapabilityChatCompletions))
	}
	var missing *Account
	require.False(t, missing.DeepSeekToolsAllowedForGroup(&allowed))
	for _, platform := range []string{PlatformOpenAI, PlatformDeepseek} {
		account := deepSeekToolsScopeAccount(1112, []int64{223})
		account.Platform = platform
		require.True(t, account.DeepSeekToolsAllowedForGroup(&allowed))
		account.Type = AccountTypeOAuth
		require.False(t, account.DeepSeekToolsAllowedForGroup(&allowed))
	}
	account := deepSeekToolsScopeAccount(1112, []int64{223})
	account.Platform = PlatformGrok
	require.False(t, account.DeepSeekToolsAllowedForGroup(&allowed))
}

func TestDeepSeekToolsEligibilityRequiresRealGroupScope(t *testing.T) {
	account := deepSeekToolsScopeAccount(1112, []int64{223})
	allowed, other := int64(223), int64(81)
	scheduler := &defaultOpenAIAccountScheduler{}
	for _, groupID := range []*int64{nil, &other, &allowed} {
		ok, reason := scheduler.isAccountRequestCompatibleReason(context.Background(), &account, OpenAIAccountScheduleRequest{
			GroupID: groupID, RequiredCapability: OpenAIEndpointCapabilityDeepSeekTools,
		})
		if groupID == &allowed {
			require.True(t, ok)
			require.Empty(t, reason)
		} else {
			require.False(t, ok)
			require.Equal(t, "deepseek_tools_group_not_allowed", reason)
		}
	}
	reason := openAICompatibleAccountEligibilityFailureReasonBeforeProfit(context.Background(), &account, PlatformOpenAI, "", false, OpenAIEndpointCapabilityDeepSeekTools)
	require.Equal(t, "deepseek_tools_group_not_allowed", reason, "missing legacy scope must deny tools")
	ctx := withDeepSeekToolsSchedulingGroup(context.Background(), &allowed, OpenAIEndpointCapabilityDeepSeekTools)
	require.Empty(t, openAICompatibleAccountEligibilityFailureReasonBeforeProfit(ctx, &account, PlatformOpenAI, "", false, OpenAIEndpointCapabilityDeepSeekTools))
	ctx = withDeepSeekToolsSchedulingGroup(ctx, &other, OpenAIEndpointCapabilityDeepSeekTools)
	require.Equal(t, "deepseek_tools_group_not_allowed", openAICompatibleAccountEligibilityFailureReasonBeforeProfit(ctx, &account, PlatformOpenAI, "", false, OpenAIEndpointCapabilityDeepSeekTools))
}

func TestDeepSeekToolsSchedulerModesIsolateSharedGroup(t *testing.T) {
	defer resetOpenAIAdvancedSchedulerSettingCacheForTest()
	for _, advanced := range []bool{false, true} {
		for _, loadBatch := range []bool{false, true} {
			for _, sticky := range []bool{false, true} {
				name := "advanced=" + strconv.FormatBool(advanced) + "/batch=" + strconv.FormatBool(loadBatch) + "/sticky=" + strconv.FormatBool(sticky)
				t.Run(name, func(t *testing.T) {
					ordinary := deepSeekToolsScopeAccount(1109, nil)
					permitted := deepSeekToolsScopeAccount(1112, []int64{223})
					permitted.Priority = 10
					permitted.Credentials = map[string]any{"model_mapping": map[string]any{"unrelated-model": "upstream-model"}}
					svc := newLegacySchedulerDecisionTestService([]Account{ordinary, permitted}, loadBatch, schedulerTestConcurrencyCache{})
					svc.rateLimitService = newOpenAIAdvancedSchedulerRateLimitService(strconv.FormatBool(advanced))
					allowed, other := int64(223), int64(81)
					session := ""
					if sticky {
						session = "tools-scope-sticky"
						require.NoError(t, svc.setStickySessionAccountID(context.Background(), &allowed, session, ordinary.ID, time.Hour))
					}
					selection, _, err := svc.SelectAccountWithSchedulerForCapability(context.Background(), &allowed, "", session, "", nil, OpenAIUpstreamTransportHTTPSSE, OpenAIEndpointCapabilityDeepSeekTools, false, false, false)
					require.NoError(t, err)
					require.NotNil(t, selection)
					require.Equal(t, permitted.ID, selection.Account.ID, "empty model tools must skip ordinary accounts and ignore chat model mappings")
					releaseLegacySchedulerDecisionSelection(selection)
					require.NoError(t, svc.getOpenAIWSStateStore().BindResponseAccount(context.Background(), other, "resp_tools_scope", permitted.ID, time.Hour))
					selection, _, err = svc.SelectAccountWithSchedulerForCapability(context.Background(), &other, "resp_tools_scope", session, "", nil, OpenAIUpstreamTransportHTTPSSE, OpenAIEndpointCapabilityDeepSeekTools, false, false, false)
					require.ErrorIs(t, err, ErrNoAvailableAccounts)
					require.Nil(t, selection, "shared group must stay denied, including a previous-response binding")
					selection, _, err = svc.SelectAccountWithSchedulerForCapability(context.Background(), &other, "", "", "gpt-5.1", nil, OpenAIUpstreamTransportHTTPSSE, OpenAIEndpointCapabilityChatCompletions, false, false, false)
					require.NoError(t, err, "ordinary chat eligibility must remain unchanged")
					require.NotNil(t, selection)
					releaseLegacySchedulerDecisionSelection(selection)
				})
			}
		}
	}
}

func TestDeepSeekToolsSmartRoutingReservationRevalidatesAndReleases(t *testing.T) {
	defer resetOpenAIAdvancedSchedulerSettingCacheForTest()
	account := deepSeekToolsScopeAccount(1112, []int64{223})
	groupID := int64(81)
	releases := 0
	svc := newLegacySchedulerDecisionTestService([]Account{account}, false, schedulerTestConcurrencyCache{})
	svc.rateLimitService = newOpenAIAdvancedSchedulerRateLimitService("false")
	ctx := context.WithValue(context.Background(), smartRoutingSelectionContextKey{}, &smartRoutingSelection{
		groupID: groupID, model: "",
		selection: &AccountSelectionResult{Account: &account, Acquired: true, ReleaseFunc: func() { releases++ }},
	})
	selection, _, err := svc.SelectAccountWithSchedulerForCapability(ctx, &groupID, "", "", "", nil, OpenAIUpstreamTransportHTTPSSE, OpenAIEndpointCapabilityDeepSeekTools, false, false, false)
	require.ErrorIs(t, err, ErrNoAvailableAccounts)
	require.Nil(t, selection)
	require.Equal(t, 1, releases, "rejecting a preselected chat slot must release it")
}

func TestDeepSeekToolsDBRecheckRejectsRevokedPermission(t *testing.T) {
	account := deepSeekToolsScopeAccount(1112, []int64{223})
	revoked := account
	revoked.Extra = nil
	groupID := int64(223)
	for _, snapshot := range []bool{false, true} {
		svc := &OpenAIGatewayService{
			accountRepo: schedulerTestOpenAIAccountRepo{accounts: []Account{revoked}},
			cfg:         &config.Config{RunMode: config.RunModeStandard},
		}
		if snapshot {
			svc.schedulerSnapshot = &SchedulerSnapshotService{cache: &openAISnapshotCacheStub{}}
		}
		require.Nil(t, svc.recheckSelectedOpenAIAccountFromDBBeforeProfit(context.Background(), &account, &groupID, PlatformOpenAI, "", false, OpenAIEndpointCapabilityDeepSeekTools), "snapshot=%v must consult current permission", snapshot)
	}
}

func TestDeepSeekToolsAdvancedFreshScopeRevocationReleasesSlot(t *testing.T) {
	stale := deepSeekToolsScopeAccount(1112, []int64{223})
	revoked := stale
	revoked.Extra = nil
	groupID := int64(223)
	acquired, released := []int64{}, []int64{}
	svc := &OpenAIGatewayService{
		accountRepo: schedulerTestOpenAIAccountRepo{accounts: []Account{revoked}},
		cfg:         &config.Config{RunMode: config.RunModeStandard},
		schedulerSnapshot: &SchedulerSnapshotService{cache: &openAISnapshotCacheStub{
			accountsByID: map[int64]*Account{stale.ID: &stale},
		}},
		concurrencyService: NewConcurrencyService(schedulerTestConcurrencyCache{acquiredIDs: &acquired, releasedIDs: &released}),
	}
	scheduler := &defaultOpenAIAccountScheduler{service: svc}
	ctx := withDeepSeekToolsSchedulingGroup(context.Background(), &groupID, OpenAIEndpointCapabilityDeepSeekTools)
	selection, _, err := scheduler.tryAcquireOpenAISelectionOrder(ctx, OpenAIAccountScheduleRequest{
		GroupID: &groupID, Platform: PlatformOpenAI, RequiredCapability: OpenAIEndpointCapabilityDeepSeekTools,
	}, []openAIAccountCandidateScore{{account: &stale, loadInfo: &AccountLoadInfo{AccountID: stale.ID}}})
	require.NoError(t, err)
	require.Nil(t, selection)
	require.Equal(t, []int64{stale.ID}, acquired)
	require.Equal(t, []int64{stale.ID}, released, "revocation between snapshot selection and DB admission must release the acquired slot")
}
