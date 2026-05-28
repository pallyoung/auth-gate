package dto

import (
	"time"

	"github.com/pallyoung/auth-gate/packages/server/internal/store"
)

type AuthRuleConfig struct {
	HeaderName string `json:"header_name,omitempty"`
	Username   string `json:"username,omitempty"`
	LoginMode  string `json:"login_mode,omitempty"`
}

type AuthRuleConfigWriteRequest struct {
	HeaderName string `json:"header_name,omitempty"`
	Secret     string `json:"secret,omitempty"`
	Username   string `json:"username,omitempty"`
	Password   string `json:"password,omitempty"`
	LoginMode  string `json:"login_mode,omitempty"`
}

type AuthRuleConfigUpdateRequest struct {
	HeaderName *string `json:"header_name,omitempty"`
	Secret     *string `json:"secret,omitempty"`
	Username   *string `json:"username,omitempty"`
	Password   *string `json:"password,omitempty"`
	LoginMode  *string `json:"login_mode,omitempty"`
}

type AuthRule struct {
	ID                   string         `json:"id"`
	RouteID              string         `json:"route_id"`
	Type                 string         `json:"type"`
	Config               AuthRuleConfig `json:"config"`
	Whitelist            []string       `json:"whitelist,omitempty"`
	RateLimit            int            `json:"rate_limit"`
	Burst                int            `json:"burst"`
	CORSAllowedOrigins   string         `json:"cors_allowed_origins,omitempty"`
	CORSAllowedMethods   string         `json:"cors_allowed_methods,omitempty"`
	CORSAllowedHeaders   string         `json:"cors_allowed_headers,omitempty"`
	CORSAllowCredentials bool           `json:"cors_allow_credentials"`
	CORSMaxAge           int            `json:"cors_max_age"`
	CreatedAt            time.Time      `json:"created_at"`
	UpdatedAt            time.Time      `json:"updated_at"`
}

type AuthRuleCreateRequest struct {
	RouteID              string                     `json:"route_id" binding:"required"`
	Type                 string                     `json:"type"`
	Config               AuthRuleConfigWriteRequest `json:"config"`
	Whitelist            []string                   `json:"whitelist,omitempty"`
	RateLimit            int                        `json:"rate_limit"`
	Burst                int                        `json:"burst"`
	CORSAllowedOrigins   string                     `json:"cors_allowed_origins,omitempty"`
	CORSAllowedMethods   string                     `json:"cors_allowed_methods,omitempty"`
	CORSAllowedHeaders   string                     `json:"cors_allowed_headers,omitempty"`
	CORSAllowCredentials bool                       `json:"cors_allow_credentials"`
	CORSMaxAge           int                        `json:"cors_max_age"`
}

type AuthRuleUpdateRequest struct {
	RouteID              *string                     `json:"route_id,omitempty"`
	Type                 *string                     `json:"type,omitempty"`
	Config               AuthRuleConfigUpdateRequest `json:"config"`
	Whitelist            *[]string                   `json:"whitelist,omitempty"`
	RateLimit            *int                        `json:"rate_limit,omitempty"`
	Burst                *int                        `json:"burst,omitempty"`
	CORSAllowedOrigins   *string                     `json:"cors_allowed_origins,omitempty"`
	CORSAllowedMethods   *string                     `json:"cors_allowed_methods,omitempty"`
	CORSAllowedHeaders   *string                     `json:"cors_allowed_headers,omitempty"`
	CORSAllowCredentials *bool                       `json:"cors_allow_credentials,omitempty"`
	CORSMaxAge           *int                        `json:"cors_max_age,omitempty"`
}

func AuthRuleResponse(rule store.AuthRule) AuthRule {
	return AuthRule{
		ID:      rule.ID,
		RouteID: rule.RouteID,
		Type:    rule.Type,
		Config: AuthRuleConfig{
			HeaderName: rule.Config.HeaderName,
			Username:   rule.Config.Username,
			LoginMode:  rule.Config.LoginMode,
		},
		Whitelist:            rule.Whitelist,
		RateLimit:            rule.RateLimit,
		Burst:                rule.Burst,
		CORSAllowedOrigins:   rule.CORSAllowedOrigins,
		CORSAllowedMethods:   rule.CORSAllowedMethods,
		CORSAllowedHeaders:   rule.CORSAllowedHeaders,
		CORSAllowCredentials: rule.CORSAllowCredentials,
		CORSMaxAge:           rule.CORSMaxAge,
		CreatedAt:            rule.CreatedAt,
		UpdatedAt:            rule.UpdatedAt,
	}
}

func AuthRuleListResponse(rules []store.AuthRule) []AuthRule {
	result := make([]AuthRule, 0, len(rules))
	for _, rule := range rules {
		result = append(result, AuthRuleResponse(rule))
	}
	return result
}
