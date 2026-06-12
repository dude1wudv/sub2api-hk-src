package service

import (
	"context"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/domain"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

// DailyBalanceGrant 是每日余额赠额的领域模型：一笔带 24h 有效期、绑定专属分组的额度桶。
// 区别于 User.Balance 单一长期余额池——每日额度多笔并存、各自过期、按到期时间 FIFO 消耗。
type DailyBalanceGrant struct {
	ID        int64
	UserID    int64
	GroupID   int64
	Amount    float64
	Remaining float64
	Status    string
	Source    string
	SourceRef *string
	GrantedAt time.Time
	ExpiresAt time.Time
	CreatedAt time.Time
}

// DailyGrantRepository 是每日余额赠额的持久化接口。
//
// 并发安全要点：AtomicDecrement 必须以 DB 行级条件更新实现
// （UPDATE ... SET remaining = remaining - amount WHERE id=? AND status='active' AND remaining >= amount），
// 避免并发请求重复消费同一笔 Grant。
type DailyGrantRepository interface {
	// CreateGrant 创建一笔新的每日额度赠额，返回其 ID。
	CreateGrant(ctx context.Context, grant *DailyBalanceGrant) (int64, error)

	// SumActiveRemaining 返回某用户在某分组下所有“有效”Grant 的剩余额度合计
	// （status=active 且 expires_at>now，惰性过滤过期）。
	SumActiveRemaining(ctx context.Context, userID, groupID int64, now time.Time) (float64, error)

	// ListActiveByExpiry 返回某用户某分组下有效 Grant，按 expires_at 升序（先到期的先消费）。
	ListActiveByExpiry(ctx context.Context, userID, groupID int64, now time.Time) ([]DailyBalanceGrant, error)

	// AtomicDecrement 原子扣减指定 Grant 的剩余额度，条件为 status=active 且 remaining>=amount。
	// 返回是否成功扣减（false 表示该 Grant 已不满足条件，调用方应跳到下一笔）。
	// 当扣减后 remaining 归零时，应在同一更新内将 status 置为 exhausted。
	AtomicDecrement(ctx context.Context, grantID int64, amount float64) (bool, error)

	// MarkExpired 将所有 status=active 且 expires_at<now 的 Grant 标记为 expired，返回受影响行数。
	MarkExpired(ctx context.Context, now time.Time) (int64, error)

	// ListByUser 返回某用户的 Grant 列表（管理员/用户面板可见性，按 created_at 倒序）。
	ListByUser(ctx context.Context, userID int64) ([]DailyBalanceGrant, error)
}

// 每日余额功能错误。
var (
	ErrDailyGrantGroupNotExclusive = infraerrors.BadRequest("DAILY_GRANT_GROUP_NOT_EXCLUSIVE", "group is not a daily-balance group")
	ErrDailyGrantAmountInvalid     = infraerrors.BadRequest("DAILY_GRANT_AMOUNT_INVALID", "daily grant amount must be positive")
)

// DailyGrantValidity 是每日额度的有效期（24 小时）。
const DailyGrantValidity = 24 * time.Hour

// dailyGrantBalanceInvalidator 抽象「失效用户余额缓存」能力，避免直接耦合 *BillingCacheService。
type dailyGrantBalanceInvalidator interface {
	InvalidateUserBalance(ctx context.Context, userID int64) error
}

// DailyGrantService 负责每日余额赠额的发放与查询。
type DailyGrantService struct {
	grantRepo    DailyGrantRepository
	groupRepo    GroupRepository
	billingCache dailyGrantBalanceInvalidator
}

// NewDailyGrantService 构造每日余额服务。billingCache 可为 nil（仅跳过缓存失效）。
func NewDailyGrantService(grantRepo DailyGrantRepository, groupRepo GroupRepository, billingCache dailyGrantBalanceInvalidator) *DailyGrantService {
	return &DailyGrantService{grantRepo: grantRepo, groupRepo: groupRepo, billingCache: billingCache}
}

// GrantDailyInput 描述一次每日额度发放。
type GrantDailyInput struct {
	UserID    int64
	GroupID   int64
	Amount    float64
	Source    string  // domain.DailyGrantSource*（默认 admin）
	SourceRef *string // 兑换码 code 或操作者标识
}

// GrantDaily 给用户发放一笔 24h 有效的每日额度，绑定到指定专属分组。
//
//   - 校验金额 > 0。
//   - 校验目标分组存在且 daily_balance_enabled=true（否则拒绝，避免发到普通分组）。
//   - 创建 Grant（expires_at = now + 24h），失效用户余额缓存使预检立即可见。
func (s *DailyGrantService) GrantDaily(ctx context.Context, in GrantDailyInput) (*DailyBalanceGrant, error) {
	if in.Amount <= 0 {
		return nil, ErrDailyGrantAmountInvalid
	}

	group, err := s.groupRepo.GetByID(ctx, in.GroupID)
	if err != nil {
		return nil, err
	}
	if !group.DailyBalanceEnabled {
		return nil, ErrDailyGrantGroupNotExclusive
	}

	source := in.Source
	if source == "" {
		source = domain.DailyGrantSourceAdmin
	}

	now := time.Now()
	grant := &DailyBalanceGrant{
		UserID:    in.UserID,
		GroupID:   in.GroupID,
		Amount:    in.Amount,
		Remaining: in.Amount,
		Status:    domain.DailyGrantStatusActive,
		Source:    source,
		SourceRef: in.SourceRef,
		GrantedAt: now,
		ExpiresAt: now.Add(DailyGrantValidity),
	}
	id, err := s.grantRepo.CreateGrant(ctx, grant)
	if err != nil {
		return nil, err
	}
	grant.ID = id

	// 失效余额缓存：让预检（checkBalanceEligibility）立即看到新额度。
	if s.billingCache != nil {
		if invErr := s.billingCache.InvalidateUserBalance(ctx, in.UserID); invErr != nil {
			// 非致命：缓存最终一致，记录由上层日志兜底。
			_ = invErr
		}
	}
	return grant, nil
}

// ListUserGrants 返回某用户的全部 Grant（管理员/用户面板可见性）。
func (s *DailyGrantService) ListUserGrants(ctx context.Context, userID int64) ([]DailyBalanceGrant, error) {
	return s.grantRepo.ListByUser(ctx, userID)
}

// ActiveRemaining 返回某用户在某专属分组下当前有效的每日额度剩余合计。
func (s *DailyGrantService) ActiveRemaining(ctx context.Context, userID, groupID int64) (float64, error) {
	return s.grantRepo.SumActiveRemaining(ctx, userID, groupID, time.Now())
}
