package cli

import (
	"path/filepath"
	"strings"
	"testing"
)

// cpx is only ever the globally required binary (#1543), so the resolver has to
// look where `lerd composer global require` actually puts it and say so plainly
// when it is not there, rather than failing inside the container with "could
// not open input file".
func TestCpxBinPath_followsComposerHome(t *testing.T) {
	home := t.TempDir()
	t.Setenv("COMPOSER_HOME", home)

	want := filepath.Join(home, "vendor", "bin", "cpx")
	if got := cpxBinPath(); got != want {
		t.Errorf("cpxBinPath() = %q, want %q", got, want)
	}
}

func TestCpxBinPath_honoursXDG(t *testing.T) {
	home := t.TempDir()
	t.Setenv("COMPOSER_HOME", "")
	setTestHome(t, home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, "xdg"))

	want := filepath.Join(home, "xdg", "composer", "vendor", "bin", "cpx")
	if got := cpxBinPath(); got != want {
		t.Errorf("cpxBinPath() = %q, want %q", got, want)
	}
}

func TestRunCpx_missingBinaryNamesTheFix(t *testing.T) {
	t.Setenv("COMPOSER_HOME", t.TempDir())

	err := runCpx([]string{"--version"})
	if err == nil {
		t.Fatal("expected an error when cpx is not installed")
	}
	if !strings.Contains(err.Error(), "composer global require cpx/cpx") {
		t.Errorf("error should name the command that installs it, got: %v", err)
	}
}
