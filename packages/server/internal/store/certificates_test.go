package store

import (
	"testing"
	"time"
)

func TestSQLite_GetCertificateHandlesUnsetValidityDates(t *testing.T) {
	db := newTestSQLite(t)

	cert := &Certificate{
		ID:                "cert-1",
		Name:              "Example",
		Domain:            "*.example.com",
		DNSProvider:       "cloudflare",
		DNSProviderConfig: "{}",
		Status:            CertStatusPending,
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
	}
	if err := db.CreateCertificate(cert); err != nil {
		t.Fatalf("CreateCertificate() error = %v", err)
	}

	got, err := db.GetCertificate(cert.ID)
	if err != nil {
		t.Fatalf("GetCertificate() error = %v", err)
	}
	if got == nil {
		t.Fatal("GetCertificate() = nil, want certificate")
	}
	if !got.NotBefore.IsZero() {
		t.Fatalf("NotBefore = %v, want zero time", got.NotBefore)
	}
	if !got.NotAfter.IsZero() {
		t.Fatalf("NotAfter = %v, want zero time", got.NotAfter)
	}
	if !got.RenewAt.IsZero() {
		t.Fatalf("RenewAt = %v, want zero time", got.RenewAt)
	}
}

func TestSQLite_ListCertificatesHandlesUnsetValidityDates(t *testing.T) {
	db := newTestSQLite(t)

	cert := &Certificate{
		ID:                "cert-1",
		Name:              "Example",
		Domain:            "*.example.com",
		DNSProvider:       "cloudflare",
		DNSProviderConfig: "{}",
		Status:            CertStatusPending,
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
	}
	if err := db.CreateCertificate(cert); err != nil {
		t.Fatalf("CreateCertificate() error = %v", err)
	}

	certs, err := db.ListCertificates()
	if err != nil {
		t.Fatalf("ListCertificates() error = %v", err)
	}
	if len(certs) != 1 {
		t.Fatalf("len(ListCertificates()) = %d, want 1", len(certs))
	}
	if !certs[0].NotBefore.IsZero() {
		t.Fatalf("NotBefore = %v, want zero time", certs[0].NotBefore)
	}
	if !certs[0].NotAfter.IsZero() {
		t.Fatalf("NotAfter = %v, want zero time", certs[0].NotAfter)
	}
	if !certs[0].RenewAt.IsZero() {
		t.Fatalf("RenewAt = %v, want zero time", certs[0].RenewAt)
	}
}

func TestSQLite_GetCertificateHandlesUnsetValidityDatesAfterUpdate(t *testing.T) {
	db := newTestSQLite(t)

	cert := &Certificate{
		ID:                "cert-1",
		Name:              "Example",
		Domain:            "*.example.com",
		DNSProvider:       "cloudflare",
		DNSProviderConfig: "{}",
		Status:            CertStatusPending,
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
	}
	if err := db.CreateCertificate(cert); err != nil {
		t.Fatalf("CreateCertificate() error = %v", err)
	}

	cert.Status = CertStatusFailed
	if err := db.UpdateCertificate(cert); err != nil {
		t.Fatalf("UpdateCertificate() error = %v", err)
	}

	got, err := db.GetCertificate(cert.ID)
	if err != nil {
		t.Fatalf("GetCertificate() after update error = %v", err)
	}
	if got == nil {
		t.Fatal("GetCertificate() after update = nil, want certificate")
	}
	if !got.NotBefore.IsZero() {
		t.Fatalf("NotBefore after update = %v, want zero time", got.NotBefore)
	}
	if !got.NotAfter.IsZero() {
		t.Fatalf("NotAfter after update = %v, want zero time", got.NotAfter)
	}
	if !got.RenewAt.IsZero() {
		t.Fatalf("RenewAt after update = %v, want zero time", got.RenewAt)
	}
}
