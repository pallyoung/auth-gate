package localca

import (
	"crypto/x509"
	"encoding/pem"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLoadOrCreateGeneratesAndReloads(t *testing.T) {
	dir := t.TempDir()

	first, err := LoadOrCreate(dir)
	if err != nil {
		t.Fatalf("first LoadOrCreate: %v", err)
	}
	if first.Cert == nil || first.Key == nil {
		t.Fatal("expected cert and key")
	}
	if !first.Cert.IsCA {
		t.Fatal("expected CA cert")
	}
	if _, err := os.Stat(filepath.Join(dir, caSubDir, caCertFilename)); err != nil {
		t.Fatalf("ca.crt should exist: %v", err)
	}

	second, err := LoadOrCreate(dir)
	if err != nil {
		t.Fatalf("second LoadOrCreate: %v", err)
	}
	if first.Cert.SerialNumber.Cmp(second.Cert.SerialNumber) != 0 {
		t.Fatal("expected same CA on second load")
	}
}

func TestSignCertificateProducesValidLeaf(t *testing.T) {
	dir := t.TempDir()
	ca, err := LoadOrCreate(dir)
	if err != nil {
		t.Fatalf("LoadOrCreate: %v", err)
	}

	domain := "*.example.com"
	certPEM, keyPEM, nb, na, err := ca.SignCertificate(domain, 90)
	if err != nil {
		t.Fatalf("SignCertificate: %v", err)
	}

	if nb.IsZero() || na.IsZero() {
		t.Fatal("expected non-zero validity times")
	}
	if na.Sub(nb) < 89*24*time.Hour {
		t.Fatalf("expected ~90 day validity, got %v", na.Sub(nb))
	}

	cb, _ := pem.Decode(certPEM)
	if cb == nil {
		t.Fatal("cert PEM decode failed")
	}
	leaf, err := x509.ParseCertificate(cb.Bytes)
	if err != nil {
		t.Fatalf("parse leaf: %v", err)
	}
	if leaf.Subject.CommonName != domain {
		t.Fatalf("CN: got %q, want %q", leaf.Subject.CommonName, domain)
	}
	if len(leaf.DNSNames) != 1 || leaf.DNSNames[0] != domain {
		t.Fatalf("SAN: got %v, want [%q]", leaf.DNSNames, domain)
	}

	kb, _ := pem.Decode(keyPEM)
	if kb == nil || kb.Type != "RSA PRIVATE KEY" {
		t.Fatal("key PEM decode failed")
	}
	if _, err := x509.ParsePKCS1PrivateKey(kb.Bytes); err != nil {
		t.Fatalf("parse key: %v", err)
	}

	pool := x509.NewCertPool()
	pool.AddCert(ca.Cert)
	if _, err := leaf.Verify(x509.VerifyOptions{
		DNSName: "foo.example.com",
		Roots:   pool,
	}); err != nil {
		t.Fatalf("leaf should verify under CA: %v", err)
	}
}

func TestSignCertificateRejectsEmptyDomain(t *testing.T) {
	dir := t.TempDir()
	ca, _ := LoadOrCreate(dir)
	if _, _, _, _, err := ca.SignCertificate("", 30); err == nil {
		t.Fatal("expected error for empty domain")
	}
}
