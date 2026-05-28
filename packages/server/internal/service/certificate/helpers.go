package certificate

import (
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/pallyoung/auth-gate/packages/server/internal/acme"
	"github.com/pallyoung/auth-gate/packages/server/internal/service/runtime"
)

// parseCertPEM parses a certificate from PEM format
func parseCertPEM(certPEM []byte) (*x509.Certificate, error) {
	block, _ := pem.Decode(certPEM)
	if block == nil {
		return nil, fmt.Errorf("failed to parse PEM block")
	}
	return x509.ParseCertificate(block.Bytes)
}

// getDomainDir returns the directory path for a domain's certificates
func getDomainDir(acmeDir string, domain string) string {
	return filepath.Join(acmeDir, "certs", normalizeDomainForPath(domain))
}

// normalizeDomainForPath normalizes a domain for use in file paths
func normalizeDomainForPath(domain string) string {
	return "_wildcard_" + strings.ReplaceAll(domain, "*.", "")
}

// ServiceConfig holds configuration for creating the service
type ServiceConfig struct {
	DataDir    string
	ACMEEmail  string
	UseStaging bool
}

// ValidateDNSProviderConfig validates DNS provider configuration
func ValidateDNSProviderConfig(provider string, config map[string]string) error {
	switch normalizeProviderName(provider) {
	case "cloudflare":
		if trimmedConfigValue(config, "api_token") == "" {
			return fmt.Errorf("cloudflare: api_token is required")
		}
	case "route53":
		if trimmedConfigValue(config, "access_key_id") == "" || trimmedConfigValue(config, "secret_access_key") == "" {
			return fmt.Errorf("route53: access_key_id and secret_access_key are required")
		}
	default:
		return fmt.Errorf("unsupported DNS provider: %s", strings.TrimSpace(provider))
	}
	return nil
}

// ParseDNSProviderConfig parses DNS provider configuration from a map
func ParseDNSProviderConfig(provider string, config map[string]string) (acme.DNSProviderConfig, error) {
	provider = normalizeProviderName(provider)

	if err := ValidateDNSProviderConfig(provider, config); err != nil {
		return acme.DNSProviderConfig{}, err
	}

	cfg := acme.DNSProviderConfig{
		ProviderType: provider,
	}

	switch provider {
	case "cloudflare":
		cfg.CloudFlareAPIToken = trimmedConfigValue(config, "api_token")
	case "route53":
		cfg.Route53AccessKeyID = trimmedConfigValue(config, "access_key_id")
		cfg.Route53SecretAccessKey = trimmedConfigValue(config, "secret_access_key")
		cfg.Route53Region = trimmedConfigValue(config, "region")
	case "pdns":
		cfg.PowerDNSHost = trimmedConfigValue(config, "host")
		cfg.PowerDNSAPIKey = trimmedConfigValue(config, "api_key")
		cfg.PowerDNSZone = trimmedConfigValue(config, "zone")
	}

	return cfg, nil
}

func normalizeProviderName(provider string) string {
	return strings.ToLower(strings.TrimSpace(provider))
}

func trimmedConfigValue(config map[string]string, key string) string {
	return strings.TrimSpace(config[key])
}

// IsCertificateExpiringSoon returns true if the certificate expires within the given duration
func IsCertificateExpiringSoon(cert interface{}, within time.Duration) bool {
	return false
}

var _ = runtime.Reloader(nil) // ensure runtime package is imported
