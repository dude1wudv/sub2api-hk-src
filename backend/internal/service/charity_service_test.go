package service

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type stubCharityRepo struct {
	settings  *CharityPoolSettings
	groups    map[int64]*CharityGroup
	nextID    int64
	daily     map[string][]CharityGroupDailyUsage
	ipDaily   map[string]*CharityIPDailyUsage
	trend     []CharityDailyTrendPoint
}

func newStubCharityRepo() *stubCharityRepo {
	sysUID := int64(999)
	return &stubCharityRepo{
		settings: &CharityPoolSettings{
			Enabled:          true,
			DailyBudgetUSD:   100.0,
			MaxConcurrency:   20,
			AutoRotateDaily:  true,
			RotationTimezone: CharityDefaultTimezone,
			SiteTitle:        "XJTU Free AI",
			HostAllowlist:    []string{"free.xjtu.shop"},
			SystemUserID:     &sysUID,
		},
		groups:  map[int64]*CharityGroup{},
		nextID:  1,
		daily:   map[string][]CharityGroupDailyUsage{},
		ipDaily: map[string]*CharityIPDailyUsage{},
	}
}

func (r *stubCharityRepo) GetSettings(context.Context) (*CharityPoolSettings, error) {
	cp := *r.settings
	cp.HostAllowlist = append([]string(nil), r.settings.HostAllowlist...)
	return &cp, nil
}
func (r *stubCharityRepo) UpdateSettings(_ context.Context, s *CharityPoolSettings) (*CharityPoolSettings, error) {
	cp := *s
	cp.SystemUserID = r.settings.SystemUserID
	r.settings = &cp
	return r.GetSettings(context.Background())
}
func (r *stubCharityRepo) SetSystemUserID(_ context.Context, uid int64) error {
	r.settings.SystemUserID = &uid
	return nil
}
func (r *stubCharityRepo) ListGroups(context.Context) ([]CharityGroup, error) {
	out := make([]CharityGroup, 0, len(r.groups))
	for _, g := range r.groups {
		out = append(out, *g)
	}
	return out, nil
}
func (r *stubCharityRepo) GetGroupByID(_ context.Context, id int64) (*CharityGroup, error) {
	g, ok := r.groups[id]
	if !ok {
		return nil, ErrCharityGroupNotFound
	}
	cp := *g
	return &cp, nil
}
func (r *stubCharityRepo) GetGroupByGroupID(_ context.Context, gid int64) (*CharityGroup, error) {
	for _, g := range r.groups {
		if g.GroupID == gid {
			cp := *g
			return &cp, nil
		}
	}
	return nil, ErrCharityGroupNotFound
}
func (r *stubCharityRepo) GetGroupByActiveAPIKeyID(_ context.Context, keyID int64) (*CharityGroup, error) {
	for _, g := range r.groups {
		if g.ActiveAPIKeyID != nil && *g.ActiveAPIKeyID == keyID {
			cp := *g
			return &cp, nil
		}
	}
	return nil, ErrCharityGroupNotFound
}
func (r *stubCharityRepo) CreateGroup(_ context.Context, g *CharityGroup) (*CharityGroup, error) {
	cp := *g
	cp.ID = r.nextID
	r.nextID++
	cp.GroupStatus = StatusActive
	r.groups[cp.ID] = &cp
	return r.GetGroupByID(context.Background(), cp.ID)
}
func (r *stubCharityRepo) UpdateGroup(_ context.Context, g *CharityGroup) (*CharityGroup, error) {
	cur, ok := r.groups[g.ID]
	if !ok {
		return nil, ErrCharityGroupNotFound
	}
	cur.DisplayName = g.DisplayName
	cur.Description = g.Description
	cur.Enabled = g.Enabled
	cur.DailyLimitUSD = g.DailyLimitUSD
	cur.MaxConcurrency = g.MaxConcurrency
	cur.SortOrder = g.SortOrder
	return r.GetGroupByID(context.Background(), g.ID)
}
func (r *stubCharityRepo) DeleteGroup(_ context.Context, id int64) error {
	delete(r.groups, id)
	return nil
}
func (r *stubCharityRepo) SetActiveAPIKey(_ context.Context, id int64, keyID *int64, rotatedDate string, rotatedAt time.Time) error {
	cur, ok := r.groups[id]
	if !ok {
		return ErrCharityGroupNotFound
	}
	cur.ActiveAPIKeyID = keyID
	cur.LastRotatedDate = rotatedDate
	cur.LastRotatedAt = &rotatedAt
	return nil
}
func (r *stubCharityRepo) GetDailyGroupUsage(_ context.Context, usageDate string) ([]CharityGroupDailyUsage, error) {
	return r.daily[usageDate], nil
}
func (r *stubCharityRepo) GetDailyIPUsage(_ context.Context, usageDate string, clientIP string) (*CharityIPDailyUsage, error) {
	if u, ok := r.ipDaily[usageDate+"|"+clientIP]; ok {
		return u, nil
	}
	return &CharityIPDailyUsage{UsageDate: usageDate, ClientIP: clientIP}, nil
}
func (r *stubCharityRepo) GetDailyTrend(context.Context, int) ([]CharityDailyTrendPoint, error) {
	return r.trend, nil
}

