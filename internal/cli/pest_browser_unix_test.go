//go:build !windows

package cli

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"os/exec"
	"strings"
	"testing"
	"time"
)

// The plan parser drives the whole install: one line per component carrying the
// primary URL and the mirror Playwright falls back to when the CDN is
// unreachable, with ffmpeg listed once even though Playwright repeats it per
// browser.
func TestPestBrowserPlanAwk_ParsesDryRun(t *testing.T) {
	cmd := exec.Command("awk", pestBrowserPlanAwk)
	cmd.Stdin = strings.NewReader(dryRunPlan)
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("awk: %v", err)
	}
	got := strings.Split(strings.TrimSpace(string(out)), "\n")
	want := []string{
		"/root/.cache/ms-playwright/chromium-1194\thttps://cdn.playwright.dev/builds/chromium/1194/chromium-linux-arm64.zip\thttps://playwright.download.prss.microsoft.com/builds/chromium/1194/chromium-linux-arm64.zip",
		"/root/.cache/ms-playwright/ffmpeg-1011\thttps://cdn.playwright.dev/builds/ffmpeg/1011/ffmpeg-linux-arm64.zip\thttps://playwright.download.prss.microsoft.com/builds/ffmpeg/1011/ffmpeg-linux-arm64.zip",
		"/root/.cache/ms-playwright/chromium_headless_shell-1194\thttps://cdn.playwright.dev/builds/chromium/1194/chromium-headless-shell-linux-arm64.zip\thttps://playwright.download.prss.microsoft.com/builds/chromium/1194/chromium-headless-shell-linux-arm64.zip",
	}
	if len(got) != len(want) {
		t.Fatalf("parsed %d lines, want %d:\n%s", len(got), len(want), out)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("line %d = %q, want %q", i, got[i], want[i])
		}
	}
}

// Playwright only prints fallbacks for components it mirrors, so a block without
// one still has to yield a usable line rather than dropping the component.
func TestPestBrowserPlanAwk_HandlesMissingFallback(t *testing.T) {
	plan := `browser: chromium version 141.0.7390.37
  Install location:    /root/.cache/ms-playwright/chromium-1194
  Download url:        https://cdn.playwright.dev/builds/chromium/1194/chromium-linux-arm64.zip
`
	cmd := exec.Command("awk", pestBrowserPlanAwk)
	cmd.Stdin = strings.NewReader(plan)
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("awk: %v", err)
	}
	want := "/root/.cache/ms-playwright/chromium-1194\thttps://cdn.playwright.dev/builds/chromium/1194/chromium-linux-arm64.zip\t"
	if got := strings.TrimRight(string(out), "\n"); got != want {
		t.Errorf("parsed %q, want %q", got, want)
	}
}

// The link filename is the sha1 of the playwright-core package path and the
// contents are that same path; Playwright reads it back to resolve the pinned
// revisions, so a different shape silently reads as a broken link.
func TestPestBrowserInstall_LinkNamingMatchesPlaywright(t *testing.T) {
	const pkgPath = "/home/dev/site/node_modules/playwright-core"
	sum := sha1.Sum([]byte(pkgPath))
	want := hex.EncodeToString(sum[:])

	cmd := exec.Command("sh", "-c", `printf '%s' "$1" | sha1sum | cut -d" " -f1`, "sh", pkgPath)
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("sha1sum: %v", err)
	}
	if got := strings.TrimSpace(string(out)); got != want {
		t.Errorf("link filename = %q, want sha1 of the package path %q", got, want)
	}
}

// Boot succeeds the moment the ready marker is printed, even though the server
// then keeps running; the watcher must detect it and stop the process.
func TestWatchPlaywrightBoot_DetectsReadyMarker(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "sh", "-c", "echo 'Listening on ws://127.0.0.1:1/'; sleep 30")
	ok, out := watchPlaywrightBoot(ctx, cmd)
	if !ok {
		t.Fatalf("expected boot success, got false with output %q", out)
	}
}

// The whole point of #677: when the server dies before listening, the doctor
// must surface the process's real output, not a bare failure.
func TestWatchPlaywrightBoot_SurfacesOutputOnFailure(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "sh", "-c", "echo 'boom: the real reason' >&2; exit 1")
	ok, out := watchPlaywrightBoot(ctx, cmd)
	if ok {
		t.Fatal("expected boot failure")
	}
	if !strings.Contains(out, "boom: the real reason") {
		t.Errorf("expected the real output surfaced, got %q", out)
	}
}

func TestWatchPlaywrightBoot_TimesOut(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 150*time.Millisecond)
	defer cancel()
	cmd := exec.CommandContext(ctx, "sh", "-c", "sleep 30")
	ok, out := watchPlaywrightBoot(ctx, cmd)
	if ok {
		t.Fatal("expected boot failure on timeout")
	}
	if !strings.Contains(out, "timed out") {
		t.Errorf("expected a timeout message, got %q", out)
	}
}
