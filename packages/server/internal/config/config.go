package config

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Server   ServerConfig   `yaml:"server"`
	Database DatabaseConfig `yaml:"database"`
	Auth     AuthConfig     `yaml:"auth"`

	path string `yaml:"-"` // config file path (not serialized)
}

type ServerConfig struct {
	Listen    []ListenEntry `yaml:"listen"`
	HTTPSPort int           `yaml:"https_port,omitempty"` // deprecated, kept for backward compat
}

// ListenEntry represents a single listen address with optional TLS.
type ListenEntry struct {
	Addr string `yaml:"addr"`
	TLS  bool   `yaml:"tls,omitempty"`
}

type DatabaseConfig struct {
	Path string `yaml:"path"`
}

type AuthConfig struct {
	JWTSecret              string `yaml:"jwt_secret"`
	BootstrapAdminPassword string `yaml:"bootstrap_admin_password"`
	LegacyAdminToken       string `yaml:"admin_token"`
	AllowEphemeralSecret   bool   `yaml:"allow_ephemeral_secret"`
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	cfg.path = path

	return &cfg, nil
}

func DefaultConfig() *Config {
	return &Config{
		Server: ServerConfig{
			Listen: []ListenEntry{{Addr: ":80", TLS: false}},
		},
		Database: DatabaseConfig{
			Path: "data",
		},
		Auth: AuthConfig{},
	}
}

// Save writes the config back to the file it was loaded from.
func (c *Config) Save() error {
	if c.path == "" {
		return fmt.Errorf("config: no file path set")
	}
	data, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Errorf("config: marshal: %w", err)
	}
	return os.WriteFile(c.path, data, 0644)
}

// EffectiveListenAddrs returns all HTTP (non-TLS) listen addresses.
func (c *Config) EffectiveListenAddrs() []string {
	if len(c.Server.Listen) > 0 {
		var addrs []string
		for _, e := range c.Server.Listen {
			if !e.TLS {
				addrs = append(addrs, e.Addr)
			}
		}
		if len(addrs) > 0 {
			return addrs
		}
	}
	return []string{":80"}
}

// EffectiveHTTPSAddrs returns all HTTPS (TLS) listen addresses.
func (c *Config) EffectiveHTTPSAddrs() []string {
	var addrs []string
	for _, e := range c.Server.Listen {
		if e.TLS {
			addrs = append(addrs, e.Addr)
		}
	}
	// Backward compat: if no TLS entries but HTTPSPort is set, use it
	if len(addrs) == 0 && c.Server.HTTPSPort > 0 {
		addrs = append(addrs, fmt.Sprintf(":%d", c.Server.HTTPSPort))
	}
	return addrs
}

func (c AuthConfig) JWTSecretValue() string {
	return strings.TrimSpace(c.JWTSecret)
}

func (c AuthConfig) BootstrapPasswordValue() string {
	if strings.TrimSpace(c.BootstrapAdminPassword) == "" {
		return ""
	}
	return c.BootstrapAdminPassword
}

func (c AuthConfig) HasLegacyAdminToken() bool {
	return strings.TrimSpace(c.LegacyAdminToken) != ""
}

func (c AuthConfig) AllowEphemeralJWT() bool {
	if c.AllowEphemeralSecret {
		return true
	}
	switch strings.ToLower(strings.TrimSpace(os.Getenv("APP_ENV"))) {
	case "", "dev", "development", "test":
		return true
	}
	if strings.EqualFold(strings.TrimSpace(os.Getenv("DEBUG")), "true") {
		return true
	}
	return false
}
