//go:build !windows

package cli

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/geodro/lerd/internal/config"
)

// TestMigrateSiteTLD_ReissuesCertForSecuredSiteWithWorktree pins the fix
// for the regression where migrating a secured site's TLD regenerated the
// worktree vhosts but never reissued the parent cert. SSL handshakes to
// branch.<newPrimary> would fail because the cert SANs still carried the
// old TLD's wildcard. The migration must produce a cert at the NEW primary
// path that covers the renamed worktree subdomain.
func TestMigrateSiteTLD_ReissuesCertForSecuredSiteWithWorktree(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmp)
	t.Setenv("XDG_DATA_HOME", tmp)

	binDir := filepath.Join(tmp, "lerd", "bin")
	if err := os.MkdirAll(binDir, 0755); err != nil {
		t.Fatal(err)
	}
	// Fake mkcert that writes its SAN list into the cert/key files so the
	// test can assert the new TLD's wildcard SAN was passed.
	fakeMkcert := `#!/bin/sh
CRT=""
KEY=""
SANS=""
while [ "$#" -gt 0 ]; do
  case "$1" in
    -cert-file) shift; CRT="$1" ;;
    -key-file) shift; KEY="$1" ;;
    *) SANS="$SANS $1" ;;
  esac
  shift
done
echo "$SANS" > "$CRT"
echo "FAKE-KEY" > "$KEY"
`
	if err := os.WriteFile(filepath.Join(binDir, "mkcert"), []byte(fakeMkcert), 0755); err != nil {
		t.Fatal(err)
	}

	if err := os.MkdirAll(config.NginxConfD(), 0755); err != nil {
		t.Fatal(err)
	}

	// Simulate a worktree via .git/worktrees/<entry>/ — DetectWorktrees
	// reads gitdir + HEAD from the entry dir.
	siteDir := filepath.Join(tmp, "alpha")
	if err := os.MkdirAll(siteDir, 0755); err != nil {
		t.Fatal(err)
	}
	checkout := filepath.Join(tmp, "feat-x-checkout")
	if err := os.MkdirAll(checkout, 0755); err != nil {
		t.Fatal(err)
	}
	wtMeta := filepath.Join(siteDir, ".git", "worktrees", "feat-x")
	os.MkdirAll(wtMeta, 0755)
	os.WriteFile(filepath.Join(wtMeta, "HEAD"), []byte("ref: refs/heads/feat-x\n"), 0644)
	os.WriteFile(filepath.Join(wtMeta, "gitdir"), []byte(filepath.Join(checkout, ".git")+"\n"), 0644)

	if err := os.WriteFile(filepath.Join(siteDir, ".env"), []byte("APP_URL=https://alpha.test\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(checkout, ".env"), []byte("APP_URL=https://feat-x.alpha.test\n"), 0644); err != nil {
		t.Fatal(err)
	}
	// The enable path restores HTTPS from the committed .lerd.yaml intent.
	if err := config.SaveProjectConfig(siteDir, &config.ProjectConfig{Secured: true}); err != nil {
		t.Fatalf("save .lerd.yaml: %v", err)
	}

	if err := config.AddSite(config.Site{
		Name:       "alpha",
		Path:       siteDir,
		Domains:    []string{"alpha.test"},
		PHPVersion: "8.4",
		Secured:    true,
	}); err != nil {
		t.Fatalf("AddSite: %v", err)
	}

	// Seed an OLD cert at the old primary so we can verify it's torn down.
	certsDir := filepath.Join(tmp, "lerd", "certs", "sites")
	if err := os.MkdirAll(certsDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(certsDir, "alpha.test.crt"), []byte("OLD-CERT"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(certsDir, "alpha.test.key"), []byte("OLD-KEY"), 0644); err != nil {
		t.Fatal(err)
	}

	changed := migrateSiteTLD("test", "localhost", false)
	if !slices.Equal(changed, []string{"alpha"}) {
		t.Fatalf("changed = %v, want [alpha]", changed)
	}

	// New cert must exist at the new primary.
	newCert := filepath.Join(certsDir, "alpha.localhost.crt")
	body, err := os.ReadFile(newCert)
	if err != nil {
		t.Fatalf("expected cert at %s, got %v", newCert, err)
	}
	// Cert SANs must include the renamed worktree wildcard.
	wantSAN := "*.feat-x.alpha.localhost"
	if !contains(body, wantSAN) {
		t.Errorf("cert %s missing SAN %q; got %q", newCert, wantSAN, body)
	}
	// Old cert files must be cleaned up so the certs/sites dir doesn't
	// accumulate stale entries across migrations.
	if _, err := os.Stat(filepath.Join(certsDir, "alpha.test.crt")); !os.IsNotExist(err) {
		t.Errorf("old cert should be removed after migration; stat err = %v", err)
	}
	if _, err := os.Stat(filepath.Join(certsDir, "alpha.test.key")); !os.IsNotExist(err) {
		t.Errorf("old key should be removed after migration; stat err = %v", err)
	}
}

// TestMigrateSiteTLD_DNSRoundTripRestoresHTTPS pins the fix for #749: disabling
// then re-enabling DNS must be lossless. A site secured on .test falls back to
// plain .localhost http on disable (registry flag off) but keeps its committed
// HTTPS intent in .lerd.yaml; re-enabling reads that intent and re-secures the
// site on .test, reissuing the cert and syncing the .env back to https.
func TestMigrateSiteTLD_DNSRoundTripRestoresHTTPS(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmp)
	t.Setenv("XDG_DATA_HOME", tmp)

	binDir := filepath.Join(tmp, "lerd", "bin")
	if err := os.MkdirAll(binDir, 0755); err != nil {
		t.Fatal(err)
	}
	fakeMkcert := `#!/bin/sh
CRT=""
KEY=""
while [ "$#" -gt 0 ]; do
  case "$1" in
    -cert-file) shift; CRT="$1" ;;
    -key-file) shift; KEY="$1" ;;
  esac
  shift
done
echo "CERT" > "$CRT"
echo "KEY" > "$KEY"
`
	if err := os.WriteFile(filepath.Join(binDir, "mkcert"), []byte(fakeMkcert), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(config.NginxConfD(), 0755); err != nil {
		t.Fatal(err)
	}

	siteDir := filepath.Join(tmp, "alpha")
	if err := os.MkdirAll(siteDir, 0755); err != nil {
		t.Fatal(err)
	}
	envPath := filepath.Join(siteDir, ".env")
	if err := os.WriteFile(envPath, []byte("APP_URL=https://alpha.test\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := config.SaveProjectConfig(siteDir, &config.ProjectConfig{Secured: true}); err != nil {
		t.Fatalf("save .lerd.yaml: %v", err)
	}
	if err := config.AddSite(config.Site{
		Name:       "alpha",
		Path:       siteDir,
		Domains:    []string{"alpha.test"},
		PHPVersion: "8.4",
		Secured:    true,
	}); err != nil {
		t.Fatalf("AddSite: %v", err)
	}

	// Disable DNS: site falls back to plain-http .localhost.
	if changed := migrateSiteTLD("test", "localhost", true); !slices.Equal(changed, []string{"alpha"}) {
		t.Fatalf("disable changed = %v, want [alpha]", changed)
	}
	site, err := config.FindSite("alpha")
	if err != nil {
		t.Fatalf("FindSite after disable: %v", err)
	}
	if site.PrimaryDomain() != "alpha.localhost" || site.Secured {
		t.Fatalf("after disable: domain=%q secured=%v, want alpha.localhost/false", site.PrimaryDomain(), site.Secured)
	}
	if envBytes, _ := os.ReadFile(envPath); !contains(envBytes, "APP_URL=http://alpha.localhost") {
		t.Errorf("after disable .env = %q, want http://alpha.localhost", envBytes)
	}
	if proj, _ := config.LoadProjectConfig(siteDir); !proj.Secured {
		t.Fatalf("after disable: .lerd.yaml secured intent lost")
	}

	// Re-enable DNS: site returns to .test and HTTPS is restored from intent.
	if changed := migrateSiteTLD("localhost", "test", false); !slices.Equal(changed, []string{"alpha"}) {
		t.Fatalf("enable changed = %v, want [alpha]", changed)
	}
	site, err = config.FindSite("alpha")
	if err != nil {
		t.Fatalf("FindSite after enable: %v", err)
	}
	if site.PrimaryDomain() != "alpha.test" || !site.Secured {
		t.Fatalf("after enable: domain=%q secured=%v, want alpha.test/true", site.PrimaryDomain(), site.Secured)
	}
	if envBytes, _ := os.ReadFile(envPath); !contains(envBytes, "APP_URL=https://alpha.test") {
		t.Errorf("after enable .env = %q, want https://alpha.test", envBytes)
	}
	certPath := filepath.Join(config.CertsDir(), "sites", "alpha.test.crt")
	if _, err := os.Stat(certPath); err != nil {
		t.Errorf("expected reissued cert at %s: %v", certPath, err)
	}
}