type stubCharityCache struct {
	groupCosts map[string]map[int64]float64
	ipCosts    map[string]float64
	rpmCount   map[string]int
	globalSlot map[string]bool
	groupSlot  map[int64]map[string]bool
	ipSlot     map[string]map[string]bool
	rotLocks   map[string]bool
}

func newStubCharityCache() *stubCharityCache {
	return &stubCharityCache{
		groupCosts: map[string]map[int64]float64{},
		ipCosts:    map[string]float64{},
		rpmCount:   map[string]int{},
		globalSlot: map[string]bool{},
		groupSlot:  map[int64]map[string]bool{},
		ipSlot:     map[string]map[string]bool{},
		rotLocks:   map[string]bool{},
	}
}

func (c *stubCharityCache) GetDailyGroupCosts(_ context.Context, d string) (map[int64]float64, bool, error) {
	m, ok := c.groupCosts[d]
	if !ok {
		return nil, false, nil
	}
	cp := make(map[int64]float64, len(m))
	for k, v := range m {
		cp[k] = v
	}
	return cp, true, nil
}
func (c *stubCharityCache) SetDailyGroupCosts(_ context.Context, d string, costs map[int64]float64) error {
	cp := make(map[int64]float64, len(costs))
	for k, v := range costs {
		cp[k] = v
	}
	c.groupCosts[d] = cp
	return nil
}
func (c *stubCharityCache) IncrDailyGroupCost(_ context.Context, d string, gid int64, delta float64) error {
	if _, ok := c.groupCosts[d]; ok {
		c.groupCosts[d][gid] += delta
	}
	return nil
}
func (c *stubCharityCache) GetDailyIPCost(_ context.Context, d, ip string) (float64, bool, error) {
	v, ok := c.ipCosts[d+"|"+ip]
	return v, ok, nil
}
func (c *stubCharityCache) SetDailyIPCost(_ context.Context, d, ip string, v float64) error {
	c.ipCosts[d+"|"+ip] = v
	return nil
}
func (c *stubCharityCache) IncrDailyIPCost(_ context.Context, d, ip string, delta float64) error {
	if _, ok := c.ipCosts[d+"|"+ip]; ok {
		c.ipCosts[d+"|"+ip] += delta
	}
	return nil
}
func (c *stubCharityCache) CheckAndIncrIPRPM(_ context.Context, ip string, limit int) (bool, error) {
	if limit <= 0 {
		return true, nil
	}
	if c.rpmCount[ip] >= limit {
		return false, nil
	}
	c.rpmCount[ip]++
	return true, nil
}
func (c *stubCharityCache) AcquireConcurrencySlots(_ context.Context, reqID string, globalLimit int, gid int64, groupLimit int, ip string, ipLimit int) (bool, string, error) {
	if globalLimit > 0 && len(c.globalSlot) >= globalLimit {
		return false, "global", nil
	}
	if groupLimit > 0 && len(c.groupSlot[gid]) >= groupLimit {
		return false, "group", nil
	}
	if ip != "" && ipLimit > 0 && len(c.ipSlot[ip]) >= ipLimit {
		return false, "ip", nil
	}
	c.globalSlot[reqID] = true
	if c.groupSlot[gid] == nil {
		c.groupSlot[gid] = map[string]bool{}
	}
	c.groupSlot[gid][reqID] = true
	if ip != "" {
		if c.ipSlot[ip] == nil {
			c.ipSlot[ip] = map[string]bool{}
		}
		c.ipSlot[ip][reqID] = true
	}
	return true, "", nil
}
func (c *stubCharityCache) ReleaseConcurrencySlots(_ context.Context, reqID string, gid int64, ip string) error {
	delete(c.globalSlot, reqID)
	delete(c.groupSlot[gid], reqID)
	delete(c.ipSlot[ip], reqID)
	return nil
}
func (c *stubCharityCache) GetLiveConcurrency(_ context.Context, gids []int64) (*CharityLiveConcurrency, error) {
	byGroup := make(map[int64]int, len(gids))
	for _, gid := range gids {
		byGroup[gid] = len(c.groupSlot[gid])
	}
	return &CharityLiveConcurrency{Global: len(c.globalSlot), ByGroup: byGroup}, nil
}
func (c *stubCharityCache) GetLiveIPConcurrency(_ context.Context, ip string) (int, error) {
	return len(c.ipSlot[ip]), nil
}
func (c *stubCharityCache) TryAcquireRotationLock(_ context.Context, dateKey string, _ time.Duration) (bool, error) {
	if c.rotLocks[dateKey] {
		return false, nil
	}
	c.rotLocks[dateKey] = true
	return true, nil
}

