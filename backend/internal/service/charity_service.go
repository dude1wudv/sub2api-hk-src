package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"slices"
	"sort"
	"strings"
	"sync"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"golang.org/x/crypto/bcrypt"
)

const (
	charitySnapshotTTL           = 5 * time.Second
	charitySystemUserConcurrency = 100000
	charityReleaseTimeout        = 3 * time.Second
)

type charitySnapshot struct {
	loadedAt      time.Time
	settings      *CharityPoolSettings
	groups        []CharityGroup
	byGroupID     map[int64]*CharityGroup
	byActiveKeyID map[int64]*CharityGroup
	hosts         map[string]struct{}
}

// CharityAdmissionResult 描述一次请求通过公益池准入后的结果。
type CharityAdmissionResult struct {
	IsCharity bool
	Meta      *CharityBillingMeta
	Release   func()
}

// CharityService 编排公益池配置、分组与 Key 轮换、热路径准入与公开/管理看板查询。
type CharityService struct {
	repo          CharityRepository
	cache         CharityCache
	userRepo      UserRepository
	groupRepo     GroupRepository
	apiKeyRepo    APIKeyRepository
	apiKeyService *APIKeyService
	adminService  AdminService
	plazaService  *ModelPlazaService
	modelCatalog  *GatewayModelCatalogService

	mu       sync.RWMutex
	snapshot *charitySnapshot
	nowFn    func() time.Time
}

// NewCharityService 创建公益池服务。
func NewCharityService(
	repo CharityRepository,
	cache CharityCache,
	userRepo UserRepository,
	groupRepo GroupRepository,
	apiKeyRepo APIKeyRepository,
	apiKeyService *APIKeyService,
	adminService AdminService,
	plazaService *ModelPlazaService,
	modelCatalog *GatewayModelCatalogService,
) *CharityService {
	return &CharityService{
		repo:          repo,
		cache:         cache,
		userRepo:      userRepo,
		groupRepo:     groupRepo,
		apiKeyRepo:    apiKeyRepo,
		apiKeyService: apiKeyService,
		adminService:  adminService,
		plazaService:  plazaService,
		modelCatalog:  modelCatalog,
		nowFn:         time.Now,
	}
}

func (s *CharityService) now() time.Time {
	if s != nil && s.nowFn != nil {
		return s.nowFn()
	}
	return time.Now()
}

// InvalidateSnapshot 主动失效内存配置快照。
func (s *CharityService) InvalidateSnapshot() {
	if s == nil {
		return
	}
	s.mu.Lock()
	s.snapshot = nil
	s.mu.Unlock()
}

func (s *CharityService) getSnapshot(ctx context.Context) (*charitySnapshot, error) {
	if s == nil || s.repo == nil {
		return nil, errors.New("charity service repository is nil")
	}
	s.mu.RLock()
	snap := s.snapshot
	s.mu.RUnlock()
	if snap != nil && s.now().Sub(snap.loadedAt) < charitySnapshotTTL {
		return snap, nil
	}

	settings, err := s.repo.GetSettings(ctx)
	if err != nil {
		return nil, err
	}
	groups, err := s.repo.ListGroups(ctx)
	if err != nil {
		return nil, err
	}

	byGroupID := make(map[int64]*CharityGroup, len(groups))
	byActiveKeyID := make(map[int64]*CharityGroup, len(groups))
	for i := range groups {
		g := &groups[i]
		byGroupID[g.GroupID] = g
		if g.ActiveAPIKeyID != nil && *g.ActiveAPIKeyID > 0 {
			byActiveKeyID[*g.ActiveAPIKeyID] = g
		}
	}
	hosts := make(map[string]struct{}, len(settings.HostAllowlist))
	for _, h := range settings.HostAllowlist {
		if norm := NormalizeCharityHost(h); norm != "" {
			hosts[norm] = struct{}{}
		}
	}

	fresh := &charitySnapshot{
		loadedAt:      s.now(),
		settings:      settings,
		groups:        groups,
		byGroupID:     byGroupID,
		byActiveKeyID: byActiveKeyID,
		hosts:         hosts,
	}
	s.mu.Lock()
	s.snapshot = fresh
	s.mu.Unlock()
	return fresh, nil
}

// IsCharityHost 判断给定 Host 头是否属于公益站独立域名（如 free.xjtu.shop）。
func (s *CharityService) IsCharityHost(ctx context.Context, rawHost string) bool {
	if s == nil {
		return false
	}
	norm := NormalizeCharityHost(rawHost)
	if norm == "" {
		return false
	}
	snap, err := s.getSnapshot(ctx)
	if err != nil || snap == nil {
		return norm == "free.xjtu.shop"
	}
	_, ok := snap.hosts[norm]
	return ok
}

// IsCharityAPIKey 判断给定 APIKey 是否属于公益池（含已轮换的旧公益 Key）。
func (s *CharityService) IsCharityAPIKey(ctx context.Context, apiKey *APIKey) bool {
	if s == nil || apiKey == nil {
		return false
	}
	snap, err := s.getSnapshot(ctx)
	if err != nil || snap == nil || snap.settings == nil {
		return strings.HasPrefix(apiKey.Key, CharityKeyPrefix)
	}
	if snap.settings.SystemUserID != nil && *snap.settings.SystemUserID > 0 && apiKey.UserID == *snap.settings.SystemUserID {
		return true
	}
	if _, ok := snap.byActiveKeyID[apiKey.ID]; ok {
		return true
	}
	return false
}

