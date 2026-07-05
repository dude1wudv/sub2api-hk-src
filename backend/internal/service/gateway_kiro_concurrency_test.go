package service

import (
	"context"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
)

type kiroGatewayAccountRepo struct {
	AccountRepository
	account Account
}

func (r *kiroGatewayAccountRepo) GetByID(_ context.Context, id int64) (*Account, error) {
	if id != r.account.ID {
		return nil, ErrAccountNotFound
	}
	account := r.account
	return &account, nil
}

func (r *kiroGatewayAccountRepo) ListSchedulableByPlatforms(_ context.Context, platforms []string) ([]Account, error) {
	for _, platform := range platforms {
		if platform == r.account.Platform && r.account.IsSchedulable() {
			return []Account{r.account}, nil
		}
	}
	return nil, nil
}

func (r *kiroGatewayAccountRepo) UpdateLastUsed(context.Context, int64) error { return nil }

func (r *kiroGatewayAccountRepo) BatchUpdateLastUsed(context.Context, map[int64]time.Time) error {
	return nil
}

type kiroGatewayCache struct{}

func (c *kiroGatewayCache) GetSessionAccountID(context.Context, int64, string) (int64, error) {
	return 0, ErrAccountNotFound
}
func (c *kiroGatewayCache) SetSessionAccountID(context.Context, int64, string, int64, time.Duration) error {
	return nil
}
func (c *kiroGatewayCache) RefreshSessionTTL(context.Context, int64, string, time.Duration) error {
	return nil
}
func (c *kiroGatewayCache) DeleteSessionAccountID(context.Context, int64, string) error {
	return nil
}

type kiroConcurrencyCache struct {
	lastAcquireMax int
	loadMaxByID    map[int64]int
}

func (c *kiroConcurrencyCache) AcquireAccountSlot(_ context.Context, _ int64, maxConcurrency int, _ string) (bool, error) {
	c.lastAcquireMax = maxConcurrency
	return true, nil
}

func (c *kiroConcurrencyCache) ReleaseAccountSlot(context.Context, int64, string) error { return nil }
func (c *kiroConcurrencyCache) GetAccountConcurrency(context.Context, int64) (int, error) {
	return 0, nil
}
func (c *kiroConcurrencyCache) GetAccountConcurrencyBatch(_ context.Context, accountIDs []int64) (map[int64]int, error) {
	result := make(map[int64]int, len(accountIDs))
	for _, id := range accountIDs {
		result[id] = 0
	}
	return result, nil
}
func (c *kiroConcurrencyCache) IncrementAccountWaitCount(context.Context, int64, int) (bool, error) {
	return true, nil
}
func (c *kiroConcurrencyCache) DecrementAccountWaitCount(context.Context, int64) error { return nil }
func (c *kiroConcurrencyCache) GetAccountWaitingCount(context.Context, int64) (int, error) {
	return 0, nil
}
func (c *kiroConcurrencyCache) AcquireUserSlot(context.Context, int64, int, string) (bool, error) {
	return true, nil
}
func (c *kiroConcurrencyCache) ReleaseUserSlot(context.Context, int64, string) error { return nil }
func (c *kiroConcurrencyCache) GetUserConcurrency(context.Context, int64) (int, error) {
	return 0, nil
}
func (c *kiroConcurrencyCache) IncrementWaitCount(context.Context, int64, int) (bool, error) {
	return true, nil
}
func (c *kiroConcurrencyCache) DecrementWaitCount(context.Context, int64) error { return nil }
func (c *kiroConcurrencyCache) GetAccountsLoadBatch(_ context.Context, accounts []AccountWithConcurrency) (map[int64]*AccountLoadInfo, error) {
	c.loadMaxByID = make(map[int64]int, len(accounts))
	result := make(map[int64]*AccountLoadInfo, len(accounts))
	for _, acc := range accounts {
		c.loadMaxByID[acc.ID] = acc.MaxConcurrency
		result[acc.ID] = &AccountLoadInfo{AccountID: acc.ID}
	}
	return result, nil
}
func (c *kiroConcurrencyCache) GetUsersLoadBatch(_ context.Context, users []UserWithConcurrency) (map[int64]*UserLoadInfo, error) {
	result := make(map[int64]*UserLoadInfo, len(users))
	for _, user := range users {
		result[user.ID] = &UserLoadInfo{UserID: user.ID}
	}
	return result, nil
}
func (c *kiroConcurrencyCache) CleanupExpiredAccountSlots(context.Context, int64) error {
	return nil
}
func (c *kiroConcurrencyCache) CleanupExpiredAccountSlotKeys(context.Context) error {
	return nil
}
func (c *kiroConcurrencyCache) CleanupStaleProcessSlots(context.Context, string) error {
	return nil
}

func TestGatewayServiceSelectAccountCapsKiroGatewayConcurrency(t *testing.T) {
	account := Account{
		ID:          99,
		Platform:    PlatformAnthropic,
		Type:        AccountTypeAPIKey,
		Priority:    1,
		Status:      StatusActive,
		Schedulable: true,
		Concurrency: 12,
		Extra:       map[string]any{"source": "kiro-gateway"},
	}
	repo := &kiroGatewayAccountRepo{account: account}
	cache := &kiroConcurrencyCache{}
	cfg := &config.Config{RunMode: config.RunModeSimple}
	cfg.Gateway.Scheduling.LoadBatchEnabled = true

	svc := &GatewayService{
		accountRepo:        repo,
		cache:              &kiroGatewayCache{},
		cfg:                cfg,
		concurrencyService: NewConcurrencyService(cache),
	}

	result, err := svc.SelectAccountWithLoadAwareness(context.Background(), nil, "", "claude-3-5-sonnet-20241022", nil, "", 0)

	require.NoError(t, err)
	require.NotNil(t, result)
	require.True(t, result.Acquired)
	require.Equal(t, account.ID, result.Account.ID)
	require.Equal(t, KiroGatewayMaxAccountConcurrency, cache.loadMaxByID[account.ID])
	require.Equal(t, KiroGatewayMaxAccountConcurrency, cache.lastAcquireMax)
}
