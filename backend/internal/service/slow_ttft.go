package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

const SlowTTFTConfigKey = "slow_ttft_protection"

type SlowTTFTConfig struct {
	Enabled          bool   `json:"enabled"`
	ThresholdSeconds int    `json:"threshold_seconds"`
	ConsecutiveCount int    `json:"consecutive_count"`
	WindowSeconds    int    `json:"window_seconds"`
	WindowCount      int    `json:"window_count"`
	PauseSeconds     int    `json:"pause_seconds"`
	Generation       string `json:"generation,omitempty"`
}

func DefaultSlowTTFTConfig() SlowTTFTConfig {
	return SlowTTFTConfig{ThresholdSeconds: 15, ConsecutiveCount: 2, WindowSeconds: 300, WindowCount: 3, PauseSeconds: 1800}
}
func (a *Account) SlowTTFTConfig() SlowTTFTConfig {
	c := DefaultSlowTTFTConfig()
	if a != nil && a.Extra != nil {
		if b, e := json.Marshal(a.Extra[SlowTTFTConfigKey]); e == nil && string(b) != "null" {
			_ = json.Unmarshal(b, &c)
		}
	}
	return c
}
func ValidateSlowTTFTExtra(extra map[string]any) error {
	v, ok := extra[SlowTTFTConfigKey]
	if !ok {
		return nil
	}
	b, e := json.Marshal(v)
	var c SlowTTFTConfig
	if e != nil || json.Unmarshal(b, &c) != nil || c.ThresholdSeconds < 1 || c.ThresholdSeconds > 3600 || c.ConsecutiveCount < 1 || c.ConsecutiveCount > 1000 || c.WindowCount < 1 || c.WindowCount > 1000 || c.WindowSeconds < 1 || c.WindowSeconds > 86400 || c.PauseSeconds < 1 || c.PauseSeconds > 604800 {
		return infraerrors.BadRequest("INVALID_SLOW_TTFT_CONFIG", "invalid slow first-output protection settings")
	}
	return nil
}

// Validate explicit configuration writes; the database owns configuration epochs
// so imports, bulk updates and manual clears invalidate old observations atomically.
func PrepareSlowTTFTExtra(extra map[string]any) error {
	if err := ValidateSlowTTFTExtra(extra); err != nil {
		return err
	}
	if _, ok := extra[SlowTTFTConfigKey]; !ok {
		return nil
	}
	c := (&Account{Extra: extra}).SlowTTFTConfig()
	extra[SlowTTFTConfigKey] = c
	return nil
}

type SlowTTFTObservation struct {
	Until   time.Time
	Reason  string
	Tripped bool
}
type SlowTTFTCache interface {
	ObserveSlowTTFT(context.Context, int64, SlowTTFTConfig, string, bool) (SlowTTFTObservation, error)
}
type SlowTTFTRepository interface {
	SetSlowTTFTPause(context.Context, int64, SlowTTFTConfig, time.Time, string) (bool, error)
	ClearSlowTTFTPause(context.Context, int64) error
}

func (s *RateLimitService) BeginSlowTTFT(ctx context.Context, account *Account) *SlowTTFTObserver {
	if s == nil || account == nil {
		return nil
	}
	cfg := account.SlowTTFTConfig()
	if !cfg.Enabled || cfg.ThresholdSeconds < 1 {
		return nil
	}
	if s.slowTTFTGroupExempt(ctx) {
		return nil
	}
	cache, ok := s.tempUnschedCache.(SlowTTFTCache)
	if !ok {
		return nil
	}
	repo, ok := s.accountRepo.(SlowTTFTRepository)
	if !ok {
		return nil
	}
	o := &SlowTTFTObserver{start: time.Now(), threshold: time.Duration(cfg.ThresholdSeconds) * time.Second}
	attempt := generateRequestID()
	o.report = func(slow bool) {
		bg, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
		defer cancel()
		if s.slowTTFTGroupExempt(bg) {
			return
		}
		fresh, err := s.accountRepo.GetByID(bg, account.ID)
		if err != nil || fresh == nil || fresh.SlowTTFTConfig() != cfg {
			return
		}
		if fresh.SlowTTFTUntil != nil && time.Now().Before(*fresh.SlowTTFTUntil) {
			return
		}
		result, err := cache.ObserveSlowTTFT(bg, account.ID, cfg, attempt, slow)
		if err != nil {
			slog.Warn("slow_ttft_observation_failed", "account_id", account.ID, "error", err)
			return
		}
		if result.Tripped {
			applied, err := repo.SetSlowTTFTPause(bg, account.ID, cfg, result.Until, result.Reason)
			if err != nil {
				slog.Error("slow_ttft_pause_persist_failed", "account_id", account.ID, "error", err)
			} else if !applied {
				slog.Debug("slow_ttft_observation_superseded", "account_id", account.ID)
			}
		}
	}
	o.mu.Lock()
	o.timer = time.AfterFunc(o.threshold+time.Millisecond, func() { o.finish(true, true) })
	o.stopCancel = context.AfterFunc(ctx, func() { o.Close() })
	o.mu.Unlock()
	return o
}

type SlowTTFTObserver struct {
	mu         sync.Mutex
	done       bool
	start      time.Time
	threshold  time.Duration
	timer      *time.Timer
	stopCancel func() bool
	report     func(bool)
}

func (o *SlowTTFTObserver) finish(sample, slow bool) {
	if o == nil {
		return
	}
	o.mu.Lock()
	if o.done {
		o.mu.Unlock()
		return
	}
	o.done = true
	if o.timer != nil {
		o.timer.Stop()
	}
	if o.stopCancel != nil {
		o.stopCancel()
	}
	report := o.report
	o.mu.Unlock()
	// Report in the observing call instead of launching an unordered goroutine.
	// Redis supplies the common observation order across concurrent instances.
	if sample && report != nil {
		report(slow)
	}
}
func (o *SlowTTFTObserver) FirstOutput() {
	if o != nil {
		o.finish(true, time.Since(o.start) > o.threshold)
	}
}
func (o *SlowTTFTObserver) Close() {
	if o != nil {
		o.finish(time.Since(o.start) > o.threshold, true)
	}
}

func (s *RateLimitService) SlowTTFTPaused(ctx context.Context, id int64) bool {
	if s == nil || s.slowTTFTGroupExempt(ctx) {
		return false
	}
	// The persisted epoch is authoritative: stale Redis pause keys after an edit
	// or clear must not block a newly configured account.
	if s.accountRepo != nil {
		a, err := s.accountRepo.GetByID(ctx, id)
		return err == nil && a != nil && a.SlowTTFTConfig().Enabled && a.SlowTTFTUntil != nil && time.Now().Before(*a.SlowTTFTUntil)
	}
	return false
}
func (s *adminServiceImpl) ClearSlowTTFTPause(ctx context.Context, id int64) error {
	r, ok := s.accountRepo.(SlowTTFTRepository)
	if !ok {
		return fmt.Errorf("slow TTFT repository unavailable")
	}
	return r.ClearSlowTTFTPause(ctx, id)
}
