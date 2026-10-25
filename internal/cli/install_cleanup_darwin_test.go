//go:build darwin

package cli

import (
	"os"
	"path/filepath"
	"testing"
)

func TestInstallCleanupScriptWritesAnExecutableScript(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	installCleanupScript()

	info, err := os.Stat(filepath.Join(home, ".local", "bin", "lerd-cleanup"))
	if err != nil {
		t.Fatalf("lerd-cleanup was not written: %v", err)
	}
	if info.Mode().Perm()&0o111 == 0 {
		t.Fatalf("lerd-cleanup mode = %v, want it executable", info.Mode().Perm())
	}
}
