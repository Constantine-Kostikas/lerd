package podman

import (
	"testing"
)

func withRunningPs(t *testing.T, out string) *int {
	t.Helper()
	calls := 0
	prev := runningPsFn
	runningPsFn = func() (string, error) { calls++; return out, nil }
	t.Cleanup(func() { runningPsFn = prev; dropRunningMemo() })
	dropRunningMemo()
	return &calls
}

// A php run checks every service its site uses, and each podman inspect is a
// VM round trip on macOS, so the checks share one podman ps.
func TestRunningChecksShareOnePs(t *testing.T) {
	calls := withRunningPs(t, "lerd-mysql\t1792438625-3696\nlerd-redis\t1792438626-3744\n")

	if ok, _ := ContainerRunning("lerd-mysql"); !ok {
		t.Error("lerd-mysql is in the ps output, want running")
	}
	if !ContainerRunningQuiet("lerd-redis") {
		t.Error("lerd-redis is in the ps output, want running")
	}
	if got := ContainerRunID("lerd-redis"); got != "1792438626-3744" {
		t.Errorf("ContainerRunID = %q, want the ps start time and pid", got)
	}
	if *calls != 1 {
		t.Errorf("podman ps ran %d times, want 1", *calls)
	}
}

// A unit lerd just started or stopped must not be answered from the ps taken
// before it.
func TestRunningMemoDroppedOnUnitChange(t *testing.T) {
	calls := withRunningPs(t, "lerd-mysql\t1-2\n")

	ContainerRunning("lerd-mysql")
	notifyUnitChange("lerd-mysql")
	ContainerRunning("lerd-mysql")
	if *calls != 2 {
		t.Errorf("podman ps ran %d times, want a fresh one after the unit changed", *calls)
	}
}

// A container missing from the ps falls through to its own inspect, so a
// stopped one is never reported running.
func TestRunningMemoFallsBackForAbsentContainer(t *testing.T) {
	withRunningPs(t, "lerd-mysql\t1-2\n")
	prev := execCommand
	execCommand = fakeExec("false", "", 0)
	t.Cleanup(func() { execCommand = prev })

	if ok, _ := ContainerRunning("lerd-postgres"); ok {
		t.Error("lerd-postgres is not in the ps and inspect says stopped, want not running")
	}
	if got := ContainerRunID("lerd-postgres"); got != "" {
		t.Errorf("ContainerRunID = %q for a stopped container, want empty", got)
	}
}
