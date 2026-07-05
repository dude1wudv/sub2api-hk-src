package service

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/claude"
	"github.com/Wei-Shaw/sub2api/internal/pkg/ctxkey"
	"github.com/stretchr/testify/require"
)

func testBoolPtr(v bool) *bool { return &v }

func groupPolicyContext(group *Group) context.Context {
	return context.WithValue(context.Background(), ctxkey.Group, group)
}

func newGatewayServiceWithBetaSettingsForGroupTest(settingService *SettingService) *GatewayService {
	cfg := &config.Config{}
	return NewGatewayService(
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		cfg,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		settingService,
		nil,
		nil,
		nil,
		nil,
		nil,
	)
}

func TestEvaluateOpenAIFastPolicy_GroupSwitchOverridesFilter(t *testing.T) {
	svc := newOpenAIGatewayServiceWithSettings(t, &OpenAIFastPolicySettings{
		Rules: []OpenAIFastPolicyRule{{
			ServiceTier: OpenAIFastTierPriority,
			Action:      BetaPolicyActionFilter,
			Scope:       BetaPolicyScopeAll,
		}},
	})
	account := &Account{Type: AccountTypeAPIKey}
	ctx := groupPolicyContext(&Group{
		ID:       52,
		Platform: PlatformOpenAI,
		Status:   StatusActive,
		Hydrated: true,
		ModelsListConfig: GroupModelsListConfig{
			AllowFastMode: testBoolPtr(true),
		},
	})

	action, _ := svc.evaluateOpenAIFastPolicy(ctx, account, "gpt-5.5", OpenAIFastTierPriority)

	require.Equal(t, BetaPolicyActionPass, action)
}

func TestEvaluateOpenAIFastPolicy_GroupSwitchOffFiltersPriority(t *testing.T) {
	svc := newOpenAIGatewayServiceWithSettings(t, DefaultOpenAIFastPolicySettings())
	account := &Account{Type: AccountTypeAPIKey}
	ctx := groupPolicyContext(&Group{
		ID:       52,
		Platform: PlatformOpenAI,
		Status:   StatusActive,
		Hydrated: true,
		ModelsListConfig: GroupModelsListConfig{
			AllowFastMode: testBoolPtr(false),
		},
	})

	action, _ := svc.evaluateOpenAIFastPolicy(ctx, account, "gpt-5.5", OpenAIFastTierPriority)

	require.Equal(t, BetaPolicyActionFilter, action)
}

func TestApplyOpenAIFastPolicyToBody_GroupSwitchOnDoesNotInjectPriority(t *testing.T) {
	svc := newOpenAIGatewayServiceWithSettings(t, DefaultOpenAIFastPolicySettings())
	account := &Account{Type: AccountTypeOAuth}
	ctx := groupPolicyContext(&Group{
		ID:       52,
		Platform: PlatformOpenAI,
		Status:   StatusActive,
		Hydrated: true,
		ModelsListConfig: GroupModelsListConfig{
			AllowFastMode: testBoolPtr(true),
		},
	})

	updated, err := svc.applyOpenAIFastPolicyToBody(ctx, account, "gpt-5.5", []byte(`{"model":"gpt-5.5","input":[]}`))

	require.NoError(t, err)
	require.JSONEq(t, `{"model":"gpt-5.5","input":[]}`, string(updated))
}

func TestApplyOpenAIFastPolicyToWSResponseCreate_GroupSwitchOnDoesNotInjectPriority(t *testing.T) {
	svc := newOpenAIGatewayServiceWithSettings(t, DefaultOpenAIFastPolicySettings())
	account := &Account{Type: AccountTypeOAuth}
	ctx := groupPolicyContext(&Group{
		ID:       52,
		Platform: PlatformOpenAI,
		Status:   StatusActive,
		Hydrated: true,
		ModelsListConfig: GroupModelsListConfig{
			AllowFastMode: testBoolPtr(true),
		},
	})

	updated, blocked, err := svc.applyOpenAIFastPolicyToWSResponseCreate(ctx, account, "gpt-5.5", []byte(`{"type":"response.create","model":"gpt-5.5"}`))

	require.NoError(t, err)
	require.Nil(t, blocked)
	require.JSONEq(t, `{"type":"response.create","model":"gpt-5.5"}`, string(updated))
}

func TestEvaluateBetaPolicy_GroupContext1MSwitchOverridesFilter(t *testing.T) {
	settings := &BetaPolicySettings{
		Rules: []BetaPolicyRule{{
			BetaToken: claude.BetaContext1M,
			Action:    BetaPolicyActionFilter,
			Scope:     BetaPolicyScopeAll,
		}},
	}
	raw, err := json.Marshal(settings)
	require.NoError(t, err)
	svc := newGatewayServiceWithBetaSettingsForGroupTest(
		NewSettingService(&betaPolicySettingRepoStub{values: map[string]string{
			SettingKeyBetaPolicySettings: string(raw),
		}}, nil),
	)
	ctx := groupPolicyContext(&Group{
		ID:       52,
		Platform: PlatformOpenAI,
		Status:   StatusActive,
		Hydrated: true,
		ModelsListConfig: GroupModelsListConfig{
			AllowContext1M: testBoolPtr(true),
		},
	})

	result := svc.evaluateBetaPolicy(ctx, claude.BetaContext1M, &Account{Type: AccountTypeAPIKey}, "gpt-5.5")

	_, filtered := result.filterSet[claude.BetaContext1M]
	require.False(t, filtered)
}

func TestEvaluateBetaPolicy_GroupFastAndContextSwitchOffFilters(t *testing.T) {
	svc := newGatewayServiceWithBetaSettingsForGroupTest(
		NewSettingService(&betaPolicySettingRepoStub{values: map[string]string{}}, nil),
	)
	ctx := groupPolicyContext(&Group{
		ID:       52,
		Platform: PlatformOpenAI,
		Status:   StatusActive,
		Hydrated: true,
		ModelsListConfig: GroupModelsListConfig{
			AllowFastMode:  testBoolPtr(false),
			AllowContext1M: testBoolPtr(false),
		},
	})

	result := svc.evaluateBetaPolicy(ctx, "", &Account{Type: AccountTypeAPIKey}, "gpt-5.5")

	_, fastFiltered := result.filterSet[claude.BetaFastMode]
	_, contextFiltered := result.filterSet[claude.BetaContext1M]
	require.True(t, fastFiltered)
	require.True(t, contextFiltered)
}