// AdmitRequest 在网关鉴权热路径校验公益 Key 的开关、轮换态、总预算、分组子上限、单 IP 限流与并发租约。
func (s *CharityService) AdmitRequest(ctx context.Context, apiKey *APIKey, clientIP string, skipBilling bool) (*CharityAdmissionResult, error) {
	if s == nil || apiKey == nil {
		return &CharityAdmissionResult{IsCharity: false}, nil
	}
	snap, err := s.getSnapshot(ctx)
	if err != nil {
		if strings.HasPrefix(apiKey.Key, CharityKeyPrefix) {
			return &CharityAdmissionResult{IsCharity: true}, infraerrors.ServiceUnavailable("CHARITY_UNAVAILABLE", "公益池服务暂不可用")
		}
		return &CharityAdmissionResult{IsCharity: false}, nil
	}

	isCharity := false
	if snap.settings.SystemUserID != nil && *snap.settings.SystemUserID > 0 && apiKey.UserID == *snap.settings.SystemUserID {
		isCharity = true
	} else if _, ok := snap.byActiveKeyID[apiKey.ID]; ok {
		isCharity = true
	}
	if !isCharity {
		return &CharityAdmissionResult{IsCharity: false}, nil
	}

	if !snap.settings.Enabled {
		return &CharityAdmissionResult{IsCharity: true}, ErrCharityDisabled
	}

	var cg *CharityGroup
	if apiKey.GroupID != nil {
		cg = snap.byGroupID[*apiKey.GroupID]
	}
	if cg == nil {
		cg = snap.byActiveKeyID[apiKey.ID]
	}
	if cg == nil || !cg.Enabled {
		return &CharityAdmissionResult{IsCharity: true}, ErrCharityGroupDisabled
	}
	if cg.ActiveAPIKeyID == nil || *cg.ActiveAPIKeyID != apiKey.ID {
		return &CharityAdmissionResult{IsCharity: true}, ErrCharityKeyRotated
	}

	usageDate, _, _ := CharityTimeWindow(s.now(), snap.settings.RotationTimezone)
	meta := &CharityBillingMeta{
		CharityGroupID: cg.ID,
		GroupID:        cg.GroupID,
		ClientIP:       strings.TrimSpace(clientIP),
		UsageDate:      usageDate,
	}
	if skipBilling {
		return &CharityAdmissionResult{IsCharity: true, Meta: meta, Release: func() {}}, nil
	}

	// 1. 校验总预算 & 分组子上限（折前 TotalCost 累计）
	if snap.settings.DailyBudgetUSD > 0 || cg.HasDailyLimit() {
		groupCosts, loadErr := s.loadDailyGroupCosts(ctx, usageDate)
		if loadErr != nil {
			slog.Warn("charity: failed to load daily group costs", "error", loadErr)
			return &CharityAdmissionResult{IsCharity: true}, infraerrors.ServiceUnavailable("CHARITY_QUOTA_CHECK_FAILED", "公益额度检查暂时不可用，请稍后重试")
		}
		var poolUsed float64
		for _, c := range groupCosts {
			poolUsed += c
		}
		if snap.settings.DailyBudgetUSD > 0 && poolUsed >= snap.settings.DailyBudgetUSD {
			return &CharityAdmissionResult{IsCharity: true}, ErrCharityDailyBudgetExhausted
		}
		if cg.HasDailyLimit() && groupCosts[cg.GroupID] >= *cg.DailyLimitUSD {
			return &CharityAdmissionResult{IsCharity: true}, ErrCharityGroupLimitExhausted
		}
	}

	// 2. 校验可选单 IP 日金额上限
	if snap.settings.PerIPDailyLimitUSD > 0 && meta.ClientIP != "" {
		ipUsed, loadErr := s.loadDailyIPCost(ctx, usageDate, meta.ClientIP)
		if loadErr != nil {
			slog.Warn("charity: failed to load daily ip cost", "ip", MaskCharityIP(meta.ClientIP), "error", loadErr)
			return &CharityAdmissionResult{IsCharity: true}, infraerrors.ServiceUnavailable("CHARITY_QUOTA_CHECK_FAILED", "公益额度检查暂时不可用，请稍后重试")
		}
		if ipUsed >= snap.settings.PerIPDailyLimitUSD {
			return &CharityAdmissionResult{IsCharity: true}, ErrCharityIPDailyLimitExhausted
		}
	}

	// 3. 校验可选单 IP RPM
	if snap.settings.PerIPRPMLimit > 0 && meta.ClientIP != "" {
		if s.cache == nil {
			return &CharityAdmissionResult{IsCharity: true}, infraerrors.ServiceUnavailable("CHARITY_RATE_CHECK_FAILED", "公益频率检查暂时不可用，请稍后重试")
		}
		allowed, rpmErr := s.cache.CheckAndIncrIPRPM(ctx, meta.ClientIP, snap.settings.PerIPRPMLimit)
		if rpmErr != nil {
			slog.Warn("charity: failed to check ip rpm", "error", rpmErr)
			return &CharityAdmissionResult{IsCharity: true}, infraerrors.ServiceUnavailable("CHARITY_RATE_CHECK_FAILED", "公益频率检查暂时不可用，请稍后重试")
		}
		if !allowed {
			return &CharityAdmissionResult{IsCharity: true}, ErrCharityIPRPMExceeded
		}
	}

	// 4. 获取并发租约（全局 / 分组 / 单 IP）
	release := func() {}
	concurrencyLimited := snap.settings.MaxConcurrency > 0 || cg.MaxConcurrency > 0 ||
		(meta.ClientIP != "" && snap.settings.PerIPMaxConcurrency > 0)
	if s.cache == nil && concurrencyLimited {
		return &CharityAdmissionResult{IsCharity: true}, infraerrors.ServiceUnavailable("CHARITY_CONCURRENCY_CHECK_FAILED", "公益并发检查暂时不可用，请稍后重试")
	}
	if s.cache != nil {
		reqID := generateCharitySlotRequestID()
		acquired, _, concErr := s.cache.AcquireConcurrencySlots(
			ctx,
			reqID,
			snap.settings.MaxConcurrency,
			cg.GroupID,
			cg.MaxConcurrency,
			meta.ClientIP,
			snap.settings.PerIPMaxConcurrency,
		)
		if concErr != nil {
			slog.Warn("charity: failed to acquire concurrency slot", "error", concErr)
			if concurrencyLimited {
				return &CharityAdmissionResult{IsCharity: true}, infraerrors.ServiceUnavailable("CHARITY_CONCURRENCY_CHECK_FAILED", "公益并发检查暂时不可用，请稍后重试")
			}
		} else if !acquired {
			return &CharityAdmissionResult{IsCharity: true}, ErrCharityConcurrencyExceeded
		} else {
			var once sync.Once
			release = func() {
				once.Do(func() {
					relCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), charityReleaseTimeout)
					defer cancel()
					if err := s.cache.ReleaseConcurrencySlots(relCtx, reqID, cg.GroupID, meta.ClientIP); err != nil {
						slog.Warn("charity: failed to release concurrency slot", "error", err)
					}
				})
			}
		}
	}

	return &CharityAdmissionResult{
		IsCharity: true,
		Meta:      meta,
		Release:   release,
	}, nil
}

