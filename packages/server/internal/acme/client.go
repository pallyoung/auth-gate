package acme

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-acme/lego/v4/certificate"
	"github.com/go-acme/lego/v4/challenge"
	"github.com/go-acme/lego/v4/lego"
	"github.com/go-acme/lego/v4/registration"
)

// Config holds ACME client configuration
type Config struct {
	Email       string // ACME account email
	DataDir     string // Directory for account key and certificates
	AcceptTerms bool   // Accept Let's Encrypt terms of service
	UseStaging  bool   // Use staging server (for testing)
}

// legoUser implements registration.User interface
type legoUser struct {
	email      string
	privateKey crypto.PrivateKey
	reg        *registration.Resource
}

// GetEmail returns the user's email
func (u *legoUser) GetEmail() string {
	return u.email
}

// GetRegistration returns the user's registration resource
func (u *legoUser) GetRegistration() *registration.Resource {
	return u.reg
}

// GetPrivateKey returns the user's private key
func (u *legoUser) GetPrivateKey() crypto.PrivateKey {
	return u.privateKey
}

// Client wraps the lego ACME client
type Client struct {
	config  Config
	client  *lego.Client
	user    *legoUser
	certDir string
}

// NewClient creates a new ACME client
func NewClient(cfg Config) (*Client, error) {
	if cfg.DataDir == "" {
		return nil, fmt.Errorf("data directory is required")
	}
	if cfg.Email == "" {
		return nil, fmt.Errorf("ACME email is required")
	}

	// Ensure directory exists
	if err := os.MkdirAll(cfg.DataDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create data directory: %w", err)
	}

	accountKeyPath := filepath.Join(cfg.DataDir, "account.key")

	var accountKey crypto.PrivateKey

	// Load existing account key or generate new one
	if data, err := os.ReadFile(accountKeyPath); err == nil {
		accountKey, err = parsePrivateKey(data)
		if err != nil {
			// Key is corrupt or invalid, generate new one
			accountKey, err = generatePrivateKey()
			if err != nil {
				return nil, fmt.Errorf("failed to generate account key: %w", err)
			}
			if err := savePrivateKey(accountKey, accountKeyPath); err != nil {
				return nil, fmt.Errorf("failed to save account key: %w", err)
			}
		}
	} else {
		// Generate new account key
		accountKey, err = generatePrivateKey()
		if err != nil {
			return nil, fmt.Errorf("failed to generate account key: %w", err)
		}
		if err := savePrivateKey(accountKey, accountKeyPath); err != nil {
			return nil, fmt.Errorf("failed to save account key: %w", err)
		}
	}

	// Create user
	user := &legoUser{
		email:      cfg.Email,
		privateKey: accountKey,
	}

	// Create lego config
	caURL := lego.LEDirectoryProduction
	if cfg.UseStaging {
		caURL = lego.LEDirectoryStaging
	}

	config := lego.NewConfig(user)
	config.CADirURL = caURL

	client, err := lego.NewClient(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create lego client: %w", err)
	}

	// Register account
	reg, err := client.Registration.Register(registration.RegisterOptions{
		TermsOfServiceAgreed: cfg.AcceptTerms,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to register ACME account: %w", err)
	}
	user.reg = reg

	return &Client{
		config:  cfg,
		client:  client,
		user:    user,
		certDir: filepath.Join(cfg.DataDir, "certs"),
	}, nil
}

// RequestCertificate requests a new certificate for the given domains using DNS-01 challenge
func (c *Client) RequestCertificate(domains []string, provider challenge.Provider) (certPEM, keyPEM []byte, err error) {
	// Set DNS provider
	if err := c.client.Challenge.SetDNS01Provider(provider); err != nil {
		return nil, nil, fmt.Errorf("failed to set DNS-01 provider: %w", err)
	}

	// Request certificate
	certRes, err := c.client.Certificate.Obtain(certificate.ObtainRequest{
		Domains: domains,
		Bundle:  true,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to obtain certificate: %w", err)
	}

	return certRes.Certificate, certRes.PrivateKey, nil
}

// SaveCertificate saves certificate and key to the cert directory
func (c *Client) SaveCertificate(domain string, certPEM, keyPEM []byte) (certPath, keyPath string, err error) {
	domainDir := filepath.Join(c.certDir, normalizeDomain(domain))
	if err := os.MkdirAll(domainDir, 0700); err != nil {
		return "", "", fmt.Errorf("failed to create domain directory: %w", err)
	}

	certPath = filepath.Join(domainDir, "cert.pem")
	keyPath = filepath.Join(domainDir, "key.pem")

	if err := os.WriteFile(certPath, certPEM, 0600); err != nil {
		return "", "", fmt.Errorf("failed to write certificate: %w", err)
	}
	if err := os.WriteFile(keyPath, keyPEM, 0600); err != nil {
		return "", "", fmt.Errorf("failed to write key: %w", err)
	}

	return certPath, keyPath, nil
}

// GetCertDir returns the certificate directory path
func (c *Client) GetCertDir() string {
	return c.certDir
}

// GetEmail returns the registered account email
func (c *Client) GetEmail() string {
	return c.user.email
}

// Helper functions

func generatePrivateKey() (crypto.PrivateKey, error) {
	return generateRSAKey(2048)
}

func generateRSAKey(bits int) (crypto.PrivateKey, error) {
	return rsa.GenerateKey(rand.Reader, bits)
}

func parsePrivateKey(data []byte) (crypto.PrivateKey, error) {
	block, _ := pem.Decode(data)
	if block == nil {
		return nil, fmt.Errorf("failed to parse PEM block")
	}
	key, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		return nil, err
	}
	return key, nil
}

func savePrivateKey(key crypto.PrivateKey, path string) error {
	rsaKey, ok := key.(*rsa.PrivateKey)
	if !ok {
		return fmt.Errorf("expected RSA private key")
	}
	keyBytes := x509.MarshalPKCS1PrivateKey(rsaKey)
	pemBytes := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: keyBytes,
	})
	return os.WriteFile(path, pemBytes, 0600)
}

// ValidateCertificate checks if the certificate is valid
func ValidateCertificate(certPEM []byte) error {
	cert, err := parseCertificate(certPEM)
	if err != nil {
		return fmt.Errorf("failed to parse certificate: %w", err)
	}

	now := time.Now()
	if now.Before(cert.NotBefore) {
		return fmt.Errorf("certificate not yet valid")
	}
	if now.After(cert.NotAfter) {
		return fmt.Errorf("certificate expired")
	}

	return nil
}

func parseCertificate(certPEM []byte) (*x509.Certificate, error) {
	block, _ := pem.Decode(certPEM)
	if block == nil {
		return nil, fmt.Errorf("failed to parse PEM block")
	}
	return x509.ParseCertificate(block.Bytes)
}

func normalizeDomain(domain string) string {
	return "_wildcard_" + strings.ReplaceAll(domain, "*.", "")
}