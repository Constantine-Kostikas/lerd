package nativephp

import (
	"strings"
	"testing"

	"github.com/geodro/lerd/internal/config"
)

// Each PHP version gets its own FPM listener, derived from the version so the
// port is stable across restarts without any state to persist.
func TestPortFor(t *testing.T) {
	cases := []struct {
		version string
		want    int
	}{
		{"8.1", 9481},
		{"8.4", 9484},
		{"8.5", 9485},
		{"7.4", 9474},
	}
	for _, c := range cases {
		got, err := PortFor(c.version)
		if err != nil {
			t.Fatalf("PortFor(%q): %v", c.version, err)
		}
		if got != c.want {
			t.Errorf("PortFor(%q) = %d, want %d", c.version, got, c.want)
		}
	}
}

func TestPortForRejectsGarbage(t *testing.T) {
	for _, v := range []string{"", "8", "abc", "8.x", "8.1.2"} {
		if _, err := PortFor(v); err == nil {
			t.Errorf("PortFor(%q) should have failed", v)
		}
	}
}

// Two versions must never collide, or one FPM silently shadows the other.
func TestPortForIsUnique(t *testing.T) {
	seen := map[int]string{}
	for _, v := range []string{"7.4", "8.0", "8.1", "8.2", "8.3", "8.4", "8.5", "8.6"} {
		p, err := PortFor(v)
		if err != nil {
			t.Fatalf("PortFor(%q): %v", v, err)
		}
		if prev, dup := seen[p]; dup {
			t.Fatalf("port %d shared by %s and %s", p, prev, v)
		}
		seen[p] = v
	}
}

func TestFPMConfig(t *testing.T) {
	got, err := FPMConfig("8.4", "/tmp/lerd-native-84.log")
	if err != nil {
		t.Fatalf("FPMConfig: %v", err)
	}
	// nginx dials in from the podman VM, so a loopback-only bind would refuse
	// every request.
	if !strings.Contains(got, "listen = 9484") {
		t.Errorf("want a listener on the version's port, got:\n%s", got)
	}
	if strings.Contains(got, "listen = 127.0.0.1") {
		t.Error("a loopback-only bind is unreachable from the nginx container")
	}
	// launchd supervises the process, so FPM must stay in the foreground.
	if !strings.Contains(got, "daemonize = no") {
		t.Error("FPM must not daemonize under launchd")
	}
	// The site's own env reaches the app through the FPM process environment.
	if !strings.Contains(got, "clear_env = no") {
		t.Error("clear_env must stay off")
	}
	if !strings.Contains(got, "/tmp/lerd-native-84.log") {
		t.Error("error_log should point at the given path")
	}
}

func TestFPMConfigRejectsBadVersion(t *testing.T) {
	if _, err := FPMConfig("nope", "/tmp/x.log"); err == nil {
		t.Error("FPMConfig should reject an unparseable version")
	}
}

// Regenerating must not churn the file, or every call restarts FPM for nothing.
func TestFPMConfigIsDeterministic(t *testing.T) {
	a, _ := FPMConfig("8.4", "/tmp/x.log")
	b, _ := FPMConfig("8.4", "/tmp/x.log")
	if a != b {
		t.Error("FPMConfig must be deterministic")
	}
}

// Switching a site to native with no binary installed must say so, not start a
// listener that nothing answers on.
func TestEnsureInstalledReportsMissingBinary(t *testing.T) {
	err := EnsureInstalled("8.4", "/nonexistent/php-native-8.4")
	if err == nil {
		t.Fatal("expected an error for a missing binary")
	}
	if !strings.Contains(err.Error(), "8.4") {
		t.Errorf("error should name the version, got: %v", err)
	}
}

// Workers run "php artisan ..." through a shell, so a dir holding a php ->
// native-CLI symlink goes on PATH ahead of BinDir, whose php is the container
// shim. Without this a native site's workers would silently run in a container.
func TestShimDirIsPerVersionAndSeparateFromBinDir(t *testing.T) {
	d84 := ShimDir("8.4")
	if d84 == ShimDir("8.5") {
		t.Error("shim dir must be per version")
	}
	if d84 == config.BinDir() {
		t.Error("shim dir must not be BinDir, whose php is the container shim")
	}
}

// The conf.d files the containers mount name container paths (the dump bridge
// is auto-prepended from /usr/local/etc/lerd). PHP fatals on an
// auto_prepend_file it cannot open, so the override re-points the bridge and
// its assets at the host copies rather than turning capture off.
func TestOverrideIniPointsTheBridgeAtHostPaths(t *testing.T) {
	got := overrideIni("8.4")
	if strings.Contains(got, "/usr/local/etc/lerd") {
		t.Errorf("override must not carry container paths:\n%s", got)
	}
	for _, want := range []string{"auto_prepend_file=", "dump-bridge.php", "lerd.assets_dir=", "lerd.dump_host=tcp://127.0.0.1:9913"} {
		if !strings.Contains(got, want) {
			t.Errorf("override missing %q:\n%s", want, got)
		}
	}
	// An empty prepend would disable dump()/dd() entirely.
	if strings.Contains(got, "auto_prepend_file=\n") {
		t.Errorf("the prepend must point somewhere, not be cleared:\n%s", got)
	}
}

// It must be scanned after the copied container inis or it cannot win.
func TestOverrideDirIsScannedLast(t *testing.T) {
	dirs := IniScanDirs("8.4")
	last := dirs[len(dirs)-1]
	if last != OverrideDir("8.4") {
		t.Errorf("last scan dir = %q, want the override dir %q", last, OverrideDir("8.4"))
	}
}

// A dev laptop leaves sites idle for hours, and pm=dynamic keeps start_servers
// workers resident per version the whole time. ondemand lets an untouched pool
// fall to zero children; the master holds the opcache shared memory, so nothing
// is re-warmed when one is forked again.
func TestFPMConfigLetsIdlePoolsFallToZero(t *testing.T) {
	got, err := FPMConfig("8.4", "/tmp/x.log")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, "pm = ondemand") {
		t.Errorf("want an ondemand pool, got:\n%s", got)
	}
	if !strings.Contains(got, "pm.process_idle_timeout") {
		t.Error("ondemand without an idle timeout never reaps a worker")
	}
	// ondemand ignores these, and leaving them in reads as if they applied.
	for _, dead := range []string{"pm.start_servers", "pm.min_spare_servers", "pm.max_spare_servers"} {
		if strings.Contains(got, dead) {
			t.Errorf("%s has no meaning under ondemand:\n%s", dead, got)
		}
	}
	// The ceiling still has to be there, or one site can fork without bound.
	if !strings.Contains(got, "pm.max_children") {
		t.Error("a pool with no ceiling can fork without bound")
	}
}
