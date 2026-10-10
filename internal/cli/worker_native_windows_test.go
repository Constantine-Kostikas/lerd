//go:build windows

package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/geodro/lerd/internal/config"
)

// A native site has no FPM container to exec into, so its workers run on
// Windows through the host path, where php is lerd's php.cmd.
func TestWindowsNativeSiteWorkerRunsOnTheHost(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	prev := nativeSite
	nativeSite = func(string) bool { return true }
	t.Cleanup(func() { nativeSite = prev })

	site := `C:\Sites\shop`
	if _, err := writeWorkerUnitFile("lerd-queue-shop", "Queue Worker", "shop", site, "8.4", "php artisan queue:work", "always", "", "lerd-php84-fpm", "", false); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(filepath.Join(config.DataDir(), "units", "lerd-queue-shop.json"))
	if err != nil {
		t.Fatal(err)
	}
	def := string(b)
	for _, want := range []string{`"worker-exec"`, `"--shell"`, `"C:\\Sites\\shop"`, `"php artisan queue:work"`} {
		if !strings.Contains(def, want) {
			t.Errorf("unit lacks %s:\n%s", want, def)
		}
	}
	if strings.Contains(def, "podman") {
		t.Errorf("a native site's worker still execs into a container:\n%s", def)
	}
}
