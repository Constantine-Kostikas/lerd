package cli

import (
	"bytes"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/geodro/lerd/internal/config"
	"github.com/geodro/lerd/internal/tools"
)

// The runtime a re-routed command needs is installed by a command the
// declaration names, because two packages that both escape the container do not
// install the same way: nativephp/mobile installs through native:install-mobile
// precisely because the desktop package already owns native:install (#1651).
func TestHostCommandMissingMsg_namesTheDeclaredInstallCommand(t *testing.T) {
	hc := config.HostCommand{
		Args:           "artisan native:*",
		Binary:         "vendor/nativephp/php-bin/bin/host/php",
		InstallCommand: "native:install-mobile",
	}

	msg := hostCommandMissingMsg("artisan", hc)
	if !strings.Contains(msg, "vendor/nativephp/php-bin/bin/host/php") {
		t.Errorf("message = %q, want it to name the missing binary", msg)
	}
	if !strings.Contains(msg, "lerd run native:install-mobile") {
		t.Errorf("message = %q, want it to name the declared install command", msg)
	}
}

// A declaration that names no install command is not guessed at, since guessing
// is how a mobile project was told to run the desktop installer.
func TestHostCommandMissingMsg_genericWhenNoneDeclared(t *testing.T) {
	hc := config.HostCommand{Args: "artisan native:*", Binary: "vendor/bin/runtime"}

	msg := hostCommandMissingMsg("artisan", hc)
	if !strings.Contains(msg, "vendor/bin/runtime") {
		t.Errorf("message = %q, want it to name the missing binary", msg)
	}
	if strings.Contains(msg, "native:install") {
		t.Errorf("message = %q, must not name a command no declaration carries", msg)
	}
	if strings.Contains(msg, "lerd run ") {
		t.Errorf("message = %q, must not point at a command it cannot name", msg)
	}
}

func TestMissingPHPExtensions_nothingMissingWhenTheBinaryHasThemAll(t *testing.T) {
	bin := stubPHPModules(t, "[PHP Modules]\nPCNTL\nposix\n")

	if got := missingPHPExtensions(bin, []string{"posix", "pcntl"}); len(got) != 0 {
		t.Errorf("missing = %v, want none (the module list is not case sensitive)", got)
	}
}

// A probe that cannot run says nothing about the binary, and rerouting a command
// on a failed question would move work into a container the declaration never
// asked for. The command runs where it was declared to run and fails there.
func TestMissingPHPExtensions_silentWhenTheProbeFails(t *testing.T) {
	if got := missingPHPExtensions(filepath.Join(t.TempDir(), "absent"), []string{"posix"}); got != nil {
		t.Errorf("missing = %v, want nil when the binary cannot be asked", got)
	}
}

func TestMissingPHPExtensions_noneDeclared(t *testing.T) {
	if got := missingPHPExtensions("/nonexistent/php", nil); got != nil {
		t.Errorf("missing = %v, want nil without a declaration to check", got)
	}
}

// stubPHPModules writes an executable that answers `php -m` with the given list.
func stubPHPModules(t *testing.T, modules string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "php")
	script := "#!/bin/sh\ncat <<'MODS'\n" + modules + "MODS\n"
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatalf("write stub: %v", err)
	}
	return path
}

// The fallback PHP must never be installed as "php": that name in lerd's bin
// directory is the shim into the container, and a static binary written over it
// would take every `php` on the machine with it.
func TestHostPHPPath_doesNotShadowTheContainerShim(t *testing.T) {
	path := hostPHPPath("php-host-8.3")
	if base := filepath.Base(path); base == "php" {
		t.Errorf("host php installs as %q, which is lerd's php shim", base)
	}
	if filepath.Dir(path) != config.BinDir() {
		t.Errorf("host php at %q, want it in lerd's bin directory", path)
	}
}

