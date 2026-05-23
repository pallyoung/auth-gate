package store

import (
	"encoding/json"
	"time"
)

type Backend struct {
	URL             string `json:"url"`
	Weight          int    `json:"weight"`
	DialTimeoutMs   int    `json:"dial_timeout_ms,omitempty"`
	ReadTimeoutMs  int    `json:"read_timeout_ms,omitempty"`
	WriteTimeoutMs int    `json:"write_timeout_ms,omitempty"`
	MaxIdleConns    int    `json:"max_idle_conns,omitempty"`
	RewriteTarget   string `json:"rewrite_target,omitempty"`
	RedirectCode    int    `json:"redirect_code,omitempty"`
}

type Route struct {
	ID             string    `json:"id"`
	Name           string    `json:"name"`
	Host           string    `json:"host"`
	PathPrefix     string    `json:"path_prefix"`
	Backend        string    `json:"backend"` // 单后端兼容（无Backends时用）
	Backends       []Backend `json:"backends,omitempty"` // 多后端负载均衡
	StripPrefix    bool      `json:"strip_prefix"`
	Enabled        bool      `json:"enabled"`
	Priority       int       `json:"priority"`
	TLSCert        string    `json:"tls_cert,omitempty"`
	TLSKey         string    `json:"tls_key,omitempty"`
	TLSEnabled     bool      `json:"tls_enabled"`
	TimeoutMs      int       `json:"timeout_ms,omitempty"`
	RetryAttempts  int       `json:"retry_attempts,omitempty"`
	PathMatchMode   string    `json:"path_match_mode,omitempty"` // "prefix"|"exact"|"regex"
	RewriteTarget   string    `json:"rewrite_target,omitempty"`  // rewrite target (e.g. /new$1)
	RedirectCode    int       `json:"redirect_code,omitempty"`   // 301|302 for external redirects
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

func (r *Route) EffectiveBackends() []Backend {
	if len(r.Backends) > 0 {
		return r.Backends
	}
	if r.Backend != "" {
		return []Backend{{URL: r.Backend, Weight: 1}}
	}
	return nil
}

type AuthRule struct {
	ID         string     `json:"id"`
	RouteID    string     `json:"route_id"`
	Type       string     `json:"type"` // none, apikey, bearer, basic
	Config     AuthConfig `json:"config"`
	Whitelist  []string   `json:"whitelist"`  // IPs/ CIDRs excluded from rate limiting
	RateLimit  int        `json:"rate_limit"` // max requests per second
	Burst      int        `json:"burst"`      // allowed burst size
	// CORS configuration
	CORSAllowedOrigins string `json:"cors_allowed_origins,omitempty"` // comma-separated or "*"
	CORSAllowedMethods string `json:"cors_allowed_methods,omitempty"` // e.g. "GET,POST,OPTIONS"
	CORSAllowedHeaders string `json:"cors_allowed_headers,omitempty"` // e.g. "Authorization,Content-Type"
	CORSAllowCredentials bool   `json:"cors_allow_credentials"`
	CORSMaxAge           int    `json:"cors_max_age"` // seconds, default 86400
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
}

type AuthConfig struct {
	HeaderName string `json:"header_name,omitempty"`
	Secret     string `json:"secret,omitempty"`
	Username   string `json:"username,omitempty"` // for basic auth
	Password   string `json:"password,omitempty"` // for basic auth
	LoginMode  string `json:"login_mode,omitempty"`
}

func (a AuthConfig) ToJSON() string {
	data, _ := json.Marshal(a)
	return string(data)
}

func ParseAuthConfig(s string) AuthConfig {
	var cfg AuthConfig
	json.Unmarshal([]byte(s), &cfg)
	return cfg
}

// Certificate represents an SSL certificate provisioned via ACME
type Certificate struct {
	ID                string    `json:"id"`
	Name              string    `json:"name"`
	Domain            string    `json:"domain"` // e.g., "*.example.com" or "example.com"
	CertPath          string    `json:"cert_path"`
	KeyPath           string    `json:"key_path"`
	DNSProvider       string    `json:"dns_provider"`        // "cloudflare", "route53", "pdns"
	DNSProviderConfig string    `json:"dns_provider_config"` // encrypted JSON
	Status            string    `json:"status"`            // "pending", "active", "renewing", "failed"
	NotBefore         time.Time `json:"not_before"`
	NotAfter          time.Time `json:"not_after"`
	RenewAt           time.Time `json:"renew_at"` // NotAfter - 30 days
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

const (
	CertStatusPending  = "pending"
	CertStatusActive   = "active"
	CertStatusRenewing = "renewing"
	CertStatusFailed   = "failed"
)
