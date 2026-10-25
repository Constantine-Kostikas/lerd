package cli

import (
	"errors"
	"strings"
	"testing"
)

// withWSLSystemdSeams fakes the distro and records each step enableWSLSystemd takes.
func withWSLSystemdSeams(t *testing.T, isWSL, booted bool, setupErr, shutdownErr error) *[]string {
	t.Helper()
	origWSL, origBooted, origEnable, origShutdown, origInstalled := installIsWSL, installSystemdBooted, enableSystemd, shutdownWSL, systemdInstalled
	t.Cleanup(func() {
		installIsWSL, installSystemdBooted, enableSystemd, shutdownWSL, systemdInstalled = origWSL, origBooted, origEnable, origShutdown, origInstalled
	})
	systemdInstalled = func() bool { return true }
	var steps []string
	installIsWSL = func() bool { return isWSL }
	installSystemdBooted = func() bool { return booted }
	enableSystemd = func() error { steps = append(steps, "setup"); return setupErr }
	shutdownWSL = func() error { steps = append(steps, "shutdown"); return shutdownErr }
	return &steps
}

func TestEnableWSLSystemd_leavesARunningSystemdAlone(t *testing.T) {
	for _, isWSL := range []bool{true, false} {
		steps := withWSLSystemdSeams(t, isWSL, isWSL, nil, nil)
		if err := enableWSLSystemd(false); err != nil || len(*steps) > 0 {
			t.Errorf("wsl=%v: err = %v, steps = %v", isWSL, err, *steps)
		}
	}
}

func TestEnableWSLSystemd_setsUpThenRestartsWSL(t *testing.T) {
	steps := withWSLSystemdSeams(t, true, false, nil, nil)
	err := enableWSLSystemd(false)
	if err == nil || !strings.Contains(err.Error(), "lerd install") {
		t.Errorf("the run should end saying to install again, got %v", err)
	}
	if strings.Join(*steps, ",") != "setup,shutdown" {
		t.Errorf("steps = %v, want setup then shutdown", *steps)
	}
}

func TestEnableWSLSystemd_failedSetupKeepsWSLRunning(t *testing.T) {
	steps := withWSLSystemdSeams(t, true, false, errors.New("no sudo"), nil)
	err := enableWSLSystemd(false)
	if err == nil {
		t.Error("a failed wsl.conf write should fail the install")
	}
	if strings.Join(*steps, ",") != "setup" {
		t.Errorf("steps = %v, want no shutdown", *steps)
	}
}

func TestEnableWSLSystemd_unattendedOnlyExplains(t *testing.T) {
	steps := withWSLSystemdSeams(t, true, false, nil, nil)
	if err := enableWSLSystemd(true); err == nil || len(*steps) > 0 {
		t.Errorf("err = %v, steps = %v, want the manual steps and nothing run", err, *steps)
	}
}

func TestEnableWSLSystemd_noInteropAsksForTheRestart(t *testing.T) {
	steps := withWSLSystemdSeams(t, true, false, nil, errors.New("exec format error"))
	err := enableWSLSystemd(false)
	if err == nil || !strings.Contains(err.Error(), "wsl --shutdown") {
		t.Errorf("err = %v, want it to ask for wsl --shutdown", err)
	}
	if strings.Join(*steps, ",") != "setup,shutdown" {
		t.Errorf("steps = %v", *steps)
	}
}

func TestEnableWSLSystemd_missingSystemdDoesNotRestart(t *testing.T) {
	steps := withWSLSystemdSeams(t, true, false, nil, nil)
	systemdInstalled = func() bool { return false }
	err := enableWSLSystemd(false)
	if err == nil || !strings.Contains(err.Error(), "install the systemd package") {
		t.Errorf("err = %v, want it to ask for systemd", err)
	}
	if len(*steps) > 0 {
		t.Errorf("steps = %v, want nothing run", *steps)
	}
}