// RecordBilledUsage 在主账单事务提交成功后递增 Redis 今日折前消耗缓存。
func (s *CharityService) RecordBilledUsage(ctx context.Context, usageDate string, groupID int64, clientIP string, costUSD float64) {
	if s == nil || s.cache == nil || costUSD <= 0 || groupID <= 0 {
		return
	}
	if strings.TrimSpace(usageDate) == "" {
		usageDate, _, _ = CharityTimeWindow(s.now(), CharityDefaultTimezone)
	}
	if err := s.cache.IncrDailyGroupCost(ctx, usageDate, groupID, costUSD); err != nil {
		slog.Warn("charity: incr daily group cost cache failed", "group_id", groupID, "error", err)
	}
	if ipStr := strings.TrimSpace(clientIP); ipStr != "" {
		if err := s.cache.IncrDailyIPCost(ctx, usageDate, ipStr, costUSD); err != nil {
			slog.Warn("charity: incr daily ip cost cache failed", "error", err)
		}
	}
}

func (s *CharityService) loadDailyGroupCosts(ctx context.Context, usageDate string) (map[int64]float64, error) {
	rows, err := s.repo.GetDailyGroupUsage(ctx, usageDate)
	if err != nil {
		return nil, err
	}
	costs := make(map[int64]float64, len(rows))
	for _, r := range rows {
		costs[r.GroupID] = r.TotalCostUSD
	}
	return costs, nil
}

func (s *CharityService) loadDailyIPCost(ctx context.Context, usageDate string, clientIP string) (float64, error) {
	row, err := s.repo.GetDailyIPUsage(ctx, usageDate, clientIP)
	if err != nil {
		return 0, err
	}
	return row.TotalCostUSD, nil
}

// EnsureSystemUser 确保公益池专用内部宿主用户存在且为活跃状态。
func (s *CharityService) EnsureSystemUser(ctx context.Context) (*User, error) {
	settings, err := s.repo.GetSettings(ctx)
	if err != nil {
		return nil, err
	}
	if settings.SystemUserID != nil && *settings.SystemUserID > 0 {
		u, getErr := s.userRepo.GetByID(ctx, *settings.SystemUserID)
		if getErr == nil && u != nil {
			return u, nil
		}
	}

	existing, err := s.userRepo.GetByEmail(ctx, CharitySystemUserEmail)
	if err == nil && existing != nil {
		if err := s.repo.SetSystemUserID(ctx, existing.ID); err != nil {
			return nil, err
		}
		s.InvalidateSnapshot()
		return existing, nil
	}

	// 生成不可猜测的随机密码哈希，该系统账号仅供公益池内部挂载 Key 使用。
	rawPass := make([]byte, 32)
	_, _ = rand.Read(rawPass)
	hashBytes, err := bcrypt.GenerateFromPassword([]byte(hex.EncodeToString(rawPass)), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("hash charity system user password: %w", err)
	}

	newUser := &User{
		Email:        CharitySystemUserEmail,
		Username:     CharitySystemUsername,
		Notes:        "System internal owner for free.xjtu.shop charity pool keys",
		PasswordHash: string(hashBytes),
		Role:         RoleUser,
		Balance:      0,
		Concurrency:  charitySystemUserConcurrency,
		Status:       StatusActive,
	}
	if err := s.userRepo.Create(ctx, newUser); err != nil {
		// 并发创建回退查一次
		if retryUser, retryErr := s.userRepo.GetByEmail(ctx, CharitySystemUserEmail); retryErr == nil && retryUser != nil {
			_ = s.repo.SetSystemUserID(ctx, retryUser.ID)
			s.InvalidateSnapshot()
			return retryUser, nil
		}
		return nil, fmt.Errorf("create charity system user: %w", err)
	}
	if err := s.repo.SetSystemUserID(ctx, newUser.ID); err != nil {
		return nil, err
	}
	s.InvalidateSnapshot()
	return newUser, nil
}

