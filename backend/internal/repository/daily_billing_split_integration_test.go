//go:build integration

package repository

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/Wei-Shaw/sub2api/internal/domain"
	"github.com/Wei-Shaw/sub2api/internal/service"
)

// dailyBillingFixture 创建一个「每日余额」专属分组 + 用户 + apikey + account，
// 返回 repo、各 ID 及一个 grant 工厂。
type dailyBillingFixture struct {
	repo       service.UsageBillingRepository
	grantRepo  *dailyGrantRepository
	userID     int64
	groupID    int64
	apiKeyID   int64
	accountID  int64
	multiplier float64
}

func newDailyBillingFixture(t *testing.T, balance, fallbackMultiplier float64) dailyBillingFixture {
	t.Helper()
	client := testEntClient(t)
	repo := NewUsageBillingRepository(client, integrationDB)
	grantRepo := &dailyGrantRepository{client: integrationEntClient, sql: integrationDB}

	user := mustCreateUser(t, client, &service.User{
		Email:        fmt.Sprintf("daily-bill-%s@example.com", uuid.NewString()),
		PasswordHash: "hash",
		Balance:      balance,
	})
	group := mustCreateGroup(t, client, &service.Group{
		Name:                    "daily-grp-" + uuid.NewString(),
		Platform:                service.PlatformAnthropic,
		RateMultiplier:          1.0,
		Status:                  service.StatusActive,
		SubscriptionType:        service.SubscriptionTypeStandard,
		DailyBalanceEnabled:     true,
		DailyFallbackMultiplier: fallbackMultiplier,
	})
	apiKey := mustCreateApiKey(t, client, &service.APIKey{
		UserID:  user.ID,
		Key:     "sk-daily-" + uuid.NewString(),
		Name:    "daily",
		GroupID: &group.ID,
	})
	account := mustCreateAccount(t, client, &service.Account{
		Name: "daily-acct-" + uuid.NewString(),
		Type: service.AccountTypeAPIKey,
	})

	t.Cleanup(func() {
		_, _ = integrationDB.ExecContext(context.Background(),
			"DELETE FROM daily_balance_grants WHERE user_id = $1", user.ID)
	})

	return dailyBillingFixture{
		repo:       repo,
		grantRepo:  grantRepo,
		userID:     user.ID,
		groupID:    group.ID,
		apiKeyID:   apiKey.ID,
		accountID:  account.ID,
		multiplier: fallbackMultiplier,
	}
}

func (f dailyBillingFixture) addGrant(t *testing.T, remaining float64, expiresIn time.Duration) int64 {
	t.Helper()
	id, err := f.grantRepo.CreateGrant(context.Background(), &service.DailyBalanceGrant{
		UserID:    f.userID,
		GroupID:   f.groupID,
		Amount:    remaining,
		Remaining: remaining,
		Status:    domain.DailyGrantStatusActive,
		Source:    domain.DailyGrantSourceAdmin,
		ExpiresAt: time.Now().Add(expiresIn),
	})
	require.NoError(t, err)
	return id
}

func (f dailyBillingFixture) apply(t *testing.T, base float64) *service.UsageBillingApplyResult {
	t.Helper()
	res, err := f.repo.Apply(context.Background(), &service.UsageBillingCommand{
		RequestID:   uuid.NewString(),
		APIKeyID:    f.apiKeyID,
		UserID:      f.userID,
		AccountID:   f.accountID,
		AccountType: service.AccountTypeAPIKey,
		GroupID:     &f.groupID,
		BalanceCost: base,
	})
	require.NoError(t, err)
	require.NotNil(t, res)
	require.True(t, res.Applied)
	return res
}

func (f dailyBillingFixture) balance(t *testing.T) float64 {
	t.Helper()
	var b float64
	require.NoError(t, integrationDB.QueryRowContext(context.Background(),
		"SELECT balance FROM users WHERE id = $1", f.userID).Scan(&b))
	return b
}

func (f dailyBillingFixture) grantRemaining(t *testing.T, id int64) (float64, string) {
	t.Helper()
	var rem float64
	var status string
	require.NoError(t, integrationDB.QueryRowContext(context.Background(),
		"SELECT remaining, status FROM daily_balance_grants WHERE id = $1", id).Scan(&rem, &status))
	return rem, status
}

