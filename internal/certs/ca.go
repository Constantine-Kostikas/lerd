package certs

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha1" // the SubjectKeyId hash RFC 5280 names, not a security boundary
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/asn1"
	"encoding/pem"
	"errors"
	"fmt"
	"math/big"
	"net"
	"os"
	"os/user"
	"path/filepath"
	"time"
)

// The CA and site certificates lerd issues itself, in mkcert's layout and
// format: the same CAROOT, file names, key encoding and subject names, so a CA
// either one generated keeps signing for the other and nothing gets retrusted.

const (
	caCertName = "rootCA.pem"
	caKeyName  = "rootCA-key.pem"
)

// userAndHostname is the OU every certificate carries, as mkcert writes it.
func userAndHostname() string {
	var s string
	u, err := user.Current()
	if err == nil {
		s = u.Username + "@"
	}
	if h, err := os.Hostname(); err == nil {
		s += h
	}
	if err == nil && u.Name != "" && u.Name != u.Username {
		s += " (" + u.Name + ")"
	}
	return s
}

func randomSerial() (*big.Int, error) {
	return rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
}

// EnsureCA generates the root CA unless CAROOT already holds one. A CAROOT
// with only one of the two files is refused: generating over it would orphan
// every certificate the missing half signed or verified.
func EnsureCA() error {
	root, err := CARoot()
	if err != nil {
		return fmt.Errorf("resolving CAROOT: %w", err)
	}
	_, certErr := os.Stat(filepath.Join(root, caCertName))
	_, keyErr := os.Stat(filepath.Join(root, caKeyName))
	switch {
	case certErr == nil && keyErr == nil:
		return nil
	case certErr == nil:
		return fmt.Errorf("%s has %s but no %s", root, caCertName, caKeyName)
	case keyErr == nil:
		return fmt.Errorf("%s has %s but no %s", root, caKeyName, caCertName)
	}
	return newCA(root)
}

func newCA(root string) error {
	key, err := rsa.GenerateKey(rand.Reader, 3072)
	if err != nil {
		return err
	}
	spki, err := x509.MarshalPKIXPublicKey(key.Public())
	if err != nil {
		return err
	}
	var parsed struct {
		Algorithm pkix.AlgorithmIdentifier
		PublicKey asn1.BitString
	}
	if _, err := asn1.Unmarshal(spki, &parsed); err != nil {
		return err
	}
	skid := sha1.Sum(parsed.PublicKey.Bytes)
	serial, err := randomSerial()
	if err != nil {
		return err
	}
	owner := userAndHostname()
	tpl := &x509.Certificate{
		SerialNumber: serial,
		Subject: pkix.Name{
			Organization:       []string{"mkcert development CA"},
			OrganizationalUnit: []string{owner},
			CommonName:         "mkcert " + owner,
		},
		SubjectKeyId:          skid[:],
		NotBefore:             time.Now(),
		NotAfter:              time.Now().AddDate(10, 0, 0),
		KeyUsage:              x509.KeyUsageCertSign,
		BasicConstraintsValid: true,
		IsCA:                  true,
		MaxPathLenZero:        true,
	}
	der, err := x509.CreateCertificate(rand.Reader, tpl, tpl, key.Public(), key)
	if err != nil {
		return err
	}
	keyDER, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(root, 0755); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(root, caKeyName), pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: keyDER}), 0400); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(root, caCertName), pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}), 0644)
}

func loadCA() (*x509.Certificate, crypto.Signer, error) {
	root, err := CARoot()
	if err != nil {
		return nil, nil, fmt.Errorf("resolving CAROOT: %w", err)
	}
	der := firstCertDER(readFileNoErr(filepath.Join(root, caCertName)))
	if der == nil {
		return nil, nil, fmt.Errorf("no CA certificate in %s", filepath.Join(root, caCertName))
	}
	cert, err := x509.ParseCertificate(der)
	if err != nil {
		return nil, nil, err
	}
	block, _ := pem.Decode(readFileNoErr(filepath.Join(root, caKeyName)))
	if block == nil || block.Type != "PRIVATE KEY" {
		return nil, nil, fmt.Errorf("no PKCS#8 private key in %s", filepath.Join(root, caKeyName))
	}
	key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, nil, err
	}
	signer, ok := key.(crypto.Signer)
	if !ok {
		return nil, nil, errors.New("CA key cannot sign")
	}
	return cert, signer, nil
}

// writeLeaf signs a server certificate for hosts with the root CA and writes
// it with its key. Lifetime and usages are mkcert's: 27 months stays under
// the 825 days Apple platforms accept for a server certificate.
func writeLeaf(hosts []string, certFile, keyFile string) error {
	caCert, caKey, err := loadCA()
	if err != nil {
		return err
	}
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return err
	}
	serial, err := randomSerial()
	if err != nil {
		return err
	}
	tpl := &x509.Certificate{
		SerialNumber: serial,
		Subject: pkix.Name{
			Organization:       []string{"mkcert development certificate"},
			OrganizationalUnit: []string{userAndHostname()},
		},
		NotBefore:   time.Now(),
		NotAfter:    time.Now().AddDate(0, 27, 0),
		KeyUsage:    x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}
	for _, h := range hosts {
		if ip := net.ParseIP(h); ip != nil {
			tpl.IPAddresses = append(tpl.IPAddresses, ip)
		} else {
			tpl.DNSNames = append(tpl.DNSNames, h)
		}
	}
	der, err := x509.CreateCertificate(rand.Reader, tpl, caCert, key.Public(), caKey)
	if err != nil {
		return err
	}
	keyDER, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		return err
	}
	if err := os.WriteFile(keyFile, pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: keyDER}), 0600); err != nil {
		return err
	}
	return os.WriteFile(certFile, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}), 0644)
}