func TestCharityAdmitRequest_PoolAndPerGroupSubCaps(t *testing.T) {
	repo := newStubCharityRepo()
	cache := newStubCharityCache()
	svc := NewCharityService(repo, cache, nil, nil, nil, nil, nil, nil)
	fixedNow := time.Date(2026, 11, 28, 10, 0, 0, 0, time.FixedZone("CST", 8*3600))
	svc.nowFn = func() time.Time { return fixedNow }

	claudeLimit := 50.0
	claudeKeyID := int64(101)
	deepseekKeyID := int64(102)
	claudeGID := int64(10)
	deepseekGID := int64(20)

	repo.groups[1] = &CharityGroup{
		ID:              1,
		GroupID:         claudeGID,
		DisplayName:     "Claude 公益组",
		Enabled:         true,
		DailyLimitUSD:   &claudeLimit,
		MaxConcurrency:  2,
		ActiveAPIKeyID:  &claudeKeyID,
		ActiveAPIKey:    "sk-free-claude",
		LastRotatedDate: "2026-11-28",
		GroupStatus:     StatusActive,
	}
	repo.groups[2] = &CharityGroup{
		ID:              2,
		GroupID:         deepseekGID,
		DisplayName:     "DeepSeek 公益组",
		Enabled:         true,
		DailyLimitUSD:   nil, // 不设单独子上限，共享总池
		MaxConcurrency:  0,
		ActiveAPIKeyID:  &deepseekKeyID,
		ActiveAPIKey:    "sk-free-deepseek",
		LastRotatedDate: "2026-11-28",
		GroupStatus:     StatusActive,
	}

	// Claude 组已用 $50（触及单组子上限），但总池才用 $50 / $100
	repo.daily["2026-11-28"] = []CharityGroupDailyUsage{
		{UsageDate: "2026-11-28", GroupID: claudeGID, TotalCostUSD: 50.0, RequestCount: 10},
	}

	claudeKey := &APIKey{ID: claudeKeyID, UserID: 999, GroupID: &claudeGID, Key: "sk-free-claude", Status: StatusAPIKeyActive}
	deepseekKey := &APIKey{ID: deepseekKeyID, UserID: 999, GroupID: &deepseekGID, Key: "sk-free-deepseek", Status: StatusAPIKeyActive}

	// Claude 组应因单组子上限返回 429
	_, err := svc.AdmitRequest(context.Background(), claudeKey, "10.0.0.1", false)
	require.ErrorIs(t, err, ErrCharityGroupLimitExhausted)

	// DeepSeek 组未设独立子上限，应正常放行并共享剩余 $50 总池
	res, err := svc.AdmitRequest(context.Background(), deepseekKey, "10.0.0.1", false)
	require.NoError(t, err)
	require.True(t, res.IsCharity)
	require.NotNil(t, res.Meta)
	require.Equal(t, deepseekGID, res.Meta.GroupID)
	require.Equal(t, "2026-11-28", res.Meta.UsageDate)
	res.Release()

	// 计入 DeepSeek 折前消费 $50，总池达到 $100 → DeepSeek 也被拦截
	svc.RecordBilledUsage(context.Background(), "2026-11-28", deepseekGID, "10.0.0.1", 50.0)
	_, err = svc.AdmitRequest(context.Background(), deepseekKey, "10.0.0.2", false)
	require.ErrorIs(t, err, ErrCharityDailyBudgetExhausted)

	// 昨日旧轮换 Key 应返回 ErrCharityKeyRotated
	oldKey := &APIKey{ID: 888, UserID: 999, GroupID: &deepseekGID, Key: "sk-free-old", Status: StatusAPIKeyExpired}
	_, err = svc.AdmitRequest(context.Background(), oldKey, "10.0.0.2", false)
	require.ErrorIs(t, err, ErrCharityKeyRotated)
}

