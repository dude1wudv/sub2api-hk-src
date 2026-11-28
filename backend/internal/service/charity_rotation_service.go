package service

import (
	"context"
	"log/slog"
	"sync"
	"time"
)

const charityRotationCheckInterval = 30 * time.Second

// CharityRotationService 定时检测北京时间 00:00 跨日，自动轮换公益池各分组的公共 API Key。
type CharityRotationService struct {
	charityService *CharityService

	mu     sync.Mutex
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// NewCharityRotationService 创建并配置每日 Key 轮换后台服务。
func NewCharityRotationService(charityService *CharityService) *CharityRotationService {
	return &CharityRotationService{charityService: charityService}
}

// Start 启动后台检测协程，并在启动时立即做一次跨日补偿检查。
func (s *CharityRotationService) Start() {
	if s == nil || s.charityService == nil {
		return
	}
	s.mu.Lock()
	if s.cancel != nil {
		s.mu.Unlock()
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	s.cancel = cancel
	s.wg.Add(1)
	s.mu.Unlock()

	go func() {
		defer s.wg.Done()
		s.runOnce(ctx)

		ticker := time.NewTicker(charityRotationCheckInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				s.runOnce(ctx)
			}
		}
	}()
}

// Stop 优雅停止后台检测协程。
func (s *CharityRotationService) Stop() {
	if s == nil {
		return
	}
	s.mu.Lock()
	cancel := s.cancel
	s.cancel = nil
	s.mu.Unlock()
	if cancel != nil {
		cancel()
	}
	s.wg.Wait()
}

func (s *CharityRotationService) runOnce(ctx context.Context) {
	checkCtx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()
	if _, err := s.charityService.RotateDueDailyKeys(checkCtx); err != nil {
		slog.Warn("charity: rotation check failed", "error", err)
	}
}
