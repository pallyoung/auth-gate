package certificate

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/go-acme/lego/v4/challenge"
	"github.com/google/uuid"
	"github.com/pallyoung/auth-gate/packages/server/internal/acme"
	"github.com/pallyoung/auth-gate/packages/server/internal/store"
	"github.com/pallyoung/auth-gate/packages/server/internal/service/runtime"
)

// Service handles certificate provisioning and renewal
type Service struct {
	db        *store.SQLite
	reloader  runtime.Reloader
	acmeDir   string
	acmeEmail string
	acme      *acme.Client
	mu        sync.RWMutex

	// Renewal
	renewer   *Renewer
	stopCh    chan struct{}
}

// Config holds service configuration
type Config struct {
	DataDir   string // Base directory for ACME data and certificates
	ACMEEmail string // Email for ACME account
	UseStaging bool  // Use Let's Encrypt staging (for testing)
}

// NewService creates a new certificate service
func NewService(db *store.SQLite, cfg Config, reloader runtime.Reloader) (*Service, error) {
	acmeDir := filepath.Join(cfg.DataDir, "acme")

	svc := &Service{
		db:        db,
		reloader:  reloader,
		acmeDir:   acmeDir,
		acmeEmail: cfg.ACMEEmail,
		stopCh:    make(chan struct{}),
	}

	// Initialize ACME client
	client, err := acme.NewClient(acme.Config{
		Email:       cfg.ACMEEmail,
		DataDir:     acmeDir,
		AcceptTerms: true,
		UseStaging:  cfg.UseStaging,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create ACME client: %w", err)
	}
	svc.acme = client

	return svc, nil
}

// ProvisionInput holds input for certificate provisioning
type ProvisionInput struct {
	Name       string
	Domain     string // e.g., "*.example.com" or "example.com"
	DNSProvider acme.DNSProviderConfig
}

// Provision creates a new certificate for the given domain
func (s *Service) Provision(_ context.Context, input ProvisionInput) (*store.Certificate, error) {
	// Validate domain
	if err := validateDomain(input.Domain); err != nil {
		return nil, newError(ErrCodeInvalidDomain, err.Error(), nil)
	}

	// Check if certificate already exists for this domain
	existing, err := s.db.GetCertificateByDomain(input.Domain)
	if err != nil {
		return nil, newError(ErrCodeDatabase, "failed to check existing certificate", err)
	}
	if existing != nil {
		return nil, newError(ErrCodeDomainExists, "certificate already exists for domain: "+input.Domain, nil)
	}

	// Create certificate record
	cert := &store.Certificate{
		ID:               uuid.New().String(),
		Name:             input.Name,
		Domain:           input.Domain,
		Status:           store.CertStatusPending,
		DNSProvider:      input.DNSProvider.ProviderType,
		DNSProviderConfig: encryptProviderConfig(input.DNSProvider),
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}

	// Save to database
	if err := s.db.CreateCertificate(cert); err != nil {
		return nil, newError(ErrCodeDatabase, "failed to create certificate record", err)
	}

	// Provision in background
	go s.provisionCertificate(cert.ID, input)

	return cert, nil
}

// provisionCertificate does the actual certificate provisioning
func (s *Service) provisionCertificate(certID string, input ProvisionInput) {
	cert, err := s.db.GetCertificate(certID)
	if err != nil || cert == nil {
		log.Printf("[cert] failed to get certificate %s: %v", certID, err)
		return
	}

	// Create DNS provider
	provider, err := acme.NewDNSProvider(input.DNSProvider)
	if err != nil {
		log.Printf("[cert] failed to create DNS provider for %s: %v", input.Domain, err)
		s.updateStatus(certID, store.CertStatusFailed, "DNS provider error: "+err.Error())
		return
	}

	// Request certificate
	domains := parseDomain(input.Domain)
	certPEM, keyPEM, err := s.acme.RequestCertificate(domains, provider)
	if err != nil {
		log.Printf("[cert] failed to request certificate for %s: %v", input.Domain, err)
		s.updateStatus(certID, store.CertStatusFailed, "ACME error: "+err.Error())
		return
	}

	// Save certificate to filesystem
	certPath, keyPath, err := s.acme.SaveCertificate(normalizeDomain(input.Domain), certPEM, keyPEM)
	if err != nil {
		log.Printf("[cert] failed to save certificate for %s: %v", input.Domain, err)
		s.updateStatus(certID, store.CertStatusFailed, "Failed to save certificate: "+err.Error())
		return
	}

	// Update certificate record
	cert.CertPath = certPath
	cert.KeyPath = keyPath
	cert.Status = store.CertStatusActive

	// Parse certificate to get expiration dates
	if x509Cert, err := parseCertPEM(certPEM); err == nil {
		cert.NotBefore = x509Cert.NotBefore
		cert.NotAfter = x509Cert.NotAfter
		cert.RenewAt = cert.NotAfter.AddDate(0, 0, -30) // Renew 30 days before expiry
	} else {
		cert.NotAfter = time.Now().AddDate(0, 0, 90) // Default 90 days
		cert.RenewAt = cert.NotAfter.AddDate(0, 0, -30)
	}

	if err := s.db.UpdateCertificate(cert); err != nil {
		log.Printf("[cert] failed to update certificate record %s: %v", certID, err)
		return
	}

	// Trigger route reload
	if s.reloader != nil {
		s.reloader.Reload()
	}

	log.Printf("[cert] successfully provisioned certificate for %s, expires at %s", input.Domain, cert.NotAfter.Format(time.RFC3339))
}

// Renew renews an existing certificate
func (s *Service) Renew(id string) error {
	cert, err := s.db.GetCertificate(id)
	if err != nil {
		return newError(ErrCodeDatabase, "failed to get certificate", err)
	}
	if cert == nil {
		return newError(ErrCodeCertNotFound, "certificate not found: "+id, nil)
	}

	// Create DNS provider from stored config
	providerConfig := decryptProviderConfig(cert.DNSProviderConfig)
	provider, err := acme.NewDNSProvider(providerConfig)
	if err != nil {
		return newError(ErrCodeDNSProvider, "failed to create DNS provider", err)
	}

	// Update status
	cert.Status = store.CertStatusRenewing
	if err := s.db.UpdateCertificate(cert); err != nil {
		return newError(ErrCodeDatabase, "failed to update certificate status", err)
	}

	// Renew in background
	go func() {
		s.renewCertificate(cert, provider)
	}()

	return nil
}

// renewCertificate performs the actual renewal
func (s *Service) renewCertificate(cert *store.Certificate, provider challenge.Provider) {
	domains := parseDomain(cert.Domain)

	certPEM, keyPEM, err := s.acme.RequestCertificate(domains, provider)
	if err != nil {
		log.Printf("[cert] failed to renew certificate for %s: %v", cert.Domain, err)
		s.updateStatus(cert.ID, store.CertStatusFailed, "Renewal error: "+err.Error())
		return
	}

	// Save certificate to filesystem
	certPath, keyPath, err := s.acme.SaveCertificate(normalizeDomain(cert.Domain), certPEM, keyPEM)
	if err != nil {
		log.Printf("[cert] failed to save renewed certificate for %s: %v", cert.Domain, err)
		s.updateStatus(cert.ID, store.CertStatusFailed, "Failed to save certificate: "+err.Error())
		return
	}

	// Update certificate record
	cert.CertPath = certPath
	cert.KeyPath = keyPath
	cert.Status = store.CertStatusActive

	// Parse certificate to get expiration dates
	if x509Cert, err := parseCertPEM(certPEM); err == nil {
		cert.NotBefore = x509Cert.NotBefore
		cert.NotAfter = x509Cert.NotAfter
		cert.RenewAt = cert.NotAfter.AddDate(0, 0, -30)
	}

	if err := s.db.UpdateCertificate(cert); err != nil {
		log.Printf("[cert] failed to update certificate record %s: %v", cert.ID, err)
		return
	}

	// Trigger route reload
	if s.reloader != nil {
		s.reloader.Reload()
	}

	log.Printf("[cert] successfully renewed certificate for %s, expires at %s", cert.Domain, cert.NotAfter.Format(time.RFC3339))
}

// StartRenewer starts the background renewal checker
func (s *Service) StartRenewer(interval time.Duration) {
	s.renewer = &Renewer{svc: s, interval: interval}
	go s.renewer.Start()
}

// StopRenewer stops the background renewal checker
func (s *Service) StopRenewer() {
	if s.renewer != nil {
		s.renewer.Stop()
	}
}

// List returns all certificates
func (s *Service) List() ([]store.Certificate, error) {
	return s.db.ListCertificates()
}

// Get returns a certificate by ID
func (s *Service) Get(id string) (*store.Certificate, error) {
	return s.db.GetCertificate(id)
}

// Delete removes a certificate
func (s *Service) Delete(id string) error {
	cert, err := s.db.GetCertificate(id)
	if err != nil {
		return newError(ErrCodeDatabase, "failed to get certificate", err)
	}
	if cert == nil {
		return newError(ErrCodeCertNotFound, "certificate not found: "+id, nil)
	}

	// Delete certificate files
	if cert.CertPath != "" {
		os.Remove(cert.CertPath)
	}
	if cert.KeyPath != "" {
		os.Remove(cert.KeyPath)
	}

	// Delete from database
	if err := s.db.DeleteCertificate(id); err != nil {
		return newError(ErrCodeDatabase, "failed to delete certificate", err)
	}

	return nil
}

// Helper functions

func (s *Service) updateStatus(id string, status string, message string) {
	cert, err := s.db.GetCertificate(id)
	if err != nil || cert == nil {
		return
	}
	cert.Status = status
	s.db.UpdateCertificate(cert)
	if message != "" {
		log.Printf("[cert] %s: %s", status, message)
	}
}

func validateDomain(domain string) error {
	if domain == "" {
		return fmt.Errorf("domain is required")
	}

	// Basic validation: must be a valid domain format
	domain = strings.TrimPrefix(domain, "*.")

	// Check for basic format (at least one dot for multi-level domain)
	if len(domain) < 4 || strings.Count(domain, ".") < 1 {
		return fmt.Errorf("invalid domain format: %s", domain)
	}

	return nil
}

func parseDomain(domain string) []string {
	domains := []string{domain}

	// For wildcard, also include the base domain for validation
	if strings.HasPrefix(domain, "*.") {
		baseDomain := strings.TrimPrefix(domain, "*.")
		domains = append(domains, baseDomain)
	}

	return domains
}

func normalizeDomain(domain string) string {
	return strings.ReplaceAll(domain, "*", "wildcard")
}

func encryptProviderConfig(config acme.DNSProviderConfig) string {
	// Simple encryption - in production, use proper encryption
	// For now, just JSON encode (not secure for sensitive data)
	data, _ := json.Marshal(config)
	return string(data)
}

func decryptProviderConfig(encoded string) acme.DNSProviderConfig {
	var config acme.DNSProviderConfig
	json.Unmarshal([]byte(encoded), &config)
	return config
}