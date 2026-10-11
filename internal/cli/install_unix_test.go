//go:build !windows

package cli

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// The NetworkManager dispatcher restarts lerd-dns on every interface event, so
// a resume that brings several links back at once can fire more restarts than
// systemd's default allows and park the unit in failed for good (#1087). The
// directive only works in [Unit], which is how #1087 shipped broken.
func TestDNSServiceHasNoStartRateLimit(t *testing.T) {
	unit, _, _ := strings.Cut(dnsServiceContent("/bin/lerd", "127.0.0.1"), "[Service]")
	if !strings.Contains(unit, "StartLimitIntervalSec=0") {
		t.Errorf("StartLimitIntervalSec=0 must sit in [Unit]:\n%s", unit)
	}
}

func TestDNSServiceRunsTheBuiltInServer(t *testing.T) {
	got := dnsServiceContent("/home/u/.local/bin/lerd", "127.0.0.1")
	for _, want := range []string{
		"ExecStart=/home/u/.local/bin/lerd dns-serve --listen 127.0.0.1\n",
		"Restart=always",
		"WantedBy=default.target",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("unit is missing %q:\n%s", want, got)
		}
	}
}

// Where lerd-dns answers the LAN itself it has to take every interface, or
// lan:expose would leave remote machines with nothing on lanIP:5300.
func TestDNSListenHostFollowsWhoBindsTheLAN(t *testing.T) {
	prev := lerdDNSBindsLANPort
	t.Cleanup(func() { lerdDNSBindsLANPort = prev })

	lerdDNSBindsLANPort = false
	if got := dnsListenHost(); got != "127.0.0.1" {
		t.Errorf("behind the forwarder lerd-dns must stay on loopback, got %s", got)
	}
	lerdDNSBindsLANPort = true
	if got := dnsListenHost(); got != "0.0.0.0" {
		t.Errorf("answering the LAN itself needs every interface, got %s", got)
	}
}

// A shim outlives the binary it was written against: `brew upgrade lerd` moves
// lerd into a new keg and the recorded path stops resolving. The shim has to
// reach for lerd on PATH rather than fail with no such file.
func TestShimPreambleFallsBackToLerdOnPath(t *testing.T) {
	dir := t.TempDir()
	fake := filepath.Join(dir, "lerd")
	if err := os.WriteFile(fake, []byte("#!/bin/sh\necho \"ran $*\"\n"), 0755); err != nil {
		t.Fatal(err)
	}
	script := filepath.Join(dir, "php")
	body := shimPreamble("/retired/keg/bin/lerd") + "exec \"$LERD\" php \"$@\"\n"
	if err := os.WriteFile(script, []byte(body), 0755); err != nil {
		t.Fatal(err)
	}

	cmd := exec.Command(script, "-v")
	cmd.Env = append(os.Environ(), "PATH="+dir)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("shim failed: %v\n%s\n%s", err, out, body)
	}
	if strings.TrimSpace(string(out)) != "ran php -v" {
		t.Errorf("shim did not fall back to lerd on PATH, got: %q", out)
	}
}

func TestDetectSystemNode_findsNvmDirEvenWhenPathIsEmpty(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	t.Setenv("PATH", filepath.Join(tmp, "lerd", "bin"))

	nvm := filepath.Join(tmp, ".nvm", "versions", "node", "v22.0.0")
	if err := os.MkdirAll(nvm, 0755); err != nil {
		t.Fatal(err)
	}
	got := detectSystemNode()
	if got == "" {
		t.Fatal("detectSystemNode returned empty even though ~/.nvm contains a version")
	}
	if !strings.Contains(got, ".nvm") {
		t.Errorf("expected detectSystemNode to point at the nvm dir, got %q", got)
	}
}

// An nvm with no Node versions in it is invisible to detectSystemNode, so
// detectNvm is what makes the management question reach that user at all.
func TestDetectNvm(t *testing.T) {
	t.Run("empty nvm dir under NVM_DIR counts", func(t *testing.T) {
		isolateNvmLookup(t)
		tmp := t.TempDir()
		t.Setenv("NVM_DIR", tmp)
		if detectNvm() {
			t.Error("detectNvm true for a directory with no nvm.sh")
		}
		if err := os.WriteFile(filepath.Join(tmp, "nvm.sh"), []byte("#!/bin/bash\n"), 0644); err != nil {
			t.Fatal(err)
		}
		if !detectNvm() {
			t.Error("detectNvm false even though $NVM_DIR holds nvm.sh")
		}
	})

	t.Run("falls back to ~/.nvm", func(t *testing.T) {
		isolateNvmLookup(t)
		tmp := t.TempDir()
		t.Setenv("NVM_DIR", "")
		t.Setenv("HOME", tmp)
		if err := os.MkdirAll(filepath.Join(tmp, ".nvm"), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(tmp, ".nvm", "nvm.sh"), []byte("#!/bin/bash\n"), 0644); err != nil {
			t.Fatal(err)
		}
		if !detectNvm() {
			t.Error("detectNvm false for an nvm.sh in the default ~/.nvm")
		}
	})
}

func TestDetectSystemNode_findsNpmInPath(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)

	binDir := filepath.Join(tmp, "shim-bin")
	if err := os.MkdirAll(binDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(binDir, "npm"), []byte("#!/bin/sh\n"), 0755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", binDir)
	got := detectSystemNode()
	if got == "" {
		t.Fatal("detectSystemNode returned empty even though npm is on PATH")
	}
	if !strings.HasSuffix(got, "/npm") {
		t.Errorf("expected detectSystemNode to find npm, got %q", got)
	}
}
