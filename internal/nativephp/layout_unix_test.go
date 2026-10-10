//go:build !windows

package nativephp

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/geodro/lerd/internal/config"
)

func TestBinaryPathIsVersioned(t *testing.T) {
	a := BinaryPath("8.4")
	b := BinaryPath("8.5")
	if a == b {
		t.Fatal("each PHP version needs its own binary path")
	}
	if !strings.HasSuffix(a, "php-native-8.4") {
		t.Errorf("BinaryPath(8.4) = %q, want a php-native-8.4 suffix", a)
	}
	// Must never collide with the container shim named "php" in the same dir.
	if filepathBase(a) == "php" {
		t.Error("native binary must not be named php; that is the container shim")
	}
}

// FPM serves requests, the CLI runs artisan and composer. They are separate
// binaries and must not share a path.
func TestFPMBinaryPathDiffersFromCLI(t *testing.T) {
	if BinaryPath("8.4") == FPMBinaryPath("8.4") {
		t.Fatal("cli and fpm binaries must not share a path")
	}
	if !strings.Contains(FPMBinaryPath("8.4"), "fpm") {
		t.Errorf("FPMBinaryPath should be identifiable, got %q", FPMBinaryPath("8.4"))
	}
}

func TestEnsureShimCreatesAndRepoints(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_DATA_HOME", tmp)
	first := filepath.Join(tmp, "php-native-8.4")
	if err := os.WriteFile(first, []byte("#!/bin/sh\n"), 0755); err != nil {
		t.Fatal(err)
	}
	dir, err := EnsureShim("8.4", first)
	if err != nil {
		t.Fatalf("EnsureShim: %v", err)
	}
	got, err := os.Readlink(filepath.Join(dir, "php"))
	if err != nil || got != first {
		t.Fatalf("shim link = %q (%v), want %q", got, err, first)
	}

	// A reinstall points the same link somewhere new rather than failing.
	second := filepath.Join(tmp, "php-native-8.4.new")
	if err := os.WriteFile(second, []byte("#!/bin/sh\n"), 0755); err != nil {
		t.Fatal(err)
	}
	if _, err := EnsureShim("8.4", second); err != nil {
		t.Fatalf("EnsureShim repoint: %v", err)
	}
	if got, _ := os.Readlink(filepath.Join(dir, "php")); got != second {
		t.Errorf("shim link = %q, want %q after repoint", got, second)
	}
}

// The settings page lists what is installed. Under the native runtime that is
// the binaries on disk, not the container images, or it offers versions that
// cannot serve anything.
func TestListInstalledFindsNativeBinaries(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_DATA_HOME", tmp)
	binDir := config.BinDir()
	if err := os.MkdirAll(binDir, 0755); err != nil {
		t.Fatal(err)
	}
	// Both SAPIs are needed for a version to be usable.
	for _, name := range []string{"php-native-8.3", "php-native-fpm-8.3", "php-native-8.4", "php-native-fpm-8.4"} {
		if err := os.WriteFile(filepath.Join(binDir, name), []byte("#!/bin/sh\n"), 0755); err != nil {
			t.Fatal(err)
		}
	}
	// A CLI with no FPM cannot serve, so it must not be listed.
	if err := os.WriteFile(filepath.Join(binDir, "php-native-8.5"), []byte("#!/bin/sh\n"), 0755); err != nil {
		t.Fatal(err)
	}

	got := ListInstalled()
	want := []string{"8.3", "8.4"}
	if len(got) != len(want) {
		t.Fatalf("ListInstalled = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("ListInstalled = %v, want %v", got, want)
		}
	}
}

// Query capture is an engine-level extension. When the build ships one next to
// the binary the runtime has to load it, and when it does not the ini must stay
// silent rather than naming a file PHP would warn about on every request.
func TestOverrideLoadsDevtoolsOnlyWhenPresent(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_DATA_HOME", tmp)

	if got := overrideIni("8.4"); strings.Contains(got, "lerd_devtools") {
		t.Errorf("no extension present, so nothing should be loaded:\n%s", got)
	}

	so := filepath.Join(ModulesDir("8.4"), "lerd_devtools-8.4.so")
	if err := os.MkdirAll(filepath.Dir(so), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(so, []byte("x"), 0644); err != nil {
		t.Fatal(err)
	}
	got := overrideIni("8.4")
	// lerd_devtools registers a zend_module_entry, so it loads with extension=.
	// zend_extension= is for the other kind and PHP refuses the module outright.
	if !strings.Contains(got, "extension="+so) {
		t.Errorf("a present extension must be loaded:\n%s", got)
	}
	if strings.Contains(got, "zend_extension="+so) {
		t.Errorf("lerd_devtools is a module, not a zend extension:\n%s", got)
	}
}

func TestDevtoolsExtensionPathFindsTheVersionsModule(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	dir := ModulesDir("8.4")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	so := filepath.Join(dir, "lerd_devtools-8.4.so")
	if err := os.WriteFile(so, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := DevtoolsExtensionPath("8.4"); got != so {
		t.Errorf("DevtoolsExtensionPath = %q, want %q", got, so)
	}
	// A version with no build of the collector must report nothing rather than
	// a path, or PHP warns on every request for a file that is not there.
	if got := DevtoolsExtensionPath("8.5"); got != "" {
		t.Errorf("DevtoolsExtensionPath(8.5) = %q, want empty", got)
	}
}

// A native FPM must load the same conf.d files the containers bind-mount, in the
// same order, or php:ini, dumps, the debug window and xdebug all stop working
// the moment a site switches runtime.
func TestIniScanDirsOrdering(t *testing.T) {
	dirs := IniScanDirs("8.4")
	if len(dirs) < 2 {
		t.Fatalf("expected several scan dirs, got %v", dirs)
	}
	wantOrder := []string{"mail", "shared", "devtools", "dumps", "8.4"}
	var idx []int
	for _, want := range wantOrder {
		found := -1
		for i, d := range dirs {
			if filepathBase(d) == want {
				found = i
				break
			}
		}
		if found < 0 {
			t.Fatalf("scan dirs %v missing %q", dirs, want)
		}
		idx = append(idx, found)
	}
	for i := 1; i < len(idx); i++ {
		if idx[i] <= idx[i-1] {
			t.Errorf("scan dir %q must come after %q; got %v", wantOrder[i], wantOrder[i-1], dirs)
		}
	}
}

func filepathBase(p string) string {
	for i := len(p) - 1; i >= 0; i-- {
		if p[i] == '/' {
			return p[i+1:]
		}
	}
	return p
}
