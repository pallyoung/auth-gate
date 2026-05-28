package certificate

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/pallyoung/auth-gate/packages/server/internal/acme"
	"github.com/pallyoung/auth-gate/packages/server/internal/store"
)

func newTestDB(t *testing.T) *store.SQLite {
	t.Helper()

	db, err := store.NewSQLite(filepath.Join(t.TempDir(), "auth-gate.db"))
	if err != nil {
		t.Fatalf("NewSQLite() error = %v", err)
	}
	t.Cleanup(func() {
		_ = db.Close()
	})
	return db
}

func newTestService(t *testing.T) (*Service, *store.SQLite) {
	t.Helper()

	db := newTestDB(t)
	svc := &Service{db: db}
	t.Cleanup(func() {
		waitForService(t, svc)
	})
	return svc, db
}

func waitForService(t *testing.T, svc *Service) {
	t.Helper()

	waiter, ok := any(svc).(interface{ Wait() })
	if !ok {
		t.Fatal("Service does not implement Wait")
	}
	waiter.Wait()
}

func TestService_ListWrapsDatabaseErrors(t *testing.T) {
	db := newTestDB(t)
	svc := &Service{db: db}

	if err := db.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	_, err := svc.List()
	if err == nil {
		t.Fatal("List() error = nil, want wrapped database error")
	}
	if got := Code(err); got != ErrCodeDatabase {
		t.Fatalf("Code(List() error) = %q, want %q", got, ErrCodeDatabase)
	}
	if got := Message(err); got != "failed to list certificates" {
		t.Fatalf("Message(List() error) = %q, want %q", got, "failed to list certificates")
	}
}

func TestService_GetWrapsDatabaseErrors(t *testing.T) {
	db := newTestDB(t)
	svc := &Service{db: db}

	if err := db.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	_, err := svc.Get("cert-1")
	if err == nil {
		t.Fatal("Get() error = nil, want wrapped database error")
	}
	if got := Code(err); got != ErrCodeDatabase {
		t.Fatalf("Code(Get() error) = %q, want %q", got, ErrCodeDatabase)
	}
	if got := Message(err); got != "failed to get certificate" {
		t.Fatalf("Message(Get() error) = %q, want %q", got, "failed to get certificate")
	}
}

func TestService_ProvisionRejectsWhitespaceOnlyName(t *testing.T) {
	db := newTestDB(t)
	svc := &Service{db: db}

	_, err := svc.Provision(context.Background(), ProvisionInput{
		Name:   "   ",
		Domain: "*.example.com",
		DNSProvider: acme.DNSProviderConfig{
			ProviderType:       "cloudflare",
			CloudFlareAPIToken: "cf_test_token",
		},
	})
	if err == nil {
		t.Fatal("Provision() error = nil, want invalid certificate name error")
	}
	if got := Code(err); got != ErrCodeInvalidName {
		t.Fatalf("Code(Provision() error) = %q, want %q", got, ErrCodeInvalidName)
	}
	if got := Message(err); got != "certificate name required" {
		t.Fatalf("Message(Provision() error) = %q, want %q", got, "certificate name required")
	}

	certs, err := db.ListCertificates()
	if err != nil {
		t.Fatalf("ListCertificates() error = %v", err)
	}
	if len(certs) != 0 {
		t.Fatalf("len(ListCertificates()) = %d, want 0", len(certs))
	}
}

