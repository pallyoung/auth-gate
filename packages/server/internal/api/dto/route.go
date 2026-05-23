package dto

import (
	"time"

	"github.com/pallyoung/auth-gate/packages/server/internal/store"
)

type Route struct {
	ID           string          `json:"id"`
	Name         string          `json:"name"`
	Host         string          `json:"host"`
	PathPrefix   string          `json:"path_prefix"`
	Backend      string          `json:"backend"`
	StripPrefix  bool            `json:"strip_prefix"`
	Enabled      bool            `json:"enabled"`
	Priority     int             `json:"priority"`
	TLSCert      string          `json:"tls_cert,omitempty"`
	TLSKey       string          `json:"tls_key,omitempty"`
	TLSEnabled   bool            `json:"tls_enabled"`
	Backends     []store.Backend `json:"backends,omitempty"`
	PathMatchMode string         `json:"path_match_mode,omitempty"`
	RewriteTarget string         `json:"rewrite_target,omitempty"`
	RedirectCode  int            `json:"redirect_code,omitempty"`
	CreatedAt    time.Time       `json:"created_at"`
	UpdatedAt    time.Time       `json:"updated_at"`
}

type RouteWriteRequest struct {
	Name          string          `json:"name"`
	Host          string          `json:"host"`
	PathPrefix    string          `json:"path_prefix"`
	Backend       string          `json:"backend" binding:"required"`
	StripPrefix   bool            `json:"strip_prefix"`
	Enabled       bool            `json:"enabled"`
	Priority      int             `json:"priority"`
	TLSCert       string          `json:"tls_cert"`
	TLSKey        string          `json:"tls_key"`
	TLSEnabled    bool            `json:"tls_enabled"`
	Backends      []store.Backend `json:"backends"`
	PathMatchMode string          `json:"path_match_mode"`
	RewriteTarget string          `json:"rewrite_target"`
	RedirectCode  int             `json:"redirect_code"`
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