func (s *CharityService) ensureSystemUserAllowedGroup(ctx context.Context, user *User, groupID int64) error {
	if user == nil || groupID <= 0 {
		return nil
	}
	if slices.Contains(user.AllowedGroups, groupID) && user.Concurrency >= charitySystemUserConcurrency {
		return nil
	}
	updated := *user
	if !slices.Contains(updated.AllowedGroups, groupID) {
		updated.AllowedGroups = append(append([]int64(nil), user.AllowedGroups...), groupID)
	}
	if updated.Concurrency < charitySystemUserConcurrency {
		updated.Concurrency = charitySystemUserConcurrency
	}
	if err := s.userRepo.Update(ctx, &updated, UserUpdateFields{
		AllowedGroups: true,
		Concurrency:   true,
	}); err != nil {
		return fmt.Errorf("update charity system user allowed groups: %w", err)
	}
	if s.apiKeyService != nil {
		s.apiKeyService.InvalidateAuthCacheByUserID(ctx, user.ID)
	}
	return nil
}

// GetAdminOverview 返回管理员端公益池完整总览。
func (s *CharityService) GetAdminOverview(ctx context.Context) (*CharityAdminOverview, error) {
	settings, err := s.repo.GetSettings(ctx)
	if err != nil {
		return nil, err
	}
	groups, err := s.repo.ListGroups(ctx)
	if err != nil {
		return nil, err
	}
	usageDate, nextResetAt, _ := CharityTimeWindow(s.now(), settings.RotationTimezone)
	dailyRows, err := s.repo.GetDailyGroupUsage(ctx, usageDate)
	if err != nil {
		return nil, err
	}
	trend, err := s.repo.GetDailyTrend(ctx, 14)
	if err != nil {
		return nil, err
	}

	usageByGroup := make(map[int64]CharityGroupDailyUsage, len(dailyRows))
	var usedToday float64
	var reqToday int64
	var tokensToday int64
	for _, row := range dailyRows {
		usageByGroup[row.GroupID] = row
		usedToday += row.TotalCostUSD
		reqToday += row.RequestCount
		tokensToday += row.TotalTokens
	}

	groupIDs := make([]int64, 0, len(groups))
	for _, g := range groups {
		groupIDs = append(groupIDs, g.GroupID)
	}
	var live *CharityLiveConcurrency
	if s.cache != nil {
		live, _ = s.cache.GetLiveConcurrency(ctx, groupIDs)
	}
	if live == nil {
		live = &CharityLiveConcurrency{ByGroup: map[int64]int{}}
	}

	adminGroups := make([]CharityAdminGroupView, 0, len(groups))
	for _, g := range groups {
		u := usageByGroup[g.GroupID]
		adminGroups = append(adminGroups, CharityAdminGroupView{
			CharityGroup:       g,
			UsedTodayUSD:       roundUSD(u.TotalCostUSD),
			RequestCountToday:  u.RequestCount,
			TotalTokensToday:   u.TotalTokens,
			CurrentConcurrency: live.ByGroup[g.GroupID],
		})
	}

	var remaining *float64
	if settings.DailyBudgetUSD > 0 {
		r := math.Max(0, settings.DailyBudgetUSD-usedToday)
		r = roundUSD(r)
		remaining = &r
	}

	return &CharityAdminOverview{
		Settings:           *settings,
		UsageDate:          usageDate,
		NextResetAt:        nextResetAt,
		UsedTodayUSD:       roundUSD(usedToday),
		RemainingTodayUSD:  remaining,
		RequestCountToday:  reqToday,
		TotalTokensToday:   tokensToday,
		CurrentConcurrency: live.Global,
		Groups:             adminGroups,
		Trend:              trend,
	}, nil
}

// UpdateSettings 更新公益池全局配置。
func (s *CharityService) UpdateSettings(ctx context.Context, input *UpdateCharitySettingsInput) (*CharityPoolSettings, error) {
	if input == nil {
		return nil, infraerrors.BadRequest("INVALID_INPUT", "配置参数不能为空")
	}
	if input.DailyBudgetUSD < 0 || input.PerIPDailyLimitUSD < 0 {
		return nil, infraerrors.BadRequest("INVALID_BUDGET", "额度配置不能为负数")
	}
	if input.MaxConcurrency < 0 || input.PerIPMaxConcurrency < 0 || input.PerIPRPMLimit < 0 {
		return nil, infraerrors.BadRequest("INVALID_LIMIT", "并发与限流配置不能为负数")
	}
	tz := strings.TrimSpace(input.RotationTimezone)
	if tz == "" {
		tz = CharityDefaultTimezone
	}
	if _, err := time.LoadLocation(tz); err != nil {
		return nil, infraerrors.BadRequest("INVALID_TIMEZONE", "无效的时区名称")
	}

	hosts := make([]string, 0, len(input.HostAllowlist))
	seen := make(map[string]struct{}, len(input.HostAllowlist))
	for _, h := range input.HostAllowlist {
		norm := NormalizeCharityHost(h)
		if norm == "" {
			continue
		}
		if _, dup := seen[norm]; dup {
			continue
		}
		seen[norm] = struct{}{}
		hosts = append(hosts, norm)
	}
	if len(hosts) == 0 {
		hosts = []string{"free.xjtu.shop"}
	}

	title := strings.TrimSpace(input.SiteTitle)
	if title == "" {
		title = "XJTU Free AI"
	}

	updated, err := s.repo.UpdateSettings(ctx, &CharityPoolSettings{
		Enabled:             input.Enabled,
		DailyBudgetUSD:      input.DailyBudgetUSD,
		MaxConcurrency:      input.MaxConcurrency,
		PerIPDailyLimitUSD:  input.PerIPDailyLimitUSD,
		PerIPMaxConcurrency: input.PerIPMaxConcurrency,
		PerIPRPMLimit:       input.PerIPRPMLimit,
		AutoRotateDaily:     input.AutoRotateDaily,
		RotationTimezone:    tz,
		SiteTitle:           title,
		SiteNotice:          strings.TrimSpace(input.SiteNotice),
		HostAllowlist:       hosts,
	})
	if err != nil {
		return nil, err
	}
	s.InvalidateSnapshot()
	return updated, nil
}

