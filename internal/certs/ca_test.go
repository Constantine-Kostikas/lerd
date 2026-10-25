package certs

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"os"
	"path/filepath"
	"slices"
	"testing"
	"time"
)

// useTestCA points CAROOT at an empty temp dir, so issuing never reads or
// generates the developer's real CA.
func useTestCA(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	t.Setenv("CAROOT", root)
	return root
}

func parseCertFile(t *testing.T, path string) *x509.Certificate {
	t.Helper()
	der := firstCertDER(readFileNoErr(path))
	if der == nil {
		t.Fatalf("no certificate in %s", path)
	}
	cert, err := x509.ParseCertificate(der)
	if err != nil {
		t.Fatal(err)
	}
	return cert
}

func certDNSNames(t *testing.T, path string) []string {
	t.Helper()
	return parseCertFile(t, path).DNSNames
}

func TestEnsureCA_generatesMkcertLayout(t *testing.T) {
	root := useTestCA(t)
	if err := EnsureCA(); err != nil {
		t.Fatal(err)
	}

	ca := parseCertFile(t, filepath.Join(root, "rootCA.pem"))
	if !ca.IsCA || !ca.MaxPathLenZero || ca.KeyUsage != x509.KeyUsageCertSign {
		t.Errorf("CA constraints: IsCA=%v MaxPathLenZero=%v KeyUsage=%v", ca.IsCA, ca.MaxPathLenZero, ca.KeyUsage)
	}
	if got := ca.Subject.Organization; !slices.Equal(got, []string{"mkcert development CA"}) {
		t.Errorf("CA organization = %v, browser trust lookups key on mkcert's name", got)
	}
	block, _ := pem.Decode(readFileNoErr(filepath.Join(root, "rootCA-key.pem")))
	if block == nil || block.Type != "PRIVATE KEY" {
		t.Fatalf("CA key is not a PKCS#8 PEM block")
	}
	if info, _ := os.Stat(filepath.Join(root, "rootCA-key.pem")); info.Mode().Perm() != 0400 {
		t.Errorf("CA key mode = %v, want 0400", info.Mode().Perm())
	}
}

func TestEnsureCA_keepsAnExistingCA(t *testing.T) {
	root := useTestCA(t)
	if err := EnsureCA(); err != nil {
		t.Fatal(err)
	}
	before := readFileNoErr(filepath.Join(root, "rootCA.pem"))
	if err := EnsureCA(); err != nil {
		t.Fatal(err)
	}
	if string(readFileNoErr(filepath.Join(root, "rootCA.pem"))) != string(before) {
		t.Error("EnsureCA replaced a CA that was already there")
	}
}

func TestEnsureCA_refusesHalfACA(t *testing.T) {
	root := useTestCA(t)
	if err := os.WriteFile(filepath.Join(root, "rootCA.pem"), []byte("CA"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := EnsureCA(); err == nil {
		t.Fatal("EnsureCA generated over a CAROOT missing only its key")
	}
	if string(readFileNoErr(filepath.Join(root, "rootCA.pem"))) != "CA" {
		t.Error("EnsureCA touched the remaining CA file")
	}
}

func TestIssueCert_leafVerifiesAgainstTheCA(t *testing.T) {
	root := useTestCA(t)
	dir := t.TempDir()
	if err := IssueCertForce("site.test", []string{"site.test"}, dir); err != nil {
		t.Fatal(err)
	}

	pool := x509.NewCertPool()
	pool.AddCert(parseCertFile(t, filepath.Join(root, "rootCA.pem")))
	leaf := parseCertFile(t, filepath.Join(dir, "site.test.crt"))
	for _, host := range []string{"site.test", "feature.site.test"} {
		if _, err := leaf.Verify(x509.VerifyOptions{DNSName: host, Roots: pool}); err != nil {
			t.Errorf("leaf does not verify for %s: %v", host, err)
		}
	}
	// Apple platforms reject a server certificate valid for more than 825 days.
	if life := leaf.NotAfter.Sub(leaf.NotBefore); life > 825*24*time.Hour {
		t.Errorf("leaf lifetime %v exceeds 825 days", life)
	}
	if info, _ := os.Stat(filepath.Join(dir, "site.test.key")); info.Mode().Perm() != 0600 {
		t.Errorf("leaf key mode = %v, want 0600", info.Mode().Perm())
	}
}

// A CA generated before lerd issued certificates itself, by mkcert, has to keep
// signing: writing one independently stands in for that.
func TestIssueCert_signsWithACAItDidNotGenerate(t *testing.T) {
	root := useTestCA(t)
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	tpl := &x509.Certificate{
		SerialNumber:          big.NewInt(42),
		Subject:               pkix.Name{Organization: []string{"mkcert development CA"}},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().AddDate(10, 0, 0),
		KeyUsage:              x509.KeyUsageCertSign,
		BasicConstraintsValid: true,
		IsCA:                  true,
	}
	der, err := x509.CreateCertificate(rand.Reader, tpl, tpl, key.Public(), key)
	if err != nil {
		t.Fatal(err)
	}
	keyDER, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		t.Fatal(err)
	}
	os.WriteFile(filepath.Join(root, "rootCA.pem"), pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}), 0644)        //nolint:errcheck
	os.WriteFile(filepath.Join(root, "rootCA-key.pem"), pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: keyDER}), 0400) //nolint:errcheck

	dir := t.TempDir()
	if err := IssueCertForce("site.test", []string{"site.test"}, dir); err != nil {
		t.Fatal(err)
	}
	if string(readFileNoErr(filepath.Join(root, "rootCA.pem"))) != string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})) {
		t.Error("issuing replaced the existing CA")
	}
	pool := x509.NewCertPool()
	pool.AddCert(parseCertFile(t, filepath.Join(root, "rootCA.pem")))
	if _, err := parseCertFile(t, filepath.Join(dir, "site.test.crt")).Verify(x509.VerifyOptions{DNSName: "site.test", Roots: pool}); err != nil {
		t.Errorf("leaf not signed by the existing CA: %v", err)
	}
}
