package serviceops

import (
	"testing"
	"time"

	"github.com/geodro/lerd/internal/podman"
)

// A running service is probed once per run of its unit: every php or composer
// command checks it, and under WSL each probe is a third of a second.
func TestEnsureServiceRunning_probesOncePerActivation(t *testing.T) {
	withServiceHome(t)
	prevLife := podman.UnitLifecycle
	podman.UnitLifecycle = activeLifecycle{}
	t.Cleanup(func() { podman.UnitLifecycle = prevLife })
	probes := 0
	prevWait := waitReadyFn
	waitReadyFn = func(string, time.Duration) error { probes++; return nil }
	t.Cleanup(func() { waitReadyFn = prevWait })
	id := "run-1"
	prevID := activationIDFn
	activationIDFn = func(string) string { return id }
	t.Cleanup(func() { activationIDFn = prevID })

	for i := 0; i < 3; i++ {
		if err := EnsureServiceRunning("mysql"); err != nil {
			t.Fatal(err)
		}
	}
	if probes != 1 {
		t.Fatalf("probed %d times in one run of the unit, want 1", probes)
	}
	id = "run-2"
	if err := EnsureServiceRunning("mysql"); err != nil {
		t.Fatal(err)
	}
	if probes != 2 {
		t.Errorf("a restarted unit must be probed again, probed %d times", probes)
	}
}

// Without an id to go by, as when podman cannot be asked, every call still probes.
func TestEnsureServiceRunning_probesWithoutAnActivationID(t *testing.T) {
	withServiceHome(t)
	prevLife := podman.UnitLifecycle
	podman.UnitLifecycle = activeLifecycle{}
	t.Cleanup(func() { podman.UnitLifecycle = prevLife })
	probes := 0
	prevWait := waitReadyFn
	waitReadyFn = func(string, time.Duration) error { probes++; return nil }
	t.Cleanup(func() { waitReadyFn = prevWait })
	prevID := activationIDFn
	activationIDFn = func(string) string { return "" }
	t.Cleanup(func() { activationIDFn = prevID })

	for i := 0; i < 2; i++ {
		if err := EnsureServiceRunning("mysql"); err != nil {
			t.Fatal(err)
		}
	}
	if probes != 2 {
		t.Errorf("probed %d times, want every call", probes)
	}
}
