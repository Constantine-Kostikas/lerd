package nativephp

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func seedWindowsBuild(t *testing.T, version string, files ...string) {
	t.Helper()
	for _, f := range files {
		p := filepath.Join(BuildDir(version), f)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

func TestWindowsBinariesLiveInTheBuildDir(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	if got, want := BinaryPath("8.4"), filepath.Join(BuildDir("8.4"), "php.exe"); got != want {
		t.Errorf("BinaryPath = %s, want %s", got, want)
	}
	if got, want := FPMBinaryPath("8.4"), filepath.Join(BuildDir("8.4"), "php-cgi.exe"); got != want {
		t.Errorf("FPMBinaryPath = %s, want %s", got, want)
	}
}

func TestWindowsListInstalledNeedsTheCLIAndPHPCGI(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	seedWindowsBuild(t, "8.4", "php.exe", "php-cgi.exe")
	seedWindowsBuild(t, "8.3", "php.exe")
	if got := ListInstalled(); !reflect.DeepEqual(got, []string{"8.4"}) {
		t.Errorf("ListInstalled = %v, want [8.4]", got)
	}
}

func TestWindowsEnsureInstalledAcceptsAnExeWithNoExecuteBit(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	seedWindowsBuild(t, "8.4", "php-cgi.exe")
	if err := EnsureInstalled("8.4", FPMBinaryPath("8.4")); err != nil {
		t.Errorf("EnsureInstalled: %v", err)
	}
}

func TestWindowsModulesAreDLLs(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	mods := ModulesDir("8.4")
	if err := os.MkdirAll(mods, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, n := range []string{"xdebug.dll", "lerd_devtools-8.4.dll"} {
		if err := os.WriteFile(filepath.Join(mods, n), []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if got := XdebugExtensionPath("8.4"); got != filepath.Join(mods, "xdebug.dll") {
		t.Errorf("XdebugExtensionPath = %q", got)
	}
	if got := DevtoolsExtensionPath("8.4"); got != filepath.Join(mods, "lerd_devtools-8.4.dll") {
		t.Errorf("DevtoolsExtensionPath = %q", got)
	}
}

func TestWindowsIniPointsAtTheBuild(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	if dirs := IniScanDirs("8.4"); dirs[0] != BuildConfDir("8.4") {
		t.Errorf("first scan dir = %s, want the build's conf.d", dirs[0])
	}
	want := "extension_dir=" + filepath.Join(BuildDir("8.4"), "ext") + "\n"
	if ini := overrideIni("8.4"); !strings.Contains(ini, want) {
		t.Errorf("override ini lacks %q:\n%s", want, ini)
	}
}
