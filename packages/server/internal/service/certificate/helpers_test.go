package certificate

import "testing"

func TestParseDNSProviderConfig_RejectsMissingRequiredCloudflareToken(t *testing.T) {
	_, err := ParseDNSProviderConfig("cloudflare", map[string]string{})
	if err == nil {
		t.Fatal("ParseDNSProviderConfig() error = nil, want invalid provider config error")
	}
	if got := err.Error(); got != "cloudflare: api_token is required" {
		t.Fatalf("ParseDNSProviderConfig() error = %q, want %q", got, "cloudflare: api_token is required")
	}
}

func TestParseDNSProviderConfig_RejectsWhitespaceOnlyCloudflareToken(t *testing.T) {
	_, err := ParseDNSProviderConfig("cloudflare", map[string]string{
		"api_token": "   ",
	})
	if err == nil {
		t.Fatal("ParseDNSProviderConfig() error = nil, want invalid provider config error")
	}
	if got := err.Error(); got != "cloudflare: api_token is required" {
		t.Fatalf("ParseDNSProviderConfig() error = %q, want %q", got, "cloudflare: api_token is required")
	}
}

func TestParseDNSProviderConfig_RejectsMissingRequiredRoute53Keys(t *testing.T) {
	_, err := ParseDNSProviderConfig("route53", map[string]string{
		"access_key_id": "AKIAEXAMPLE",
	})
	if err == nil {
		t.Fatal("ParseDNSProviderConfig() error = nil, want invalid provider config error")
	}
	if got := err.Error(); got != "route53: access_key_id and secret_access_key are required" {
		t.Fatalf(
			"ParseDNSProviderConfig() error = %q, want %q",
			got,
			"route53: access_key_id and secret_access_key are required",
		)
	}
}

func TestParseDNSProviderConfig_RejectsWhitespaceOnlyRoute53Keys(t *testing.T) {
	_, err := ParseDNSProviderConfig("route53", map[string]string{
		"access_key_id":     "   ",
		"secret_access_key": "\t",
	})
	if err == nil {
		t.Fatal("ParseDNSProviderConfig() error = nil, want invalid provider config error")
	}
	if got := err.Error(); got != "route53: access_key_id and secret_access_key are required" {
		t.Fatalf(
			"ParseDNSProviderConfig() error = %q, want %q",
			got,
			"route53: access_key_id and secret_access_key are required",
		)
	}
}

func TestParseDNSProviderConfig_RejectsUnsupportedManualProvider(t *testing.T) {
	_, err := ParseDNSProviderConfig("manual", map[string]string{})
	if err == nil {
		t.Fatal("ParseDNSProviderConfig() error = nil, want unsupported provider error")
	}
	if got := err.Error(); got != "unsupported DNS provider: manual" {
		t.Fatalf("ParseDNSProviderConfig() error = %q, want %q", got, "unsupported DNS provider: manual")
	}
}

func TestParseDNSProviderConfig_RejectsUnsupportedPowerDNSProvider(t *testing.T) {
	_, err := ParseDNSProviderConfig("pdns", map[string]string{
		"host":    "https://dns.example.com",
		"api_key": "pdns-secret",
	})
	if err == nil {
		t.Fatal("ParseDNSProviderConfig() error = nil, want unsupported provider error")
	}
	if got := err.Error(); got != "unsupported DNS provider: pdns" {
		t.Fatalf("ParseDNSProviderConfig() error = %q, want %q", got, "unsupported DNS provider: pdns")
	}
}
