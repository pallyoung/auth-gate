package certificate

import (
	"context"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/pallyoung/auth-gate/packages/server/internal/localca"
	"github.com/pallyoung/auth-gate/packages/server/internal/service/runtime"
	"github.com/pallyoung/auth-gate/packages/server/internal/store"
)

// Default durations for locally issued certificates.
const (
	defaultLeafDays = 90
	renewOffsetDays = 30
)

// Service handles certificate provisioning, import, and re-signing.
//
// Two sources are supported:
//   - SourceLocalCA: signed by the bundled local CA. Auto-renewed in the
//     background 30 days before NotAfter.
//   - SourceImported: pasted/uploaded PEM. Never auto-renewed; Resign
//     returns an error instructing the user to re-import.
type Service struct {
	db       store.Store
	reloader runtime.Reloader

	ca       *localca.CA
	certDir  string
	renewer  *Renewer
	mu       sync.Mutex
}

type Config struct {
	DataDir string
	CA      *localca.CA
}

func NewService(db store.Store, cfg Config, reloader runtime.Reloader) (*Service, error) {
	if cfg.CA == nil {
		return nil, fmt.Errorf("certificate service: local CA is required")
	}
	certDir := filepath.Join(cfg.DataDir, "certs")
	if err := os.MkdirAll(certDir, 0o755); err != nil {
		return nil, fmt.Errorf("create cert dir: %w", err)
	}
	return &Service{
		db:       db,
		reloader: reloader,
		ca:       cfg.CA,
		certDir:  certDir,
	}, nil
}

