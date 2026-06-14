package dto

import (
	"testing"
	"time"

	"github.com/pallyoung/auth-gate/packages/server/internal/store"
)

func TestCertificateResponseFromStore_IncludesSource(t *testing.T) {
	response := CertificateResponseFromStore(store.Certificate{
		ID:        "cert-1",
		Name:      "Wildcard",
		Domain:    "*.example.com",
		Source:    store.SourceLocalCA,
		CAID:      "ca-default",
		Status:    store.CertStatusActive,
		NotBefore: time.Date(2026, time.June, 1, 0, 0, 0, 0, time.UTC),
		NotAfter:  time.Date(2026, time.September, 1, 0, 0, 0, 0, time.UTC),
		RenewAt:   time.Date(2026, time.August, 1, 0, 0, 0, 0, time.UTC),
		CreatedAt: time.Date(2026, time.June, 1, 0, 0, 0, 0, time.UTC),
		UpdatedAt: time.Date(2026, time.June, 2, 0, 0, 0, 0, time.UTC),
	})

	if response.Source != store.SourceLocalCA {
		t.Fatalf("Source = %q, want %q", response.Source, store.SourceLocalCA)
	}
	if response.CAID != "ca-default" {
		t.Fatalf("CAID = %q, want %q", response.CAID, "ca-default")
	}
	if response.NotBefore == "" {
		t.Error("expected NotBefore populated")
	}
	if response.RenewAt == "" {
		t.Error("expected RenewAt populated")
	}
}

func TestCertificateResponseFromStore_ZeroTimesAreEmpty(t *testing.T) {
	response := CertificateResponseFromStore(store.Certificate{
		ID:        "cert-1",
		Domain:    "x.example.com",
		Source:    store.SourceImported,
		Status:    store.CertStatusActive,
		CreatedAt: time.Date(2026, time.June, 1, 0, 0, 0, 0, time.UTC),
		UpdatedAt: time.Date(2026, time.June, 2, 0, 0, 0, 0, time.UTC),
	})

	if response.NotBefore != "" {
		t.Errorf("NotBefore = %q, want empty", response.NotBefore)
	}
	if response.NotAfter != "" {
		t.Errorf("NotAfter = %q, want empty", response.NotAfter)
	}
	if response.RenewAt != "" {
		t.Errorf("RenewAt = %q, want empty", response.RenewAt)
	}
}

func TestCertificateListResponseFromStore(t *testing.T) {
	list := CertificateListResponseFromStore([]store.Certificate{
		{ID: "1", Domain: "a.example.com", Source: store.SourceLocalCA, Status: store.CertStatusActive,
			CreatedAt: time.Now(), UpdatedAt: time.Now()},
		{ID: "2", Domain: "b.example.com", Source: store.SourceImported, Status: store.CertStatusActive,
			CreatedAt: time.Now(), UpdatedAt: time.Now()},
	})
	if len(list) != 2 {
		t.Fatalf("len = %d, want 2", len(list))
	}
	if list[0].Domain != "a.example.com" {
		t.Errorf("list[0] = %+v", list[0])
	}
}