func TestCharityAdmitRequest_ConcurrencyAndIPLimits(t *testing.T) {
	repo := newStubCharityRepo()
	repo.settings.MaxConcurrency = 2
	repo.settings.PerIPMaxConcurrency = 1
	repo.settings.PerIPRPMLimit = 2
	cache := newStubCharityCache()
	svc := NewCharityService(repo, cache, nil, nil, nil, nil, nil, nil)

	gid := int64(10)
	kid := int64(101)
	repo.groups[1] = &CharityGroup{
		ID:             1,
		GroupID:        gid,
		Enabled:        true,
		MaxConcurrency: 2,
		ActiveAPIKeyID: &kid,
		ActiveAPIKey:   "sk-free-test",
		GroupStatus:    StatusActive,
	}
	key := &APIKey{ID: kid, UserID: 999, GroupID: &gid, Key: "sk-free-test", Status: StatusAPIKeyActive}

	r1, err := svc.AdmitRequest(context.Background(), key, "1.2.3.4", false)
	require.NoError(t, err)

	// 同 IP 并发上限 1 → 第二并发立即被拒绝
	_, err = svc.AdmitRequest(context.Background(), key, "1.2.3.4", false)
	require.ErrorIs(t, err, ErrCharityConcurrencyExceeded)

	// 释放后再次请求成功，但消耗第 2 次 RPM（注意上面拒绝前已计入 RPM）
	r1.Release()
	_, err = svc.AdmitRequest(context.Background(), key, "1.2.3.4", false)
	require.ErrorIs(t, err, ErrCharityIPRPMExceeded)

	// 换另一 IP 可正常请求
	r2, err := svc.AdmitRequest(context.Background(), key, "5.6.7.8", false)
	require.NoError(t, err)
	r2.Release()
}

func TestBuildUsageBillingCommand_CharityUsesPreDiscountTotalCostAndZeroBalance(t *testing.T) {
	gid := int64(10)
	cmd := buildUsageBillingCommand("req-charity-1", &UsageLog{
		Model:        "gpt-5.4",
		InputTokens:  1000,
		OutputTokens: 500,
	}, &postUsageBillingParams{
		Cost: &CostBreakdown{
			TotalCost:  1.25, // 折前 1x 含 Fast/Ultra 档位倍率
			ActualCost: 0.25, // 即使分组有 0.2x 折扣，公益池也必须按折前 TotalCost 扣减
		},
		User:    &User{ID: 999},
		APIKey:  &APIKey{ID: 101, UserID: 999, GroupID: &gid},
		Account: &Account{ID: 55, Type: AccountTypeAPIKey},
		Charity: &CharityBillingMeta{
			GroupID:   gid,
			UsageDate: "2026-11-28",
			ClientIP:  "10.1.2.3",
		},
	})
	require.NotNil(t, cmd)
	require.Equal(t, 0.0, cmd.BalanceCost)
	require.Equal(t, 0.0, cmd.SubscriptionCost)
	require.Equal(t, gid, cmd.CharityGroupID)
	require.Equal(t, "2026-11-28", cmd.CharityUsageDate)
	require.InDelta(t, 1.25, cmd.CharityCostUSD, 1e-8)
	require.Equal(t, int64(1500), cmd.CharityTokens)
	require.Equal(t, "10.1.2.3", cmd.CharityClientIP)
}
