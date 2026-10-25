package ui

import (
	"testing"
	"time"
)

// A finished build must re-probe the patch and drop the status snapshot
// before the client is told it is done, or the card keeps the old number until
// the snapshot's TTL expires.
func TestRefreshAfterPHPBuild(t *testing.T) {
	stubNativeRuntime(t, false)
	origPoll, origPatch := pollContainersFn, refreshPHPPatchFn
	t.Cleanup(func() {
		pollContainersFn, refreshPHPPatchFn = origPoll, origPatch
		snapshots.status.mu.Lock()
		snapshots.status.data, snapshots.status.at = nil, time.Time{}
		snapshots.status.mu.Unlock()
	})

	polled := false
	probed := ""
	pollContainersFn = func() { polled = true }
	refreshPHPPatchFn = func(version string) { probed = version }

	snapshots.status.mu.Lock()
	snapshots.status.data, snapshots.status.at = []byte(`{"php":[]}`), time.Now()
	snapshots.status.mu.Unlock()

	refreshAfterPHPBuild("8.5")

	if !polled {
		t.Error("container cache was not polled")
	}
	if probed != "8.5" {
		t.Errorf("probed patch for %q, want 8.5", probed)
	}
	snapshots.status.mu.Lock()
	at := snapshots.status.at
	snapshots.status.mu.Unlock()
	if !at.IsZero() {
		t.Error("status snapshot still fresh, the card would report the old image")
	}
}
