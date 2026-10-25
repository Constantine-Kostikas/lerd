package cli

import (
	"io"
	"testing"

	"github.com/geodro/lerd/internal/config"
)

// The dashboard's Install goes through InstallPHPVersion. Under the native
// runtime it has to fetch the published build and bring its pool up, the way
// `lerd use` plus `lerd start` would, rather than build an image nothing serves.
func TestInstallPHPVersionFetchesTheNativeBuild(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	cfg, err := config.LoadGlobal()
	if err != nil {
		t.Fatal(err)
	}
	cfg.PHP.Runtime = config.PHPRuntimeNative
	if err := config.SaveGlobal(cfg); err != nil {
		t.Fatal(err)
	}

	var fetched, inied, started string
	origInstall, origStart, origIni := nativeInstallFn, nativeStartFn, ensureXdebugIni
	nativeInstallFn = func(version string, _ io.Writer) error { fetched = version; return nil }
	ensureXdebugIni = func(version string) error { inied = version; return nil }
	nativeStartFn = func(version string) error { started = version; return nil }
	defer func() { nativeInstallFn, nativeStartFn, ensureXdebugIni = origInstall, origStart, origIni }()

	if err := InstallPHPVersion("8.4", io.Discard); err != nil {
		t.Fatal(err)
	}
	if fetched != "8.4" || inied != "8.4" || started != "8.4" {
		t.Errorf("fetched %q, xdebug ini %q, started %q, want all 8.4", fetched, inied, started)
	}
}
