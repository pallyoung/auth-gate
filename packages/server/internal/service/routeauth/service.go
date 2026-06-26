package routeauth

import (
	"database/sql"
	"errors"
	"strings"

	"github.com/pallyoung/auth-gate/packages/server/internal/service/runtime"
	"github.com/pallyoung/auth-gate/packages/server/internal/store"
)

const (
	ErrCodeRouteAuthNotFound  = "route_auth_not_found"
	ErrCodeRouteNotFound      = "route_not_found"
	ErrCodeRouteAuthStoreFailure = "route_auth_store_failure"
)

type Error struct {
	code    string
	message string
	cause   error
}

func (e *Error) Error() string   { return e.message }
func (e *Error) Unwrap() error   { return e.cause }
func Code(err error) string {
	var target *Error
	if errors.As(err, &target) {
		return target.code
	}
	return ""
}

func newError(code, message string, cause error) error {
	return &Error{code: code, message: message, cause: cause}
}

type UpdateInput struct {
	ApiKeyEnabled    *bool
	ApiKeyHeader     *string
	GatewayEnabled   *bool
	GatewayLoginMode *string
	Whitelist        []string
	RateLimit        *int
	Burst            *int
	CORSAllowedOrigins   *string
	CORSAllowedMethods   *string
	CORSAllowedHeaders   *string
	CORSAllowCredentials *bool
	CORSMaxAge           *int
}

type Service struct {
	db       store.Store
	reloader runtime.Reloader
}

func NewService(db store.Store, reloader runtime.Reloader) *Service {
	return &Service{db: db, reloader: reloader}
}

func (s *Service) Get(routeID string) (*store.RouteAuthConfig, error) {
	cfg, err := s.db.GetRouteAuthConfig(routeID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			// Return empty config (all disabled)
			return &store.RouteAuthConfig{RouteID: routeID}, nil
		}
		return nil, newError(ErrCodeRouteAuthStoreFailure, "failed to get route auth config", err)
	}
	return cfg, nil
}

func (s *Service) Update(routeID string, input UpdateInput) (*store.RouteAuthConfig, error) {
	// Verify route exists
	if _, err := s.db.GetRoute(routeID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, newError(ErrCodeRouteNotFound, "route not found", err)
		}
		return nil, newError(ErrCodeRouteAuthStoreFailure, "failed to verify route", err)
	}

	// Load existing or start fresh
	existing, err := s.db.GetRouteAuthConfig(routeID)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, newError(ErrCodeRouteAuthStoreFailure, "failed to get route auth config", err)
	}
	if existing == nil {
		existing = &store.RouteAuthConfig{RouteID: routeID}
	}

	// Merge fields
	if input.ApiKeyEnabled != nil {
		existing.ApiKeyEnabled = *input.ApiKeyEnabled
	}
	if input.ApiKeyHeader != nil {
		existing.ApiKeyHeader = strings.TrimSpace(*input.ApiKeyHeader)
	}
	if input.GatewayEnabled != nil {
		existing.GatewayEnabled = *input.GatewayEnabled
	}
	if input.GatewayLoginMode != nil {
		existing.GatewayLoginMode = strings.TrimSpace(*input.GatewayLoginMode)
	}
	if input.Whitelist != nil {
		existing.Whitelist = normalizeStringSlice(input.Whitelist)
	}
	if input.RateLimit != nil {
		existing.RateLimit = *input.RateLimit
	}
	if input.Burst != nil {
		existing.Burst = *input.Burst
	}
	if input.CORSAllowedOrigins != nil {
		existing.CORSAllowedOrigins = normalizeCommaSeparated(*input.CORSAllowedOrigins)
	}
	if input.CORSAllowedMethods != nil {
		existing.CORSAllowedMethods = normalizeCommaSeparated(*input.CORSAllowedMethods)
	}
	if input.CORSAllowedHeaders != nil {
		existing.CORSAllowedHeaders = normalizeCommaSeparated(*input.CORSAllowedHeaders)
	}
	if input.CORSAllowCredentials != nil {
		existing.CORSAllowCredentials = *input.CORSAllowCredentials
	}
	if input.CORSMaxAge != nil {
		existing.CORSMaxAge = *input.CORSMaxAge
	}

	// Default gateway login mode
	if existing.GatewayEnabled && existing.GatewayLoginMode == "" {
		existing.GatewayLoginMode = "form"
	}

	if err := s.db.CreateOrUpdateRouteAuthConfig(existing); err != nil {
		return nil, newError(ErrCodeRouteAuthStoreFailure, "failed to save route auth config", err)
	}

	s.reload()
	return existing, nil
}

func (s *Service) Delete(routeID string) error {
	if err := s.db.DeleteRouteAuthConfig(routeID); err != nil {
		return newError(ErrCodeRouteAuthStoreFailure, "failed to delete route auth config", err)
	}
	s.reload()
	return nil
}

func (s *Service) reload() {
	if s.reloader != nil {
		s.reloader.Reload()
	}
}

func normalizeStringSlice(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	normalized := make([]string, 0, len(values))
	for _, v := range values {
		trimmed := strings.TrimSpace(v)
		if trimmed != "" {
			normalized = append(normalized, trimmed)
		}
	}
	if len(normalized) == 0 {
		return nil
	}
	return normalized
}

func normalizeCommaSeparated(value string) string {
	parts := strings.Split(value, ",")
	normalized := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			normalized = append(normalized, trimmed)
		}
	}
	return strings.Join(normalized, ",")
}
