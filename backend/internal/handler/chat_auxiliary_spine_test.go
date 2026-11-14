//go:build unit

package handler

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestChatAuxiliaryUsageAlwaysUsesMandatoryQueue(t *testing.T) {
	pool := service.NewUsageRecordWorkerPoolWithOptions(service.UsageRecordWorkerPoolOptions{WorkerCount: 1, QueueSize: 1, TaskTimeout: time.Second, OverflowPolicy: "drop"})
	t.Cleanup(pool.Stop)
	h := &OpenAIGatewayHandler{usageRecordWorkerPool: pool}
	started, release := make(chan struct{}), make(chan struct{})
	pool.Submit(func(context.Context) { close(started); <-release })
	<-started
	defer close(release)
	pool.Submit(func(context.Context) {})
	var called atomic.Bool
	result := &service.OpenAIForwardResult{ChatAuxiliaryUsage: &service.ChatAuxiliaryUsage{PriceBasis: service.ChatAuxiliaryPriceBasis, Provider: "deepseek"}}
	h.submitOpenAIUsageRecordTask(t.Context(), result, func(context.Context) { called.Store(true) })
	require.True(t, called.Load(), "metadata-only and error-terminal bills cannot be dropped")
	require.Zero(t, result.SearchCount, "must not fake model search usage")
}
