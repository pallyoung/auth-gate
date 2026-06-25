package localca

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"time"
)

// SubjectInfo carries optional X.509 subject fields for leaf certificates.
// When a field is empty the corresponding pkix.Name attribute is omitted.
type SubjectInfo struct {
	Organization       string
	OrganizationalUnit string
	Country            string
	Province           string
	Locality           string
}

// CA is a self-signed certificate authority stored on disk.
// Use LoadOrCreate to obtain one rooted at the data directory.
type CA struct {
	Cert    *x509.Certificate
	Key     *rsa.PrivateKey
	CertPEM []byte
	KeyPEM  []byte
}

const (
	defaultCAName      = "Auth Gate Local CA"
	defaultCAValidity  = 100 * 365 * 24 * time.Hour
	defaultCAKeySize   = 2048
	caCertFilename     = "ca.crt"
	caKeyFilename      = "ca.key"
	caSubDir           = "ca"
	defaultLeafDays    = 90
	leafKeySize        = 2048
)

// LoadOrCreate returns the CA stored under <dataDir>/ca/, generating it
// (self-signed, 100-year) on first run. The CA key is written with 0600.
func LoadOrCreate(dataDir string) (*CA, error) {
	dir := filepath.Join(dataDir, caSubDir)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("create ca dir: %w", err)
	}

	certPath := filepath.Join(dir, caCertFilename)
	keyPath := filepath.Join(dir, caKeyFilename)

	if certPEM, err := os.ReadFile(certPath); err == nil {
		if keyPEM, kerr := os.ReadFile(keyPath); kerr == nil {
			return parseCA(certPEM, keyPEM)
		}
	}

	return generateAndPersist(dir, certPath, keyPath)
}

func generateAndPersist(dir, certPath, keyPath string) (*CA, error) {
	key, err := rsa.GenerateKey(rand.Reader, defaultCAKeySize)
	if err != nil {
		return nil, fmt.Errorf("generate ca key: %w", err)
	}

	serial, err := randomSerial()
	if err != nil {
		return nil, err
	}

	now := time.Now()
	tmpl := &x509.Certificate{
		SerialNumber: serial,
		Subject: pkix.Name{
			CommonName:   defaultCAName,
			Organization: []string{"Auth Gate"},
		},
		NotBefore:             now.Add(-time.Hour),
		NotAfter:              now.Add(defaultCAValidity),
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
		BasicConstraintsValid: true,
		IsCA:                  true,
		MaxPathLen:            1,
	}

	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		return nil, fmt.Errorf("self-sign ca: %w", err)
	}
	cert, err := x509.ParseCertificate(der)
	if err != nil {
		return nil, fmt.Errorf("parse ca cert: %w", err)
	}

	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
	keyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(key),
	})

	if err := os.WriteFile(certPath, certPEM, 0o644); err != nil {
		return nil, fmt.Errorf("write ca cert: %w", err)
	}
	if err := os.WriteFile(keyPath, keyPEM, 0o600); err != nil {
		return nil, fmt.Errorf("write ca key: %w", err)
	}

	return &CA{Cert: cert, Key: key, CertPEM: certPEM, KeyPEM: keyPEM}, nil
}

func parseCA(certPEM, keyPEM []byte) (*CA, error) {
	cb, _ := pem.Decode(certPEM)
	if cb == nil || cb.Type != "CERTIFICATE" {
		return nil, fmt.Errorf("ca: invalid certificate PEM")
	}
	cert, err := x509.ParseCertificate(cb.Bytes)
	if err != nil {
		return nil, fmt.Errorf("ca: parse certificate: %w", err)
	}
	if !cert.IsCA {
		return nil, fmt.Errorf("ca: certificate is not a CA")
	}

	kb, _ := pem.Decode(keyPEM)
	if kb == nil || kb.Type != "RSA PRIVATE KEY" {
		return nil, fmt.Errorf("ca: invalid key PEM")
	}
	key, err := x509.ParsePKCS1PrivateKey(kb.Bytes)
	if err != nil {
		return nil, fmt.Errorf("ca: parse key: %w", err)
	}

	return &CA{Cert: cert, Key: key, CertPEM: certPEM, KeyPEM: keyPEM}, nil
}

// SignCertificate issues a leaf certificate for the given domain, valid for
// the given number of days. Returns PEM-encoded cert and key, plus NotBefore
// and NotAfter. If info is non-nil its non-empty fields are added to the
// certificate Subject.
func (c *CA) SignCertificate(domain string, days int, info *SubjectInfo) (certPEM, keyPEM []byte, notBefore, notAfter time.Time, err error) {
	if domain == "" {
		return nil, nil, time.Time{}, time.Time{}, fmt.Errorf("domain is required")
	}
	if days <= 0 {
		days = defaultLeafDays
	}

	key, err := rsa.GenerateKey(rand.Reader, leafKeySize)
	if err != nil {
		return nil, nil, time.Time{}, time.Time{}, fmt.Errorf("generate leaf key: %w", err)
	}

	serial, err := randomSerial()
	if err != nil {
		return nil, nil, time.Time{}, time.Time{}, err
	}

	now := time.Now()
	notBefore = now.Add(-time.Hour)
	notAfter = now.Add(time.Duration(days) * 24 * time.Hour)

	subject := pkix.Name{
		CommonName: domain,
	}
	if info != nil {
		if info.Organization != "" {
			subject.Organization = []string{info.Organization}
		}
		if info.OrganizationalUnit != "" {
			subject.OrganizationalUnit = []string{info.OrganizationalUnit}
		}
		if info.Country != "" {
			subject.Country = []string{info.Country}
		}
		if info.Province != "" {
			subject.Province = []string{info.Province}
		}
		if info.Locality != "" {
			subject.Locality = []string{info.Locality}
		}
	}

	tmpl := &x509.Certificate{
		SerialNumber: serial,
		Subject:      subject,
		NotBefore:   notBefore,
		NotAfter:    notAfter,
		KeyUsage:    x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		DNSNames:    []string{domain},
	}

	der, err := x509.CreateCertificate(rand.Reader, tmpl, c.Cert, &key.PublicKey, c.Key)
	if err != nil {
		return nil, nil, time.Time{}, time.Time{}, fmt.Errorf("sign leaf: %w", err)
	}

	certPEM = pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
	keyPEM = pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(key),
	})
	return certPEM, keyPEM, notBefore, notAfter, nil
}

func randomSerial() (*big.Int, error) {
	limit := new(big.Int).Lsh(big.NewInt(1), 128)
	s, err := rand.Int(rand.Reader, limit)
	if err != nil {
		return nil, fmt.Errorf("random serial: %w", err)
	}
	return s, nil
}
