//go:build linux

package certs

import (
	"bytes"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
)

// ErrNoSystemTrustStore reports that no known CA anchor directory exists on
// this system, as on NixOS and other distros where trust is declarative.
var ErrNoSystemTrustStore = errors.New("no system CA trust store found")

// systemTrustStore is one distro family's CA anchor directory and the command
// that rebuilds the aggregated trust bundle from it.
type systemTrustStore struct {
	dir      string
	filename string
	command  []string
}

// systemTrustStores lists the anchor layouts mkcert recognises, in the same
// order; the first whose directory exists wins. A var so tests can substitute
// temp paths.
var systemTrustStores = []systemTrustStore{
	{"/etc/pki/ca-trust/source/anchors", "lerd-mkcert-rootCA.pem", []string{"update-ca-trust", "extract"}},       // Fedora/RHEL
	{"/usr/local/share/ca-certificates", "lerd-mkcert-rootCA.crt", []string{"update-ca-certificates"}},           // Debian/Ubuntu
	{"/etc/ca-certificates/trust-source/anchors", "lerd-mkcert-rootCA.crt", []string{"trust", "extract-compat"}}, // Arch
	{"/usr/share/pki/trust/anchors", "lerd-mkcert-rootCA.crt", []string{"update-ca-certificates"}},               // openSUSE
}

// updateSystemTrust refreshes the aggregated trust bundle. A var so tests can
// stub out the real command.
var updateSystemTrust = func(command []string) error {
	return exec.Command(command[0], command[1:]...).Run()
}

// TrustCAInSystemStore installs the given mkcert root CA PEM into whichever
// system trust store this distro uses. The caller must be root. Idempotent: a
// byte-identical CA already in place is a no-op, so re-running on every package
// upgrade is cheap.
//
// mkcert normally does this itself via an interactive sudo, which a package
// maintainer script cannot answer. `lerd bootstrap --trust-ca` runs as root and
// installs the user-generated CA directly instead.
func TrustCAInSystemStore(caPEM []byte) error {
	for _, store := range systemTrustStores {
		if info, err := os.Stat(store.dir); err != nil || !info.IsDir() {
			continue
		}
		caFile := filepath.Join(store.dir, store.filename)
		if existing, err := os.ReadFile(caFile); err == nil && bytes.Equal(existing, caPEM) {
			return nil
		}
		if err := os.WriteFile(caFile, caPEM, 0644); err != nil {
			return err
		}
		return updateSystemTrust(store.command)
	}
	return ErrNoSystemTrustStore
}

// UntrustCAFromSystemStore removes, as root, the anchor TrustCAInSystemStore
// wrote and any other holding caPEM's certificate, since installs before #1194
// let mkcert write its own under a name only mkcert knew.
func UntrustCAFromSystemStore(caPEM []byte) error {
	var sawStore bool
	for _, store := range systemTrustStores {
		if info, err := os.Stat(store.dir); err != nil || !info.IsDir() {
			continue
		}
		sawStore = true
		files := anchorFiles(store, firstCertDER(caPEM))
		for _, f := range files {
			if err := os.Remove(f); err != nil {
				return err
			}
		}
		if len(files) == 0 {
			continue
		}
		if err := updateSystemTrust(store.command); err != nil {
			return err
		}
	}
	if !sawStore {
		return ErrNoSystemTrustStore
	}
	return nil
}

// anchorFiles lists the anchors in store that lerd put there: its own file,
// and any file whose certificate is der.
func anchorFiles(store systemTrustStore, der []byte) []string {
	own := filepath.Join(store.dir, store.filename)
	var files []string
	if _, err := os.Stat(own); err == nil {
		files = append(files, own)
	}
	if der == nil {
		return files
	}
	entries, _ := os.ReadDir(store.dir)
	for _, e := range entries {
		path := filepath.Join(store.dir, e.Name())
		if path != own && e.Type().IsRegular() && bytes.Equal(firstCertDER(readFileNoErr(path)), der) {
			files = append(files, path)
		}
	}
	return files
}

// SystemTrustAnchorPresent reports whether an anchor lerd put there is on
// disk, so an uninstall only escalates to root when there is something to remove.
func SystemTrustAnchorPresent() bool {
	der := rootCADER()
	for _, store := range systemTrustStores {
		if len(anchorFiles(store, der)) > 0 {
			return true
		}
	}
	return false
}
