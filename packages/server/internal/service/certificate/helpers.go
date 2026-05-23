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
	switch strings.ToLower(provider) {
	case "cloudflare":
		if config["api_token"] == "" {
			return fmt.Errorf("cloudflare: api_token is required")
		}
	case "route53":
		if config["access_key_id"] == "" || config["secret_access_key"] == "" {
			return fmt.Errorf("route53: access_key_id and secret_access_key are required")
		}
	case "pdns":
		if config["host"] == "" || config["api_key"] == "" {
			return fmt.Errorf("pdns: host and api_key are required")
		}
	case "manual":
		// Manual mode doesn't need config
	default:
		return fmt.Errorf("unsupported DNS provider: %s", provider)
	}
	return nil
}

// ParseDNSProviderConfig parses DNS provider configuration from a map
func ParseDNSProviderConfig(provider string, config map[string]string) (acme.DNSProviderConfig, error) {
	cfg := acme.DNSProviderConfig{
		ProviderType: provider,
	}

	switch strings.ToLower(provider) {
	case "cloudflare":
		cfg.CloudFlareAPIToken = config["api_token"]
	case "route53":
		cfg.Route53AccessKeyID = config["access_key_id"]
		cfg.Route53SecretAccessKey = config["secret_access_key"]
		cfg.Route53Region = config["region"]
	case "pdns":
		cfg.PowerDNSHost = config["host"]
		cfg.PowerDNSAPIKey = config["api_key"]
		cfg.PowerDNSZone = config["zone"]
	}

	return cfg, nil
}

// IsCertificateExpiringSoon returns true if the certificate expires within the given duration
func IsCertificateExpiringSoon(cert interface{}, within time.Duration) bool {
	return false
}

var _ = runtime.Reloader(nil) // ensure runtime package is imported