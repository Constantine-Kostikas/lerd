package cli

import (
	"slices"
	"testing"

	"github.com/geodro/lerd/internal/config"
)

func TestPHPRuntimeFromArgs(t *testing.T) {
	if _, show, err := phpRuntimeFromArgs(nil); err != nil || !show {
		t.Errorf("no argument should mean show, got show=%v err=%v", show, err)
	}
	for _, mode := range []string{config.PHPRuntimeNative, config.PHPRuntimeContainer} {
		got, show, err := phpRuntimeFromArgs([]string{mode})
		if err != nil || show || got != mode {
			t.Errorf("%q: got (%q,%v,%v)", mode, got, show, err)
		}
	}
	if _, _, err := phpRuntimeFromArgs([]string{"nonsense"}); err == nil {
		t.Error("an unknown mode should be rejected, not silently normalised")
	}
}

// Going native leaves nothing for the shared FPM containers to serve, and on a
// 4GB VM that is memory worth reclaiming. Only the versions lerd actually runs
// a container for are touched.
func TestFPMUnitsToStop(t *testing.T) {
	got := fpmUnitsFor([]string{"8.4", "8.5"})
	want := map[string]bool{"lerd-php84-fpm": true, "lerd-php85-fpm": true}
	if len(got) != len(want) {
		t.Fatalf("got %v, want two units", got)
	}
	for _, u := range got {
		if !want[u] {
			t.Errorf("unexpected unit %q", u)
		}
	}
	if len(fpmUnitsFor(nil)) != 0 {
		t.Error("no versions should mean no units")
	}
}

// A fresh install has no FPM images, so narrowing the versions a switch to
// native fetches by what the container runtime installed fetched nothing and
// left every site on a version with no binary behind it.
func TestNativeVersionsWantedIgnoresContainerImages(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	reg := &config.SiteRegistry{Sites: []config.Site{
		{Name: "app", Path: t.TempDir(), PHPVersion: "8.4"},
		{Name: "old", Path: t.TempDir(), PHPVersion: "8.3", Paused: true},
	}}
	if err := config.SaveSites(reg); err != nil {
		t.Fatal(err)
	}
	cfg, err := config.LoadGlobal()
	if err != nil {
		t.Fatal(err)
	}
	cfg.PHP.DefaultVersion = "8.5"
	if err := config.SaveGlobal(cfg); err != nil {
		t.Fatal(err)
	}
	if got := nativeVersionsWanted(); !slices.Equal(got, []string{"8.4", "8.5"}) {
		t.Errorf("wanted = %v, want [8.4 8.5]", got)
	}
}

// A version a switch to native is about to fetch has no xdebug ini of its own
// yet, so the refresh has to cover it too or xdebug never loads there.
func TestXdebugRefreshCoversNativeVersionsWanted(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	reg := &config.SiteRegistry{Sites: []config.Site{{Name: "app", Path: t.TempDir(), PHPVersion: "8.4"}}}
	if err := config.SaveSites(reg); err != nil {
		t.Fatal(err)
	}
	if got := xdebugRefreshVersions(config.PHPRuntimeNative, []string{"8.3"}); !slices.Contains(got, "8.3") || !slices.Contains(got, "8.4") {
		t.Errorf("native refresh = %v, want 8.3 and 8.4", got)
	}
	if got := xdebugRefreshVersions(config.PHPRuntimeContainer, []string{"8.3"}); !slices.Equal(got, []string{"8.3"}) {
		t.Errorf("container refresh = %v, want [8.3]", got)
	}
}
