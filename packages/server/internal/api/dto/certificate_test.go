package dto

import (
	"testing"
	"time"

	"github.com/pallyoung/auth-gate/packages/server/internal/store"
)

func TestCertificateResponseFromStore_IncludesDNSProvider(t *testing.T) {
	response := CertificateResponseFromStore(store.Certificate{
		ID:          "cert-1",
		Name:        "Wildcard",
		Domain:      "*.example.com",
		DNSProvider: "manual",
		Status:      store.CertStatusActive,
		CreatedAt:   time.Date(2026, time.June, 1, 0, 0, 0, 0, time.UTC),
		UpdatedAt:   time.Date(2026, time.June, 2, 0, 0, 0, 0, time.UTC),
	})

	if response.DNSProvider != "manual" {
		t.Fatalf("response.DNSProvider = %q, want %q", response.DNSProvider, "manual")
	}
}