// 场景1：纯 Grant —— base ≤ grant 总额，overflow=0，余额不动。
func TestDailyBilling_PureGrant(t *testing.T) {
	f := newDailyBillingFixture(t, 100, 1.5)
	g := f.addGrant(t, 10, 24*time.Hour)

	res := f.apply(t, 4) // base=4，全部由 grant 覆盖

	require.InDelta(t, 4.0, res.DailyGrantSpent, 1e-9)
	require.InDelta(t, 0.0, res.LongTermSpent, 1e-9)
	require.InDelta(t, 100.0, f.balance(t), 1e-9, "余额不应变动")

	rem, status := f.grantRemaining(t, g)
	require.InDelta(t, 6.0, rem, 1e-9)
	require.Equal(t, domain.DailyGrantStatusActive, status)
}

// 场景2：纯长期 —— 无有效 Grant，全额 ×倍率 从余额扣。
func TestDailyBilling_PureLongTerm(t *testing.T) {
	f := newDailyBillingFixture(t, 100, 1.5)
	// 一笔已过期的 Grant，不应被消费
	expired := f.addGrant(t, 50, -time.Hour)

	res := f.apply(t, 8) // base=8，无有效 grant → 8*1.5=12 从余额

	require.InDelta(t, 0.0, res.DailyGrantSpent, 1e-9)
	require.InDelta(t, 12.0, res.LongTermSpent, 1e-9)
	require.InDelta(t, 88.0, f.balance(t), 1e-9)

	rem, _ := f.grantRemaining(t, expired)
	require.InDelta(t, 50.0, rem, 1e-9, "过期 grant 不应被扣")
}

// 场景3：跨界拆分 —— 部分 Grant + 部分余额×倍率。
func TestDailyBilling_SplitAcrossBoundary(t *testing.T) {
	f := newDailyBillingFixture(t, 100, 1.5)
	g := f.addGrant(t, 6, 24*time.Hour)

	res := f.apply(t, 10) // grant 扣 6，overflow 4 → 4*1.5=6 从余额

	require.InDelta(t, 6.0, res.DailyGrantSpent, 1e-9)
	require.InDelta(t, 6.0, res.LongTermSpent, 1e-9)
	require.InDelta(t, 94.0, f.balance(t), 1e-9)

	rem, status := f.grantRemaining(t, g)
	require.InDelta(t, 0.0, rem, 1e-9)
	require.Equal(t, domain.DailyGrantStatusExhausted, status)
}

// 场景4：多 Grant FIFO + 恰好用尽最早到期的。
func TestDailyBilling_FIFOExhaust(t *testing.T) {
	f := newDailyBillingFixture(t, 100, 2.0)
	sooner := f.addGrant(t, 3, 2*time.Hour)  // 先到期
	later := f.addGrant(t, 10, 20*time.Hour) // 后到期

	res := f.apply(t, 5) // 先扣 sooner 的 3，再扣 later 的 2

	require.InDelta(t, 5.0, res.DailyGrantSpent, 1e-9)
	require.InDelta(t, 0.0, res.LongTermSpent, 1e-9)
	require.InDelta(t, 100.0, f.balance(t), 1e-9)

	rs, ss := f.grantRemaining(t, sooner)
	require.InDelta(t, 0.0, rs, 1e-9)
	require.Equal(t, domain.DailyGrantStatusExhausted, ss)

	rl, sl := f.grantRemaining(t, later)
	require.InDelta(t, 8.0, rl, 1e-9)
	require.Equal(t, domain.DailyGrantStatusActive, sl)
}