// AddGroup 将现有分组（或一步新建的专属公益分组）加入公益池并自动签发首个当日公共 Key。
func (s *CharityService) AddGroup(ctx context.Context, input *CreateCharityGroupInput) (*CharityGroup, error) {
	if input == nil {
		return nil, infraerrors.BadRequest("INVALID_INPUT", "分组参数不能为空")
	}
	if input.MaxConcurrency < 0 {
		return nil, infraerrors.BadRequest("INVALID_CONCURRENCY", "并发上限不能为负数")
	}
	if input.DailyLimitUSD != nil && *input.DailyLimitUSD < 0 {
		return nil, infraerrors.BadRequest("INVALID_DAILY_LIMIT", "每日子上限不能为负数")
	}

	var targetGroupID int64
	if input.GroupID != nil && *input.GroupID > 0 {
		g, err := s.groupRepo.GetByID(ctx, *input.GroupID)
		if err != nil || g == nil {
			return nil, infraerrors.NotFound("GROUP_NOT_FOUND", "目标分组不存在")
		}
		targetGroupID = g.ID
	} else {
		name := strings.TrimSpace(input.CreateGroupName)
		platform := strings.TrimSpace(strings.ToLower(input.CreateGroupPlatform))
		if name == "" || platform == "" {
			return nil, infraerrors.BadRequest("INVALID_GROUP", "请选择已有分组或填写新建分组的名称与平台")
		}
		if s.adminService == nil {
			return nil, infraerrors.ServiceUnavailable("ADMIN_SERVICE_UNAVAILABLE", "无法自动创建底层分组")
		}
		createdGroup, err := s.adminService.CreateGroup(ctx, &CreateGroupInput{
			Name:             name,
			Description:      strings.TrimSpace(input.CreateGroupDescription),
			Platform:         platform,
			RateMultiplier:   1.0,
			IsExclusive:      true,
			SubscriptionType: SubscriptionTypeStandard,
		})
		if err != nil {
			return nil, err
		}
		targetGroupID = createdGroup.ID
	}

	var normalizedLimit *float64
	if input.DailyLimitUSD != nil && *input.DailyLimitUSD > 0 {
		v := *input.DailyLimitUSD
		normalizedLimit = &v
	}

	created, err := s.repo.CreateGroup(ctx, &CharityGroup{
		GroupID:        targetGroupID,
		DisplayName:    strings.TrimSpace(input.DisplayName),
		Description:    strings.TrimSpace(input.Description),
		Enabled:        input.Enabled,
		DailyLimitUSD:  normalizedLimit,
		MaxConcurrency: input.MaxConcurrency,
		SortOrder:      input.SortOrder,
	})
	if err != nil {
		return nil, err
	}

	// 自动签发当日公共 Key 并授权系统宿主用户访问该分组
	rotated, err := s.RotateGroupKey(ctx, created.ID)
	if err != nil {
		return nil, err
	}
	s.InvalidateSnapshot()
	return rotated, nil
}

// UpdateGroup 编辑公益分组配置（子上限、并发、启用开关、展示名/排序）。
func (s *CharityService) UpdateGroup(ctx context.Context, id int64, input *UpdateCharityGroupInput) (*CharityGroup, error) {
	if input == nil {
		return nil, infraerrors.BadRequest("INVALID_INPUT", "更新参数不能为空")
	}
	if input.MaxConcurrency < 0 {
		return nil, infraerrors.BadRequest("INVALID_CONCURRENCY", "并发上限不能为负数")
	}
	if input.DailyLimitUSD != nil && *input.DailyLimitUSD < 0 {
		return nil, infraerrors.BadRequest("INVALID_DAILY_LIMIT", "每日子上限不能为负数")
	}
	var normalizedLimit *float64
	if input.DailyLimitUSD != nil && *input.DailyLimitUSD > 0 {
		v := *input.DailyLimitUSD
		normalizedLimit = &v
	}
	updated, err := s.repo.UpdateGroup(ctx, &CharityGroup{
		ID:             id,
		DisplayName:    strings.TrimSpace(input.DisplayName),
		Description:    strings.TrimSpace(input.Description),
		Enabled:        input.Enabled,
		DailyLimitUSD:  normalizedLimit,
		MaxConcurrency: input.MaxConcurrency,
		SortOrder:      input.SortOrder,
	})
	if err != nil {
		return nil, err
	}
	s.InvalidateSnapshot()
	return updated, nil
}

// RemoveGroup 将分组移出公益池并失效其当前公共 Key（保留底层 Group 与账号）。
func (s *CharityService) RemoveGroup(ctx context.Context, id int64) error {
	existing, err := s.repo.GetGroupByID(ctx, id)
	if err != nil {
		return err
	}
	if existing.ActiveAPIKeyID != nil && *existing.ActiveAPIKeyID > 0 {
		s.expireAPIKeyBestEffort(ctx, *existing.ActiveAPIKeyID, existing.ActiveAPIKey)
	}
	if err := s.repo.DeleteGroup(ctx, id); err != nil {
		return err
	}
	s.InvalidateSnapshot()
	return nil
}

