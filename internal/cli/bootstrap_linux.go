//go:build linux

package cli

import (
	"errors"
	"fmt"
	"os"
	"os/user"
	"path/filepath"

	"github.com/geodro/lerd/internal/certs"
	"github.com/geodro/lerd/internal/dns"
	"github.com/geodro/lerd/internal/feedback"
)

// writeDNSSudoers is the seam tests override to keep the sudoers write off the
// real /etc/sudoers.d.
var writeDNSSudoers = dns.WriteSudoersForUser

// runBootstrapSystem performs the root prerequisites the per-user install
// relies on: the unprivileged-port sysctl, systemd linger, and the DNS sudoers
// rule that lets the install configure the resolver without prompting. Both
// install routes come through here, a package maintainer script directly and an
// interactive install by re-executing itself under sudo. skipSudoers leaves out
// the resolver grant for an install that manages no DNS and would never use it.
func runBootstrapSystem(target string, skipSudoers bool) error {
	feedback.Header("Bootstrapping system for lerd")

	if err := writePortDropIn(unprivPortDropIn, bootstrapRunner); err != nil {
		feedback.Warn("enabling unprivileged ports: %v", err)
	} else {
		feedback.Done("unprivileged ports enabled for 80/443")
	}

	if target == "" {
		return nil
	}
	if err := enableLinger(target, bootstrapRunner); err != nil {
		feedback.Warn("enabling linger for %s: %v", target, err)
	} else {
		feedback.Done("systemd linger enabled for " + target)
	}
	if dns.HostOwnsResolver() {
		feedback.Note("NixOS owns the resolver; skipping DNS sudoers")
		return nil
	}
	if skipSudoers {
		return nil
	}
	if err := writeDNSSudoers(target); err != nil {
		feedback.Warn("installing DNS sudoers rule: %v", err)
	} else {
		feedback.Done("DNS sudoers rule installed for " + target)
	}
	return nil
}

// runBootstrapUntrustCA drops lerd's CA from the system trust store on
// uninstall. caRoot names the CA so an anchor an older install let mkcert
// write is found too; without it only lerd's own anchor goes.
func runBootstrapUntrustCA(caRoot string) error {
	var pem []byte
	if caRoot != "" {
		pem, _ = os.ReadFile(filepath.Join(caRoot, "rootCA.pem"))
	}
	if err := certs.UntrustCAFromSystemStore(pem); err != nil {
		if errors.Is(err, certs.ErrNoSystemTrustStore) {
			return nil
		}
		return fmt.Errorf("removing CA from system store: %w", err)
	}
	feedback.Done("lerd CA removed from the system store")
	return nil
}

// runBootstrapTrustCA trusts the user's root CA in the system store, as root.
// A missing CA is no error (localhost installs have none). Without caRoot the
// target user's default is used, since resolving it as root finds root's.
func runBootstrapTrustCA(target, caRoot string) error {
	if target == "" && caRoot == "" {
		return fmt.Errorf("--trust-ca needs a target user (pass --user)")
	}
	if caRoot == "" {
		u, err := user.Lookup(target)
		if err != nil {
			return fmt.Errorf("looking up user %s: %w", target, err)
		}
		caRoot = filepath.Join(u.HomeDir, ".local", "share", "mkcert")
	}
	caPath := filepath.Join(caRoot, "rootCA.pem")
	pem, err := os.ReadFile(caPath)
	if err != nil {
		feedback.Note("no lerd CA at " + caPath + " yet, skipping system trust")
		return nil
	}
	if err := certs.TrustCAInSystemStore(pem); err != nil {
		if errors.Is(err, certs.ErrNoSystemTrustStore) {
			feedback.Note("no system CA trust store on this distro; trust " + caPath + " through your system configuration (NixOS: security.pki.certificateFiles)")
			return nil
		}
		return fmt.Errorf("trusting CA in system store: %w", err)
	}
	feedback.Done("lerd CA trusted in the system store")
	return nil
}
