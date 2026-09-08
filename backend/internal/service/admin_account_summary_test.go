package service

import (
	"context"
	"strings"
	"testing"
)

type summaryAccountRepositorySpy struct {
	AccountRepository
	used bool
}

func (repository *summaryAccountRepositorySpy) ListForSummaryWithFilters(
	context.Context, string, string, string, string, int64, string,
) ([]Account, error) {
	repository.used = true
	return []Account{{Platform: PlatformOpenAI, Type: AccountTypeOAuth, Status: StatusActive, Schedulable: true}}, nil
}

func TestGetAccountSummaryUsesLightweightRepositoryWhenAvailable(t *testing.T) {
	repository := &summaryAccountRepositorySpy{}
	service := &adminServiceImpl{accountRepo: repository}
	summary, err := service.GetAccountSummary(context.Background(), "", "", "", "", 0, "")
	if err != nil {
		t.Fatal(err)
	}
	if !repository.used || summary.Total != 1 || summary.OpenAI != 1 {
		t.Fatalf("lightweight summary path not used: used=%v summary=%#v", repository.used, summary)
	}
}

func TestOpenAIOAuthUsageSummaryQueryUsesResetBaseline(t *testing.T) {
	query := openAIOAuthUsageSummaryQuery()
	for _, want := range []string{settingKeyOpenAIOAuthUsageBaselineAccountIDs, settingKeyOpenAIOAuthUsageCountAfter,
		"a.id IN (SELECT id FROM baseline_account_ids)", "a.created_at >= (SELECT count_after FROM oauth_usage_baseline)",
		"COUNT(DISTINCT tu.account_id)::bigint AS accounts_with_usage"} {
		if !strings.Contains(query, want) {
			t.Fatalf("query missing %q", want)
		}
	}
}

func TestAccountUsageAccumulatorSummary(t *testing.T) {
	var accumulator accountUsageAccumulator
	accumulator.add(25)
	accumulator.add(100)
	summary := accumulator.summary()
	if summary.Sampled != 2 || summary.Exhausted != 1 {
		t.Fatalf("unexpected sample counts: %#v", summary)
	}
	if summary.UsedPercent == nil || *summary.UsedPercent != 62.5 {
		t.Fatalf("unexpected used percent: %#v", summary.UsedPercent)
	}
	if summary.RemainingPercent == nil || *summary.RemainingPercent != 37.5 {
		t.Fatalf("unexpected remaining percent: %#v", summary.RemainingPercent)
	}
}

func TestBuildAccountSummaryOpenAIOAuthWindows(t *testing.T) {
	accounts := []Account{
		{Platform: PlatformOpenAI, Type: AccountTypeOAuth, Status: StatusActive, Schedulable: true, Extra: map[string]any{"codex_5h_used_percent": 10.0, "codex_7d_used_percent": 80.0}},
		{Platform: PlatformOpenAI, Type: AccountTypeOAuth, Status: StatusActive, Schedulable: true, Extra: map[string]any{"codex_5h_used_percent": 40.0, "codex_7d_used_percent": 20.0}},
		{Platform: PlatformOpenAI, Type: AccountTypeAPIKey, Status: StatusActive, Schedulable: true, Extra: map[string]any{"codex_5h_used_percent": 5.0}},
	}
	summary := buildAccountSummary(accounts)
	if summary.Codex5h.Sampled != 2 || summary.Codex5h.UsedPercent == nil || *summary.Codex5h.UsedPercent != 25 {
		t.Fatalf("unexpected Codex 5h summary: %#v", summary.Codex5h)
	}
	if summary.OAuthPool.Total != 2 || summary.OAuthPool.Remaining7dPercent == nil || *summary.OAuthPool.Remaining7dPercent != 50 {
		t.Fatalf("unexpected OAuth pool summary: %#v", summary.OAuthPool)
	}
}