func TestService_ProvisionTrimsDomainBeforePersistingAndDuplicateChecks(t *testing.T) {
	svc, db := newTestService(t)

	first, err := svc.Provision(context.Background(), ProvisionInput{
		Name:   "Wildcard",
		Domain: " *.example.com ",
		DNSProvider: acme.DNSProviderConfig{
			ProviderType: "manual",
		},
	})
	if err != nil {
		t.Fatalf("Provision() first error = %v", err)
	}
	if first.Domain != "*.example.com" {
		t.Fatalf("first.Domain = %q, want %q", first.Domain, "*.example.com")
	}

	stored, err := db.GetCertificate(first.ID)
	if err != nil {
		t.Fatalf("GetCertificate() error = %v", err)
	}
	if stored == nil {
		t.Fatal("GetCertificate() = nil, want stored certificate")
	}
	if stored.Domain != "*.example.com" {
		t.Fatalf("stored.Domain = %q, want %q", stored.Domain, "*.example.com")
	}

	_, err = svc.Provision(context.Background(), ProvisionInput{
		Name:   "Wildcard 2",
		Domain: "*.example.com",
		DNSProvider: acme.DNSProviderConfig{
			ProviderType: "manual",
		},
	})
	if err == nil {
		t.Fatal("Provision() second error = nil, want duplicate domain error")
	}
	if got := Code(err); got != ErrCodeDomainExists {
		t.Fatalf("Code(second Provision() error) = %q, want %q", got, ErrCodeDomainExists)
	}
}

func TestService_WaitDrainsBackgroundProvisioning(t *testing.T) {
	svc, db := newTestService(t)

	cert, err := svc.Provision(context.Background(), ProvisionInput{
		Name:   "Manual",
		Domain: "*.example.com",
		DNSProvider: acme.DNSProviderConfig{
			ProviderType: "manual",
		},
	})
	if err != nil {
		t.Fatalf("Provision() error = %v", err)
	}

	waitForService(t, svc)

	stored, err := db.GetCertificate(cert.ID)
	if err != nil {
		t.Fatalf("GetCertificate() error = %v", err)
	}
	if stored == nil {
		t.Fatal("GetCertificate() = nil, want stored certificate")
	}
	if stored.Status != store.CertStatusFailed {
		t.Fatalf("stored.Status = %q, want %q", stored.Status, store.CertStatusFailed)
	}
}

func TestService_RenewRejectsUnsupportedManualProvider(t *testing.T) {
	db := newTestDB(t)
	svc := &Service{db: db}

	cert := &store.Certificate{
		ID:                "cert-manual",
		Name:              "Manual wildcard",
		Domain:            "*.example.com",
		DNSProvider:       "manual",
		DNSProviderConfig: encryptProviderConfig(acme.DNSProviderConfig{ProviderType: "manual"}),
		Status:            store.CertStatusActive,
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
	}
	if err := db.CreateCertificate(cert); err != nil {
		t.Fatalf("CreateCertificate() error = %v", err)
	}

	err := svc.Renew(cert.ID)
	if err == nil {
		t.Fatal("Renew() error = nil, want unsupported provider error")
	}
	if got := Code(err); got != ErrCodeDNSProvider {
		t.Fatalf("Code(Renew() error) = %q, want %q", got, ErrCodeDNSProvider)
	}
	if got := Message(err); got != "failed to create DNS provider" {
		t.Fatalf("Message(Renew() error) = %q, want %q", got, "failed to create DNS provider")
	}
}

func TestService_RenewRejectsUnsupportedPowerDNSProvider(t *testing.T) {
	db := newTestDB(t)
	svc := &Service{db: db}

	cert := &store.Certificate{
		ID:          "cert-pdns",
		Name:        "PowerDNS wildcard",
		Domain:      "*.example.com",
		DNSProvider: "pdns",
		DNSProviderConfig: encryptProviderConfig(acme.DNSProviderConfig{
			ProviderType:   "pdns",
			PowerDNSHost:   "https://dns.example.com",
			PowerDNSAPIKey: "pdns-secret",
			PowerDNSZone:   "example.com",
		}),
		Status:    store.CertStatusActive,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	if err := db.CreateCertificate(cert); err != nil {
		t.Fatalf("CreateCertificate() error = %v", err)
	}

	err := svc.Renew(cert.ID)
	if err == nil {
		t.Fatal("Renew() error = nil, want unsupported provider error")
	}
	if got := Code(err); got != ErrCodeDNSProvider {
		t.Fatalf("Code(Renew() error) = %q, want %q", got, ErrCodeDNSProvider)
	}
	if got := Message(err); got != "failed to create DNS provider" {
		t.Fatalf("Message(Renew() error) = %q, want %q", got, "failed to create DNS provider")
	}
}