// RotateGroupKey 为指定公益分组立即签发新公共 Key 并失效旧 Key。
func (s *CharityService) RotateGroupKey(ctx context.Context, id int64) (*CharityGroup, error) {
	cg, err := s.repo.GetGroupByID(ctx, id)
	if err != nil {
		return nil, err
	}
	settings, err := s.repo.GetSettings(ctx)
	if err != nil {
		return nil, err
	}
	sysUser, err := s.EnsureSystemUser(ctx)
	if err != nil {
		return nil, err
	}
	if err := s.ensureSystemUserAllowedGroup(ctx, sysUser, cg.GroupID); err != nil {
		return nil, err
	}

	usageDate, _, _ := CharityTimeWindow(s.now(), settings.RotationTimezone)
	newKeyStr, err := generateCharityAPIKey()
	if err != nil {
		return nil, fmt.Errorf("generate charity api key: %w", err)
	}

	groupID := cg.GroupID
	apiKey := &APIKey{
		UserID:  sysUser.ID,
		Key:     newKeyStr,
		Name:    fmt.Sprintf("charity-%d-%s", cg.GroupID, usageDate),
		GroupID: &groupID,
		Status:  StatusAPIKeyActive,
	}
	if err := s.apiKeyRepo.Create(ctx, apiKey); err != nil {
		return nil, fmt.Errorf("create charity api key: %w", err)
	}

	oldKeyID := cg.ActiveAPIKeyID
	oldKeyStr := cg.ActiveAPIKey

	rotatedAt := s.now()
	if err := s.repo.SetActiveAPIKey(ctx, cg.ID, &apiKey.ID, usageDate, rotatedAt); err != nil {
		return nil, err
	}
	if oldKeyID != nil && *oldKeyID > 0 && *oldKeyID != apiKey.ID {
		s.expireAPIKeyBestEffort(ctx, *oldKeyID, oldKeyStr)
	}
	if s.apiKeyService != nil {
		s.apiKeyService.InvalidateAuthCacheByKey(ctx, newKeyStr)
	}
	s.InvalidateSnapshot()
	return s.repo.GetGroupByID(ctx, cg.ID)
}

// RotateAllKeys 一键轮换全部公益分组公共 Key。
func (s *CharityService) RotateAllKeys(ctx context.Context) ([]CharityGroup, error) {
	groups, err := s.repo.ListGroups(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]CharityGroup, 0, len(groups))
	for _, g := range groups {
		rotated, rotErr := s.RotateGroupKey(ctx, g.ID)
		if rotErr != nil {
			return out, rotErr
		}
		out = append(out, *rotated)
	}
	return out, nil
}

// RotateDueDailyKeys 供后台调度器在每日 00:00 (Asia/Shanghai) 自动轮换所有到期分组的公共 Key。
func (s *CharityService) RotateDueDailyKeys(ctx context.Context) (int, error) {
	settings, err := s.repo.GetSettings(ctx)
	if err != nil || settings == nil || !settings.Enabled || !settings.AutoRotateDaily {
		return 0, err
	}
	usageDate, _, _ := CharityTimeWindow(s.now(), settings.RotationTimezone)
	groups, err := s.repo.ListGroups(ctx)
	if err != nil {
		return 0, err
	}
	due := make([]CharityGroup, 0)
	for _, g := range groups {
		if g.LastRotatedDate != usageDate || g.ActiveAPIKeyID == nil || g.ActiveAPIKey == "" {
			due = append(due, g)
		}
	}
	if len(due) == 0 {
		return 0, nil
	}
	if s.cache != nil {
		locked, lockErr := s.cache.TryAcquireRotationLock(ctx, usageDate, 45*time.Second)
		if lockErr != nil {
			return 0, lockErr
		}
		if !locked {
			return 0, nil
		}
	}
	rotatedCount := 0
	for _, g := range due {
		if _, rotErr := s.RotateGroupKey(ctx, g.ID); rotErr != nil {
			slog.Error("charity: daily key rotation failed", "charity_group_id", g.ID, "group_id", g.GroupID, "error", rotErr)
			continue
		}
		rotatedCount++
	}
	if rotatedCount > 0 {
		slog.Info("charity: completed daily key rotation", "date", usageDate, "rotated_groups", rotatedCount)
	}
	return rotatedCount, nil
}

func (s *CharityService) expireAPIKeyBestEffort(ctx context.Context, keyID int64, rawKey string) {
	if s.apiKeyRepo != nil {
		if err := s.apiKeyRepo.Update(ctx, &APIKey{
			ID:     keyID,
			Status: StatusAPIKeyExpired,
		}, APIKeyUpdateFields{Status: true}); err != nil {
			slog.Warn("charity: failed to expire rotated api key", "api_key_id", keyID, "error", err)
		}
	}
	if s.apiKeyService != nil && rawKey != "" {
		s.apiKeyService.InvalidateAuthCacheByKey(ctx, rawKey)
	}
}

