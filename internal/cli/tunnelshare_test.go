package cli

import (
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"
)

// ── parseTunnelURL ────────────────────────────────────────────────────────────

func TestParseTunnelURL(t *testing.T) {
	cases := []struct {
		tool string
		line string
		want string
	}{
		{"ngrok", `{"addr":"http://localhost:80","lvl":"info","msg":"started tunnel","obj":"tunnels","url":"https://84c1-1-2-3-4.ngrok-free.app"}`, "https://84c1-1-2-3-4.ngrok-free.app"},
		{"ngrok", `{"lvl":"info","msg":"client session established","obj":"csess"}`, ""},
		{"cloudflare", `2026-07-25T08:00:00Z INF |  https://plum-lakes-agree.trycloudflare.com                        |`, "https://plum-lakes-agree.trycloudflare.com"},
		{"cloudflare", `2026-07-25T08:00:00Z INF Requesting new quick Tunnel on trycloudflare.com...`, ""},
		{"expose", `Expose-URL:         https://mysite.sharedwithexpose.com`, "https://mysite.sharedwithexpose.com"},
		{"expose", `Public HTTPS:       https://mysite.sharedwithexpose.com`, "https://mysite.sharedwithexpose.com"},
		{"expose", `Visit https://expose.dev/docs to get started`, ""},
		{"serveo", `Forwarding HTTP traffic from https://f3a1c2.serveo.net`, "https://f3a1c2.serveo.net"},
		{"serveo", `Press g to start a GUI session and ctrl-c to quit.`, ""},
		{"localhost-run", `a1b2c3d4.lhr.life tunneled with tls termination, https://a1b2c3d4.lhr.life`, "https://a1b2c3d4.lhr.life"},
		{"localhost-run", `** visit https://localhost.run/docs/ for documentation **`, ""},
		{"pinggy", `https://dztqm-84-232-173-137.free.pinggy.net`, "https://dztqm-84-232-173-137.free.pinggy.net"},
		{"pinggy", `https://vurbr-84-232-173-137.run.pinggy-free.link`, "https://vurbr-84-232-173-137.run.pinggy-free.link"},
		{"pinggy", `https://myapp.a.pinggy.link`, "https://myapp.a.pinggy.link"},
		{"pinggy", `http://dztqm-84-232-173-137.free.pinggy.net`, ""},
		{"pinggy", `Your tunnel will expire in 60 minutes. Upgrade to Pinggy Pro to get unrestricted tunnels. https://dashboard.pinggy.io`, ""},
	}
	for _, c := range cases {
		got, ok := parseTunnelURL(c.tool, c.line)
		if c.want == "" {
			if ok {
				t.Errorf("parseTunnelURL(%s, %q) matched %q, want no match", c.tool, c.line, got)
			}
			continue
		}
		if !ok || got != c.want {
			t.Errorf("parseTunnelURL(%s, %q) = %q, %v; want %q", c.tool, c.line, got, ok, c.want)
		}
	}
}

// ── tool status / auto pick ───────────────────────────────────────────────────

func TestShareTools_listsEveryToolWithLabel(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	tools := ShareTools().Tools
	if len(tools) != len(shareTools) {
		t.Fatalf("got %d tools, want %d", len(tools), len(shareTools))
	}
	for _, tool := range tools {
		if tool.Label == "" || tool.Binary == "" {
			t.Errorf("tool %s missing label or binary: %+v", tool.Name, tool)
		}
	}
}

func TestShareToolCanonicalName(t *testing.T) {
	cases := []struct {
		tool *shareTool
		want string
	}{
		{&shareTool{mode: shareModeNgrok}, "ngrok"},
		{&shareTool{mode: shareModeCloudflare}, "cloudflare"},
		{&shareTool{mode: shareModeExpose}, "expose"},
		{&shareTool{mode: shareModeSSH, ssh: sshProviders["serveo"]}, "serveo"},
		{&shareTool{mode: shareModeSSH, ssh: sshProviders["localhost-run"]}, "localhost-run"},
		{&shareTool{mode: shareModeSSH, ssh: sshProviders["pinggy"]}, "pinggy"},
	}
	for _, c := range cases {
		if got := shareToolCanonicalName(c.tool); got != c.want {
			t.Errorf("shareToolCanonicalName(%+v) = %q, want %q", c.tool, got, c.want)
		}
	}
}

