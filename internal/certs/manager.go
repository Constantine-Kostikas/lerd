package certs

import (
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"sync/atomic"
	"time"
)

// certReissueWindow is how close to NotAfter a leaf cert may drift before the
// IssueCert reuse path stops trusting it and reissues. Mirrors the 30-day
// threshold `lerd status` warns at, so the self-heal kicks in as the warning
// starts rather than waiting for the cert to actually expire.
const certReissueWindow = 30 * 24 * time.Hour

// issueCertMu serialises issueCertAtomic calls per primaryDomain. Two
// concurrent reissues for the same site (e.g. boot scanWorktrees racing the
// watcher's syncWorktree on the same site) must not interleave their
// renames — pre-fix both used a fixed "<primary>.crt.new" tempfile path,
// so one would clobber the other's tempfile or rename a partially-flushed
// file. Lock per domain so unrelated sites still issue in parallel.
var issueCertMu sync.Map // map[string]*sync.Mutex

func lockForDomain(domain string) *sync.Mutex {
	if m, ok := issueCertMu.Load(domain); ok {
		return m.(*sync.Mutex)
	}
	m, _ := issueCertMu.LoadOrStore(domain, &sync.Mutex{})
	return m.(*sync.Mutex)
}

// writeLeafFn signs the leaf; a var so tests can make the write fail midway.
var writeLeafFn = writeLeaf

// tempSuffixSeq increments per call to issueCertAtomic so concurrent
// callers (across processes too) don't share a tempfile path even if the
// per-domain mutex is bypassed somehow.
var tempSuffixSeq atomic.Uint64

// IssueCert issues a certificate named after primaryDomain covering every domain
// and its wildcard. An existing pair is reused only while it is valid and clear
// of certReissueWindow, so a start or watcher pass self-heals an aging cert.
func IssueCert(primaryDomain string, allDomains []string, certsDir string) error {
	certFile := filepath.Join(certsDir, primaryDomain+".crt")
	keyFile := filepath.Join(certsDir, primaryDomain+".key")
	if _, certErr := os.Stat(certFile); certErr == nil {
		if _, keyErr := os.Stat(keyFile); keyErr == nil {
			if !certNeedsReissue(certFile, certReissueWindow) {
				return nil
			}
		}
	}
	return issueCertAtomic(primaryDomain, allDomains, certsDir)
}

// certNeedsReissue reports whether the PEM cert at path should be reissued: it
// returns true when the file is unreadable, not a parseable certificate, or
// within window of (or past) its NotAfter. An unreadable or malformed cert is
// treated as needing reissue rather than trusted.
func certNeedsReissue(path string, window time.Duration) bool {
	data, err := os.ReadFile(path)
	if err != nil {
		return true
	}
	block, _ := pem.Decode(data)
	if block == nil {
		return true
	}
	parsed, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return true
	}
	return time.Until(parsed.NotAfter) < window
}

// IssueCertForce regenerates the certificate even if files exist, renaming
// atomically so a failure keeps the previous pair: a missing cert trips
// RepairVhosts into flipping the site to plain HTTP.
func IssueCertForce(primaryDomain string, allDomains []string, certsDir string) error {
	return issueCertAtomic(primaryDomain, allDomains, certsDir)
}

func issueCertAtomic(primaryDomain string, allDomains []string, certsDir string) error {
	mu := lockForDomain(primaryDomain)
	mu.Lock()
	defer mu.Unlock()

	if err := os.MkdirAll(certsDir, 0755); err != nil {
		return err
	}

	certFile := filepath.Join(certsDir, primaryDomain+".crt")
	keyFile := filepath.Join(certsDir, primaryDomain+".key")
	// Per-call unique suffix: pid + monotonic seq + ns time. Ensures
	// cross-process concurrent issuers (e.g. lerd-watcher and lerd-ui)
	// don't collide on the .new path even when the in-process mutex
	// can't help.
	suffix := ".new." + strconv.Itoa(os.Getpid()) + "." + strconv.FormatUint(tempSuffixSeq.Add(1), 10) + "." + strconv.FormatInt(time.Now().UnixNano(), 10)
	tmpCert := certFile + suffix
	tmpKey := keyFile + suffix

	var sans []string
	for _, d := range allDomains {
		sans = append(sans, d, "*."+d)
	}

	// Generating the CA on first issue, as mkcert did, keeps a site secured
	// before install ever ran working.
	if err := EnsureCA(); err != nil {
		return fmt.Errorf("certificate for %s: %w", primaryDomain, err)
	}
	if err := writeLeafFn(sans, tmpCert, tmpKey); err != nil {
		os.Remove(tmpCert) //nolint:errcheck
		os.Remove(tmpKey)  //nolint:errcheck
		return fmt.Errorf("certificate for %s: %w", primaryDomain, err)
	}

	// Swap the new cert and key in with os.Rename, which atomically replaces
	// the target in place. Crucially the previous cert is COPIED aside, not
	// moved: moving it would unlink certFile for the instant between the two
	// renames, and a concurrent `nginx -s reload` (the watcher or UI reissuing
	// the same site while the CLI reloads) that lands in that window crashes
	// with "cannot load certificate ... No such file". Copying keeps a complete
	// cert at certFile at every moment while still letting us roll back to the
	// previous cert if the key rename fails (a cert/key mismatch is worse than a
	// transient issue failure: nginx refuses to start the site).
	bakCert := certFile + ".bak." + strconv.Itoa(os.Getpid())
	hadPrevCert := false
	if _, err := os.Stat(certFile); err == nil {
		if err := copyFile(certFile, bakCert); err != nil {
			os.Remove(tmpCert) //nolint:errcheck
			os.Remove(tmpKey)  //nolint:errcheck
			return fmt.Errorf("backing up cert for %s: %w", primaryDomain, err)
		}
		hadPrevCert = true
	}
	if err := os.Rename(tmpCert, certFile); err != nil {
		os.Remove(tmpCert) //nolint:errcheck
		os.Remove(tmpKey)  //nolint:errcheck
		if hadPrevCert {
			os.Remove(bakCert) //nolint:errcheck
		}
		return fmt.Errorf("renaming cert for %s: %w", primaryDomain, err)
	}
	if err := os.Rename(tmpKey, keyFile); err != nil {
		os.Remove(tmpKey) //nolint:errcheck
		if hadPrevCert {
			os.Rename(bakCert, certFile) //nolint:errcheck — atomic restore of prev cert
		} else {
			os.Remove(certFile) //nolint:errcheck
		}
		return fmt.Errorf("renaming key for %s: %w", primaryDomain, err)
	}
	if hadPrevCert {
		os.Remove(bakCert) //nolint:errcheck
	}
	return nil
}

// copyFile copies src to dst, creating or truncating dst. Used to back up the
// previous cert without unlinking it, so the live cert path is never absent.
func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close() //nolint:errcheck
	out, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close() //nolint:errcheck
		return err
	}
	return out.Close()
}
