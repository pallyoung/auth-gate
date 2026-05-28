package acme

import (
	"fmt"
	"os"
	"strings"

	"github.com/go-acme/lego/v4/challenge"
	cloudflareprovider "github.com/go-acme/lego/v4/providers/dns/cloudflare"
	route53provider "github.com/go-acme/lego/v4/providers/dns/route53"
)

// DNSProviderConfig holds configuration for different DNS providers
type DNSProviderConfig struct {
	// Common fields
	ProviderType string // "cloudflare", "route53"

	// CloudFlare
	CloudFlareAPIToken string

	// Route53
	Route53AccessKeyID     string
	Route53SecretAccessKey string
	Route53Region          string

	// PowerDNS
	PowerDNSHost   string
	PowerDNSAPIKey string
	PowerDNSZone   string
}

// NewDNSProvider creates a DNS provider based on configuration
func NewDNSProvider(config DNSProviderConfig) (challenge.Provider, error) {
	switch strings.ToLower(config.ProviderType) {
	case "cloudflare":
		return newCloudFlareProvider(config.CloudFlareAPIToken)
	case "route53":
		return newRoute53Provider(config.Route53AccessKeyID, config.Route53SecretAccessKey, config.Route53Region)
	default:
		return nil, fmt.Errorf("unsupported DNS provider: %s", config.ProviderType)
	}
}

// cloudFlareProvider implements DNS-01 challenge for CloudFlare
type cloudFlareProvider struct {
	delegate *cloudflareprovider.DNSProvider
}

func newCloudFlareProvider(apiToken string) (*cloudFlareProvider, error) {
	if apiToken == "" {
		return nil, fmt.Errorf("CloudFlare API token is required")
	}

	cfg := cloudflareprovider.NewDefaultConfig()
	cfg.AuthToken = apiToken

	provider, err := cloudflareprovider.NewDNSProviderConfig(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create CloudFlare provider: %w", err)
	}
	return &cloudFlareProvider{delegate: provider}, nil
}

func (p *cloudFlareProvider) Present(domain, token, keyAuth string) error {
	return p.delegate.Present(domain, token, keyAuth)
}

func (p *cloudFlareProvider) CleanUp(domain, token, keyAuth string) error {
	return p.delegate.CleanUp(domain, token, keyAuth)
}

// route53Provider implements DNS-01 challenge for AWS Route53
type route53Provider struct {
	delegate *route53provider.DNSProvider
}

func newRoute53Provider(accessKeyID, secretAccessKey, region string) (*route53Provider, error) {
	// Route53 provider reads from environment variables AWS_ACCESS_KEY_ID, AWS_SECRET_ACCESS_KEY, AWS_REGION
	// Set them programmatically via environment
	if accessKeyID != "" {
		os.Setenv("AWS_ACCESS_KEY_ID", accessKeyID)
	}
	if secretAccessKey != "" {
		os.Setenv("AWS_SECRET_ACCESS_KEY", secretAccessKey)
	}
	if region != "" {
		os.Setenv("AWS_REGION", region)
	}

	provider, err := route53provider.NewDNSProvider()
	if err != nil {
		return nil, fmt.Errorf("failed to create Route53 provider: %w", err)
	}
	return &route53Provider{delegate: provider}, nil
}

func (p *route53Provider) Present(domain, token, keyAuth string) error {
	return p.delegate.Present(domain, token, keyAuth)
}

func (p *route53Provider) CleanUp(domain, token, keyAuth string) error {
	return p.delegate.CleanUp(domain, token, keyAuth)
}

// powerDNSProvider implements DNS-01 challenge for PowerDNS
// This is a simplified implementation that returns an error
// PowerDNS support requires the pdns provider package
type powerDNSProvider struct {
	host   string
	apiKey string
	zone   string
}

func newPowerDNSProvider(host, apiKey, zone string) (*powerDNSProvider, error) {
	if host == "" || apiKey == "" {
		return nil, fmt.Errorf("PowerDNS host and API key are required")
	}
	return &powerDNSProvider{
		host:   host,
		apiKey: apiKey,
		zone:   zone,
	}, nil
}

func (p *powerDNSProvider) Present(domain, token, keyAuth string) error {
	// PowerDNS provider would need to be implemented here
	// For now, return an error indicating it needs implementation
	return fmt.Errorf("PowerDNS provider not yet implemented - use CloudFlare or Route53")
}

func (p *powerDNSProvider) CleanUp(domain, token, keyAuth string) error {
	return nil
}

// manualProvider allows manual DNS configuration
type manualProvider struct{}

func newManualProvider() (*manualProvider, error) {
	return &manualProvider{}, nil
}

func (p *manualProvider) Present(domain, token, keyAuth string) error {
	// In manual mode, user must manually create DNS TXT record
	return fmt.Errorf("MANUAL DNS MODE: Please create TXT record:\n  Name: _acme-challenge.%s\n  Value: %s\n\nThen wait for DNS propagation and call the API again...", domain, keyAuth[:32]+"...")
}

func (p *manualProvider) CleanUp(domain, token, keyAuth string) error {
	// In manual mode, user should manually delete DNS TXT record
	return nil
}
