//go:build !windows

package cli

import (
	"os"
	"path/filepath"

	"strings"
	"testing"
)

func TestPickShareTool_explicitNgrok_present(t *testing.T) {
	t.Setenv("PATH", fakeBin(t, "ngrok"))
	tool, err := pickShareTool(true, false, false, false, false, false, "", "", "", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tool.mode != shareModeNgrok {
		t.Errorf("mode = %v, want shareModeNgrok", tool.mode)
	}
}

func TestPickShareTool_explicitCloudflare_present(t *testing.T) {
	t.Setenv("PATH", fakeBin(t, "cloudflared"))
	tool, err := pickShareTool(false, true, false, false, false, false, "", "", "", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tool.mode != shareModeCloudflare {
		t.Errorf("mode = %v, want shareModeCloudflare", tool.mode)
	}
}

func TestPickShareTool_explicitExpose_present(t *testing.T) {
	t.Setenv("PATH", fakeBin(t, "expose"))
	tool, err := pickShareTool(false, false, true, false, false, false, "", "", "", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tool.mode != shareModeExpose {
		t.Errorf("mode = %v, want shareModeExpose", tool.mode)
	}
}

func TestPickShareTool_autoDetect_ngrokFirst(t *testing.T) {
	t.Setenv("PATH", fakeBin(t, "ngrok", "cloudflared", "expose", "ssh"))
	tool, err := pickShareTool(false, false, false, false, false, false, "", "", "", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tool.mode != shareModeNgrok {
		t.Errorf("mode = %v, want shareModeNgrok (ngrok takes priority)", tool.mode)
	}
}

func TestPickShareTool_autoDetect_cloudflareBeforeExpose(t *testing.T) {
	t.Setenv("PATH", fakeBin(t, "cloudflared", "expose", "ssh"))
	tool, err := pickShareTool(false, false, false, false, false, false, "", "", "", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tool.mode != shareModeCloudflare {
		t.Errorf("mode = %v, want shareModeCloudflare", tool.mode)
	}
}

func TestPickShareTool_autoDetect_expose(t *testing.T) {
	t.Setenv("PATH", fakeBin(t, "expose", "ssh"))
	tool, err := pickShareTool(false, false, false, false, false, false, "", "", "", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tool.mode != shareModeExpose {
		t.Errorf("mode = %v, want shareModeExpose", tool.mode)
	}
}

func TestPickShareTool_autoDetect_sshFallback(t *testing.T) {
	t.Setenv("PATH", fakeBin(t, "ssh"))
	tool, err := pickShareTool(false, false, false, false, false, false, "", "", "", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tool.mode != shareModeSSH {
		t.Errorf("mode = %v, want shareModeSSH", tool.mode)
	}
	if tool.ssh.host != "localhost.run" {
		t.Errorf("ssh.host = %q, want localhost.run", tool.ssh.host)
	}
}

func TestPickShareTool_domain_withExplicitCloudflare(t *testing.T) {
	t.Setenv("PATH", fakeBin(t, "ngrok", "cloudflared"))
	tool, err := pickShareTool(false, true, false, false, false, false, "dev.example.com", "", "", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tool.mode != shareModeCloudflare {
		t.Errorf("mode = %v, want shareModeCloudflare", tool.mode)
	}
	if tool.domain != "dev.example.com" {
		t.Errorf("domain = %q, want dev.example.com", tool.domain)
	}
}

func TestPickShareTool_domain_withCloudflareDefault(t *testing.T) {
	t.Setenv("PATH", fakeBin(t, "ngrok", "cloudflared"))
	tool, err := pickShareTool(false, false, false, false, false, false, "dev.example.com", "cloudflare", "", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tool.mode != shareModeCloudflare {
		t.Errorf("mode = %v, want shareModeCloudflare", tool.mode)
	}
	if tool.domain != "dev.example.com" {
		t.Errorf("domain = %q, want dev.example.com", tool.domain)
	}
}

func TestPickShareTool_domain_impliesCloudflare(t *testing.T) {
	t.Setenv("PATH", fakeBin(t, "ngrok", "cloudflared"))
	// --domain is Cloudflare-only, so it selects the tool with no flag at all and
	// outranks a configured default pointing somewhere else.
	for _, defaultTool := range []string{"", "ngrok"} {
		tool, err := pickShareTool(false, false, false, false, false, false, "dev.example.com", defaultTool, "", "")
		if err != nil {
			t.Errorf("default %q: unexpected error: %v", defaultTool, err)
			continue
		}
		if tool.mode != shareModeCloudflare {
			t.Errorf("default %q: mode = %v, want shareModeCloudflare", defaultTool, tool.mode)
		}
		if tool.domain != "dev.example.com" {
			t.Errorf("default %q: domain = %q, want dev.example.com", defaultTool, tool.domain)
		}
	}
}

// ngrok reserves domains of its own, so --domain means the same thing there: a
// public URL that survives the next run.
func TestPickShareTool_domain_withExplicitNgrok(t *testing.T) {
	t.Setenv("PATH", fakeBin(t, "ngrok", "cloudflared"))
	tool, err := pickShareTool(true, false, false, false, false, false, "myapp.ngrok.app", "", "", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tool.mode != shareModeNgrok {
		t.Errorf("mode = %v, want shareModeNgrok", tool.mode)
	}
	if tool.ngrok.domain != "myapp.ngrok.app" {
		t.Errorf("ngrok.domain = %q, want myapp.ngrok.app", tool.ngrok.domain)
	}
}

func TestPickShareTool_defaultTool_used(t *testing.T) {
	t.Setenv("PATH", fakeBin(t, "ngrok", "cloudflared", "expose", "ssh"))
	cases := []struct {
		defaultTool string
		wantMode    shareMode
		wantSSHHost string
	}{
		{"ngrok", shareModeNgrok, ""},
		{"cloudflare", shareModeCloudflare, ""},
		{"expose", shareModeExpose, ""},
		{"serveo", shareModeSSH, "serveo.net"},
		{"localhost-run", shareModeSSH, "localhost.run"},
		{"pinggy", shareModeSSH, "free.pinggy.io"},
	}
	for _, c := range cases {
		tool, err := pickShareTool(false, false, false, false, false, false, "", c.defaultTool, "", "")
		if err != nil {
			t.Errorf("default %q: unexpected error: %v", c.defaultTool, err)
			continue
		}
		if tool.mode != c.wantMode {
			t.Errorf("default %q: mode = %v, want %v", c.defaultTool, tool.mode, c.wantMode)
		}
		if tool.ssh.host != c.wantSSHHost {
			t.Errorf("default %q: ssh.host = %q, want %q", c.defaultTool, tool.ssh.host, c.wantSSHHost)
		}
	}
}

func TestPickShareTool_defaultTool_flagOverrides(t *testing.T) {
	t.Setenv("PATH", fakeBin(t, "ngrok", "cloudflared"))
	tool, err := pickShareTool(true, false, false, false, false, false, "", "cloudflare", "", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tool.mode != shareModeNgrok {
		t.Errorf("mode = %v, want shareModeNgrok (flag beats default)", tool.mode)
	}
}

func TestEnsureCloudflareTunnel_happyPath_logsInWhenNoCert(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("TUNNEL_ORIGIN_CERT", "")
	if err := os.MkdirAll(filepath.Join(home, ".cloudflared"), 0755); err != nil {
		t.Fatal(err)
	}
	logFile := fakeCloudflared(t, "created", 0, "routed", 0)

	name, err := ensureCloudflareTunnel("mysite", "dev.example.com", true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if name != "lerd-mysite" {
		t.Errorf("name = %q, want lerd-mysite", name)
	}
	calls := readCalls(t, logFile)
	for _, want := range []string{"tunnel login", "tunnel create lerd-mysite", "tunnel route dns lerd-mysite dev.example.com"} {
		if !strings.Contains(calls, want) {
			t.Errorf("calls %q missing %q", calls, want)
		}
	}
}

func TestEnsureCloudflareTunnel_skipsLoginWhenCertExists(t *testing.T) {
	cert := filepath.Join(t.TempDir(), "cert.pem")
	if err := os.WriteFile(cert, []byte("x"), 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("TUNNEL_ORIGIN_CERT", cert)
	logFile := fakeCloudflared(t, "created", 0, "routed", 0)

	if _, err := ensureCloudflareTunnel("mysite", "dev.example.com", true); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.Contains(readCalls(t, logFile), "tunnel login") {
		t.Error("tunnel login was called despite existing cert")
	}
}

func TestEnsureCloudflareTunnel_toleratesExistingTunnelAndRoute(t *testing.T) {
	certWithTunnelCredentials(t)
	fakeCloudflared(t, "tunnel with name already exists", 1, "record with that host already exists", 1)

	var name string
	var err error
	out := captureStdout(t, func() { name, err = ensureCloudflareTunnel("mysite", "dev.example.com", true) })
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if name != "lerd-mysite" {
		t.Errorf("name = %q, want lerd-mysite", name)
	}
	if !strings.Contains(out, "already exists") {
		t.Errorf("expected a note about the existing record, got %q", out)
	}
}

// A tunnel that exists on the account with no local credentials is recovered
// automatically: lerd deletes the orphaned tunnel and creates a fresh one,
// rather than failing and asking the user to do it by hand.
func TestEnsureCloudflareTunnel_missingCredentialsRecreatesTunnel(t *testing.T) {
	certWithoutTunnelCredentials(t)
	logFile := fakeCloudflaredRecovering(t)

	name, err := ensureCloudflareTunnel("mysite", "dev.example.com", true)
	if err != nil {
		t.Fatalf("expected auto-recovery, got error: %v", err)
	}
	if name != "lerd-mysite" {
		t.Errorf("name = %q, want lerd-mysite", name)
	}
	calls := readCalls(t, logFile)
	if !strings.Contains(calls, "tunnel delete lerd-mysite") {
		t.Errorf("the orphaned tunnel was not deleted:\n%s", calls)
	}
	// Two creates: the first hits "already exists", the recovery one succeeds.
	if strings.Count(calls, "tunnel create lerd-mysite") < 2 {
		t.Errorf("the tunnel was not recreated after delete:\n%s", calls)
	}
	// The DNS record still points at the deleted tunnel, so the route must
	// force it onto the new one (else Cloudflare serves error 1033).
	if !strings.Contains(calls, "tunnel route dns --overwrite-dns lerd-mysite dev.example.com") {
		t.Errorf("recreated tunnel's DNS was not force-repointed:\n%s", calls)
	}
}

// A normal (non-recovery) route must not force-overwrite the DNS, so a record a
// user aimed elsewhere is respected.
func TestEnsureCloudflareTunnel_normalRouteDoesNotOverwriteDNS(t *testing.T) {
	certWithTunnelCredentials(t)
	logFile := fakeCloudflared(t, "created", 0, "routed", 0)

	if _, err := ensureCloudflareTunnel("mysite", "dev.example.com", true); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.Contains(readCalls(t, logFile), "--overwrite-dns") {
		t.Errorf("a normal share must not overwrite DNS:\n%s", readCalls(t, logFile))
	}
}

func TestEnsureCloudflareTunnel_freshRouteWarnsAboutPropagation(t *testing.T) {
	certWithTunnelCredentials(t)
	fakeCloudflared(t, "created", 0, "Added CNAME dev.example.com which will route to this tunnel", 0)

	out := captureStdout(t, func() {
		if _, err := ensureCloudflareTunnel("mysite", "dev.example.com", true); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})
	if !strings.Contains(out, "30 minutes") {
		t.Errorf("a freshly added record should warn about resolver caching, got %q", out)
	}
}

func TestEnsureCloudflareTunnel_reusedRouteStaysQuiet(t *testing.T) {
	certWithTunnelCredentials(t)
	fakeCloudflared(t, "created", 0, "", 0)

	out := captureStdout(t, func() {
		if _, err := ensureCloudflareTunnel("mysite", "dev.example.com", true); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})
	if strings.Contains(out, "30 minutes") {
		t.Errorf("an unchanged route should not warn about propagation, got %q", out)
	}
}

func TestEnsureCloudflareTunnel_routeFailurePropagates(t *testing.T) {
	cert := filepath.Join(t.TempDir(), "cert.pem")
	if err := os.WriteFile(cert, []byte("x"), 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("TUNNEL_ORIGIN_CERT", cert)
	fakeCloudflared(t, "created", 0, "zone not found", 1)

	_, err := ensureCloudflareTunnel("mysite", "dev.example.com", true)
	if err == nil || !strings.Contains(err.Error(), "route dns") {
		t.Errorf("error = %v, want route dns failure", err)
	}
}
