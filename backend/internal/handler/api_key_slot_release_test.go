package handler

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestAPIKeyAdmissionHTTPReleaseWaitsForForwarding(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var keyReleases atomic.Int32
	userStopped := make(chan struct{})
	release := wrapUserAndAPIKeyRelease(ctx, 1, func() { close(userStopped) }, func() { keyReleases.Add(1) })
	cancel()
	select {
	case <-userStopped:
	case <-time.After(time.Second):
		t.Fatal("client cancellation did not release user slot")
	}
	require.Zero(t, keyReleases.Load(), "Key capacity must remain held while detached upstream drains")
	release()
	release()
	require.EqualValues(t, 1, keyReleases.Load(), "forwarding completion releases Key capacity exactly once")
}

func TestAPIKeyAdmissionHTTPUnlimitedStatsReleaseOnCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	keyStopped := make(chan struct{})
	release := wrapUserAndAPIKeyRelease(ctx, 0, nil, func() { close(keyStopped) })
	cancel()
	select {
	case <-keyStopped:
	case <-time.After(time.Second):
		t.Fatal("unlimited Key stats did not clean up on cancellation")
	}
	release()
}
