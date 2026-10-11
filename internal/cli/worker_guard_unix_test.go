//go:build !windows

package cli

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestBuildWorkerGuard_ExitsZeroWhenPIDAlive(t *testing.T) {
	tmp := t.TempDir()
	pidFile := filepath.Join(tmp, "worker.pid")
	if err := os.WriteFile(pidFile, []byte(testPIDString()), 0644); err != nil {
		t.Fatal(err)
	}

	a := defaultGuardArgs(pidFile, "false")
	cmd := exec.Command("sh", "-c", a.build())
	if err := cmd.Run(); err != nil {
		t.Fatalf("guard should exit 0 when pid is alive, got: %v", err)
	}
}

func TestBuildWorkerGuard_ProceedsWhenPIDStale(t *testing.T) {
	tmp := t.TempDir()
	pidFile := filepath.Join(tmp, "worker.pid")
	if err := os.WriteFile(pidFile, []byte("2147483646\n"), 0644); err != nil {
		t.Fatal(err)
	}

	marker := filepath.Join(tmp, "ran")
	a := defaultGuardArgs(pidFile, "touch "+marker)
	cmd := exec.Command("sh", "-c", a.build())
	if err := cmd.Run(); err != nil {
		t.Fatalf("guard with stale pid should run wrapped command: %v", err)
	}
	if _, err := os.Stat(marker); err != nil {
		t.Errorf("marker not created; wrapped command didn't run")
	}
}

func TestBuildWorkerGuard_ProceedsWhenPIDFileMissing(t *testing.T) {
	tmp := t.TempDir()
	pidFile := filepath.Join(tmp, "worker.pid")
	marker := filepath.Join(tmp, "ran")
	a := defaultGuardArgs(pidFile, "touch "+marker)
	cmd := exec.Command("sh", "-c", a.build())
	if err := cmd.Run(); err != nil {
		t.Fatalf("guard without pid file should run wrapped command: %v", err)
	}
	if _, err := os.Stat(marker); err != nil {
		t.Errorf("marker not created")
	}
}

func TestBuildWorkerGuard_WritesPIDFileBeforeExec(t *testing.T) {
	tmp := t.TempDir()
	pidFile := filepath.Join(tmp, "worker.pid")
	a := defaultGuardArgs(pidFile, "test -s "+pidFile)
	cmd := exec.Command("sh", "-c", a.build())
	if err := cmd.Run(); err != nil {
		t.Fatalf("pid file should be written before wrapped command runs: %v", err)
	}
}

// TestBuildWorkerGuard_RunsOrphanCleanupBeforeExec verifies the orphan
// cleanup step (step 2) runs even when the pid file is missing — this is
// the suspend/wake codepath. We swap `podman` for a stub that records
// each argument so the test asserts the inner sh -c script contains
// both the worker command and the site path (the cwd-scoping that
// keeps cross-site sibling workers from being killed).
func TestBuildWorkerGuard_RunsOrphanCleanupBeforeExec(t *testing.T) {
	tmp := t.TempDir()
	pidFile := filepath.Join(tmp, "worker.pid")
	pkillMarker := filepath.Join(tmp, "pkill-ran")

	stub := filepath.Join(tmp, "podman-stub")
	stubScript := "#!/bin/sh\nfor a in \"$@\"; do printf '<%s>' \"$a\" >> " + pkillMarker + "; done\nexit 0\n"
	if err := os.WriteFile(stub, []byte(stubScript), 0755); err != nil {
		t.Fatal(err)
	}

	a := guardArgs{
		pidFile:   pidFile,
		podmanBin: stub,
		container: "lerd-php84-fpm",
		sitePath:  "/Users/u/parkapp",
		workerCmd: "php artisan queue:work --queue=default",
		runCmd:    "true",
	}
	cmd := exec.Command("sh", "-c", a.build())
	if err := cmd.Run(); err != nil {
		t.Fatalf("guard run failed: %v", err)
	}

	got, err := os.ReadFile(pkillMarker)
	if err != nil {
		t.Fatalf("orphan cleanup did not invoke podman: %v", err)
	}
	// The first three args are the literal `exec`, container name, and
	// `sh -c`. The fourth is the inner script (one shell arg) — must
	// contain both the worker command and the site path so the cwd
	// scoping is in effect.
	gotStr := string(got)
	for _, want := range []string{
		"<exec>",
		"<lerd-php84-fpm>",
		"<sh>",
		"<-c>",
		"php artisan queue:work --queue=default",
		"/Users/u/parkapp",
		"readlink /proc/$p/cwd",
	} {
		if !strings.Contains(gotStr, want) {
			t.Errorf("orphan cleanup missing %q in args:\n%s", want, gotStr)
		}
	}
}