// GetPublicOverview 构造面向 free.xjtu.shop 独立前端的公开状态、Key、折前价目表与当前 IP 用量。
func (s *CharityService) GetPublicOverview(ctx context.Context, clientIP string) (*CharityPublicOverview, error) {
	settings, err := s.repo.GetSettings(ctx)
	if err != nil {
		return nil, err
	}
	groups, err := s.repo.ListGroups(ctx)
	if err != nil {
		return nil, err
	}
	now := s.now()
	usageDate, nextResetAt, _ := CharityTimeWindow(now, settings.RotationTimezone)
	secondsUntilReset := int64(math.Max(0, nextResetAt.Sub(now).Seconds()))

	dailyRows, err := s.repo.GetDailyGroupUsage(ctx, usageDate)
	if err != nil {
		return nil, err
	}
	usageByGroup := make(map[int64]CharityGroupDailyUsage, len(dailyRows))
	var poolUsed float64
	var poolRequests int64
	for _, row := range dailyRows {
		usageByGroup[row.GroupID] = row
		poolUsed += row.TotalCostUSD
		poolRequests += row.RequestCount
	}

	var poolRemaining *float64
	poolExhausted := false
	if settings.DailyBudgetUSD > 0 {
		rem := math.Max(0, settings.DailyBudgetUSD-poolUsed)
		rem = roundUSD(rem)
		poolRemaining = &rem
		poolExhausted = poolUsed >= settings.DailyBudgetUSD
	}

	enabledGroups := make([]CharityGroup, 0, len(groups))
	groupIDs := make([]int64, 0, len(groups))
	for _, g := range groups {
		if g.Enabled && g.GroupStatus == StatusActive {
			enabledGroups = append(enabledGroups, g)
			groupIDs = append(groupIDs, g.GroupID)
		}
	}

	var live *CharityLiveConcurrency
	var ipConc int
	if s.cache != nil {
		live, _ = s.cache.GetLiveConcurrency(ctx, groupIDs)
		ipConc, _ = s.cache.GetLiveIPConcurrency(ctx, clientIP)
	}
	if live == nil {
		live = &CharityLiveConcurrency{ByGroup: map[int64]int{}}
	}

	modelsByGroup := s.loadPublicModelsByGroup(ctx, groupIDs)

	publicGroups := make([]CharityPublicGroupView, 0, len(enabledGroups))
	for _, g := range enabledGroups {
		u := usageByGroup[g.GroupID]
		var groupRemaining *float64
		exhausted := poolExhausted
		exhaustedReason := ""
		if poolExhausted {
			exhaustedReason = "pool_exhausted"
		}
		if g.HasDailyLimit() {
			gRem := math.Max(0, *g.DailyLimitUSD-u.TotalCostUSD)
			if poolRemaining != nil && *poolRemaining < gRem {
				gRem = *poolRemaining
			}
			gRem = roundUSD(gRem)
			groupRemaining = &gRem
			if u.TotalCostUSD >= *g.DailyLimitUSD {
				exhausted = true
				exhaustedReason = "group_limit_exhausted"
			}
		} else if poolRemaining != nil {
			r := *poolRemaining
			groupRemaining = &r
		}

		publicKey := ""
		if settings.Enabled {
			publicKey = g.ActiveAPIKey
		}

		publicGroups = append(publicGroups, CharityPublicGroupView{
			ID:                 g.ID,
			GroupID:            g.GroupID,
			Name:               g.EffectiveDisplayName(),
			Description:        g.Description,
			Platform:           g.GroupPlatform,
			Enabled:            g.Enabled,
			DailyLimitUSD:      g.DailyLimitUSD,
			UsedTodayUSD:       roundUSD(u.TotalCostUSD),
			RemainingTodayUSD:  groupRemaining,
			RequestCountToday:  u.RequestCount,
			MaxConcurrency:     g.MaxConcurrency,
			CurrentConcurrency: live.ByGroup[g.GroupID],
			PublicAPIKey:       publicKey,
			LastRotatedDate:    g.LastRotatedDate,
			LastRotatedAt:      g.LastRotatedAt,
			Exhausted:          exhausted,
			ExhaustedReason:    exhaustedReason,
			Models:             modelsByGroup[g.GroupID],
		})
	}

	ipRow, _ := s.repo.GetDailyIPUsage(ctx, usageDate, clientIP)
	if ipRow == nil {
		ipRow = &CharityIPDailyUsage{UsageDate: usageDate, ClientIP: clientIP}
	}
	var ipRem *float64
	if settings.PerIPDailyLimitUSD > 0 {
		r := roundUSD(math.Max(0, settings.PerIPDailyLimitUSD-ipRow.TotalCostUSD))
		ipRem = &r
	}

	return &CharityPublicOverview{
		Enabled:            settings.Enabled,
		SiteTitle:          settings.SiteTitle,
		SiteNotice:         settings.SiteNotice,
		UsageDate:          usageDate,
		Timezone:           settings.RotationTimezone,
		AutoRotateDaily:    settings.AutoRotateDaily,
		NextResetAt:        nextResetAt,
		SecondsUntilReset:  secondsUntilReset,
		DailyBudgetUSD:     settings.DailyBudgetUSD,
		UsedTodayUSD:       roundUSD(poolUsed),
		RemainingTodayUSD:  poolRemaining,
		RequestCountToday:  poolRequests,
		MaxConcurrency:     settings.MaxConcurrency,
		CurrentConcurrency: live.Global,
		PoolExhausted:      poolExhausted,
		Groups:             publicGroups,
		IPStatus: CharityPublicIPStatus{
			MaskedIP:           MaskCharityIP(clientIP),
			UsedTodayUSD:       roundUSD(ipRow.TotalCostUSD),
			RequestCountToday:  ipRow.RequestCount,
			DailyLimitUSD:      settings.PerIPDailyLimitUSD,
			RemainingTodayUSD:  ipRem,
			MaxConcurrency:     settings.PerIPMaxConcurrency,
			CurrentConcurrency: ipConc,
			RPMLimit:           settings.PerIPRPMLimit,
		},
	}, nil
}

