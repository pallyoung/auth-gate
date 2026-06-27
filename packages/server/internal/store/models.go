package store

import (
	"encoding/json"
	"log"
	"strings"
	"time"
)

type Backend struct {
	URL            string `json:"url"`
	Weight         int    `json:"weight"`
	DialTimeoutMs  int    `json:"dial_timeout_ms,omitempty"`
	ReadTimeoutMs  int    `json:"read_timeout_ms,omitempty"`
	WriteTimeoutMs int    `json:"write_timeout_ms,omitempty"`
	MaxIdleConns   int    `json:"max_idle_conns,omitempty"`
	RewriteTarget  string `json:"rewrite_target,omitempty"`
	RedirectCode   int    `json:"redirect_code,omitempty"`
}

type Route struct {
	ID            string    `json:"id"`
	Name          string    `json:"name"`
	Host          string    `json:"host"`
	PathPrefix    string    `json:"path_prefix"`
	Backend       string    `json:"backend"`            // 单后端兼容（无Backends时用）
	Backends      []Backend `json:"backends,omitempty"` // 多后端负载均衡
	StripPrefix   bool      `json:"strip_prefix"`
	Enabled       bool      `json:"enabled"`
	Priority      int       `json:"priority"`
	// RouteType determines how requests are handled: "proxy" (default) forwards to
	// a backend server, while "static" serves files from a local directory.
	Type       string `json:"type,omitempty"`        // "proxy" (default) or "static"
	StaticRoot string `json:"static_root,omitempty"` // local directory path for static serving
	StaticSPA  bool   `json:"static_spa,omitempty"`  // SPA fallback: serve index.html on 404
	TLSCert       string    `json:"tls_cert,omitempty"`
	TLSKey        string    `json:"tls_key,omitempty"`
	TLSEnabled    bool      `json:"tls_enabled"`
	HTTPSRedirect bool      `json:"https_redirect,omitempty"` // auto redirect HTTP -> HTTPS
	CertificateID string    `json:"certificate_id,omitempty"` // references Certificate.ID
	TimeoutMs     int       `json:"timeout_ms,omitempty"`
	RetryAttempts int       `json:"retry_attempts,omitempty"`
	PathMatchMode string    `json:"path_match_mode,omitempty"` // "prefix"|"exact"|"regex"
	HeaderName    string    `json:"header_name,omitempty"`     // match request header key
	HeaderValue   string    `json:"header_value,omitempty"`    // match request header value
	RewriteTarget string            `json:"rewrite_target,omitempty"`  // rewrite target (e.g. /new$1)
	RedirectCode  int               `json:"redirect_code,omitempty"`   // 301|302 for external redirects
	// Header manipulation (nil/empty = no-op)
	SetRequestHeaders    map[string]string `json:"set_request_headers,omitempty"`    // add/overwrite before forwarding
	RemoveRequestHeaders []string          `json:"remove_request_headers,omitempty"` // delete before forwarding
	AddResponseHeaders   map[string]string `json:"add_response_headers,omitempty"`   // add to upstream response
	RemoveResponseHeaders []string         `json:"remove_response_headers,omitempty"` // delete from upstream response
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
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
	ID        string     `json:"id"`
	RouteID   string     `json:"route_id"`
	Type      string     `json:"type"` // none, apikey, gateway
	Config    AuthConfig `json:"config"`
	Whitelist []string   `json:"whitelist"`  // IPs/ CIDRs excluded from rate limiting
	RateLimit int        `json:"rate_limit"` // max requests per second
	Burst     int        `json:"burst"`      // allowed burst size
	// CORS configuration
	CORSAllowedOrigins   string    `json:"cors_allowed_origins,omitempty"` // comma-separated or "*"
	CORSAllowedMethods   string    `json:"cors_allowed_methods,omitempty"` // e.g. "GET,POST,OPTIONS"
	CORSAllowedHeaders   string    `json:"cors_allowed_headers,omitempty"` // e.g. "Authorization,Content-Type"
	CORSAllowCredentials bool      `json:"cors_allow_credentials"`
	CORSMaxAge           int       `json:"cors_max_age"` // seconds, default 86400
	CreatedAt            time.Time `json:"created_at"`
	UpdatedAt            time.Time `json:"updated_at"`
}

type AuthConfig struct {
	HeaderName string `json:"header_name,omitempty"`
	Secret     string `json:"secret,omitempty"`
	Username   string `json:"username,omitempty"`
	Password   string `json:"password,omitempty"`
	LoginMode  string `json:"login_mode,omitempty"`
}

func (a AuthConfig) ToJSON() string {
	data, _ := json.Marshal(a)
	return string(data)
}

func ParseAuthConfig(s string) AuthConfig {
	var cfg AuthConfig
	if err := json.Unmarshal([]byte(s), &cfg); err != nil {
		log.Printf("warning: malformed auth config JSON: %v", err)
	}
	cfg.HeaderName = strings.TrimSpace(cfg.HeaderName)
	cfg.Secret = strings.TrimSpace(cfg.Secret)
	cfg.Username = strings.TrimSpace(cfg.Username)
	cfg.LoginMode = strings.TrimSpace(cfg.LoginMode)
	return cfg
}

