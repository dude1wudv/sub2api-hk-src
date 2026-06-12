package repository

import (
	"context"
	"database/sql"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/ent/dailybalancegrant"
	"github.com/Wei-Shaw/sub2api/internal/domain"
	"github.com/Wei-Shaw/sub2api/internal/service"
)

type dailyGrantRepository struct {
	client *dbent.Client
	sql    sqlExecutor
}

// NewDailyGrantRepository 构造每日余额赠额仓储。
func NewDailyGrantRepository(client *dbent.Client, sqlDB *sql.DB) service.DailyGrantRepository {
	return &dailyGrantRepository{client: client, sql: sqlDB}
}

func (r *dailyGrantRepository) CreateGrant(ctx context.Context, grant *service.DailyBalanceGrant) (int64, error) {
	client := clientFromContext(ctx, r.client)
	create := client.DailyBalanceGrant.Create().
		SetUserID(grant.UserID).
		SetGroupID(grant.GroupID).
		SetAmount(grant.Amount).
		SetRemaining(grant.Remaining).
		SetStatus(grant.Status).
		SetSource(grant.Source).
		SetExpiresAt(grant.ExpiresAt)
	if !grant.GrantedAt.IsZero() {
		create = create.SetGrantedAt(grant.GrantedAt)
	}
	if grant.SourceRef != nil {
		create = create.SetSourceRef(*grant.SourceRef)
	}
	created, err := create.Save(ctx)
	if err != nil {
		return 0, err
	}
	return created.ID, nil
}

func (r *dailyGrantRepository) SumActiveRemaining(ctx context.Context, userID, groupID int64, now time.Time) (float64, error) {
	client := clientFromContext(ctx, r.client)
	var result []struct {
		Sum float64 `json:"sum"`
	}
	err := client.DailyBalanceGrant.Query().
		Where(
			dailybalancegrant.UserIDEQ(userID),
			dailybalancegrant.GroupIDEQ(groupID),
			dailybalancegrant.StatusEQ(domain.DailyGrantStatusActive),
			dailybalancegrant.ExpiresAtGT(now),
		).
		Aggregate(dbent.As(dbent.Sum(dailybalancegrant.FieldRemaining), "sum")).
		Scan(ctx, &result)
	if err != nil {
		return 0, err
	}
	if len(result) == 0 {
		return 0, nil
	}
	return result[0].Sum, nil
}

func (r *dailyGrantRepository) ListActiveByExpiry(ctx context.Context, userID, groupID int64, now time.Time) ([]service.DailyBalanceGrant, error) {
	client := clientFromContext(ctx, r.client)
	rows, err := client.DailyBalanceGrant.Query().
		Where(
			dailybalancegrant.UserIDEQ(userID),
			dailybalancegrant.GroupIDEQ(groupID),
			dailybalancegrant.StatusEQ(domain.DailyGrantStatusActive),
			dailybalancegrant.ExpiresAtGT(now),
			dailybalancegrant.RemainingGT(0),
		).
		Order(dbent.Asc(dailybalancegrant.FieldExpiresAt)).
		All(ctx)
	if err != nil {
		return nil, err
	}
	return mapDailyGrants(rows), nil
}

// AtomicDecrement 原子扣减指定 Grant 的剩余额度。
//
// 并发安全：条件更新 WHERE status='active' AND remaining>=amount，依赖 DB 行锁串行化
// 并发请求，确保同一笔 Grant 不会被重复消费。受影响行数为 0 表示该 Grant 已不满足条件，
// 调用方应跳到下一笔。扣减后剩余归零时，同一条语句内把 status 置为 exhausted。
func (r *dailyGrantRepository) AtomicDecrement(ctx context.Context, grantID int64, amount float64) (bool, error) {
	if amount <= 0 {
		return false, nil
	}
	// clientFromContext 在事务上下文中返回 tx client，其 ExecContext 复用同一连接，
	// 使 Grant 递减与余额扣减落在同一事务（T4 拆分结算依赖此点）。
	client := clientFromContext(ctx, r.client)
	res, err := client.ExecContext(ctx, `
		UPDATE daily_balance_grants
		SET remaining = remaining - $1,
			status = CASE WHEN remaining - $1 <= 0 THEN 'exhausted' ELSE status END
		WHERE id = $2
		  AND status = 'active'
		  AND remaining >= $1
	`, amount, grantID)
	if err != nil {
		return false, err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return false, err
	}
	return affected > 0, nil
}

func (r *dailyGrantRepository) MarkExpired(ctx context.Context, now time.Time) (int64, error) {
	client := clientFromContext(ctx, r.client)
	n, err := client.DailyBalanceGrant.Update().
		Where(
			dailybalancegrant.StatusEQ(domain.DailyGrantStatusActive),
			dailybalancegrant.ExpiresAtLT(now),
		).
		SetStatus(domain.DailyGrantStatusExpired).
		Save(ctx)
	if err != nil {
		return 0, err
	}
	return int64(n), nil
}

func (r *dailyGrantRepository) ListByUser(ctx context.Context, userID int64) ([]service.DailyBalanceGrant, error) {
	client := clientFromContext(ctx, r.client)
	rows, err := client.DailyBalanceGrant.Query().
		Where(dailybalancegrant.UserIDEQ(userID)).
		Order(dbent.Desc(dailybalancegrant.FieldCreatedAt)).
		All(ctx)
	if err != nil {
		return nil, err
	}
	return mapDailyGrants(rows), nil
}

func mapDailyGrants(rows []*dbent.DailyBalanceGrant) []service.DailyBalanceGrant {
	out := make([]service.DailyBalanceGrant, 0, len(rows))
	for _, row := range rows {
		out = append(out, mapDailyGrant(row))
	}
	return out
}

func mapDailyGrant(row *dbent.DailyBalanceGrant) service.DailyBalanceGrant {
	grant := service.DailyBalanceGrant{
		ID:        row.ID,
		UserID:    row.UserID,
		GroupID:   row.GroupID,
		Amount:    row.Amount,
		Remaining: row.Remaining,
		Status:    row.Status,
		Source:    row.Source,
		GrantedAt: row.GrantedAt,
		ExpiresAt: row.ExpiresAt,
		CreatedAt: row.CreatedAt,
	}
	if row.SourceRef != nil {
		grant.SourceRef = row.SourceRef
	}
	return grant
}
