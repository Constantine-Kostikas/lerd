package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/geodro/lerd/internal/config"
)

// A shim whose binary is where it says it is must be left exactly as written:
// the heal is a repair, not a chance to repoint a working install at whatever
// binary happens to be running.
func TestHealShimBinaryPathsLeavesWorkingShimsAlone(t *testing.T) {
	binDir, current := shimHealFixture(t)
	other := filepath.Join(filepath.Dir(current), "lerd-other")
	if err := os.WriteFile(other, []byte("#!/bin/sh\n"), 0755); err != nil {
		t.Fatal(err)
	}
	body := "#!/bin/sh\nexec " + other + " php \"$@\"\n"
	writeShim(t, binDir, "php", body)

	healShimBinaryPaths(current, binaryGone)

	if got := readShim(t, binDir, "php"); got != body {
		t.Errorf("php shim was rewritten:\n%s", got)
	}
}

// Nothing is rewritten when the replacement is not on disk either, so a probe
// that cannot resolve the binary never makes matters worse.
func TestHealShimBinaryPathsSkipsWhenTheBinaryIsMissing(t *testing.T) {
	binDir, current := shimHealFixture(t)
	body := "#!/bin/sh\nexec /gone/lerd php \"$@\"\n"
	writeShim(t, binDir, "php", body)

	healShimBinaryPaths(filepath.Join(filepath.Dir(current), "absent"), binaryGone)

	if got := readShim(t, binDir, "php"); got != body {
		t.Errorf("php shim was rewritten:\n%s", got)
	}
}

// A repair rewrites files the user did not ask lerd to touch, so the start has
// to name what it moved.
func TestRepairSummaryNamesWhatMoved(t *testing.T) {
	summary := repairSummary([]string{"lerd-ui"}, []string{"php", "composer"})

	for _, want := range []string{"lerd-ui", "php", "composer", config.LerdBinary()} {
		if !strings.Contains(summary, want) {
			t.Errorf("summary %q does not mention %q", summary, want)
		}
	}
}

// A lerd that resolves to a directory is not something to heal towards, so the
// walk stops rather than writing nonsense into every shim.
func TestHealShimBinaryPathsIgnoresADirectory(t *testing.T) {
	binDir, current := shimHealFixture(t)
	body := "#!/bin/sh\nexec /gone/lerd php \"$@\"\n"
	writeShim(t, binDir, "php", body)

	if healed := healShimBinaryPaths(filepath.Dir(current), binaryGone); len(healed) != 0 {
		t.Errorf("healShimBinaryPaths() = %v; want nothing repaired", healed)
	}
	if got := readShim(t, binDir, "php"); got != body {
		t.Errorf("php shim was rewritten:\n%s", got)
	}
}

// shimHealFixture points lerd's data dir at a temp dir and returns the shim dir
// alongside an installed lerd binary to heal towards.
func shimHealFixture(t *testing.T) (binDir, lerdBin string) {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("XDG_DATA_HOME", dir)
	binDir = config.BinDir()
	if err := os.MkdirAll(binDir, 0755); err != nil {
		t.Fatal(err)
	}
	lerdBin = filepath.Join(dir, "opt", "lerd", "bin", "lerd")
	if err := os.MkdirAll(filepath.Dir(lerdBin), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(lerdBin, []byte("#!/bin/sh\n"), 0755); err != nil {
		t.Fatal(err)
	}
	return binDir, lerdBin
}

func writeShim(t *testing.T, binDir, name, body string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(binDir, name), []byte(body), 0755); err != nil {
		t.Fatal(err)
	}
}

func readShim(t *testing.T, binDir, name string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(binDir, name))
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}
