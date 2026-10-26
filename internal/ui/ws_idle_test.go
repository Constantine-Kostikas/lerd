package ui

import (
	"testing"
	"time"
)

// chooseInterval is the only branching point that decides the cache poll
// cadence, over (visible>0, focused, idle) plus edge values for the counter.
func TestChooseInterval(t *testing.T) {
	cases := []struct {
		name    string
		visible int32
		focused bool
		idle    bool
		want    time.Duration
	}{
		{"focused tab, active session", 1, true, false, intervalFocused},
		{"focused tab, idle session", 1, true, true, intervalIdle},
		{"tab open behind another window", 1, false, false, intervalIdle},
		{"no tabs, active session", 0, false, false, intervalIdle},
		{"no tabs, idle session", 0, false, true, intervalIdle},
		{"many tabs, one focused", 7, true, false, intervalFocused},
		{"many tabs, idle session", 7, true, true, intervalIdle},
		{"negative counter, active session", -1, true, false, intervalIdle},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := chooseInterval(tc.visible, tc.focused, tc.idle); got != tc.want {
				t.Errorf("chooseInterval(%d, %v, %v) = %v, want %v", tc.visible, tc.focused, tc.idle, got, tc.want)
			}
		})
	}
}

// intervalFocused must be strictly shorter than intervalIdle, otherwise the
// "fast cadence on focused tabs" promise is meaningless. Pin the invariant
// so a future tweak to the constants can't silently break it.
func TestFocusedIntervalIsFasterThanIdle(t *testing.T) {
	if intervalFocused >= intervalIdle {
		t.Errorf("intervalFocused (%v) must be < intervalIdle (%v)", intervalFocused, intervalIdle)
	}
}
