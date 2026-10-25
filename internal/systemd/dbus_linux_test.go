//go:build linux

package systemd

import (
	"errors"
	"fmt"
	"testing"
	"time"

	godbus "github.com/godbus/dbus/v5"
)

// withServiceSuffix is the gate every DBus unit op passes through. A bug here
// breaks Start/Stop/Restart/Enable/Disable/IsEnabled for every caller, so the
// table covers the inputs each callsite actually emits.
func TestWithServiceSuffix(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"lerd-ui", "lerd-ui.service"},
		{"lerd-watcher", "lerd-watcher.service"},
		{"lerd-queue-myapp", "lerd-queue-myapp.service"},
		{"lerd-ui.service", "lerd-ui.service"},
		{"lerd-test.timer", "lerd-test.timer"},
		{"lerd-stripe-my.app", "lerd-stripe-my.app"},
		{"", ".service"},
	}
	for _, tc := range cases {
		if got := withServiceSuffix(tc.in); got != tc.want {
			t.Errorf("withServiceSuffix(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

// withDefaultSuffix is used by property lookups that may legitimately target
// a .timer or a .service. Bare names default to .service; explicit suffixes
// pass through verbatim so a name+".timer" composition works.
func TestWithDefaultSuffix(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"lerd-ui", "lerd-ui.service"},
		{"lerd-test.timer", "lerd-test.timer"},
		{"lerd-test.service", "lerd-test.service"},
		{"lerd-queue-myapp.timer", "lerd-queue-myapp.timer"},
		{"some.weird.name", "some.weird.name"},
	}
	for _, tc := range cases {
		if got := withDefaultSuffix(tc.in); got != tc.want {
			t.Errorf("withDefaultSuffix(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

// Timer lookups compose name+".timer" and ask DBus. The suffix rule must
// pass ".timer" through unchanged so a lookup doesn't end up querying
// "lerd-foo.timer.service".
func TestTimerSuffixIsNotDoubleAppended(t *testing.T) {
	if got := withServiceSuffix("lerd-foo.timer"); got != "lerd-foo.timer" {
		t.Errorf("withServiceSuffix(%q) = %q, want passthrough, a timer lookup would query the wrong unit", "lerd-foo.timer", got)
	}
	if got := withDefaultSuffix("lerd-foo.timer"); got != "lerd-foo.timer" {
		t.Errorf("withDefaultSuffix(%q) = %q, want passthrough", "lerd-foo.timer", got)
	}
}

// NotifyReady and NotifyStopping must be safe to call outside a systemd
// notify-socket context: bare CLI runs (lerd serve-ui from a terminal,
// go test) have no $NOTIFY_SOCKET and the underlying daemon.SdNotify
// returns (false, nil) — no panic, no error surfaced.
func TestNotifyReadyAndStoppingAreSafeWithoutSocket(t *testing.T) {
	t.Setenv("NOTIFY_SOCKET", "")
	NotifyReady()
	NotifyStopping()
}

// runUnitOpWithRetry is the stop-reliability fix for `lerd stop` leaving a
// container running with "stop … failed: canceled" (a parallel "replace" stop
// of an interdependent unit). It must re-issue only on "canceled", stop as
// soon as the job is "done", give up after maxAttempts, and never swallow a
// transport error. A nil settle (and the production settle being skipped here)
// keeps the test instant.
func TestRunUnitOpWithRetry(t *testing.T) {
	t.Run("canceled then done", func(t *testing.T) {
		results := []string{"canceled", "canceled", "done"}
		var calls int
		got, err := runUnitOpWithRetry(stopRetryAttempts, nil, func() (string, error) {
			r := results[calls]
			calls++
			return r, nil
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != "done" {
			t.Fatalf("result = %q, want done", got)
		}
		if calls != 3 {
			t.Fatalf("calls = %d, want 3 (retried until done)", calls)
		}
	})

	t.Run("gives up after maxAttempts and returns last result", func(t *testing.T) {
		var calls int
		got, err := runUnitOpWithRetry(3, nil, func() (string, error) {
			calls++
			return "canceled", nil
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != "canceled" {
			t.Fatalf("result = %q, want canceled (final result surfaced)", got)
		}
		if calls != 3 {
			t.Fatalf("calls = %d, want 3 (bounded by maxAttempts)", calls)
		}
	})

	t.Run("non-canceled non-done result is not retried", func(t *testing.T) {
		var calls int
		got, err := runUnitOpWithRetry(stopRetryAttempts, nil, func() (string, error) {
			calls++
			return "failed", nil
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != "failed" || calls != 1 {
			t.Fatalf("result=%q calls=%d, want failed/1 (no retry on hard failure)", got, calls)
		}
	})

	t.Run("done on first attempt", func(t *testing.T) {
		var calls int
		got, _ := runUnitOpWithRetry(stopRetryAttempts, nil, func() (string, error) {
			calls++
			return "done", nil
		})
		if got != "done" || calls != 1 {
			t.Fatalf("result=%q calls=%d, want done/1", got, calls)
		}
	})

	t.Run("transport error is surfaced immediately", func(t *testing.T) {
		var calls int
		_, err := runUnitOpWithRetry(stopRetryAttempts, nil, func() (string, error) {
			calls++
			return "", errUnitOpTimedOut
		})
		if err == nil || calls != 1 {
			t.Fatalf("err=%v calls=%d, want non-nil/1 (no retry on transport error)", err, calls)
		}
	})

	t.Run("maxAttempts below one is clamped to a single attempt", func(t *testing.T) {
		var calls int
		runUnitOpWithRetry(0, nil, func() (string, error) { //nolint:errcheck
			calls++
			return "canceled", nil
		})
		if calls != 1 {
			t.Fatalf("calls = %d, want 1 (clamped)", calls)
		}
	})
}

// A stop aimed at a unit systemd has never loaded is already satisfied: there
// is nothing running to stop. `lerd stop` enumerates workers a site declares,
// which on a machine that never installed one names a unit systemd does not
// know, so a clean shutdown reported a screen of failures for workers that were
// never up.
func TestUnitNotLoaded(t *testing.T) {
	missing := godbus.Error{
		Name: "org.freedesktop.systemd1.NoSuchUnit",
		Body: []any{"Unit lerd-queue-acme.service not loaded."},
	}
	if !unitNotLoaded(fmt.Errorf("stop lerd-queue-acme failed: %w", missing)) {
		t.Error("a NoSuchUnit error must read as already stopped")
	}

	// A unit that exists and genuinely failed to stop must still be an error, or
	// a stop that left a container running would report success.
	realFailure := godbus.Error{
		Name: "org.freedesktop.systemd1.JobFailed",
		Body: []any{"Job for lerd-mysql.service failed."},
	}
	if unitNotLoaded(fmt.Errorf("stop lerd-mysql failed: %w", realFailure)) {
		t.Error("a genuine stop failure was swallowed")
	}
	if unitNotLoaded(errors.New("stop lerd-mysql timed out after 30s")) {
		t.Error("a timeout was swallowed")
	}
}

// A stop has to outlast the graceful window the unit itself declares. mysql
// and postgres declare stop_timeout 60, which reaches the unit as
// TimeoutStopSec=75, so the fixed 30s wait returned as though the service had
// come down while podman was still counting; the SIGKILL that followed left
// the unit failed behind an uninstall that had already finished (#1510).
func TestStopJobWait(t *testing.T) {
	cases := []struct {
		name     string
		declared time.Duration
		want     time.Duration
	}{
		{"a unit that declares nothing keeps the historic wait", 0, jobWaitFloor},
		{"an unreadable window keeps the historic wait", -1, jobWaitFloor},
		{"a window shorter than the floor keeps the floor", 5 * time.Second, jobWaitFloor},
		{"the default service window still fits the floor", 20 * time.Second, jobWaitFloor},
		{"a database's window outlasts its own SIGKILL", 75 * time.Second, 85 * time.Second},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := stopJobWait(tc.declared); got != tc.want {
				t.Errorf("stopJobWait(%s) = %s, want %s", tc.declared, got, tc.want)
			}
		})
	}
}

// The timeout message reports the wait that actually elapsed. It used to say
// 30s unconditionally, which names the wrong number the moment the wait comes
// from the unit, and unitNotLoaded reads these messages.
func TestUnitOpTimeoutMessageReportsTheRealWait(t *testing.T) {
	err := unitOpTimeoutError("stop", "lerd-mysql", 85*time.Second)
	if got, want := err.Error(), "stop lerd-mysql timed out after 1m25s"; got != want {
		t.Errorf("message = %q, want %q", got, want)
	}
	if unitNotLoaded(err) {
		t.Error("a timeout must not read as an already-stopped unit")
	}
}