// 场景5：并发双请求争抢同一 Grant —— 无超卖、无免费。
// Grant=10，两请求各 base=8。一个吃满 grant 的一部分、另一个走余额兜底。
// 合计 grant 扣减 ≤ 10，且两请求的计费总额守恒（不免费）。
func TestDailyBilling_ConcurrentContention(t *testing.T) {
	f := newDailyBillingFixture(t, 1000, 1.5)
	g := f.addGrant(t, 10, 24*time.Hour)

	const n = 8
	const base = 4.0
	var grantTotal int64 // 以微分单位累加避免浮点竞态
	var longTotal int64
	var wg sync.WaitGroup
	wg.Add(n)
	for i := 0; i < n; i++ {
		go func() {
			defer wg.Done()
			res, err := f.repo.Apply(context.Background(), &service.UsageBillingCommand{
				RequestID:   uuid.NewString(),
				APIKeyID:    f.apiKeyID,
				UserID:      f.userID,
				AccountID:   f.accountID,
				AccountType: service.AccountTypeAPIKey,
				GroupID:     &f.groupID,
				BalanceCost: base,
			})
			require.NoError(t, err)
			atomic.AddInt64(&grantTotal, int64(res.DailyGrantSpent*1e6))
			atomic.AddInt64(&longTotal, int64(res.LongTermSpent*1e6))
		}()
	}
	wg.Wait()

	grantSpent := float64(atomic.LoadInt64(&grantTotal)) / 1e6
	// grant 总扣减不超过初始 10，且不超卖
	require.LessOrEqual(t, grantSpent, 10.0+1e-6, "grant 不应超卖")

	rem, _ := f.grantRemaining(t, g)
	require.InDelta(t, 10.0-grantSpent, rem, 1e-3, "grant 剩余应与扣减守恒")

	// 计费守恒（不免费）：每个请求 base=4，
	// 由 grant 覆盖的部分按 1.0 计、溢出部分按 1.5 计。
	// 总 base = n*4 = 32；grant 覆盖 grantSpent，溢出 (32-grantSpent) 应 ×1.5 落到余额。
	expectedLong := (float64(n)*base - grantSpent) * 1.5
	longSpent := float64(atomic.LoadInt64(&longTotal)) / 1e6
	require.InDelta(t, expectedLong, longSpent, 1e-2, "长期余额扣减应与溢出守恒，无免费")

	// 余额实际扣减应等于 longSpent
	require.InDelta(t, 1000.0-longSpent, f.balance(t), 1e-2)
}

// 回归：非专属分组（daily_balance_enabled=false）扣费与改造前逐字节一致——全额扣余额，不碰 grant。
func TestDailyBilling_NonExclusiveGroupUnchanged(t *testing.T) {
	client := testEntClient(t)
	repo := NewUsageBillingRepository(client, integrationDB)
	user := mustCreateUser(t, client, &service.User{
		Email:        fmt.Sprintf("nondaily-%s@example.com", uuid.NewString()),
		PasswordHash: "hash",
		Balance:      50,
	})
	group := mustCreateGroup(t, client, &service.Group{
		Name:             "nondaily-grp-" + uuid.NewString(),
		Platform:         service.PlatformAnthropic,
		RateMultiplier:   1.0,
		Status:           service.StatusActive,
		SubscriptionType: service.SubscriptionTypeStandard,
		// DailyBalanceEnabled 默认 false
	})
	apiKey := mustCreateApiKey(t, client, &service.APIKey{
		UserID: user.ID, Key: "sk-nd-" + uuid.NewString(), Name: "nd", GroupID: &group.ID,
	})
	account := mustCreateAccount(t, client, &service.Account{
		Name: "nd-acct-" + uuid.NewString(), Type: service.AccountTypeAPIKey,
	})

	res, err := repo.Apply(context.Background(), &service.UsageBillingCommand{
		RequestID:   uuid.NewString(),
		APIKeyID:    apiKey.ID,
		UserID:      user.ID,
		AccountID:   account.ID,
		AccountType: service.AccountTypeAPIKey,
		GroupID:     &group.ID,
		BalanceCost: 7,
	})
	require.NoError(t, err)
	require.InDelta(t, 0.0, res.DailyGrantSpent, 1e-9)
	require.InDelta(t, 7.0, res.LongTermSpent, 1e-9, "非专属分组全额扣余额")

	var b float64
	require.NoError(t, integrationDB.QueryRowContext(context.Background(),
		"SELECT balance FROM users WHERE id = $1", user.ID).Scan(&b))
	require.InDelta(t, 43.0, b, 1e-9)
}