// TestBuildWorkerGuard_SkipsOrphanCleanupWhenOuterAlive ensures the live
// host-side process short-circuit happens before any podman call —
// otherwise we'd wake the podman machine for every launchd respawn.
func TestBuildWorkerGuard_SkipsOrphanCleanupWhenOuterAlive(t *testing.T) {
	tmp := t.TempDir()
	pidFile := filepath.Join(tmp, "worker.pid")
	pkillMarker := filepath.Join(tmp, "pkill-ran")

	if err := os.WriteFile(pidFile, []byte(testPIDString()), 0644); err != nil {
		t.Fatal(err)
	}

	stub := filepath.Join(tmp, "podman-stub")
	if err := os.WriteFile(stub, []byte("#!/bin/sh\ntouch "+pkillMarker+"\n"), 0755); err != nil {
		t.Fatal(err)
	}

	a := guardArgs{
		pidFile:   pidFile,
		podmanBin: stub,
		container: "lerd-php84-fpm",
		sitePath:  "/Users/u/parkapp",
		workerCmd: "php artisan queue:work",
		runCmd:    "false",
	}
	if err := exec.Command("sh", "-c", a.build()).Run(); err != nil {
		t.Fatalf("guard should exit 0: %v", err)
	}
	if _, err := os.Stat(pkillMarker); err == nil {
		t.Error("podman was invoked even though outer process is alive")
	}
}

// TestBuildWorkerReapCommand_TargetsContainerAndCwd verifies the stop-time reap
// (persisted as a worker's .reap sidecar) invokes `podman exec <container> sh -c`
// with the same cwd-scoped kill the launch guard uses, so a stop terminates the
// in-container worker (and its file-watcher child) instead of orphaning it.
func TestBuildWorkerReapCommand_TargetsContainerAndCwd(t *testing.T) {
	tmp := t.TempDir()
	marker := filepath.Join(tmp, "podman-ran")
	stub := filepath.Join(tmp, "podman-stub")
	stubScript := "#!/bin/sh\nfor a in \"$@\"; do printf '<%s>' \"$a\" >> " + marker + "; done\nexit 0\n"
	if err := os.WriteFile(stub, []byte(stubScript), 0755); err != nil {
		t.Fatal(err)
	}

	reap := buildWorkerReapCommand(stub, "lerd-php85-fpm", "/Users/u/lferp", "php artisan horizon:listen")
	if err := exec.Command("sh", "-c", reap).Run(); err != nil {
		t.Fatalf("reap run failed: %v", err)
	}

	got, err := os.ReadFile(marker)
	if err != nil {
		t.Fatalf("reap did not invoke podman: %v", err)
	}
	gotStr := string(got)
	for _, want := range []string{
		"<exec>",
		"<lerd-php85-fpm>",
		"<sh>",
		"<-c>",
		"php artisan horizon:listen",
		"/Users/u/lferp",
		"readlink /proc/$p/cwd",
	} {
		if !strings.Contains(gotStr, want) {
			t.Errorf("reap command missing %q in podman args:\n%s", want, gotStr)
		}
	}
}