// LookupPublicKeyStatus 允许用户粘贴今日公益 Key 查询实时状态与消耗。
func (s *CharityService) LookupPublicKeyStatus(ctx context.Context, rawKey string) (*CharityKeyLookupResult, error) {
	keyStr := strings.TrimSpace(rawKey)
	settings, err := s.repo.GetSettings(ctx)
	if err != nil {
		return nil, err
	}
	usageDate, nextResetAt, _ := CharityTimeWindow(s.now(), settings.RotationTimezone)
	dailyRows, err := s.repo.GetDailyGroupUsage(ctx, usageDate)
	if err != nil {
		return nil, err
	}
	usageByGroup := make(map[int64]CharityGroupDailyUsage, len(dailyRows))
	var poolUsed float64
	for _, r := range dailyRows {
		usageByGroup[r.GroupID] = r
		poolUsed += r.TotalCostUSD
	}
	var poolRem *float64
	if settings.DailyBudgetUSD > 0 {
		r := roundUSD(math.Max(0, settings.DailyBudgetUSD-poolUsed))
		poolRem = &r
	}

	base := &CharityKeyLookupResult{
		Found:              false,
		Status:             "not_found",
		UsageDate:          usageDate,
		PoolDailyBudgetUSD: settings.DailyBudgetUSD,
		PoolUsedTodayUSD:   roundUSD(poolUsed),
		PoolRemainingUSD:   poolRem,
		NextResetAt:        nextResetAt,
	}
	if keyStr == "" {
		return base, nil
	}

	apiKey, err := s.apiKeyRepo.GetByKey(ctx, keyStr)
	if err != nil || apiKey == nil {
		return base, nil
	}
	if !s.IsCharityAPIKey(ctx, apiKey) {
		return base, nil
	}

	base.Found = true
	var cg *CharityGroup
	if apiKey.GroupID != nil {
		cg, _ = s.repo.GetGroupByGroupID(ctx, *apiKey.GroupID)
	}
	if cg == nil {
		base.Status = "disabled"
		return base, nil
	}

	base.GroupID = cg.GroupID
	base.GroupName = cg.EffectiveDisplayName()
	base.Platform = cg.GroupPlatform
	base.GroupDailyLimitUSD = cg.DailyLimitUSD
	base.LastRotatedDate = cg.LastRotatedDate

	u := usageByGroup[cg.GroupID]
	base.GroupUsedTodayUSD = roundUSD(u.TotalCostUSD)
	if cg.HasDailyLimit() {
		rem := math.Max(0, *cg.DailyLimitUSD-u.TotalCostUSD)
		if poolRem != nil && *poolRem < rem {
			rem = *poolRem
		}
		rem = roundUSD(rem)
		base.GroupRemainingUSD = &rem
	} else if poolRem != nil {
		r := *poolRem
		base.GroupRemainingUSD = &r
	}

	if !settings.Enabled || !cg.Enabled {
		base.Status = "disabled"
	} else if cg.ActiveAPIKeyID == nil || *cg.ActiveAPIKeyID != apiKey.ID || apiKey.Status != StatusAPIKeyActive {
		base.Status = "rotated_expired"
	} else {
		base.Status = "active"
	}
	return base, nil
}

func (s *CharityService) loadPublicModelsByGroup(ctx context.Context, groupIDs []int64) map[int64][]CharityPublicModel {
	out := make(map[int64][]CharityPublicModel, len(groupIDs))
	for _, groupID := range groupIDs {
		out[groupID] = []CharityPublicModel{}
		if s.modelCatalog == nil {
			continue
		}
		group, err := s.groupRepo.GetByIDLite(ctx, groupID)
		if err != nil {
			slog.Warn("charity: failed to load model group", "group_id", groupID, "error", err)
			continue
		}
		if !group.IsActive() {
			continue
		}
		available, err := s.modelCatalog.ListForGroup(ctx, group)
		if err != nil {
			slog.Warn("charity: failed to load group model catalog", "group_id", groupID, "error", err)
			continue
		}
		models := make([]CharityPublicModel, 0, len(available))
		for _, discovered := range available {
			item := CharityPublicModel{Name: discovered.ID, Platform: group.Platform}
			if s.plazaService != nil {
				metadata := s.plazaService.metadataCatalog.Resolve(item.Name)
				item.SupportsVision = slices.Contains(metadata.InputModalities, "image")
				item.SupportsReasoning = metadata.Reasoning != nil && *metadata.Reasoning
				priced := PlazaModel{Name: item.Name, Platform: item.Platform}
				s.plazaService.fillDisplayPricing(ctx, &priced, group)
				if pricing := priced.Pricing; pricing != nil {
					item.InputPricePerMillion = perTokenToPerMillion(pricing.InputPrice)
					item.OutputPricePerMillion = perTokenToPerMillion(pricing.OutputPrice)
					item.CacheReadPerMillion = perTokenToPerMillion(pricing.CacheReadPrice)
					item.CacheWritePerMillion = perTokenToPerMillion(pricing.CacheWritePrice)
				}
			}
			models = append(models, item)
		}
		sort.SliceStable(models, func(i, j int) bool {
			return models[i].Name < models[j].Name
		})
		out[groupID] = models
	}
	return out
}

func perTokenToPerMillion(perToken *float64) *float64 {
	if perToken == nil {
		return nil
	}
	v := math.Round(*perToken*1_000_000*10000) / 10000
	return &v
}

func roundUSD(v float64) float64 {
	if math.IsNaN(v) || math.IsInf(v, 0) {
		return 0
	}
	return math.Round(v*1_000_000) / 1_000_000
}

func generateCharityAPIKey() (string, error) {
	buf := make([]byte, 24)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return CharityKeyPrefix + hex.EncodeToString(buf), nil
}

func generateCharitySlotRequestID() string {
	buf := make([]byte, 12)
	_, _ = rand.Read(buf)
	return hex.EncodeToString(buf)
}
