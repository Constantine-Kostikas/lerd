package cli

import (
	"os"
	"strings"
	"testing"
)

// guardArgs is a fixture covering the new buildWorkerGuard signature.
// `true` is used as the podman binary so the orphan-cleanup step always
// succeeds quickly without contacting a real podman machine.
type guardArgs struct {
	pidFile, podmanBin, container, sitePath, workerCmd, runCmd string
}

func defaultGuardArgs(pidFile, runCmd string) guardArgs {
	return guardArgs{
		pidFile:   pidFile,
		podmanBin: "true",
		container: "lerd-php84-fpm",
		sitePath:  "/Users/test/site",
		workerCmd: "php artisan queue:work",
		runCmd:    runCmd,
	}
}

func (a guardArgs) build() string {
	return buildWorkerGuard(a.pidFile, a.podmanBin, a.container, a.sitePath, a.workerCmd, a.runCmd)
}

func TestBuildWorkerGuard_WrapsCommand(t *testing.T) {
	a := defaultGuardArgs("/tmp/lerd-queue-alpha.pid", "podman exec -w /site lerd-php84-fpm php artisan queue:work")
	got := a.build()

	for _, want := range []string{a.pidFile, a.runCmd, "kill -0", "exec ", "pgrep -f", "readlink /proc/$p/cwd", "'php artisan queue:work'", "'/Users/test/site'"} {
		if !strings.Contains(got, want) {
			t.Errorf("guard missing %q:\n%s", want, got)
		}
	}
}

// TestBuildWorkerGuard_WaitsForOrphanExit locks in the fix for a worker that
// holds a listening socket (e.g. Reverb): the orphan cleanup must SIGTERM, then
// WAIT for the process to exit, then SIGKILL stragglers — otherwise the
// replacement binds the port before the old one frees it (EADDRINUSE).
func TestBuildWorkerGuard_WaitsForOrphanExit(t *testing.T) {
	a := defaultGuardArgs("/tmp/lerd-reverb-alpha.pid", "podman exec -w /site lerd-php84-fpm php artisan reverb:start")
	got := a.build()

	for _, want := range []string{"kill -TERM", "while [ -n", "kill -KILL", "sleep 0.1"} {
		if !strings.Contains(got, want) {
			t.Errorf("guard missing orphan-exit wait construct %q:\n%s", want, got)
		}
	}
}

func testPIDString() string {
	return itoa(os.Getpid())
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var buf [20]byte
	i := len(buf)
	neg := n < 0
	if neg {
		n = -n
	}
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}
