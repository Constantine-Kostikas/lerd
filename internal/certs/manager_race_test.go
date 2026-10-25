package certs

import (
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
)

// TestIssueCertForce_certNeverAbsentDuringReissue is the regression test for the
// "cannot load certificate ... No such file" crash: issueCertAtomic must swap
// the cert in without ever unlinking the live path, so a concurrent reader (an
// nginx reload mid-reissue) always finds a complete cert. An observer stats the
// cert in a tight loop while the issuer reissues many times; any absence fails.
//
// Against the old move-the-old-cert-aside implementation this records absences;
// against the copy-then-atomic-rename implementation it records zero.
func TestIssueCertForce_certNeverAbsentDuringReissue(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_DATA_HOME", tmp)

	// The renames are under test, not the signing, so the leaf is a stub
	// that keeps the reissue loop fast.
	useTestCA(t)
	orig := writeLeafFn
	t.Cleanup(func() { writeLeafFn = orig })
	writeLeafFn = func(_ []string, certFile, keyFile string) error {
		if err := os.WriteFile(certFile, []byte("CERT"), 0644); err != nil {
			return err
		}
		return os.WriteFile(keyFile, []byte("KEY"), 0600)
	}

	certsDir := filepath.Join(tmp, "lerd", "certs", "sites")
	if err := os.MkdirAll(certsDir, 0755); err != nil {
		t.Fatal(err)
	}
	certPath := filepath.Join(certsDir, "myapp.test.crt")
	// Seed an initial cert/key so every reissue hits the "previous cert exists"
	// branch, the one that used to move the live cert out of the way.
	if err := os.WriteFile(certPath, []byte("CERT"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(certsDir, "myapp.test.key"), []byte("KEY"), 0644); err != nil {
		t.Fatal(err)
	}

	var absent int64
	done := make(chan struct{})
	stopped := make(chan struct{})
	go func() {
		defer close(stopped)
		for {
			select {
			case <-done:
				return
			default:
				if _, err := os.Stat(certPath); os.IsNotExist(err) {
					atomic.AddInt64(&absent, 1)
				}
			}
		}
	}()

	for i := 0; i < 400; i++ {
		if err := IssueCertForce("myapp.test", []string{"myapp.test"}, certsDir); err != nil {
			t.Fatalf("reissue %d: %v", i, err)
		}
	}
	close(done)
	<-stopped

	if n := atomic.LoadInt64(&absent); n > 0 {
		t.Fatalf("cert path was absent %d times during reissue; the swap unlinks the live cert", n)
	}
}
