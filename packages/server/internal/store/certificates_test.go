package store

import (
	"testing"
	"time"
)

func newTestCert(domain string) *Certificate {
	return &Certificate{
		ID:        "cert-" + domain,
		Name:      "Example " + domain,
		Domain:    domain,
		Source:    SourceLocalCA,
		CAID:      "ca-1",
		Status:    CertStatusActive,
		CertPath:  "/tmp/cert.pem",
		KeyPath:   "/tmp/key.pem",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}

func TestJSON_GetCertificateHandlesUnsetValidityDates(t *testing.T) {
	db := newTestStore(t)
	cert := newTestCert("*.example.com")

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

func TestJSON_ListCertificatesHandlesUnsetValidityDates(t *testing.T) {
	db := newTestStore(t)
	cert := newTestCert("*.example.com")

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

func TestJSON_GetCertificateHandlesUnsetValidityDatesAfterUpdate(t *testing.T) {
	db := newTestStore(t)
	cert := newTestCert("*.example.com")

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

func TestJSON_CertificateCRUDRoundTrip(t *testing.T) {
	db := newTestStore(t)
	cert := newTestCert("foo.example.com")
	cert.NotBefore = time.Now().Add(-time.Hour)
	cert.NotAfter = time.Now().Add(24 * time.Hour)
	cert.RenewAt = time.Now().Add(-time.Minute)

	if err := db.CreateCertificate(cert); err != nil {
		t.Fatalf("CreateCertificate: %v", err)
	}

	got, err := db.GetCertificateByDomain("foo.example.com")
	if err != nil {
		t.Fatalf("GetCertificateByDomain: %v", err)
	}
	if got == nil {
		t.Fatal("expected cert")
	}
	if got.Source != SourceLocalCA {
		t.Errorf("Source = %q, want %q", got.Source, SourceLocalCA)
	}
	if got.Status != CertStatusActive {
		t.Errorf("Status = %q, want %q", got.Status, CertStatusActive)
	}

	got.Status = CertStatusFailed
	if err := db.UpdateCertificate(got); err != nil {
		t.Fatalf("UpdateCertificate: %v", err)
	}
	refreshed, _ := db.GetCertificate(got.ID)
	if refreshed.Status != CertStatusFailed {
		t.Errorf("Status after update = %q, want failed", refreshed.Status)
	}

	if err := db.DeleteCertificate(got.ID); err != nil {
		t.Fatalf("DeleteCertificate: %v", err)
	}
	gone, _ := db.GetCertificate(got.ID)
	if gone != nil {
		t.Error("expected cert to be deleted")
	}
}

func TestJSON_ListExpiringLocalCertificates(t *testing.T) {
	db := newTestStore(t)

	active := newTestCert("active.example.com")
	active.NotAfter = time.Now().Add(24 * time.Hour)
	active.RenewAt = time.Now().Add(-time.Hour)
	if err := db.CreateCertificate(active); err != nil {
		t.Fatalf("CreateCertificate active: %v", err)
	}

	imported := newTestCert("imported.example.com")
	imported.Source = SourceImported
	imported.CAID = ""
	imported.NotAfter = time.Now().Add(24 * time.Hour)
	imported.RenewAt = time.Now().Add(-time.Hour)
	if err := db.CreateCertificate(imported); err != nil {
		t.Fatalf("CreateCertificate imported: %v", err)
	}

	far := newTestCert("far.example.com")
	far.NotAfter = time.Now().Add(365 * 24 * time.Hour)
	far.RenewAt = time.Now().Add(335 * 24 * time.Hour)
	if err := db.CreateCertificate(far); err != nil {
		t.Fatalf("CreateCertificate far: %v", err)
	}

	expiring, err := db.ListExpiringLocalCertificates(time.Now())
	if err != nil {
		t.Fatalf("ListExpiringLocalCertificates: %v", err)
	}
	if len(expiring) != 1 {
		t.Fatalf("len = %d, want 1 (imported should be excluded)", len(expiring))
	}
	if expiring[0].ID != active.ID {
		t.Errorf("ID = %q, want %q", expiring[0].ID, active.ID)
	}
}

func TestJSON_CACertificateRoundTrip(t *testing.T) {
	db := newTestStore(t)

	first, err := db.GetFirstCACertificate()
	if err != nil {
		t.Fatalf("GetFirstCACertificate (empty): %v", err)
	}
	if first != nil {
		t.Fatal("expected nil for empty ca_certificates")
	}

	ca := &CACertificate{
		ID:        "ca-1",
		Name:      "Auth Gate Local CA",
		CertPEM:   "-----BEGIN CERTIFICATE-----\nfake\n-----END CERTIFICATE-----\n",
		KeyPEM:    "-----BEGIN RSA PRIVATE KEY-----\nfake\n-----END RSA PRIVATE KEY-----\n",
		NotBefore: time.Now().Add(-time.Hour),
		NotAfter:  time.Now().Add(100 * 365 * 24 * time.Hour),
		CreatedAt: time.Now(),
	}
	if err := db.CreateCACertificate(ca); err != nil {
		t.Fatalf("CreateCACertificate: %v", err)
	}

	got, err := db.GetCACertificate(ca.ID)
	if err != nil {
		t.Fatalf("GetCACertificate: %v", err)
	}
	if got == nil || got.Name != ca.Name {
		t.Fatalf("GetCACertificate = %+v, want name %q", got, ca.Name)
	}

	list, err := db.ListCACertificates()
	if err != nil {
		t.Fatalf("ListCACertificates: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("len = %d, want 1", len(list))
	}

	first, err = db.GetFirstCACertificate()
	if err != nil {
		t.Fatalf("GetFirstCACertificate: %v", err)
	}
	if first == nil || first.ID != ca.ID {
		t.Fatal("expected to find the first CA")
	}
}