// A host command runs the project's own code, and its composer.lock was
// resolved against the project's PHP version, so the build that replaces the
// incomplete one has to be that version rather than the newest lerd pins.
func TestHostPHPTool_picksTheProjectsVersionThenTheNearest(t *testing.T) {
	m := &tools.Manifest{Tools: map[string]tools.Tool{
		"php-host-8.3": {Version: "8.3.32"},
		"php-host-8.4": {Version: "8.4.23"},
		"php-host-8.5": {Version: "8.5.8"},
		"mkcert":       {Version: "v1.4.4"},
	}}

	cases := map[string]string{
		"8.4": "php-host-8.4",
		"8.5": "php-host-8.5",
		"8.1": "php-host-8.3", // nothing pinned that low, nearest wins
		"9.0": "php-host-8.5",
	}
	for version, want := range cases {
		if got, ok := hostPHPTool(m, version); !ok || got != want {
			t.Errorf("hostPHPTool(%s) = %q, want %q", version, got, want)
		}
	}
}

func TestHostPHPTool_nothingPinned(t *testing.T) {
	if _, ok := hostPHPTool(&tools.Manifest{Tools: map[string]tools.Tool{"mkcert": {}}}, "8.4"); ok {
		t.Error("want no tool when the manifest pins no PHP")
	}
}

// The missing extensions are read inside a sentence, so they are joined like
// one rather than printed as a bare comma list.
func TestJoinAnd(t *testing.T) {
	cases := map[string]string{
		"":            "",
		"posix":       "posix",
		"posix|pcntl": "posix and pcntl",
		"a|b|c":       "a, b and c",
	}
	for input, want := range cases {
		var items []string
		if input != "" {
			items = strings.Split(input, "|")
		}
		if got := joinAnd(items); got != want {
			t.Errorf("joinAnd(%v) = %q, want %q", items, got, want)
		}
	}
}

// A pin that moved is not worth failing the command over: the build installed
// last time carries the extensions the command needs just as much, so a fetch
// that fails falls back to it and says which version is actually running.
func TestHostPHPFallback_usesTheCopyOnDiskAndSaysSo(t *testing.T) {
	dest := filepath.Join(t.TempDir(), "php-host-8.4")
	if err := os.WriteFile(dest, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer

	got, err := hostPHPFallback(dest, "8.4.20", &out, errors.New("dial tcp: timeout"))

	if err != nil || got != dest {
		t.Fatalf("fallback = (%q, %v), want the installed binary", got, err)
	}
	if !strings.Contains(out.String(), "8.4.20") || !strings.Contains(out.String(), "timeout") {
		t.Errorf("warning = %q, want the version it fell back to and why", out.String())
	}
}

// With nothing on disk there is nothing to fall back to, and the command cannot
// run at all, so the failure names what could not be fetched rather than
// surfacing three layers away as a missing extension.
func TestHostPHPFallback_failsWhenNothingIsInstalled(t *testing.T) {
	_, err := hostPHPFallback(filepath.Join(t.TempDir(), "absent"), "", io.Discard, errors.New("HTTP 503"))

	if err == nil || !strings.Contains(err.Error(), "503") {
		t.Errorf("err = %v, want it to carry the download failure", err)
	}
}

// Nothing lerd fetches starts before the user has seen how big it is, the same
// promise the image pull disclosure makes. A manifest that pins no size for a
// platform still announces the download rather than saying nothing.
func TestHostPHPDownloadLabel_disclosesTheSize(t *testing.T) {
	with := hostPHPDownloadLabel("8.5.8", 12931817)
	if !strings.Contains(with, "8.5.8") || !strings.Contains(with, "MiB") {
		t.Errorf("label = %q, want the version and a human size", with)
	}
	without := hostPHPDownloadLabel("8.5.8", 0)
	if !strings.Contains(without, "8.5.8") || strings.Contains(without, "(") {
		t.Errorf("label = %q, want no size when the manifest pins none", without)
	}
}
