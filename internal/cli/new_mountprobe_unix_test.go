//go:build !windows

package cli

import (
	"os"
	"path/filepath"
	"testing"
)

// The Podman Machine VM always shares the host home, so a project under it
// needs no round trip either.
func TestContainerSeesHostDirSkipsHome(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	probed := false
	dir := filepath.Join(home, "Sites")
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	if !containerSeesHostDirOn("darwin", dir, func(string) bool { probed = true; return false }) {
		t.Error("a path under the host home must be reported as visible")
	}
	if probed {
		t.Error("a path under the host home must not be probed")
	}
}
