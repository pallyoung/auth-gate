package certificate

import (
	"crypto"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"strings"

	"github.com/pallyoung/auth-gate/packages/server/internal/service/runtime"
)

// normalizeDomainForPath turns a domain (which may start with "*.") into a
// filesystem-safe directory name.
func normalizeDomainForPath(domain string) string {
	return "_wildcard_" + strings.ReplaceAll(domain, "*.", "")
}

// parsePrivateKey decodes a PEM RSA private key.
func parsePrivateKey(pemBytes []byte) (crypto.PrivateKey, error) {
	block, _ := pem.Decode(pemBytes)
	if block == nil {
		return nil, fmt.Errorf("invalid PEM block")
	}
	if key, err := x509.ParsePKCS1PrivateKey(block.Bytes); err == nil {
		return key, nil
	}
	if key, err := x509.ParsePKCS8PrivateKey(block.Bytes); err == nil {
		if rsaKey, ok := key.(*rsa.PrivateKey); ok {
			return rsaKey, nil
		}
		return nil, fmt.Errorf("PKCS8 key is not RSA")
	}
	return nil, fmt.Errorf("could not parse private key (tried PKCS1 and PKCS8)")
}

// validateKeyMatchesCertificate checks that the private key and certificate
// form a matching pair by comparing their public key modulus.
func validateKeyMatchesCertificate(certPEM, keyPEM []byte) error {
	cb, _ := pem.Decode(certPEM)
	if cb == nil {
		return fmt.Errorf("invalid certificate PEM")
	}
	cert, err := x509.ParseCertificate(cb.Bytes)
	if err != nil {
		return fmt.Errorf("parse certificate: %w", err)
	}
	key, err := parsePrivateKey(keyPEM)
	if err != nil {
		return err
	}
	rsaKey, ok := key.(*rsa.PrivateKey)
	if !ok {
		return fmt.Errorf("private key is not RSA")
	}
	certPub, ok := cert.PublicKey.(*rsa.PublicKey)
	if !ok {
		return fmt.Errorf("certificate public key is not RSA")
	}
	if certPub.N.Cmp(rsaKey.N) != 0 {
		return fmt.Errorf("certificate and private key do not match")
	}
	return nil
}

// validateDomainMatchesCertificate ensures the requested domain is the
// certificate's CN or appears in its SANs (with wildcard matching).
func validateDomainMatchesCertificate(domain string, cert *x509.Certificate) error {
	if cert.Subject.CommonName == domain {
		return nil
	}
	for _, san := range cert.DNSNames {
		if matchDomain(san, domain) {
			return nil
		}
	}
	return fmt.Errorf("domain %q is not covered by certificate (CN=%q, SANs=%v)",
		domain, cert.Subject.CommonName, cert.DNSNames)
}

// matchDomain handles wildcard SAN matching per RFC 6125: a certificate with
// "*.example.com" matches "foo.example.com" but not "example.com" or
// "a.b.example.com".
func matchDomain(certName, requested string) bool {
	if certName == requested {
		return true
	}
	if !strings.HasPrefix(certName, "*.") {
		return false
	}
	base := strings.TrimPrefix(certName, "*.")
	if !strings.HasSuffix(requested, "."+base) {
		return false
	}
	prefix := strings.TrimSuffix(requested, "."+base)
	if prefix == "" {
		return false
	}
	if strings.Contains(prefix, ".") {
		return false
	}
	return true
}

var _ = runtime.Reloader(nil)
var _ = errors.New
