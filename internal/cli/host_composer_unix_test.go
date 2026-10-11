//go:build !windows

package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/geodro/lerd/internal/feedback"
)

func TestHostComposerOnPathFindsTheOneBehindTheShim(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_DATA_HOME", tmp)
	binDir := filepath.Join(tmp, "lerd", "bin")
	writeExecutable(t, filepath.Join(binDir, "composer"))
	hostDir := filepath.Join(tmp, "usr", "bin")
	host := filepath.Join(hostDir, "composer")
	writeExecutable(t, host)
	t.Setenv("PATH", strings.Join([]string{binDir, hostDir}, string(os.PathListSeparator)))

	if got := hostComposerOnPath(); got != host {
		t.Errorf("got %q, want %q", got, host)
	}
}

func TestNoteShadowedComposerNamesTheHostCopy(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_DATA_HOME", tmp)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(tmp, "config"))
	hostDir := filepath.Join(tmp, "usr", "bin")
	host := filepath.Join(hostDir, "composer")
	writeExecutable(t, host)
	t.Setenv("PATH", hostDir)

	var buf bytes.Buffer
	restore := feedback.SetTestWriter(&buf)
	defer restore()

	noteShadowedComposer()

	if !strings.Contains(buf.String(), host) {
		t.Errorf("install should name the composer it fronts, got %q", buf.String())
	}
}
