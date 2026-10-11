//go:build !windows

package cli

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/geodro/lerd/internal/tools"
)

// The mise lerd installs is stamped, which is what makes it lerd's to bring
// back to the pin later, unlike one the user installed in the same place.
func TestInstallMise_stampsItAsLerds(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("LERD_TOOLS_URL", "http://127.0.0.1:1/tools.yaml")

	if err := installMise(miseRelease(t, "2026.9.12"), home, io.Discard); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(home, ".local", "bin", "mise")); err != nil {
		t.Fatalf("mise not extracted: %v", err)
	}
	if v := tools.InstalledVersion("mise"); strings.TrimPrefix(v, "v") != "2026.9.12" {
		t.Fatalf("InstalledVersion(mise) = %q, want 2026.9.12 from lerd's own mise", v)
	}
}

// tools:update reaches mise through the same per-tool entry the dashboard
// uses, and puts the pinned build back over a self-updated one.
func TestUpdateTool_bringsMiseBackToThePin(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("LERD_TOOLS_URL", "http://127.0.0.1:1/tools.yaml")
	if err := installMise(miseRelease(t, "2026.10.2"), home, io.Discard); err != nil {
		t.Fatal(err)
	}

	if err := updateTool(miseRelease(t, "2026.9.12"), "mise"); err != nil {
		t.Fatalf("updateTool(mise) = %v", err)
	}
	out, err := os.ReadFile(filepath.Join(home, ".local", "bin", "mise"))
	if err != nil || !bytes.Contains(out, []byte("2026.9.12")) {
		t.Fatalf("mise on disk after the update:\n%s", out)
	}
	if v := tools.InstalledVersion("mise"); strings.TrimPrefix(v, "v") != "2026.9.12" {
		t.Fatalf("InstalledVersion(mise) = %q, want 2026.9.12", v)
	}
}
