package routes

import (
	"database/sql"
	"errors"
	"net/url"
	"strings"

	"github.com/pallyoung/auth-gate/packages/server/internal/service/runtime"
	"github.com/pallyoung/auth-gate/packages/server/internal/store"
)

const (
	ErrCodeRouteNotFound         = "route_not_found"
	ErrCodeMissingRouteFields    = "missing_route_fields"
	ErrCodeInvalidRoutePathPrefix = "invalid_route_path_prefix"
	ErrCodeInvalidRouteBackend   = "invalid_route_backend"
	ErrCodeRouteStoreFailure     = "route_store_failure"
)

type Error struct {
	code    string
	message string
	cause   error
}

func (e *Error) Error() string {
	return e.message
}

func (e *Error) Unwrap() error {
	return e.cause
}

func Code(err error) string {
	var target *Error
	if errors.As(err, &target) {
		return target.code
	}
	return ""
}

func newError(code, message string, cause error) error {
	return &Error{
		code:    code,
		message: message,
		cause:   cause,
	}
}

type Service struct {
	db       *store.SQLite
	reloader runtime.Reloader
}

type CreateInput struct {
	Name        string
	Host        string
	PathPrefix  string
	Backend     string
	StripPrefix bool
	Enabled     bool
	Priority    int
}

type UpdateInput = CreateInput

func NewService(db *store.SQLite, reloader runtime.Reloader) *Service {
	return &Service{
		db:       db,
		reloader: reloader,
	}
}

func (s *Service) List() ([]store.Route, error) {
	routes, err := s.db.ListRoutes()
	if err != nil {
		return nil, newError(ErrCodeRouteStoreFailure, "failed to list routes", err)
	}
	return routes, nil
}

func (s *Service) Get(id string) (*store.Route, error) {
	route, err := s.db.GetRoute(id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, newError(ErrCodeRouteNotFound, "route not found", err)
		}
		return nil, newError(ErrCodeRouteStoreFailure, "failed to get route", err)
	}
	return route, nil
}

func (s *Service) Create(input CreateInput) (*store.Route, error) {
	route := &store.Route{
		Name:        strings.TrimSpace(input.Name),
		Host:        strings.TrimSpace(input.Host),
		PathPrefix:  strings.TrimSpace(input.PathPrefix),
		Backend:     strings.TrimSpace(input.Backend),
		StripPrefix: input.StripPrefix,
		Enabled:     input.Enabled,
		Priority:    input.Priority,
	}
	if err := validate(route); err != nil {
		return nil, err
	}
	if err := s.db.CreateRoute(route); err != nil {
		return nil, newError(ErrCodeRouteStoreFailure, "failed to create route", err)
	}
	s.reload()
	return route, nil
}

func (s *Service) Update(id string, input UpdateInput) (*store.Route, error) {
	route := &store.Route{
		ID:          id,
		Name:        strings.TrimSpace(input.Name),
		Host:        strings.TrimSpace(input.Host),
		PathPrefix:  strings.TrimSpace(input.PathPrefix),
		Backend:     strings.TrimSpace(input.Backend),
		StripPrefix: input.StripPrefix,
		Enabled:     input.Enabled,
		Priority:    input.Priority,
	}
	if err := validate(route); err != nil {
		return nil, err
	}
	if err := s.db.UpdateRoute(route); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, newError(ErrCodeRouteNotFound, "route not found", err)
		}
		return nil, newError(ErrCodeRouteStoreFailure, "failed to update route", err)
	}
	s.reload()
	return route, nil
}

func (s *Service) Delete(id string) error {
	if _, err := s.Get(id); err != nil {
		return err
	}
	if err := s.db.DeleteRoute(id); err != nil {
		return newError(ErrCodeRouteStoreFailure, "failed to delete route", err)
	}
	s.reload()
	return nil
}

func (s *Service) reload() {
	if s.reloader != nil {
		s.reloader.Reload()
	}
}

func validate(route *store.Route) error {
	if route.PathPrefix == "" || route.Backend == "" {
		return newError(ErrCodeMissingRouteFields, "path_prefix and backend required", nil)
	}
	if !strings.HasPrefix(route.PathPrefix, "/") {
		return newError(ErrCodeInvalidRoutePathPrefix, "path_prefix must start with /", nil)
	}
	backendURL, err := url.Parse(route.Backend)
	if err != nil || backendURL.Scheme == "" || backendURL.Host == "" {
		return newError(ErrCodeInvalidRouteBackend, "backend must be a valid http or https URL", err)
	}
	if backendURL.Scheme != "http" && backendURL.Scheme != "https" {
		return newError(ErrCodeInvalidRouteBackend, "backend must be a valid http or https URL", nil)
	}
	return nil
}
