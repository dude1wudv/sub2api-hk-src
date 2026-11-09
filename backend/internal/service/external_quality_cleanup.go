package service

import (
	"context"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
)

type externalQualityPruner interface {
	PruneExternalQuality(context.Context) error
}

// This maintenance loop shares the existing worker's cancellation lifecycle;
// it holds no artwork or event cache and is independent of passive rollups.
func (s *ChannelMonitorV2Aggregator) externalQualityCleanupLoop(ctx context.Context) {
	pruner, ok := s.repo.(externalQualityPruner)
	if !ok {
		return
	}
	cleanup := func() {
		pruneCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
		defer cancel()
		if err := pruner.PruneExternalQuality(pruneCtx); err != nil && ctx.Err() == nil {
			logger.LegacyPrintf("service.external_quality", "retention cleanup failed: %v", err)
		}
	}
	cleanup()
	ticker := time.NewTicker(8 * time.Hour)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			cleanup()
		}
	}
}