// ProvisionLocal signs a new certificate with the local CA. If info is
// non-nil its non-empty fields are embedded in the certificate Subject.
// Synchronous.
func (s *Service) ProvisionLocal(_ context.Context, name, domain string, days int, info *localca.SubjectInfo) (*store.Certificate, error) {
	var err error
	name, err = normalizeCertificateName(name)
	if err != nil {
		return nil, err
	}
	domain, err = normalizeCertificateDomain(domain)
	if err != nil {
		return nil, newError(ErrCodeInvalidDomain, err.Error(), nil)
	}

	if existing, err := s.db.GetCertificateByDomain(domain); err != nil {
		return nil, newError(ErrCodeDatabase, "check existing certificate", err)
	} else if existing != nil {
		return nil, newError(ErrCodeDomainExists, "certificate already exists for domain: "+domain, nil)
	}

	certPEM, keyPEM, nb, na, err := s.ca.SignCertificate(domain, days, info)
	if err != nil {
		return nil, newError(ErrCodeLocalCA, "sign certificate: "+err.Error(), err)
	}

	certPath, keyPath, err := s.writeCertFiles(domain, certPEM, keyPEM)
	if err != nil {
		return nil, err
	}

	cert := &store.Certificate{
		ID:        uuid.New().String(),
		Name:      name,
		Domain:    domain,
		CertPath:  certPath,
		KeyPath:   keyPath,
		Source:    store.SourceLocalCA,
		CAID:      s.caID(),
		Status:    store.CertStatusActive,
		NotBefore: nb,
		NotAfter:  na,
		RenewAt:   na.AddDate(0, 0, -renewOffsetDays),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	if info != nil {
		cert.Organization = info.Organization
		cert.OrganizationalUnit = info.OrganizationalUnit
		cert.Country = info.Country
		cert.Province = info.Province
		cert.Locality = info.Locality
	}
	if err := s.db.CreateCertificate(cert); err != nil {
		s.removeCertFiles(certPath, keyPath)
		return nil, newError(ErrCodeDatabase, "save certificate", err)
	}
	s.triggerReload()
	return cert, nil
}

// Import stores a user-supplied certificate and key after validating them.
func (s *Service) Import(_ context.Context, name, domain, certPEM, keyPEM string) (*store.Certificate, error) {
	var err error
	name, err = normalizeCertificateName(name)
	if err != nil {
		return nil, err
	}
	domain, err = normalizeCertificateDomain(domain)
	if err != nil {
		return nil, newError(ErrCodeInvalidDomain, err.Error(), nil)
	}

	leaf, err := parseLeafCertificate([]byte(certPEM))
	if err != nil {
		return nil, newError(ErrCodeInvalidPEM, "parse certificate: "+err.Error(), err)
	}
	if err := validateKeyMatchesCertificate([]byte(certPEM), []byte(keyPEM)); err != nil {
		return nil, newError(ErrCodeInvalidPEM, "key does not match certificate: "+err.Error(), err)
	}
	if err := validateDomainMatchesCertificate(domain, leaf); err != nil {
		return nil, newError(ErrCodeDomainMismatch, err.Error(), err)
	}

	if existing, err := s.db.GetCertificateByDomain(domain); err != nil {
		return nil, newError(ErrCodeDatabase, "check existing certificate", err)
	} else if existing != nil {
		return nil, newError(ErrCodeDomainExists, "certificate already exists for domain: "+domain, nil)
	}

	certPath, keyPath, err := s.writeCertFiles(domain, []byte(certPEM), []byte(keyPEM))
	if err != nil {
		return nil, err
	}

	cert := &store.Certificate{
		ID:        uuid.New().String(),
		Name:      name,
		Domain:    domain,
		CertPath:  certPath,
		KeyPath:   keyPath,
		Source:    store.SourceImported,
		Status:    store.CertStatusActive,
		NotBefore: leaf.NotBefore,
		NotAfter:  leaf.NotAfter,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	if err := s.db.CreateCertificate(cert); err != nil {
		s.removeCertFiles(certPath, keyPath)
		return nil, newError(ErrCodeDatabase, "save certificate", err)
	}
	s.triggerReload()
	return cert, nil
}

// Resign re-issues a local-CA certificate with the same domain. Imported
// certificates must be re-imported by the user instead.
func (s *Service) Resign(id string) (*store.Certificate, error) {
	cert, err := s.db.GetCertificate(id)
	if err != nil {
		return nil, newError(ErrCodeDatabase, "get certificate", err)
	}
	if cert == nil {
		return nil, newError(ErrCodeCertNotFound, "certificate not found: "+id, nil)
	}
	if cert.Source != store.SourceLocalCA {
		return nil, newError(ErrCodeImportedCannotResign,
			"imported certificates cannot be auto-renewed; re-import the certificate instead", nil)
	}

	// Rebuild subject info from stored fields so re-signed certs preserve org.
	var info *localca.SubjectInfo
	if cert.Organization != "" || cert.OrganizationalUnit != "" || cert.Country != "" || cert.Province != "" || cert.Locality != "" {
		info = &localca.SubjectInfo{
			Organization:       cert.Organization,
			OrganizationalUnit: cert.OrganizationalUnit,
			Country:            cert.Country,
			Province:           cert.Province,
			Locality:           cert.Locality,
		}
	}
	certPEM, keyPEM, nb, na, err := s.ca.SignCertificate(cert.Domain, defaultLeafDays, info)
	if err != nil {
		return nil, newError(ErrCodeLocalCA, "re-sign certificate: "+err.Error(), err)
	}
	if _, _, err := s.writeCertFiles(cert.Domain, certPEM, keyPEM); err != nil {
		return nil, err
	}
	cert.NotBefore = nb
	cert.NotAfter = na
	cert.RenewAt = na.AddDate(0, 0, -renewOffsetDays)
	cert.Status = store.CertStatusActive
	if err := s.db.UpdateCertificate(cert); err != nil {
		return nil, newError(ErrCodeDatabase, "update certificate", err)
	}
	s.triggerReload()
	return cert, nil
}

// List returns all certificates.
func (s *Service) List() ([]store.Certificate, error) {
	certs, err := s.db.ListCertificates()
	if err != nil {
		return nil, newError(ErrCodeDatabase, "list certificates", err)
	}
	return certs, nil
}

// Get returns a single certificate by ID.
func (s *Service) Get(id string) (*store.Certificate, error) {
	cert, err := s.db.GetCertificate(id)
	if err != nil {
		return nil, newError(ErrCodeDatabase, "get certificate", err)
	}
	return cert, nil
}

// GetCAExport returns the CA cert PEM plus identifying metadata for the
// /api/ca endpoint.
func (s *Service) GetCAExport() (certPEM string, name string, notAfter time.Time, err error) {
	if s.ca == nil {
		return "", "", time.Time{}, newError(ErrCodeLocalCA, "no local CA loaded", nil)
	}
	return string(s.ca.CertPEM), s.ca.Cert.Subject.CommonName, s.ca.Cert.NotAfter, nil
}

// Delete removes a certificate and its files.
func (s *Service) Delete(id string) error {
	cert, err := s.db.GetCertificate(id)
	if err != nil {
		return newError(ErrCodeDatabase, "get certificate", err)
	}
	if cert == nil {
		return newError(ErrCodeCertNotFound, "certificate not found: "+id, nil)
	}
	if cert.CertPath != "" {
		_ = os.Remove(cert.CertPath)
	}
	if cert.KeyPath != "" {
		_ = os.Remove(cert.KeyPath)
	}
	if err := s.db.DeleteCertificate(id); err != nil {
		return newError(ErrCodeDatabase, "delete certificate", err)
	}
	s.triggerReload()
	return nil
}

// StartRenewer begins the background re-signer that handles local-CA
// certificates approaching expiration.
func (s *Service) StartRenewer(interval time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.renewer != nil {
		return
	}
	s.renewer = &Renewer{svc: s, interval: interval}
	go s.renewer.run()
}

// StopRenewer stops the background re-signer. Safe to call multiple times.
func (s *Service) StopRenewer() {
	s.mu.Lock()
	r := s.renewer
	s.renewer = nil
	s.mu.Unlock()
	if r != nil {
		r.stop()
	}
}

// Internal: file handling and reloader plumbing

func (s *Service) writeCertFiles(domain string, certPEM, keyPEM []byte) (certPath, keyPath string, err error) {
	dir := filepath.Join(s.certDir, normalizeDomainForPath(domain))
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", "", newError(ErrCodeFilesystem, "create cert dir: "+err.Error(), err)
	}
	certPath = filepath.Join(dir, "cert.pem")
	keyPath = filepath.Join(dir, "key.pem")
	if err := os.WriteFile(certPath, certPEM, 0o644); err != nil {
		return "", "", newError(ErrCodeFilesystem, "write cert: "+err.Error(), err)
	}
	if err := os.WriteFile(keyPath, keyPEM, 0o600); err != nil {
		_ = os.Remove(certPath)
		return "", "", newError(ErrCodeFilesystem, "write key: "+err.Error(), err)
	}
	return certPath, keyPath, nil
}

func (s *Service) removeCertFiles(certPath, keyPath string) {
	if certPath != "" {
		_ = os.Remove(certPath)
	}
	if keyPath != "" {
		_ = os.Remove(keyPath)
	}
}

func (s *Service) triggerReload() {
	if s.reloader == nil {
		return
	}
	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("[cert] reloader panic: %v", r)
			}
		}()
		s.reloader.Reload()
	}()
}

