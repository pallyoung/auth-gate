package dto

import (
	"time"

	"github.com/pallyoung/auth-gate/packages/server/internal/store"
)

type Route struct {
	ID            string          `json:"id"`
	Name          string          `json:"name"`
	Host          string          `json:"host"`
	PathPrefix    string          `json:"path_prefix"`
	Backend       string          `json:"backend"`
	StripPrefix   bool            `json:"strip_prefix"`
	Enabled       bool            `json:"enabled"`
	Priority      int             `json:"priority"`
	TLSCert       string          `json:"tls_cert,omitempty"`
	TLSKey        string          `json:"tls_key,omitempty"`
	TLSEnabled    bool            `json:"tls_enabled"`
	HTTPSRedirect bool            `json:"https_redirect"`
	CertificateID string          `json:"certificate_id,omitempty"`
	TimeoutMs     int             `json:"timeout_ms,omitempty"`
	RetryAttempts int             `json:"retry_attempts,omitempty"`
	Backends      []store.Backend `json:"backends,omitempty"`
	PathMatchMode string          `json:"path_match_mode,omitempty"`
	RewriteTarget string          `json:"rewrite_target,omitempty"`
	RedirectCode  int             `json:"redirect_code,omitempty"`
	CreatedAt     time.Time       `json:"created_at"`
	UpdatedAt     time.Time       `json:"updated_at"`
}

type RouteCreateRequest struct {
	Name          string          `json:"name"`
	Host          string          `json:"host"`
	PathPrefix    string          `json:"path_prefix"`
	Backend       string          `json:"backend"`
	StripPrefix   bool            `json:"strip_prefix"`
	Enabled       bool            `json:"enabled"`
	Priority      int             `json:"priority"`
	TLSCert       string          `json:"tls_cert"`
	TLSKey        string          `json:"tls_key"`
	TLSEnabled    bool            `json:"tls_enabled"`
	HTTPSRedirect bool            `json:"https_redirect"`
	CertificateID string          `json:"certificate_id"`
	TimeoutMs     int             `json:"timeout_ms"`
	RetryAttempts int             `json:"retry_attempts"`
	Backends      []store.Backend `json:"backends"`
	PathMatchMode string          `json:"path_match_mode"`
	RewriteTarget string          `json:"rewrite_target"`
	RedirectCode  int             `json:"redirect_code"`
}

type RouteUpdateRequest struct {
	Name          *string          `json:"name,omitempty"`
	Host          *string          `json:"host,omitempty"`
	PathPrefix    *string          `json:"path_prefix,omitempty"`
	Backend       *string          `json:"backend,omitempty"`
	StripPrefix   *bool            `json:"strip_prefix,omitempty"`
	Enabled       *bool            `json:"enabled,omitempty"`
	Priority      *int             `json:"priority,omitempty"`
	TLSCert       *string          `json:"tls_cert,omitempty"`
	TLSKey        *string          `json:"tls_key,omitempty"`
	TLSEnabled    *bool            `json:"tls_enabled,omitempty"`
	HTTPSRedirect *bool            `json:"https_redirect,omitempty"`
	CertificateID *string          `json:"certificate_id,omitempty"`
	TimeoutMs     *int             `json:"timeout_ms,omitempty"`
	RetryAttempts *int             `json:"retry_attempts,omitempty"`
	Backends      *[]store.Backend `json:"backends,omitempty"`
	PathMatchMode *string          `json:"path_match_mode,omitempty"`
	RewriteTarget *string          `json:"rewrite_target,omitempty"`
	RedirectCode  *int             `json:"redirect_code,omitempty"`
}

func RouteResponse(route store.Route) Route {
	return Route{
		ID:            route.ID,
		Name:          route.Name,
		Host:          route.Host,
		PathPrefix:    route.PathPrefix,
		Backend:       route.Backend,
		StripPrefix:   route.StripPrefix,
		Enabled:       route.Enabled,
		Priority:      route.Priority,
		TLSCert:       route.TLSCert,
		TLSKey:        route.TLSKey,
		TLSEnabled:    route.TLSEnabled,
		HTTPSRedirect: route.HTTPSRedirect,
		CertificateID: route.CertificateID,
		TimeoutMs:     route.TimeoutMs,
		RetryAttempts: route.RetryAttempts,
		Backends:      route.Backends,
		PathMatchMode: route.PathMatchMode,
		RewriteTarget: route.RewriteTarget,
		RedirectCode:  route.RedirectCode,
		CreatedAt:     route.CreatedAt,
		UpdatedAt:     route.UpdatedAt,
	}
}

func RouteListResponse(routes []store.Route) []Route {
	result := make([]Route, 0, len(routes))
	for _, route := range routes {
		result = append(result, RouteResponse(route))
	}
	return result
}