func TestResolveTunnelTool_rejectsUnknownName(t *testing.T) {
	_, err := resolveTunnelTool("wireguard", "", "", "")
	if err == nil || !strings.Contains(err.Error(), "unknown tunnel tool") {
		t.Fatalf("expected unknown-tool error, got %v", err)
	}
}

// ── tunnel process lifecycle ──────────────────────────────────────────────────

func shellTunnel(script string) *exec.Cmd {
	return exec.Command("/bin/sh", "-c", script)
}

func waitTunnelGone(t *testing.T, key string) {
	t.Helper()
	for range 100 {
		if _, ok := tunnelStatusByKey(key); !ok {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("tunnel for %s still reported after waiting", key)
}

func TestTunnelStop_noopWhenNotRunning(t *testing.T) {
	tunnelStateHome(t)
	if err := TunnelStop("tunnel-test-none", ""); err != nil {
		t.Fatalf("expected no-op, got %v", err)
	}
}

// ── worktree tunnels ──────────────────────────────────────────────────────────

func TestTunnelKey_separatesWorktreeFromSite(t *testing.T) {
	if got := tunnelKey("rapids", ""); got != "rapids" {
		t.Errorf("tunnelKey with no branch = %q, want the bare site name", got)
	}
	if tunnelKey("rapids", "feature") == tunnelKey("rapids", "") {
		t.Error("a worktree tunnel must not share the site's key")
	}
	if tunnelKey("rapids", "feature") == tunnelKey("rapids", "other") {
		t.Error("two branches of one site must not share a key")
	}
}

// ── named Cloudflare tunnels ─────────────────────────────────────────────────

func TestTunnelURLFromLine_parsesTheOutputWhenNothingIsKnown(t *testing.T) {
	got, ok := tunnelURLFromLine("serveo", "", "Forwarding from https://abc.serveo.net")
	if !ok || got != "https://abc.serveo.net" {
		t.Errorf("got (%q, %v), want the serveo URL", got, ok)
	}
}

// A named tunnel prints no URL at all, so the scan waits for the line that says
// it is carrying traffic and reports the hostname it already knew.
func TestTunnelURLFromLine_knownURLWaitsForTheConnection(t *testing.T) {
	known := "https://acme.example.com"
	if _, ok := tunnelURLFromLine("cloudflare", known, "INF Starting tunnel tunnelID=abc"); ok {
		t.Error("the hostname was reported before the tunnel connected")
	}
	got, ok := tunnelURLFromLine("cloudflare", known, "INF Registered tunnel connection connIndex=0 protocol=quic")
	if !ok || got != known {
		t.Errorf("got (%q, %v), want %q once the connection registered", got, ok, known)
	}
}

func TestTunnelURLFromLine_knownURLAcceptsTheOlderWording(t *testing.T) {
	known := "https://acme.example.com"
	line := "INF Connection 0e8b1c4a-1111-2222-3333-444455556666 registered"
	if got, ok := tunnelURLFromLine("cloudflare", known, line); !ok || got != known {
		t.Errorf("got (%q, %v), want %q", got, ok, known)
	}
}

// A share started in a terminal is invisible to the daemon's own registry, so
// the status has to fall through to what the CLI recorded on disk.
func TestTunnelStatus_reportsAShareStartedFromTheCLI(t *testing.T) {
	tunnelStateHome(t)
	st := tunnelState{Site: "acme", Tool: "ngrok", URL: "https://abc.ngrok.io", PID: os.Getpid()}
	if err := writeTunnelState(tunnelKey("acme", ""), st); err != nil {
		t.Fatalf("writing: %v", err)
	}

	info, ok := TunnelStatus("acme", "")
	if !ok || info.URL != st.URL || !info.External {
		t.Errorf("TunnelStatus = %+v, %v; want the CLI tunnel marked external", info, ok)
	}
}

// startSleeper runs a real process to stand in for a "lerd share" in a
// terminal, so a stop has something it can actually signal.
func startSleeper(t *testing.T) *exec.Cmd {
	t.Helper()
	c := exec.Command("/bin/sh", "-c", "sleep 30")
	if err := c.Start(); err != nil {
		t.Fatalf("starting the stand-in share: %v", err)
	}
	t.Cleanup(func() {
		_ = c.Process.Kill()
		_ = c.Wait()
	})
	return c
}
