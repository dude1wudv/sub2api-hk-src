package service

import (
	"context"
	"testing"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/stretchr/testify/require"
)

// redeemRepoGetByCodeStub 通过嵌入接口实现最小桩：只覆盖 GetByCode，
// 未实现的方法保持 nil（本测试的 guard 路径不会触达它们）。
type redeemRepoGetByCodeStub struct {
	RedeemCodeRepository
	code *RedeemCode
}

func (s *redeemRepoGetByCodeStub) GetByCode(ctx context.Context, code string) (*RedeemCode, error) {
	return s.code, nil
}

func dailyBalanceCode(groupID *int64) *RedeemCode {
	return &RedeemCode{
		ID:        1,
		Code:      "DAILY-TEST",
		Type:      RedeemTypeDailyBalance,
		Value:     10,
		Status:    StatusUnused,
		GroupID:   groupID,
		ExpiresAt: nil,
	}
}

// daily_balance 兑换码缺少 group_id 时应被拒绝（在任何事务/扣减之前）。
func TestRedeem_DailyBalance_MissingGroupID_Rejected(t *testing.T) {
	svc := &RedeemService{
		redeemRepo: &redeemRepoGetByCodeStub{code: dailyBalanceCode(nil)},
		// cache 为 nil → 跳过限流与分布式锁
		// dailyGrantRepo / groupRepo 未注入
	}
	_, err := svc.Redeem(context.Background(), 1, "DAILY-TEST")
	require.Error(t, err)
	require.True(t, infraerrors.IsBadRequest(err), "缺少 group_id 应返回 BadRequest，实际: %v", err)
}

// daily_balance 兑换码在依赖未注入（功能未启用）时应被拒绝。
func TestRedeem_DailyBalance_DepsNotInjected_Rejected(t *testing.T) {
	groupID := int64(7)
	svc := &RedeemService{
		redeemRepo: &redeemRepoGetByCodeStub{code: dailyBalanceCode(&groupID)},
		// dailyGrantRepo / groupRepo 未注入 → 拒绝
	}
	_, err := svc.Redeem(context.Background(), 1, "DAILY-TEST")
	require.Error(t, err)
	require.True(t, infraerrors.IsBadRequest(err), "依赖未注入应返回 BadRequest，实际: %v", err)
}
