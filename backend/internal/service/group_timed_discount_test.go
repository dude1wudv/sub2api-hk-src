//go:build unit

package service

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestGroupTimedDiscountOpenAt(t *testing.T) {
	loc := time.FixedZone(TimedDiscountTimezone, 8*60*60)
	group := &Group{
		TimedDiscountEnabled:     true,
		TimedDiscountStartMinute: 30,
		TimedDiscountEndMinute:   7*60 + 30,
	}

	require.True(t, group.TimedDiscountOpenAt(time.Date(2026, 7, 1, 0, 30, 0, 0, loc)))
	require.True(t, group.TimedDiscountOpenAt(time.Date(2026, 7, 1, 7, 29, 0, 0, loc)))
	require.False(t, group.TimedDiscountOpenAt(time.Date(2026, 7, 1, 7, 30, 0, 0, loc)))
	require.False(t, group.TimedDiscountOpenAt(time.Date(2026, 7, 1, 0, 29, 0, 0, loc)))
}

func TestGroupTimedDiscountOpenAtOvernightWindow(t *testing.T) {
	loc := time.FixedZone(TimedDiscountTimezone, 8*60*60)
	group := &Group{
		TimedDiscountEnabled:     true,
		TimedDiscountStartMinute: 23 * 60,
		TimedDiscountEndMinute:   2 * 60,
	}

	require.True(t, group.TimedDiscountOpenAt(time.Date(2026, 7, 1, 23, 30, 0, 0, loc)))
	require.True(t, group.TimedDiscountOpenAt(time.Date(2026, 7, 2, 1, 30, 0, 0, loc)))
	require.False(t, group.TimedDiscountOpenAt(time.Date(2026, 7, 1, 12, 0, 0, 0, loc)))
}
