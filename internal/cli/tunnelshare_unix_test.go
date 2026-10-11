//go:build !windows

package cli

import (
	"os"
	"strings"
	"syscall"
	"testing"
	"time"
)

func TestShareTools_detectsInstalledBinaries(t *testing.T) {
	t.Setenv("PATH", fakeBin(t, "ngrok", "ssh"))
	installed := map[string]bool{}
	for _, tool := range ShareTools().Tools {
		installed[tool.Name] = tool.Installed
	}
	want := map[string]bool{"ngrok": true, "cloudflare": false, "expose": false, "serveo": true, "localhost-run": true, "pinggy": true}
	for name, w := range want {
		if installed[name] != w {
			t.Errorf("tool %s installed = %v, want %v", name, installed[name], w)
		}
	}
}

func TestAutoShareToolName(t *testing.T) {
	cases := []struct {
		bins        []string
		defaultTool string
		want        string
	}{
		{[]string{"ngrok", "cloudflared"}, "", "ngrok"},
		{[]string{"cloudflared", "expose"}, "", "cloudflare"},
		{[]string{"expose", "ssh"}, "", "expose"},
		{[]string{"ssh"}, "", "localhost-run"},
		{[]string{"ngrok", "cloudflared"}, "cloudflare", "cloudflare"},
		{[]string{"ngrok"}, "cloudflare", "ngrok"}, // default not installed, fall back to auto
		{nil, "", ""},
	}
	for _, c := range cases {
		t.Setenv("PATH", fakeBin(t, c.bins...))
		if got := autoShareToolName(c.defaultTool, ""); got != c.want {
			t.Errorf("autoShareToolName(%q) with %v = %q, want %q", c.defaultTool, c.bins, got, c.want)
		}
	}
}

func TestResolveTunnelTool_explicitNameIgnoresDefault(t *testing.T) {
	t.Setenv("PATH", fakeBin(t, "ngrok", "cloudflared"))
	tool, err := resolveTunnelTool("ngrok", "cloudflare", "", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tool.mode != shareModeNgrok {
		t.Errorf("mode = %v, want shareModeNgrok", tool.mode)
	}
}

func TestStartTunnelProcess_returnsURLAndTracksTunnel(t *testing.T) {
	tunnelStateHome(t)
	site := "tunnel-test-a"
	cmd := shellTunnel(`echo "Forwarding HTTP traffic from https://abc123.serveo.net"; sleep 30`)
	url, err := startTunnelProcess(site, "serveo", cmd, nil, 5*time.Second, "", "")
	t.Cleanup(func() { _ = TunnelStop(site, "") })
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if url != "https://abc123.serveo.net" {
		t.Errorf("url = %q, want the serveo URL", url)
	}
	info, ok := TunnelStatus(site, "")
	if !ok || info.URL != url || info.Tool != "serveo" {
		t.Errorf("TunnelStatus = %+v, %v; want running serveo tunnel", info, ok)
	}
	if err := TunnelStop(site, ""); err != nil {
		t.Fatalf("stop: %v", err)
	}
	waitTunnelGone(t, site)
}

func TestStartTunnelProcess_readsStderrToo(t *testing.T) {
	tunnelStateHome(t)
	site := "tunnel-test-stderr"
	cmd := shellTunnel(`echo "INF |  https://plum-lakes-agree.trycloudflare.com  |" >&2; sleep 30`)
	url, err := startTunnelProcess(site, "cloudflare", cmd, nil, 5*time.Second, "", "")
	t.Cleanup(func() { _ = TunnelStop(site, "") })
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if url != "https://plum-lakes-agree.trycloudflare.com" {
		t.Errorf("url = %q, want the trycloudflare URL", url)
	}
	_ = TunnelStop(site, "")
	waitTunnelGone(t, site)
}

func TestStartTunnelProcess_exitBeforeURLReportsOutput(t *testing.T) {
	tunnelStateHome(t)
	site := "tunnel-test-fail"
	cmd := shellTunnel(`echo "ERR authentication failed: install your authtoken" >&2; exit 1`)
	_, err := startTunnelProcess(site, "ngrok", cmd, nil, 5*time.Second, "", "")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "authentication failed") {
		t.Errorf("error %q does not carry the tool output", err)
	}
	waitTunnelGone(t, site)
}

