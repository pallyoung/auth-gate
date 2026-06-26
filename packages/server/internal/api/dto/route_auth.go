package dto

import (
	"time"

	"github.com/pallyoung/auth-gate/packages/server/internal/store"
)

// ---- Route Auth Config ----

type RouteAuthConfigResponse struct {
	RouteID          string   `json:"route_id"`
	ApiKeyEnabled    bool     `json:"api_key_enabled"`
	ApiKeyHeader     string   `json:"api_key_header,omitempty"`
	GatewayEnabled   bool     `json:"gateway_enabled"`
	GatewayLoginMode string   `json:"gateway_login_mode,omitempty"`
	Whitelist        []string `json:"whitelist,omitempty"`
	RateLimit        int      `json:"rate_limit"`
	Burst            int      `json:"burst"`
	CORSAllowedOrigins   string `json:"cors_allowed_origins,omitempty"`
	CORSAllowedMethods   string `json:"cors_allowed_methods,omitempty"`
	CORSAllowedHeaders   string `json:"cors_allowed_headers,omitempty"`
	CORSAllowCredentials bool   `json:"cors_allow_credentials"`
	CORSMaxAge           int    `json:"cors_max_age"`
}

type RouteAuthConfigUpdateRequest struct {
	ApiKeyEnabled    *bool    `json:"api_key_enabled,omitempty"`
	ApiKeyHeader     *string  `json:"api_key_header,omitempty"`
	GatewayEnabled   *bool    `json:"gateway_enabled,omitempty"`
	GatewayLoginMode *string  `json:"gateway_login_mode,omitempty"`
	Whitelist        []string `json:"whitelist,omitempty"`
	RateLimit        *int     `json:"rate_limit,omitempty"`
	Burst            *int     `json:"burst,omitempty"`
	CORSAllowedOrigins   *string `json:"cors_allowed_origins,omitempty"`
	CORSAllowedMethods   *string `json:"cors_allowed_methods,omitempty"`
	CORSAllowedHeaders   *string `json:"cors_allowed_headers,omitempty"`
	CORSAllowCredentials *bool   `json:"cors_allow_credentials,omitempty"`
	CORSMaxAge           *int    `json:"cors_max_age,omitempty"`
}

func RouteAuthConfigResponseFromStore(cfg store.RouteAuthConfig) RouteAuthConfigResponse {
	return RouteAuthConfigResponse{
		RouteID:              cfg.RouteID,
		ApiKeyEnabled:        cfg.ApiKeyEnabled,
		ApiKeyHeader:         cfg.ApiKeyHeader,
		GatewayEnabled:       cfg.GatewayEnabled,
		GatewayLoginMode:     cfg.GatewayLoginMode,
		Whitelist:            cfg.Whitelist,
		RateLimit:            cfg.RateLimit,
		Burst:                cfg.Burst,
		CORSAllowedOrigins:   cfg.CORSAllowedOrigins,
		CORSAllowedMethods:   cfg.CORSAllowedMethods,
		CORSAllowedHeaders:   cfg.CORSAllowedHeaders,
		CORSAllowCredentials: cfg.CORSAllowCredentials,
		CORSMaxAge:           cfg.CORSMaxAge,
	}
}

// ---- API Keys ----

type ApiKeyResponse struct {
	ID         string     `json:"id"`
	RouteID    string     `json:"route_id"`
	Name       string     `json:"name"`
	KeyPrefix  string     `json:"key_prefix"`
	ExpiresAt  *time.Time `json:"expires_at"`
	Status     string     `json:"status"`
	LastUsedAt *time.Time `json:"last_used_at"`
	CreatedAt  time.Time  `json:"created_at"`
}

type ApiKeyCreateResponse struct {
	ApiKeyResponse
	Secret string `json:"secret"` // 仅创建时返回
}

// ApiKeyListItemResponse is used for list endpoints where the full secret
// should be visible and copyable by admins.
type ApiKeyListItemResponse struct {
	ApiKeyResponse
	Secret string `json:"secret"`
}

type ApiKeyCreateRequest struct {
	Name      string     `json:"name"`
	ExpiresAt *time.Time `json:"expires_at"`
}

type ApiKeyUpdateRequest struct {
	Name *string `json:"name,omitempty"`
}

func ApiKeyResponseFromKey(k store.ApiKey) ApiKeyResponse {
	return ApiKeyResponse{
		ID:         k.ID,
		RouteID:    k.RouteID,
		Name:       k.Name,
		KeyPrefix:  k.KeyPrefix,
		ExpiresAt:  k.ExpiresAt,
		Status:     k.Status,
		LastUsedAt: k.LastUsedAt,
		CreatedAt:  k.CreatedAt,
	}
}

func ApiKeyListResponse(keys []store.ApiKey) []ApiKeyResponse {
	result := make([]ApiKeyResponse, 0, len(keys))
	for _, k := range keys {
		result = append(result, ApiKeyResponseFromKey(k))
	}
	return result
}

func ApiKeyListWithSecretsResponse(keys []store.ApiKey) []ApiKeyListItemResponse {
	result := make([]ApiKeyListItemResponse, 0, len(keys))
	for _, k := range keys {
		result = append(result, ApiKeyListItemResponse{
			ApiKeyResponse: ApiKeyResponseFromKey(k),
			Secret:         k.Secret,
		})
	}
	return result
}
