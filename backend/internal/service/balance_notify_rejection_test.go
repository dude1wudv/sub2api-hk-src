package service

import (
	"context"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type rejectionNotifySettings struct {
	SettingRepository
	enabled atomic.Bool
	reads   atomic.Int64
	started chan context.Context
	resume  chan struct{}
}

func (s *rejectionNotifySettings) GetMultiple(ctx context.Context, _ []string) (map[string]string, error) {
	s.reads.Add(1)
	if s.started != nil {
		s.started <- ctx
	}
	if s.resume != nil {
		select {
		case <-s.resume:
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	return map[string]string{SettingKeyBalanceLowNotifyEnabled: strconv.FormatBool(s.enabled.Load()), SettingKeyBalanceLowNotifyThreshold: "10"}, nil
}

func (*rejectionNotifySettings) GetValue(context.Context, string) (string, error) { return "Test", nil }

type rejectionNotifyEmail struct {
	calls      atomic.Int64
	deliveries chan string
}

func (s *rejectionNotifyEmail) SendEmail(ctx context.Context, to, _, _ string) error {
	if _, ok := ctx.Deadline(); !ok {
		panic("notification delivery must have a deadline")
	}
	s.calls.Add(1)
	if s.deliveries != nil {
		s.deliveries <- to
	}
	return nil
}

func rejectionNotifyUser(id int64) *User {
	return &User{ID: id, Balance: -0.01, BalanceNotifyEnabled: true, BalanceNotifyExtraEmails: []NotifyEmailEntry{
		{Email: "verified@example.test", Verified: true}, {Email: "disabled@example.test", Verified: true, Disabled: true}, {Email: "unverified@example.test"},
	}}
}

func TestBalanceRejectionNotificationAsyncBoundedAndCooldown(t *testing.T) {
	settings := &rejectionNotifySettings{started: make(chan context.Context, 2), resume: make(chan struct{})}
	settings.enabled.Store(true)
	email := &rejectionNotifyEmail{deliveries: make(chan string, 3)}
	svc := NewBalanceNotifyService(email, settings, nil)
	t.Cleanup(svc.Stop)
	returned := make(chan struct{})
	user := rejectionNotifyUser(9)
	go func() { svc.NotifyInsufficientBalance(user); close(returned) }()
	select {
	case <-returned:
	case <-time.After(2 * time.Second):
		t.Fatal("auth notification blocked on settings")
	}
	select {
	case ctx := <-settings.started:
		_, ok := ctx.Deadline()
		require.True(t, ok)
	case <-time.After(2 * time.Second):
		t.Fatal("notification task did not run")
	}
	var wg sync.WaitGroup
	for range 50 {
		wg.Add(1)
		go func() { defer wg.Done(); svc.NotifyInsufficientBalance(user) }()
	}
	wg.Wait()
	require.Equal(t, int64(1), settings.reads.Load(), "same-user concurrent rejections must share the cooldown")
	close(settings.resume)
	select {
	case to := <-email.deliveries:
		require.Equal(t, "verified@example.test", to)
	case <-time.After(2 * time.Second):
		t.Fatal("notification was not delivered")
	}
	svc.NotifyInsufficientBalance(user)
	require.Equal(t, int64(1), settings.reads.Load())
	svc.exhaustedNotified.Set(strconv.FormatInt(user.ID, 10), true, time.Nanosecond)
	require.Eventually(t, func() bool { _, exists := svc.exhaustedNotified.Get(strconv.FormatInt(user.ID, 10)); return !exists }, time.Second, time.Millisecond)
	svc.NotifyInsufficientBalance(user)
	select {
	case <-email.deliveries:
	case <-time.After(2 * time.Second):
		t.Fatal("expired cooldown did not permit delivery")
	}
	svc.usageNotifications.pool.StopAndWait()
	require.Equal(t, int64(2), email.calls.Load())
}

func TestBalanceRejectionNotificationRespectsSwitches(t *testing.T) {
	for _, tt := range []struct {
		name                string
		global, userEnabled bool
		balance             float64
		reads               int64
	}{
		{"global disabled", false, true, 0, 1}, {"user disabled", true, false, 0, 0}, {"positive balance", true, true, 0.01, 0},
	} {
		t.Run(tt.name, func(t *testing.T) {
			settings := &rejectionNotifySettings{}
			settings.enabled.Store(tt.global)
			email := &rejectionNotifyEmail{}
			svc := NewBalanceNotifyService(email, settings, nil)
			t.Cleanup(svc.Stop)
			user := rejectionNotifyUser(8)
			user.BalanceNotifyEnabled, user.Balance = tt.userEnabled, tt.balance
			svc.NotifyInsufficientBalance(user)
			svc.usageNotifications.pool.StopAndWait()
			require.Equal(t, tt.reads, settings.reads.Load())
			require.Zero(t, email.calls.Load())
		})
	}
}

func TestBalanceRejectionNotificationQueueFullDoesNotConsumeCooldown(t *testing.T) {
	settings := &rejectionNotifySettings{}
	settings.enabled.Store(true)
	svc := NewBalanceNotifyService(&rejectionNotifyEmail{}, settings, nil)
	t.Cleanup(svc.Stop)
	started := make(chan struct{}, usageNotificationWorkers)
	for range usageNotificationWorkers {
		require.True(t, svc.usageNotifications.submit(func(ctx context.Context) { started <- struct{}{}; <-ctx.Done() }))
	}
	for range usageNotificationWorkers {
		<-started
	}
	for range usageNotificationQueueSize {
		require.True(t, svc.usageNotifications.submit(func(context.Context) {}))
	}
	user := rejectionNotifyUser(10)
	svc.NotifyInsufficientBalance(user)
	_, reserved := svc.exhaustedNotified.Get(strconv.FormatInt(user.ID, 10))
	require.False(t, reserved, "a dropped notification must be retryable")
	require.Zero(t, settings.reads.Load())
}
