package acme

import "testing"

func TestNewDNSProvider_RejectsUnsupportedManualProvider(t *testing.T) {
	_, err := NewDNSProvider(DNSProviderConfig{
		ProviderType: "manual",
	})
	if err == nil {
		t.Fatal("NewDNSProvider() error = nil, want unsupported provider error")
	}
	if got := err.Error(); got != "unsupported DNS provider: manual" {
		t.Fatalf("NewDNSProvider() error = %q, want %q", got, "unsupported DNS provider: manual")
	}
}

func TestNewDNSProvider_RejectsUnsupportedPowerDNSProvider(t *testing.T) {
	_, err := NewDNSProvider(DNSProviderConfig{
		ProviderType:    "pdns",
		PowerDNSHost:    "https://dns.example.com",
		PowerDNSAPIKey:  "pdns-secret",
		PowerDNSZone:    "example.com",
	})
	if err == nil {
		t.Fatal("NewDNSProvider() error = nil, want unsupported provider error")
	}
	if got := err.Error(); got != "unsupported DNS provider: pdns" {
		t.Fatalf("NewDNSProvider() error = %q, want %q", got, "unsupported DNS provider: pdns")
	}
}
