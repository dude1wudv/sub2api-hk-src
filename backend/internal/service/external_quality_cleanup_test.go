package service

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type externalQualityPrunerStub struct {
	ChannelMonitorV2Repository
	called chan struct{}
}

func (r *externalQualityPrunerStub) PruneExternalQuality(context.Context) error {
	r.called <- struct{}{}
	return nil
}

func TestExternalQualityCleanupStartsAndHonorsWorkerCancellation(t *testing.T) {
	repo := &externalQualityPrunerStub{called: make(chan struct{}, 1)}
	aggregator := NewChannelMonitorV2Aggregator(repo, nil, nil)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan struct{})
	go func() { aggregator.externalQualityCleanupLoop(ctx); close(done) }()
	select {
	case <-repo.called:
	case <-time.After(time.Second):
		t.Fatal("initial cleanup did not run")
	}
	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("cleanup did not stop")
	}
	require.Empty(t, repo.called)
}
