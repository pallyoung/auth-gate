package store

import (
	"encoding/json"
	"time"
)

type Route struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Host        string    `json:"host"`
	PathPrefix  string    `json:"path_prefix"`
	Backend     string    `json:"backend"`
	StripPrefix bool      `json:"strip_prefix"`
	Enabled     bool      `json:"enabled"`
	Priority    int       `json:"priority"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type AuthRule struct {
	ID        string     `json:"id"`
	RouteID   string     `json:"route_id"`
	Type       string    `json:"type"` // none, apikey, bearer, basic
	Config     AuthConfig `json:"config"`
	Whitelist  []string  `json:"whitelist"`
	RateLimit  int       `json:"rate_limit"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

type AuthConfig struct {
	HeaderName string `json:"header_name,omitempty"`
	Secret     string `json:"secret,omitempty"`
	JWKSUrl    string `json:"jwks_url,omitempty"`
	Issuer     string `json:"issuer,omitempty"`
	Audience   string `json:"audience,omitempty"`
	Username   string `json:"username,omitempty"` // for basic auth
	Password   string `json:"password,omitempty"` // for basic auth
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
