package service

import (
	"context"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/ctxkey"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

const (
	CharitySystemUserEmail       = "charity-pool@system.local"
	CharitySystemUsername        = "charity-pool"
	CharityDefaultTimezone       = "Asia/Shanghai"
	CharityDefaultDailyBudgetUSD = 100.0
	CharityDefaultMaxConcurrency = 20
	CharityKeyPrefix             = "sk-free-"
	CharityDefaultHost           = "free.xjtu.shop"
)

var (
	ErrCharityDisabled              = infraerrors.ServiceUnavailable("CHARITY_DISABLED", "公益池当前未开启")
	ErrCharityGroupDisabled         = infraerrors.Forbidden("CHARITY_GROUP_DISABLED", "该公益分组当前已停用")
	ErrCharityGroupNotFound         = infraerrors.NotFound("CHARITY_GROUP_NOT_FOUND", "公益分组不存在")
	ErrCharityGroupAlreadyExists    = infraerrors.Conflict("CHARITY_GROUP_EXISTS", "该分组已加入公益池")
	ErrCharityKeyRotated            = infraerrors.Unauthorized("CHARITY_KEY_ROTATED", "该公益 Key 已随每日轮换失效，请访问公益站复制今日最新 Key")
	ErrCharityHostKeyNotAllowed     = infraerrors.Forbidden("CHARITY_HOST_KEY_NOT_ALLOWED", "公益域名仅允许使用公益池公共 Key")
	ErrCharityDailyBudgetExhausted  = infraerrors.TooManyRequests("CHARITY_DAILY_BUDGET_EXHAUSTED", "今日全站公益共享额度已用完，将于北京时间 00:00 自动重置")
	ErrCharityGroupLimitExhausted   = infraerrors.TooManyRequests("CHARITY_GROUP_LIMIT_EXHAUSTED", "该公益分组今日子额度已用完，请切换其他公益分组或等待北京时间 00:00 重置")
	ErrCharityIPDailyLimitExhausted = infraerrors.TooManyRequests("CHARITY_IP_DAILY_LIMIT_EXHAUSTED", "当前网络今日公益额度已达上限，将于北京时间 00:00 重置")
	ErrCharityIPRPMExceeded         = infraerrors.TooManyRequests("CHARITY_IP_RPM_EXCEEDED", "当前网络请求过于频繁，请稍后再试")
	ErrCharityConcurrencyExceeded   = infraerrors.TooManyRequests("CHARITY_CONCURRENCY_EXCEEDED", "公益池当前并发已满，请稍后重试")
)

// CharityBillingMeta 保存在请求上下文中，供网关后扣流程识别公益请求。
type CharityBillingMeta struct {
	CharityGroupID int64
	GroupID        int64
	APIKeyID       int64
	UserID         int64
	ClientIP       string
	UsageDate      string
}

func CharityAdmissionFromContext(ctx context.Context) *CharityBillingMeta {
	if ctx == nil {
		return nil
	}
	meta, _ := ctx.Value(ctxkey.CharityBilling).(*CharityBillingMeta)
	if meta == nil || meta.GroupID <= 0 {
		return nil
	}
	return meta
}

func CharityAdmittedKey(ctx context.Context, key *APIKey) bool {
	meta := CharityAdmissionFromContext(ctx)
	return meta != nil && key != nil && meta.APIKeyID == key.ID && meta.UserID == key.UserID
}

// CharityPoolSettings 对应 charity_pool_settings 单行配置。
type CharityPoolSettings struct {
	Enabled             bool      `json:"enabled"`
	DailyBudgetUSD      float64   `json:"daily_budget_usd"`
	MaxConcurrency      int       `json:"max_concurrency"`
	PerIPDailyLimitUSD  float64   `json:"per_ip_daily_limit_usd"`
	PerIPMaxConcurrency int       `json:"per_ip_max_concurrency"`
	PerIPRPMLimit       int       `json:"per_ip_rpm_limit"`
	AutoRotateDaily     bool      `json:"auto_rotate_daily"`
	RotationTimezone    string    `json:"rotation_timezone"`
	SiteTitle           string    `json:"site_title"`
	SiteNotice          string    `json:"site_notice"`
	HostAllowlist       []string  `json:"host_allowlist"`
	SystemUserID        *int64    `json:"system_user_id,omitempty"`
	CreatedAt           time.Time `json:"created_at"`
	UpdatedAt           time.Time `json:"updated_at"`
}

// CharityGroup 对应 charity_groups 中的一条公益分组配置。
type CharityGroup struct {
	ID              int64      `json:"id"`
	GroupID         int64      `json:"group_id"`
	DisplayName     string     `json:"display_name"`
	Description     string     `json:"description"`
	Enabled         bool       `json:"enabled"`
	DailyLimitUSD   *float64   `json:"daily_limit_usd"` // nil 或 <= 0 表示不设单独子上限，共享总池
	MaxConcurrency  int        `json:"max_concurrency"`
	SortOrder       int        `json:"sort_order"`
	ActiveAPIKeyID  *int64     `json:"active_api_key_id,omitempty"`
	ActiveAPIKey    string     `json:"active_api_key"`
	LastRotatedDate string     `json:"last_rotated_date"`
	LastRotatedAt   *time.Time `json:"last_rotated_at,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`

	// 关联底层分组摘要
	GroupName          string `json:"group_name"`
	GroupPlatform      string `json:"group_platform"`
	GroupStatus        string `json:"group_status"`
	AccountCount       int64  `json:"account_count"`
	ActiveAccountCount int64  `json:"active_account_count"`
}

// EffectiveDisplayName 返回对外展示名称。
func (g *CharityGroup) EffectiveDisplayName() string {
	if g == nil {
		return ""
	}
	if name := strings.TrimSpace(g.DisplayName); name != "" {
		return name
	}
	return g.GroupName
}

// HasDailyLimit 判断是否设置了独立的分组每日子上限。
func (g *CharityGroup) HasDailyLimit() bool {
	return g != nil && g.DailyLimitUSD != nil && *g.DailyLimitUSD > 0
}

// CharityGroupDailyUsage 单个分组的当日折前用量。
type CharityGroupDailyUsage struct {
	UsageDate    string  `json:"usage_date"`
	GroupID      int64   `json:"group_id"`
	TotalCostUSD float64 `json:"total_cost_usd"`
	RequestCount int64   `json:"request_count"`
	TotalTokens  int64   `json:"total_tokens"`
}

// CharityIPDailyUsage 单个 IP 的当日折前用量。
type CharityIPDailyUsage struct {
	UsageDate    string  `json:"usage_date"`
	ClientIP     string  `json:"client_ip"`
	TotalCostUSD float64 `json:"total_cost_usd"`
	RequestCount int64   `json:"request_count"`
}

// CharityDailyTrendPoint 每日汇总趋势点（管理后台统计用）。
type CharityDailyTrendPoint struct {
	UsageDate    string  `json:"usage_date"`
	TotalCostUSD float64 `json:"total_cost_usd"`
	RequestCount int64   `json:"request_count"`
	TotalTokens  int64   `json:"total_tokens"`
}

// CharityLiveConcurrency 当前实时并发快照。
type CharityLiveConcurrency struct {
	Global  int           `json:"global"`
	ByGroup map[int64]int `json:"by_group"`
}

// CharityPublicModel 公益站对外展示的模型与折前 1× 参考单价（USD per 1M tokens）。
type CharityPublicModel struct {
	Name                  string   `json:"name"`
	Platform              string   `json:"platform"`
	InputPricePerMillion  *float64 `json:"input_price_per_million,omitempty"`
	OutputPricePerMillion *float64 `json:"output_price_per_million,omitempty"`
	CacheReadPerMillion   *float64 `json:"cache_read_per_million,omitempty"`
	CacheWritePerMillion  *float64 `json:"cache_write_per_million,omitempty"`
	SupportsVision        bool     `json:"supports_vision,omitempty"`
	SupportsReasoning     bool     `json:"supports_reasoning,omitempty"`
}

// CharityPublicGroupView 公益用户界面单个分组展示数据。
type CharityPublicGroupView struct {
	ID                 int64                `json:"id"`
	GroupID            int64                `json:"group_id"`
	Name               string               `json:"name"`
	Description        string               `json:"description"`
	Platform           string               `json:"platform"`
	Enabled            bool                 `json:"enabled"`
	DailyLimitUSD      *float64             `json:"daily_limit_usd"` // null = 不设单组上限，共享总池
	UsedTodayUSD       float64              `json:"used_today_usd"`
	RemainingTodayUSD  *float64             `json:"remaining_today_usd"` // nil 表示总池不限且单组不限
	RequestCountToday  int64                `json:"request_count_today"`
	MaxConcurrency     int                  `json:"max_concurrency"`
	CurrentConcurrency int                  `json:"current_concurrency"`
	PublicAPIKey       string               `json:"public_api_key"`
	LastRotatedDate    string               `json:"last_rotated_date"`
	LastRotatedAt      *time.Time           `json:"last_rotated_at,omitempty"`
	Exhausted          bool                 `json:"exhausted"`
	ExhaustedReason    string               `json:"exhausted_reason,omitempty"`
	Models             []CharityPublicModel `json:"models"`
}

// CharityPublicIPStatus 当前访问者 IP 的当日状态与限流规则。
type CharityPublicIPStatus struct {
	MaskedIP           string   `json:"masked_ip"`
	UsedTodayUSD       float64  `json:"used_today_usd"`
	RequestCountToday  int64    `json:"request_count_today"`
	DailyLimitUSD      float64  `json:"daily_limit_usd"`
	RemainingTodayUSD  *float64 `json:"remaining_today_usd"`
	MaxConcurrency     int      `json:"max_concurrency"`
	CurrentConcurrency int      `json:"current_concurrency"`
	RPMLimit           int      `json:"rpm_limit"`
}

// CharityPublicOverview 公益站（free.xjtu.shop）公开接口返回体。
type CharityPublicOverview struct {
	Enabled            bool                     `json:"enabled"`
	SiteTitle          string                   `json:"site_title"`
	SiteNotice         string                   `json:"site_notice"`
	UsageDate          string                   `json:"usage_date"`
	Timezone           string                   `json:"timezone"`
	AutoRotateDaily    bool                     `json:"auto_rotate_daily"`
	NextResetAt        time.Time                `json:"next_reset_at"`
	SecondsUntilReset  int64                    `json:"seconds_until_reset"`
	DailyBudgetUSD     float64                  `json:"daily_budget_usd"`
	UsedTodayUSD       float64                  `json:"used_today_usd"`
	RemainingTodayUSD  *float64                 `json:"remaining_today_usd"`
	RequestCountToday  int64                    `json:"request_count_today"`
	MaxConcurrency     int                      `json:"max_concurrency"`
	CurrentConcurrency int                      `json:"current_concurrency"`
	PoolExhausted      bool                     `json:"pool_exhausted"`
	Groups             []CharityPublicGroupView `json:"groups"`
	IPStatus           CharityPublicIPStatus    `json:"ip_status"`
}

// CharityAdminGroupView 管理员端单个公益分组视图。
type CharityAdminGroupView struct {
	CharityGroup
	UsedTodayUSD       float64 `json:"used_today_usd"`
	RequestCountToday  int64   `json:"request_count_today"`
	TotalTokensToday   int64   `json:"total_tokens_today"`
	CurrentConcurrency int     `json:"current_concurrency"`
}

// CharityAdminOverview 管理员后台公益池总览。
type CharityAdminOverview struct {
	Settings           CharityPoolSettings      `json:"settings"`
	UsageDate          string                   `json:"usage_date"`
	NextResetAt        time.Time                `json:"next_reset_at"`
	UsedTodayUSD       float64                  `json:"used_today_usd"`
	RemainingTodayUSD  *float64                 `json:"remaining_today_usd"`
	RequestCountToday  int64                    `json:"request_count_today"`
	TotalTokensToday   int64                    `json:"total_tokens_today"`
	CurrentConcurrency int                      `json:"current_concurrency"`
	Groups             []CharityAdminGroupView  `json:"groups"`
	Trend              []CharityDailyTrendPoint `json:"trend"`
}

// CharityKeyLookupResult 公开按 Key 查询实时状态。
type CharityKeyLookupResult struct {
	Found              bool      `json:"found"`
	Status             string    `json:"status"` // active | rotated_expired | disabled | not_found
	GroupID            int64     `json:"group_id,omitempty"`
	GroupName          string    `json:"group_name,omitempty"`
	Platform           string    `json:"platform,omitempty"`
	UsageDate          string    `json:"usage_date"`
	PoolDailyBudgetUSD float64   `json:"pool_daily_budget_usd"`
	PoolUsedTodayUSD   float64   `json:"pool_used_today_usd"`
	PoolRemainingUSD   *float64  `json:"pool_remaining_usd"`
	GroupDailyLimitUSD *float64  `json:"group_daily_limit_usd"`
	GroupUsedTodayUSD  float64   `json:"group_used_today_usd"`
	GroupRemainingUSD  *float64  `json:"group_remaining_usd"`
	LastRotatedDate    string    `json:"last_rotated_date,omitempty"`
	NextResetAt        time.Time `json:"next_reset_at"`
}

// UpdateCharitySettingsInput 管理端更新公益池全局配置入参。
type UpdateCharitySettingsInput struct {
	Enabled             bool     `json:"enabled"`
	DailyBudgetUSD      float64  `json:"daily_budget_usd"`
	MaxConcurrency      int      `json:"max_concurrency"`
	PerIPDailyLimitUSD  float64  `json:"per_ip_daily_limit_usd"`
	PerIPMaxConcurrency int      `json:"per_ip_max_concurrency"`
	PerIPRPMLimit       int      `json:"per_ip_rpm_limit"`
	AutoRotateDaily     bool     `json:"auto_rotate_daily"`
	RotationTimezone    string   `json:"rotation_timezone"`
	SiteTitle           string   `json:"site_title"`
	SiteNotice          string   `json:"site_notice"`
	HostAllowlist       []string `json:"host_allowlist"`
}

// CreateCharityGroupInput 管理端将分组加入公益池（或一步新建专属分组）入参。
type CreateCharityGroupInput struct {
	// 方式 1：绑定已有分组
	GroupID *int64 `json:"group_id"`
	// 方式 2：当 GroupID 为空时，一步新建一个专属公益分组
	CreateGroupName        string `json:"create_group_name"`
	CreateGroupPlatform    string `json:"create_group_platform"`
	CreateGroupDescription string `json:"create_group_description"`

	DisplayName    string   `json:"display_name"`
	Description    string   `json:"description"`
	Enabled        bool     `json:"enabled"`
	DailyLimitUSD  *float64 `json:"daily_limit_usd"` // nil 或 <= 0 表示不设单独子上限
	MaxConcurrency int      `json:"max_concurrency"`
	SortOrder      int      `json:"sort_order"`
}

// UpdateCharityGroupInput 管理端编辑公益分组入参。
type UpdateCharityGroupInput struct {
	DisplayName    string   `json:"display_name"`
	Description    string   `json:"description"`
	Enabled        bool     `json:"enabled"`
	DailyLimitUSD  *float64 `json:"daily_limit_usd"`
	MaxConcurrency int      `json:"max_concurrency"`
	SortOrder      int      `json:"sort_order"`
}

// CharityRepository 公益池数据库仓储接口。
type CharityRepository interface {
	GetSettings(ctx context.Context) (*CharityPoolSettings, error)
	UpdateSettings(ctx context.Context, settings *CharityPoolSettings) (*CharityPoolSettings, error)
	SetSystemUserID(ctx context.Context, userID int64) error

	ListGroups(ctx context.Context) ([]CharityGroup, error)
	GetGroupByID(ctx context.Context, id int64) (*CharityGroup, error)
	GetGroupByGroupID(ctx context.Context, groupID int64) (*CharityGroup, error)
	GetGroupByActiveAPIKeyID(ctx context.Context, apiKeyID int64) (*CharityGroup, error)
	CreateGroup(ctx context.Context, g *CharityGroup) (*CharityGroup, error)
	UpdateGroup(ctx context.Context, g *CharityGroup) (*CharityGroup, error)
	DeleteGroup(ctx context.Context, id int64) error
	SetActiveAPIKey(ctx context.Context, id int64, apiKeyID *int64, rotatedDate string, rotatedAt time.Time) error

	GetDailyGroupUsage(ctx context.Context, usageDate string) ([]CharityGroupDailyUsage, error)
	GetDailyIPUsage(ctx context.Context, usageDate string, clientIP string) (*CharityIPDailyUsage, error)
	GetDailyTrend(ctx context.Context, days int) ([]CharityDailyTrendPoint, error)
}

// CharityCache 公益池 Redis 热路径缓存（用量、并发槽、IP RPM、轮换锁）。
type CharityCache interface {
	GetDailyGroupCosts(ctx context.Context, usageDate string) (map[int64]float64, bool, error)
	SetDailyGroupCosts(ctx context.Context, usageDate string, costs map[int64]float64) error
	IncrDailyGroupCost(ctx context.Context, usageDate string, groupID int64, costUSD float64) error

	GetDailyIPCost(ctx context.Context, usageDate string, clientIP string) (float64, bool, error)
	SetDailyIPCost(ctx context.Context, usageDate string, clientIP string, costUSD float64) error
	IncrDailyIPCost(ctx context.Context, usageDate string, clientIP string, costUSD float64) error

	CheckAndIncrIPRPM(ctx context.Context, clientIP string, limit int) (bool, error)

	AcquireConcurrencySlots(ctx context.Context, requestID string, globalLimit int, groupID int64, groupLimit int, clientIP string, ipLimit int) (acquired bool, reason string, err error)
	ReleaseConcurrencySlots(ctx context.Context, requestID string, groupID int64, clientIP string) error
	GetLiveConcurrency(ctx context.Context, groupIDs []int64) (*CharityLiveConcurrency, error)
	GetLiveIPConcurrency(ctx context.Context, clientIP string) (int, error)

	TryAcquireRotationLock(ctx context.Context, dateKey string, ttl time.Duration) (bool, error)
}

// CharityTimeWindow 返回在配置时区下的当前日期字符串（YYYY-MM-DD）及下次 00:00 重置时刻。
func CharityTimeWindow(now time.Time, tzName string) (usageDate string, nextResetAt time.Time, loc *time.Location) {
	if strings.TrimSpace(tzName) == "" {
		tzName = CharityDefaultTimezone
	}
	var err error
	loc, err = time.LoadLocation(tzName)
	if err != nil {
		loc = time.FixedZone("CST", 8*3600)
	}
	localNow := now.In(loc)
	usageDate = localNow.Format("2006-01-02")
	nextResetAt = time.Date(localNow.Year(), localNow.Month(), localNow.Day()+1, 0, 0, 0, 0, loc)
	return usageDate, nextResetAt, loc
}

// NormalizeCharityHost 标准化 Host 头（去端口、转小写、去首尾空白）。
func NormalizeCharityHost(rawHost string) string {
	h := strings.TrimSpace(strings.ToLower(rawHost))
	if h == "" {
		return ""
	}
	if strings.HasPrefix(h, "[") {
		if idx := strings.LastIndex(h, "]"); idx > 0 {
			return h[1:idx]
		}
	}
	if idx := strings.IndexByte(h, ':'); idx > 0 {
		return h[:idx]
	}
	return h
}

// MaskCharityIP 对外脱敏展示客户端 IP，避免页面直接完整暴露局域网/公网地址。
func MaskCharityIP(rawIP string) string {
	ipStr := strings.TrimSpace(rawIP)
	if ipStr == "" {
		return "unknown"
	}
	if strings.Contains(ipStr, ".") {
		parts := strings.Split(ipStr, ".")
		if len(parts) == 4 {
			return parts[0] + "." + parts[1] + ".*." + parts[3]
		}
	}
	if strings.Contains(ipStr, ":") {
		parts := strings.Split(ipStr, ":")
		if len(parts) >= 2 {
			return parts[0] + ":" + parts[1] + ":*:*"
		}
	}
	return ipStr
}
