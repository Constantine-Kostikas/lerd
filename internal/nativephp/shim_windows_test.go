package nativephp

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/geodro/lerd/internal/config"
)

func TestWindowsShimIsACmdRunningTheNativeCLI(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	dir, err := EnsureShim("8.4", BinaryPath("8.4"))
	if err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(filepath.Join(dir, "php.cmd"))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		`set "PHP_INI_SCAN_DIR=` + IniScanDir("8.4") + `"`,
		`"` + BinaryPath("8.4") + `" %*`,
	} {
		if !strings.Contains(string(b), want) {
			t.Errorf("php.cmd lacks %q:\n%s", want, b)
		}
	}
}

func TestWindowsHostCommandResolvesPHPThroughPATHEXT(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("PATHEXT", ".COM;.EXE;.BAT;.CMD")
	if err := os.MkdirAll(config.BinDir(), 0o755); err != nil {
		t.Fatal(err)
	}
	shim := filepath.Join(config.BinDir(), "php.cmd")
	if err := os.WriteFile(shim, []byte("@echo off\r\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	site := t.TempDir()
	c, err := HostCommand(site, []string{"php", "artisan", "migrate"})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.EqualFold(c.Path, shim) {
		t.Errorf("php resolved to %s, want %s", c.Path, shim)
	}
	if c.Dir != site {
		t.Errorf("dir = %s, want %s", c.Dir, site)
	}
}

func TestWindowsLookPathInPrefersTheProjectsVendorBin(t *testing.T) {
	vendor, bin := t.TempDir(), t.TempDir()
	t.Setenv("PATHEXT", ".EXE;.BAT")
	for _, p := range []string{filepath.Join(vendor, "pest.bat"), filepath.Join(bin, "pest.bat")} {
		if err := os.WriteFile(p, nil, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	got, err := lookPathIn("pest", vendor+string(os.PathListSeparator)+bin)
	if err != nil || !strings.EqualFold(got, filepath.Join(vendor, "pest.bat")) {
		t.Errorf("lookPathIn = %q, %v; want the vendor/bin copy", got, err)
	}
	if got, _ := lookPathIn(`C:\tools\php.exe`, bin); got != `C:\tools\php.exe` {
		t.Errorf("an absolute path was looked up instead of kept: %q", got)
	}
	if _, err := lookPathIn("missing", bin); err == nil {
		t.Error("a missing command resolved")
	}
}
