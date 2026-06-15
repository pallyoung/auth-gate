package config

import (
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Server   ServerConfig   `yaml:"server"`
	Database DatabaseConfig `yaml:"database"`
	Auth     AuthConfig     `yaml:"auth"`
}

type ServerConfig struct {
	Addr string `yaml:"addr"`
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

	return &cfg, nil
}

func DefaultConfig() *Config {
	return &Config{
		Server: ServerConfig{
			Addr: ":8080",
		},
		Database: DatabaseConfig{
			Path: "data",
		},
		Auth: AuthConfig{},
	}
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