func TestStartTunnelProcess_timesOut(t *testing.T) {
	tunnelStateHome(t)
	site := "tunnel-test-slow"
	cmd := shellTunnel(`sleep 30`)
	_, err := startTunnelProcess(site, "serveo", cmd, nil, 300*time.Millisecond, "", "")
	if err == nil || !strings.Contains(err.Error(), "timed out") {
		t.Fatalf("expected timeout error, got %v", err)
	}
	waitTunnelGone(t, site)
}

func TestStartTunnelProcess_replacesExistingTunnel(t *testing.T) {
	tunnelStateHome(t)
	site := "tunnel-test-replace"
	t.Cleanup(func() { _ = TunnelStop(site, "") })
	first := shellTunnel(`echo "Forwarding HTTP traffic from https://first.serveo.net"; sleep 30`)
	if _, err := startTunnelProcess(site, "serveo", first, nil, 5*time.Second, "", ""); err != nil {
		t.Fatalf("first start: %v", err)
	}
	second := shellTunnel(`echo "Forwarding HTTP traffic from https://second.serveo.net"; sleep 30`)
	url, err := startTunnelProcess(site, "serveo", second, nil, 5*time.Second, "", "")
	if err != nil {
		t.Fatalf("second start: %v", err)
	}
	if url != "https://second.serveo.net" {
		t.Errorf("url = %q, want the second tunnel's URL", url)
	}
	info, ok := TunnelStatus(site, "")
	if !ok || info.URL != "https://second.serveo.net" {
		t.Errorf("TunnelStatus = %+v, %v; want the second tunnel", info, ok)
	}
}

func TestStartTunnelProcess_stopProxyRunsWhenProcessDies(t *testing.T) {
	tunnelStateHome(t)
	site := "tunnel-test-proxy"
	stopped := make(chan struct{})
	cmd := shellTunnel(`echo "Forwarding HTTP traffic from https://p.serveo.net"; sleep 30`)
	if _, err := startTunnelProcess(site, "serveo", cmd, func() { close(stopped) }, 5*time.Second, "", ""); err != nil {
		t.Fatalf("start: %v", err)
	}
	if err := TunnelStop(site, ""); err != nil {
		t.Fatalf("stop: %v", err)
	}
	select {
	case <-stopped:
	case <-time.After(5 * time.Second):
		t.Fatal("stopProxy was not called after the tunnel stopped")
	}
}

func TestStopAllTunnels(t *testing.T) {
	tunnelStateHome(t)
	sites := []string{"tunnel-test-all-1", "tunnel-test-all-2"}
	for _, s := range sites {
		cmd := shellTunnel(`echo "Forwarding HTTP traffic from https://` + s + `.serveo.net"; sleep 30`)
		if _, err := startTunnelProcess(s, "serveo", cmd, nil, 5*time.Second, "", ""); err != nil {
			t.Fatalf("start %s: %v", s, err)
		}
	}
	StopAllTunnels()
	for _, s := range sites {
		waitTunnelGone(t, s)
	}
}

// A worktree fronts its own domain, so its tunnel is a separate process that
// neither replaces the parent's nor is stopped along with it.
func TestStartTunnelProcess_worktreeTunnelIsIndependentOfTheSite(t *testing.T) {
	tunnelStateHome(t)
	site := "tunnel-test-wt"
	siteKey := tunnelKey(site, "")
	wtKey := tunnelKey(site, "feature")
	t.Cleanup(func() { _ = TunnelStop(site, ""); _ = TunnelStop(site, "feature") })

	parent := shellTunnel(`echo "Forwarding HTTP traffic from https://parent.serveo.net"; sleep 30`)
	if _, err := startTunnelProcess(siteKey, "serveo", parent, nil, 5*time.Second, "", ""); err != nil {
		t.Fatalf("parent start: %v", err)
	}
	wt := shellTunnel(`echo "Forwarding HTTP traffic from https://branch.serveo.net"; sleep 30`)
	if _, err := startTunnelProcess(wtKey, "serveo", wt, nil, 5*time.Second, "", ""); err != nil {
		t.Fatalf("worktree start: %v", err)
	}

	pInfo, ok := TunnelStatus(site, "")
	if !ok || pInfo.URL != "https://parent.serveo.net" {
		t.Errorf("site tunnel = %+v, %v; want the parent URL still running", pInfo, ok)
	}
	wInfo, ok := TunnelStatus(site, "feature")
	if !ok || wInfo.URL != "https://branch.serveo.net" {
		t.Errorf("worktree tunnel = %+v, %v; want the branch URL", wInfo, ok)
	}

	if err := TunnelStop(site, "feature"); err != nil {
		t.Fatalf("stop worktree: %v", err)
	}
	waitTunnelGone(t, wtKey)
	if _, ok := TunnelStatus(site, ""); !ok {
		t.Error("stopping the worktree tunnel also stopped the site's")
	}
}

