//go:build !windows

package cli

import (
	"os"
	"path/filepath"
	"testing"
)

// The ostree images keep home at /var/home behind a /home symlink, and they do
// not agree on which spelling lands in passwd: Silverblue's is /var/home/you,
// Bazzite's is /home/you. The quadlet mounts %h, so a driver under home has to
// be stored the way %h spells it whichever way the symlink runs, or the registry
// names a path that does not exist inside the container.
func TestOdbcDriverPathNamesHomeTheWayTheQuadletMountsIt(t *testing.T) {
	root := t.TempDir()
	realHome := filepath.Join(root, "var", "home", "me")
	drv := filepath.Join(realHome, "drv")
	if err := os.MkdirAll(drv, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(drv, "libodbcHDB.so"), []byte("stub"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(root, "var", "home"), filepath.Join(root, "home")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	linkHome := filepath.Join(root, "home", "me")

	for _, tc := range []struct {
		name string
		home string // what passwd says, i.e. what %h expands to
		give string // what the user types
	}{
		{"canonical home, driver given through the link", realHome, filepath.Join(linkHome, "drv", "libodbcHDB.so")},
		{"linked home, driver given canonically", linkHome, filepath.Join(realHome, "drv", "libodbcHDB.so")},
		{"linked home, driver given through the link", linkHome, filepath.Join(linkHome, "drv", "libodbcHDB.so")},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("HOME", tc.home)
			got, err := odbcDriverPath(tc.give)
			if err != nil {
				t.Fatalf("odbcDriverPath: %v", err)
			}
			want := filepath.Join(tc.home, "drv", "libodbcHDB.so")
			if got != want {
				t.Errorf("odbcDriverPath = %q, want %q so it matches the %%h mount", got, want)
			}
		})
	}
}
