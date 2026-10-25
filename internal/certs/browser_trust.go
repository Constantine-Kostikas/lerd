package certs

import (
	"crypto/x509"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// certutilLookup reports whether certutil is resolvable on PATH. Seam for tests.
var certutilLookup = func() bool {
	_, err := exec.LookPath("certutil")
	return err == nil
}

// BrowserTrustAvailable reports whether certutil (nss-tools) is there to put the
// root CA into the browser NSS stores. Without it only the system store trusts
// .test, so curl and PHP are fine but Firefox and Chrome warn.
func BrowserTrustAvailable() bool {
	return certutilLookup()
}

// nssDBDirs returns the existing NSS databases lerd writes the CA into: the
// Chromium store and the native and snap Firefox profiles, as mkcert did. A
// sandboxed (Flatpak) store is left out, since no repair could ever reach it.
var nssDBDirs = func() []string {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil
	}
	var dirs []string
	for _, dir := range []string{
		filepath.Join(home, ".pki", "nssdb"),
		filepath.Join(home, "snap", "chromium", "current", ".pki", "nssdb"),
		"/etc/pki/nssdb",
	} {
		if nssDBPresent(dir) {
			dirs = append(dirs, dir)
		}
	}
	for _, pattern := range []string{
		filepath.Join(home, ".mozilla", "firefox", "*"),
		filepath.Join(home, "Library", "Application Support", "Firefox", "Profiles", "*"),
		filepath.Join(home, "snap", "firefox", "common", ".mozilla", "firefox", "*"),
	} {
		matches, _ := filepath.Glob(pattern)
		for _, dir := range matches {
			if nssDBPresent(dir) {
				dirs = append(dirs, dir)
			}
		}
	}
	return dirs
}

// nssDBPresent reports whether dir holds an NSS certificate database. Only the
// sql-backed cert9.db counts: every browser that still ships has moved to it,
// and the legacy format needs a different certutil prefix to read at all.
func nssDBPresent(dir string) bool {
	_, err := os.Stat(filepath.Join(dir, "cert9.db"))
	return err == nil
}

// nssCertPEM exports the certificate stored under nickname in the NSS database
// at dir, or nothing when the store has none. Overridable in tests.
var nssCertPEM = func(dir, nickname string) []byte {
	out, err := exec.Command("certutil", "-d", "sql:"+dir, "-L", "-n", nickname, "-a").Output()
	if err != nil {
		return nil
	}
	return out
}

// caNSSNickname is the nickname mkcert stores the root CA under: its own label
// followed by the certificate's serial in decimal. Deriving it the way mkcert
// does is what lets the lookup ask for the CA lerd signs with today rather than
// for whatever certificate happens to wear the mkcert name.
func caNSSNickname(der []byte) string {
	cert, err := x509.ParseCertificate(der)
	if err != nil {
		return ""
	}
	return "mkcert development CA " + cert.SerialNumber.String()
}

// BrowserStoresMissingCA returns the NSS databases that do not hold the root CA
// lerd signs with. A browser reading one of those warns on every .test site even
// though curl, PHP and openssl are happy, which is the state certutil's mere
// presence on PATH says nothing about.
//
// Empty when every store has it, and equally empty when the question could not
// be asked: no certutil to ask with, no CA generated yet, or no browser store on
// the machine. An unanswered check is not a finding. The exported certificate is
// compared as DER rather than trusted for carrying the right nickname, so a CA
// regenerated after a store imported the old one reports as missing.
func BrowserStoresMissingCA() []string {
	if !BrowserTrustAvailable() {
		return nil
	}
	der := rootCADER()
	if der == nil {
		return nil
	}
	nickname := caNSSNickname(der)
	if nickname == "" {
		return nil
	}
	var missing []string
	for _, dir := range nssDBDirs() {
		if !bundleContainsDER(nssCertPEM(dir, nickname), der) {
			missing = append(missing, dir)
		}
	}
	return missing
}

// runCertutil runs certutil against one NSS database. Overridable in tests.
var runCertutil = func(args ...string) error {
	out, err := exec.Command("certutil", args...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("%w: %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

// InstallBrowserTrust imports the root CA into every NSS database missing it,
// under mkcert's nickname so BrowserStoresMissingCA finds it, and returns the
// stores that refused (a read-only system database, most often).
func InstallBrowserTrust() []error {
	root, err := CARoot()
	if err != nil {
		return []error{err}
	}
	nickname := caNSSNickname(rootCADER())
	if nickname == "" {
		return nil
	}
	var errs []error
	for _, dir := range BrowserStoresMissingCA() {
		if err := runCertutil("-A", "-d", "sql:"+dir, "-t", "C,,", "-n", nickname, "-i", filepath.Join(root, caCertName)); err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", dir, err))
		}
	}
	return errs
}

// RemoveBrowserTrust deletes the root CA from every NSS database holding it.
func RemoveBrowserTrust() {
	if !BrowserTrustAvailable() {
		return
	}
	nickname := caNSSNickname(rootCADER())
	if nickname == "" {
		return
	}
	for _, dir := range nssDBDirs() {
		if nssCertPEM(dir, nickname) != nil {
			runCertutil("-D", "-d", "sql:"+dir, "-n", nickname) //nolint:errcheck
		}
	}
}