// ApiKey represents a named API key for route-level authentication.
// Multiple ApiKeys can exist per route, each with its own expiration and lifecycle.
type ApiKey struct {
	ID         string     `json:"id"`
	RouteID    string     `json:"route_id"`
	Name       string     `json:"name"`
	KeyPrefix  string     `json:"key_prefix"`  // first 8 chars for display
	Secret     string     `json:"-"`           // full secret, never serialized to JSON
	ExpiresAt  *time.Time `json:"expires_at"`  // nil = never expires
	Status     string     `json:"status"`      // active / expired / revoked
	LastUsedAt *time.Time `json:"last_used_at"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
}

// RouteAuthConfig holds the authentication configuration for a single route.
// Each route has at most one config. Auth methods are toggled on/off independently;
// when multiple methods are enabled, any one passing is sufficient (OR logic).
type RouteAuthConfig struct {
	ID      string `json:"id"`
	RouteID string `json:"route_id" binding:"required"`

	// API Key authentication toggle
	ApiKeyEnabled bool   `json:"api_key_enabled"`
	ApiKeyHeader  string `json:"api_key_header,omitempty"` // default "X-API-Key"

	// Gateway login toggle
	GatewayEnabled   bool   `json:"gateway_enabled"`
	GatewayLoginMode string `json:"gateway_login_mode,omitempty"` // "form"

	// Shared runtime policy (applies to all enabled auth methods)
	Whitelist            []string `json:"whitelist,omitempty"`
	RateLimit            int      `json:"rate_limit"`
	Burst                int      `json:"burst"`
	CORSAllowedOrigins   string   `json:"cors_allowed_origins,omitempty"`
	CORSAllowedMethods   string   `json:"cors_allowed_methods,omitempty"`
	CORSAllowedHeaders   string   `json:"cors_allowed_headers,omitempty"`
	CORSAllowCredentials bool     `json:"cors_allow_credentials"`
	CORSMaxAge           int      `json:"cors_max_age"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// HasAuth returns true if any authentication method is enabled.
func (c *RouteAuthConfig) HasAuth() bool {
	return c.ApiKeyEnabled || c.GatewayEnabled
}

// IsApiKeyExpired returns true if the given ApiKey is expired or revoked.
func IsApiKeyExpired(key *ApiKey) bool {
	if key.Status != "active" {
		return true
	}
	if key.ExpiresAt != nil && key.ExpiresAt.Before(time.Now()) {
		return true
	}
	return false
}

// Certificate represents an SSL certificate in the registry. Certificates may
// be issued by the bundled local CA (Source = "local_ca") or imported from
// external PEM files (Source = "imported").
type Certificate struct {
	ID                 string    `json:"id"`
	Name               string    `json:"name"`
	Domain             string    `json:"domain"` // e.g., "*.example.com" or "example.com"
	CertPath           string    `json:"cert_path"`
	KeyPath            string    `json:"key_path"`
	Source             string    `json:"source"`   // "local_ca" or "imported"
	CAID               string    `json:"ca_id"`    // empty when Source = "imported"
	Status             string    `json:"status"`   // "active" or "failed"
	Organization       string    `json:"organization,omitempty"`
	OrganizationalUnit string    `json:"organizational_unit,omitempty"`
	Country            string    `json:"country,omitempty"`
	Province           string    `json:"province,omitempty"`
	Locality           string    `json:"locality,omitempty"`
	NotBefore          time.Time `json:"not_before"`
	NotAfter           time.Time `json:"not_after"`
	RenewAt            time.Time `json:"renew_at"` // NotAfter - 30 days; zero for imported
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
}

const (
	SourceLocalCA = "local_ca"
	SourceImported = "imported"

	CertStatusActive = "active"
	CertStatusFailed = "failed"
)

// HostProfile is a named collection of /etc/hosts entries. At most one profile
// may be active at a time; the mutex is enforced at the service layer.
type HostProfile struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	IsActive    bool      `json:"is_active"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// HostEntry is a single IP/hostnames line belonging to a HostProfile.
type HostEntry struct {
	ID        string    `json:"id"`
	ProfileID string    `json:"profile_id"`
	Position  int       `json:"position"`
	IP        string    `json:"ip"`
	Hostnames string    `json:"hostnames"` // space-separated
	Comment   string    `json:"comment"`
	Enabled   bool      `json:"enabled"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// PermissionGroup is a named set of per-route path permissions that can be
// assigned to multiple users, avoiding repetitive per-user configuration.
type PermissionGroup struct {
	ID         string              `json:"id"`
	Name       string              `json:"name"`
	RouteIDs   []string            `json:"route_ids,omitempty"`  // routes this group grants access to
	RoutePaths map[string][]string `json:"route_paths"`          // routeID -> allowed paths
	CreatedAt  time.Time           `json:"created_at"`
	UpdatedAt  time.Time           `json:"updated_at"`
}