func (s *Service) caID() string {
	ca, err := s.db.GetFirstCACertificate()
	if err != nil || ca == nil {
		return ""
	}
	return ca.ID
}

// Helpers used by tests and other internal packages.

func parseLeafCertificate(pemBytes []byte) (*x509.Certificate, error) {
	block, _ := pem.Decode(pemBytes)
	if block == nil {
		return nil, fmt.Errorf("invalid PEM block")
	}
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("parse x509: %w", err)
	}
	if cert.IsCA {
		return nil, fmt.Errorf("certificate is a CA, not a leaf")
	}
	return cert, nil
}

func normalizeCertificateName(name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", newError(ErrCodeInvalidName, "certificate name is required", nil)
	}
	return name, nil
}

func normalizeCertificateDomain(domain string) (string, error) {
	domain = strings.TrimSpace(domain)
	if domain == "" {
		return "", fmt.Errorf("domain is required")
	}
	if !looksLikeDomain(domain) {
		return "", fmt.Errorf("invalid domain format: %s", domain)
	}
	return domain, nil
}

func looksLikeDomain(domain string) bool {
	if len(domain) < 4 {
		return false
	}
	stripped := strings.TrimPrefix(domain, "*.")
	if !strings.Contains(stripped, ".") {
		return false
	}
	for _, r := range stripped {
		if r == '.' || r == '-' || (r >= '0' && r <= '9') || (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') {
			continue
		}
		return false
	}
	return true
}
