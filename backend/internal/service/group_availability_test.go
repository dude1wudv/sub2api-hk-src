//go:build unit

package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type availabilityHistoryRepo struct {
	GroupRepository
	groups  []Group
	changes map[int64]GroupRateChange
	err     error
	queried []int64
	since   time.Time
}

func (r *availabilityHistoryRepo) ListActive(context.Context) ([]Group, error) { return r.groups, nil }
func (r *availabilityHistoryRepo) GetRecentRateChanges(_ context.Context, ids []int64, since time.Time) (map[int64]GroupRateChange, error) {
	r.queried, r.since = ids, since
	return r.changes, r.err
}

type availabilityPersonalRates struct {
	UserGroupRateRepository
	rates map[int64]float64
	err   error
}

func (r *availabilityPersonalRates) GetByUserID(context.Context, int64) (map[int64]float64, error) {
	return r.rates, r.err
}

func TestGroupAvailabilityExplicitLaunchWindow(t *testing.T) {
	now := time.Now().UTC()
	for _, tc := range []struct {
		name           string
		offset         time.Duration
		launched, want bool
	}{
		{"unpublished newly created/imported", 0, false, false},
		{"launched now", 0, true, true},
		{"recent", -23 * time.Hour, true, true},
		{"exactly 24 hours", -24 * time.Hour, true, false},
		{"old edited group", -30 * 24 * time.Hour, true, false},
		{"scheduled", time.Hour, true, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			g := Group{CreatedAt: now, UpdatedAt: now}
			if tc.launched {
				at := now.Add(tc.offset)
				g.LaunchedAt = &at
			}
			require.Equal(t, tc.want, g.IsNewAt(now))
		})
	}
}

func TestAvailableGroupHintsPermissionsPersonalRatesAndFallback(t *testing.T) {
	now := time.Now().UTC()
	change := GroupRateChange{OldRate: 2, NewRate: 1, ChangedAt: now.Add(-time.Hour)}
	for _, tc := range []struct {
		name                 string
		historyErr, ratesErr error
		fixed                bool
		wantHint             bool
	}{
		{name: "ordinary user", wantHint: true},
		{name: "fixed equal to base", fixed: true},
		{name: "history unavailable", historyErr: errors.New("history unavailable")},
		{name: "history unavailable with fixed rate", historyErr: errors.New("history unavailable"), fixed: true},
		{name: "personal rates unknown", ratesErr: errors.New("rates unavailable")},
	} {
		t.Run(tc.name, func(t *testing.T) {
			launchRate := 1.0
			repo := &availabilityHistoryRepo{groups: []Group{
				{ID: 1, Name: "public", RateMultiplier: 1, LaunchedAt: &now, LaunchRate: &launchRate},
				{ID: 2, Name: "private", IsExclusive: true},
				{ID: 3, Name: "subscription", SubscriptionType: SubscriptionTypeSubscription},
			}, changes: map[int64]GroupRateChange{1: change, 2: change, 3: change}, err: tc.historyErr}
			rates := &availabilityPersonalRates{err: tc.ratesErr}
			if tc.fixed {
				rates.rates = map[int64]float64{1: 1}
			}
			svc := &APIKeyService{groupRepo: repo, userRepo: &visibilityUserRepo{user: &User{ID: 1}}, userSubRepo: &visibilitySubRepo{}, userGroupRateRepo: rates}
			groups, err := svc.GetAvailableGroups(context.Background(), 1)
			require.NoError(t, err)
			require.Len(t, groups, 1, "private and unsubscribed groups stay hidden")
			if tc.ratesErr == nil {
				require.Equal(t, []int64{1}, repo.queried, "history only queried for authorized groups")
				require.WithinDuration(t, now.Add(-48*time.Hour), repo.since, time.Second)
			}
			require.True(t, groups[0].IsNew, "history failures must not suppress independent launch metadata")
			require.Equal(t, tc.wantHint, groups[0].RecentRateChange != nil)
			require.Equal(t, !tc.fixed && tc.ratesErr == nil, groups[0].LaunchRate != nil)
			require.Equal(t, 1.0, groups[0].RateMultiplier, "rate hints cannot change actual pricing")
			if tc.wantHint {
				require.Equal(t, change, *groups[0].RecentRateChange)
			}
		})
	}
}

func TestAdminGroupLaunchSnapshotAndPartialUpdate(t *testing.T) {
	ctx := context.Background()
	launched := time.Now().Add(-72 * time.Hour).UTC()
	for _, publish := range []bool{false, true} {
		repo := &groupRepoStubForAdmin{}
		svc := &adminServiceImpl{groupRepo: repo}
		input := &CreateGroupInput{Name: "launch", Platform: PlatformOpenAI, RateMultiplier: 2}
		if publish {
			input.LaunchedAt = &launched
		}
		g, err := svc.CreateGroup(ctx, input)
		require.NoError(t, err)
		if publish {
			require.Equal(t, 2.0, *g.LaunchRate)
		} else {
			require.Nil(t, g.LaunchRate)
			require.Nil(t, g.LaunchedAt)
		}
	}
	rate := 2.0
	for _, tc := range []struct {
		name       string
		input      UpdateGroupInput
		wantLaunch bool
		wantRate   float64
	}{
		{name: "ordinary rate edit preserves old publication", input: UpdateGroupInput{RateMultiplier: ptrGroupAvailability(3.0)}, wantLaunch: true, wantRate: 2},
		{name: "same publication preserves snapshot", input: UpdateGroupInput{RateMultiplier: ptrGroupAvailability(3.0), LaunchedAt: &launched}, wantLaunch: true, wantRate: 2},
		{name: "republish snapshots new rate", input: UpdateGroupInput{RateMultiplier: ptrGroupAvailability(3.0), LaunchedAt: ptrGroupAvailability(time.Now())}, wantLaunch: true, wantRate: 3},
		{name: "unpublish clears snapshot", input: UpdateGroupInput{ClearLaunchedAt: true}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			repo := &groupRepoStubForAdmin{getByID: &Group{ID: 1, Name: "old", Platform: PlatformOpenAI, RateMultiplier: 2, Status: StatusActive, LaunchedAt: &launched, LaunchRate: &rate}}
			svc := &adminServiceImpl{groupRepo: repo}
			g, err := svc.UpdateGroup(ctx, 1, &tc.input)
			require.NoError(t, err)
			require.Equal(t, tc.wantLaunch, g.LaunchedAt != nil)
			if tc.wantLaunch {
				require.Equal(t, tc.wantRate, *g.LaunchRate)
			} else {
				require.Nil(t, g.LaunchRate)
			}
		})
	}
	clone := cloneGroupForDuplicate(&Group{LaunchedAt: &launched, LaunchRate: &rate}, "operation")
	require.Nil(t, clone.LaunchedAt, "copying a launched group must not automatically launch the copy")
	require.Nil(t, clone.LaunchRate)
}

func ptrGroupAvailability[T any](v T) *T { return &v }