func TestStartTunnelProcess_namedTunnelReportsItsOwnHostname(t *testing.T) {
	tunnelStateHome(t)
	site := "tunnel-test-named"
	known := "https://acme.example.com"
	cmd := shellTunnel(`echo "INF Registered tunnel connection connIndex=0"; sleep 30`)
	url, err := startTunnelProcess(site, "cloudflare", cmd, nil, 5*time.Second, known, "")
	t.Cleanup(func() { _ = TunnelStop(site, "") })
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if url != known {
		t.Errorf("url = %q, want %q", url, known)
	}
	if err := TunnelStop(site, ""); err != nil {
		t.Fatalf("stop: %v", err)
	}
	waitTunnelGone(t, site)
}

// The daemon's own tunnel is the live one; a leftover CLI entry for the same
// site must not shadow it.
func TestTunnelStatus_prefersTheDaemonsOwnTunnel(t *testing.T) {
	tunnelStateHome(t)
	site := "tunnel-test-both"
	if err := writeTunnelState(tunnelKey(site, ""), tunnelState{
		Site: site, Tool: "ngrok", URL: "https://stale.ngrok.io", PID: os.Getpid(),
	}); err != nil {
		t.Fatalf("writing: %v", err)
	}
	cmd := shellTunnel(`echo "Forwarding HTTP traffic from https://live.serveo.net"; sleep 30`)
	if _, err := startTunnelProcess(site, "serveo", cmd, nil, 5*time.Second, "", ""); err != nil {
		t.Fatalf("starting: %v", err)
	}
	t.Cleanup(func() { _ = TunnelStop(site, "") })

	info, ok := TunnelStatus(site, "")
	if !ok || info.URL != "https://live.serveo.net" || info.External {
		t.Errorf("TunnelStatus = %+v, %v; want the daemon's own tunnel", info, ok)
	}
}

// A stop acts on the tunnel the dashboard is showing. With both kinds up that
// is the daemon's own, and the CLI share behind it takes over the display.
func TestTunnelStop_leavesTheCLIShareWhenWeOwnTheLiveTunnel(t *testing.T) {
	tunnelStateHome(t)
	site := "tunnel-test-stop-both"
	sleeper := startSleeper(t)
	if err := writeTunnelState(tunnelKey(site, ""), tunnelState{
		Site: site, Tool: "ngrok", URL: "https://cli.ngrok.io", PID: sleeper.Process.Pid,
	}); err != nil {
		t.Fatalf("writing: %v", err)
	}
	cmd := shellTunnel(`echo "Forwarding HTTP traffic from https://live.serveo.net"; sleep 30`)
	if _, err := startTunnelProcess(site, "serveo", cmd, nil, 5*time.Second, "", ""); err != nil {
		t.Fatalf("starting: %v", err)
	}
	t.Cleanup(func() { _ = TunnelStop(site, "") })

	if err := TunnelStop(site, ""); err != nil {
		t.Fatalf("stop: %v", err)
	}
	info, ok := TunnelStatus(site, "")
	if !ok || info.URL != "https://cli.ngrok.io" || !info.External {
		t.Errorf("after the stop TunnelStatus = %+v, %v; want the CLI share still up", info, ok)
	}
	if err := sleeper.Process.Signal(syscall.Signal(0)); err != nil {
		t.Error("the CLI share was signalled even though we owned the live tunnel")
	}
}

func TestTunnelStop_signalsACLIShareWhenItIsTheOnlyOne(t *testing.T) {
	tunnelStateHome(t)
	site := "tunnel-test-stop-cli"
	sleeper := startSleeper(t)
	if err := writeTunnelState(tunnelKey(site, ""), tunnelState{
		Site: site, Tool: "ngrok", URL: "https://cli.ngrok.io", PID: sleeper.Process.Pid,
	}); err != nil {
		t.Fatalf("writing: %v", err)
	}

	if err := TunnelStop(site, ""); err != nil {
		t.Fatalf("stop: %v", err)
	}
	done := make(chan error, 1)
	go func() { done <- sleeper.Wait() }()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("the CLI share was never signalled")
	}
}
