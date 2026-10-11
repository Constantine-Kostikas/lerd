//go:build windows

package cli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/geodro/lerd/internal/nativephp"
	"github.com/geodro/lerd/internal/tools"
)

// stageWindowsBuild fakes what the Windows release tarball unpacks to.
func stageWindowsBuild(t *testing.T, version, marker string) string {
	t.Helper()
	stage := t.TempDir()
	files := map[string]string{
		"php-native-" + version + "/php.exe":          marker,
		"php-native-" + version + "/php-cgi.exe":      marker,
		"php-native-" + version + "/ext/php_intl.dll": marker,
		"modules/xdebug.dll":                          marker,
		"modules/lerd_devtools-" + version + ".dll":   marker,
		"conf.d/10-lerd-extensions.ini":               "extension=intl\n",
	}
	for name, body := range files {
		p := filepath.Join(stage, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return stage
}

func TestUnpackWindowsBuildPlacesTheBuildDir(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	if err := unpackNativePHP(stageWindowsBuild(t, "8.4", "a"), "8.4", "8.4.26"); err != nil {
		t.Fatal(err)
	}
	for _, p := range []string{
		nativephp.BinaryPath("8.4"),
		nativephp.FPMBinaryPath("8.4"),
		filepath.Join(nativephp.BuildDir("8.4"), "ext", "php_intl.dll"),
		filepath.Join(nativephp.BuildConfDir("8.4"), "10-lerd-extensions.ini"),
	} {
		if _, err := os.Stat(p); err != nil {
			t.Errorf("missing %s: %v", p, err)
		}
	}
	if nativephp.DevtoolsExtensionPath("8.4") == "" {
		t.Error("the query-capture collector was not installed")
	}
	if got := nativephp.ListInstalled(); len(got) != 1 || got[0] != "8.4" {
		t.Errorf("ListInstalled = %v, want [8.4]", got)
	}
	if got := tools.InstalledVersion(nativeTool("8.4")); got != "8.4.26" {
		t.Errorf("stamp = %q, want 8.4.26", got)
	}
}

func TestUnpackWindowsBuildReplacesAnEarlierPatch(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	if err := unpackNativePHP(stageWindowsBuild(t, "8.4", "old"), "8.4", "8.4.25"); err != nil {
		t.Fatal(err)
	}
	if err := unpackNativePHP(stageWindowsBuild(t, "8.4", "new"), "8.4", "8.4.26"); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(nativephp.FPMBinaryPath("8.4"))
	if err != nil || string(b) != "new" {
		t.Errorf("php-cgi.exe = %q, %v; want the new patch", b, err)
	}
	if _, err := os.Stat(nativephp.BuildDir("8.4") + ".old"); !os.IsNotExist(err) {
		t.Errorf("the previous build was left behind: %v", err)
	}
	if got := tools.InstalledVersion(nativeTool("8.4")); got != "8.4.26" {
		t.Errorf("stamp = %q, want 8.4.26", got)
	}
}

func TestUnpackWindowsBuildRefusesAnArchiveWithoutPHPCGI(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	stage := stageWindowsBuild(t, "8.4", "a")
	os.Remove(filepath.Join(stage, "php-native-8.4", "php-cgi.exe"))
	if err := unpackNativePHP(stage, "8.4", "8.4.26"); err == nil {
		t.Error("an archive with no php-cgi.exe installed")
	}
}
