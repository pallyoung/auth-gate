package routes

import (
	"database/sql"
	"errors"
	"net/http"
	"net/url"
	"regexp"
	"strings"

	"github.com/pallyoung/auth-gate/packages/server/internal/routehost"
	"github.com/pallyoung/auth-gate/packages/server/internal/service/runtime"
	"github.com/pallyoung/auth-gate/packages/server/internal/store"
)

const (
	ErrCodeRouteNotFound             = "route_not_found"
	ErrCodeMissingRouteFields        = "missing_route_fields"
	ErrCodeInvalidRoutePathPrefix    = "invalid_route_path_prefix"
	ErrCodeInvalidRoutePathMatchMode = "invalid_route_path_match_mode"
	ErrCodeInvalidRoutePathRegex     = "invalid_route_path_regex"
	ErrCodeReservedRoutePathPrefix   = "reserved_route_path_prefix"
	ErrCodeInvalidRouteHost          = "invalid_route_host"
	ErrCodeInvalidRouteBackend       = "invalid_route_backend"
	ErrCodeInvalidRouteBackendWeight = "invalid_route_backend_weight"
	ErrCodeInvalidRouteRedirectCode  = "invalid_route_redirect_code"
	ErrCodeRouteStoreFailure         = "route_store_failure"
	ErrCodeCertificateNotFound       = "certificate_not_found"
)

const controlPlaneReservedPathPrefix = "/_authgate"

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

// CertService is the subset of certificate service that the route service
// needs to resolve certificate paths from a certificate ID.
type CertService interface {
	Get(id string) (*store.Certificate, error)
}

type Service struct {
	db       store.Store
	reloader runtime.Reloader
	certSvc  CertService
}

type CreateInput struct {
	Name          string
	Host          string
	PathPrefix    string
	Backend       string
	StripPrefix   bool
	Enabled       bool
	Priority      int
	// RouteType: "proxy" (default) or "static"
	Type       string
	StaticRoot string
	StaticSPA  bool
	TLSCert       string
	TLSKey        string
	TLSEnabled    bool
	HTTPSRedirect bool
	CertificateID string
	TimeoutMs     int
	RetryAttempts int
	Backends      []store.Backend
	PathMatchMode string
	HeaderName    string
	HeaderValue   string
	RewriteTarget string
	RedirectCode  int
	// Header manipulation
	SetRequestHeaders     map[string]string
	RemoveRequestHeaders  []string
	AddResponseHeaders    map[string]string
	RemoveResponseHeaders []string
}

type UpdateInput struct {
	Name          *string
	Host          *string
	PathPrefix    *string
	Backend       *string
	StripPrefix   *bool
	Enabled       *bool
	Priority      *int
	// RouteType: "proxy" (default) or "static"
	Type       *string
	StaticRoot *string
	StaticSPA  *bool
	TLSCert       *string
	TLSKey        *string
	TLSEnabled    *bool
	HTTPSRedirect *bool
	CertificateID *string
	TimeoutMs     *int
	RetryAttempts *int
	Backends      *[]store.Backend
	PathMatchMode *string
	HeaderName    *string
	HeaderValue   *string
	RewriteTarget *string
	RedirectCode  *int
	// Header manipulation
	SetRequestHeaders     *map[string]string
	RemoveRequestHeaders  *[]string
	AddResponseHeaders    *map[string]string
	RemoveResponseHeaders *[]string
}

func NewService(db store.Store, reloader runtime.Reloader, certSvc CertService) *Service {
	return &Service{
		db:       db,
		reloader: reloader,
		certSvc:  certSvc,
	}
}

func (s *Service) List() ([]store.Route, error) {
	routes, err := s.db.ListRoutes()
	if err != nil {
		return nil, newError(ErrCodeRouteStoreFailure, "failed to list routes", err)
	}
	for i := range routes {
		routes[i] = normalizeStoredRoute(routes[i])
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
	normalized := normalizeStoredRoute(*route)
	return &normalized, nil
}

func (s *Service) Create(input CreateInput) (*store.Route, error) {
	route := &store.Route{
		Name:          strings.TrimSpace(input.Name),
		Host:          normalizeHost(input.Host),
		PathPrefix:    strings.TrimSpace(input.PathPrefix),
		Backend:       strings.TrimSpace(input.Backend),
		StripPrefix:   input.StripPrefix,
		Enabled:       input.Enabled,
		Priority:      input.Priority,
		Type:          normalizeRouteType(input.Type),
		StaticRoot:    strings.TrimSpace(input.StaticRoot),
		StaticSPA:     input.StaticSPA,
		TLSCert:       strings.TrimSpace(input.TLSCert),
		TLSKey:        strings.TrimSpace(input.TLSKey),
		TLSEnabled:    input.TLSEnabled,
		HTTPSRedirect: input.HTTPSRedirect,
		CertificateID: strings.TrimSpace(input.CertificateID),
		TimeoutMs:     input.TimeoutMs,
		RetryAttempts: input.RetryAttempts,
		Backends:      input.Backends,
		PathMatchMode: normalizePathMatchMode(input.PathMatchMode),
		HeaderName:    strings.TrimSpace(input.HeaderName),
		HeaderValue:   strings.TrimSpace(input.HeaderValue),
		RewriteTarget: strings.TrimSpace(input.RewriteTarget),
		RedirectCode:  input.RedirectCode,
		// Header manipulation
		SetRequestHeaders:     normalizeHeaderMap(input.SetRequestHeaders),
		RemoveRequestHeaders:  normalizeHeaderNames(input.RemoveRequestHeaders),
		AddResponseHeaders:    normalizeHeaderMap(input.AddResponseHeaders),
		RemoveResponseHeaders: normalizeHeaderNames(input.RemoveResponseHeaders),
	}
	if err := s.resolveCertificate(route); err != nil {
		return nil, err
	}
	route.Backends = normalizeBackends(route.Backends)
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
	route, err := s.Get(id)
	if err != nil {
		return nil, err
	}

	if input.Name != nil {
		route.Name = strings.TrimSpace(*input.Name)
	}
	if input.Host != nil {
		route.Host = normalizeHost(*input.Host)
	}
	if input.PathPrefix != nil {
		route.PathPrefix = strings.TrimSpace(*input.PathPrefix)
	}
	if input.Backend != nil {
		route.Backend = strings.TrimSpace(*input.Backend)
	}
	if input.StripPrefix != nil {
		route.StripPrefix = *input.StripPrefix
	}
	if input.Enabled != nil {
		route.Enabled = *input.Enabled
	}
	if input.Priority != nil {
		route.Priority = *input.Priority
	}
	if input.Type != nil {
		route.Type = normalizeRouteType(*input.Type)
	}
	if input.StaticRoot != nil {
		route.StaticRoot = strings.TrimSpace(*input.StaticRoot)
	}
	if input.StaticSPA != nil {
		route.StaticSPA = *input.StaticSPA
	}
	if input.TLSCert != nil {
		route.TLSCert = strings.TrimSpace(*input.TLSCert)
	}
	if input.TLSKey != nil {
		route.TLSKey = strings.TrimSpace(*input.TLSKey)
	}
	if input.TLSEnabled != nil {
		route.TLSEnabled = *input.TLSEnabled
	}
	if input.HTTPSRedirect != nil {
		route.HTTPSRedirect = *input.HTTPSRedirect
	}
	if input.CertificateID != nil {
		route.CertificateID = strings.TrimSpace(*input.CertificateID)
	}
	if input.TimeoutMs != nil {
		route.TimeoutMs = *input.TimeoutMs
	}
	if input.RetryAttempts != nil {
		route.RetryAttempts = *input.RetryAttempts
	}
	if input.Backends != nil {
		route.Backends = *input.Backends
	}
	if input.PathMatchMode != nil {
		route.PathMatchMode = normalizePathMatchMode(*input.PathMatchMode)
	}
	if input.HeaderName != nil {
		route.HeaderName = strings.TrimSpace(*input.HeaderName)
	}
	if input.HeaderValue != nil {
		route.HeaderValue = strings.TrimSpace(*input.HeaderValue)
	}
	if input.RewriteTarget != nil {
		route.RewriteTarget = strings.TrimSpace(*input.RewriteTarget)
	}
	if input.RedirectCode != nil {
		route.RedirectCode = *input.RedirectCode
	}
	// Header manipulation
	if input.SetRequestHeaders != nil {
		route.SetRequestHeaders = normalizeHeaderMap(*input.SetRequestHeaders)
	}
	if input.RemoveRequestHeaders != nil {
		route.RemoveRequestHeaders = normalizeHeaderNames(*input.RemoveRequestHeaders)
	}
	if input.AddResponseHeaders != nil {
		route.AddResponseHeaders = normalizeHeaderMap(*input.AddResponseHeaders)
	}
	if input.RemoveResponseHeaders != nil {
		route.RemoveResponseHeaders = normalizeHeaderNames(*input.RemoveResponseHeaders)
	}
	route.Backends = normalizeBackends(route.Backends)

	if err := s.resolveCertificate(route); err != nil {
		return nil, err
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
	if route.Type == "static" {
		// Static routes require a root directory, no backend needed
		if route.StaticRoot == "" {
			return newError(ErrCodeMissingRouteFields, "static_root is required for static routes", nil)
		}
	} else {
		// Proxy routes require a backend
		if route.Backend == "" && len(route.Backends) == 0 {
			return newError(ErrCodeMissingRouteFields, "backend or backends required", nil)
		}
	}
	if !isValidPathMatchMode(route.PathMatchMode) {
		return newError(ErrCodeInvalidRoutePathMatchMode, "path_match_mode must be one of prefix, exact, stop, regex, or regex_i", nil)
	}
	if isRegexPathMatchMode(route.PathMatchMode) {
		if _, err := compileRoutePathRegex(route.PathMatchMode, route.PathPrefix); err != nil {
			return newError(ErrCodeInvalidRoutePathRegex, "path_prefix must be a valid regular expression for the selected path match mode", err)
		}
	}
	if requiresLeadingSlash(route.PathMatchMode) && route.PathPrefix != "" && !strings.HasPrefix(route.PathPrefix, "/") {
		return newError(ErrCodeInvalidRoutePathPrefix, "path_prefix must start with /", nil)
	}
	if route.PathPrefix == controlPlaneReservedPathPrefix || strings.HasPrefix(route.PathPrefix, controlPlaneReservedPathPrefix+"/") {
		return newError(ErrCodeReservedRoutePathPrefix, "path_prefix conflicts with reserved control-plane paths", nil)
	}
	if !routehost.IsValid(route.Host) {
		return newError(ErrCodeInvalidRouteHost, routehost.InvalidMessage, nil)
	}
	// Backend validation only applies to proxy routes
	if route.Type != "static" {
		if route.Backend != "" {
			if err := validateBackendURL(route.Backend); err != nil {
				return err
			}
		}
		for _, backend := range route.Backends {
			if err := validateBackendURL(backend.URL); err != nil {
				return err
			}
			if backend.Weight <= 0 {
				return newError(ErrCodeInvalidRouteBackendWeight, "backend weight must be greater than 0", nil)
			}
		}
	}
	if !isValidRedirectCode(route.RedirectCode) {
		return newError(ErrCodeInvalidRouteRedirectCode, "redirect_code must be 0, 301, or 302", nil)
	}
	return nil
}

func isValidRedirectCode(code int) bool {
	return code == 0 || code == http.StatusMovedPermanently || code == http.StatusFound
}

func isValidPathMatchMode(pathMatchMode string) bool {
	switch pathMatchMode {
	case "", "exact", "stop", "regex", "regex_i":
		return true
	default:
		return false
	}
}

func isRegexPathMatchMode(pathMatchMode string) bool {
	return pathMatchMode == "regex" || pathMatchMode == "regex_i"
}

func compileRoutePathRegex(pathMatchMode, pathPrefix string) (*regexp.Regexp, error) {
	pattern := pathPrefix
	if pathMatchMode == "regex_i" {
		pattern = "(?i)" + pattern
	}
	return regexp.Compile(pattern)
}

func validateBackendURL(raw string) error {
	backendURL, err := url.Parse(raw)
	if err != nil || backendURL.Scheme == "" || backendURL.Host == "" {
		return newError(ErrCodeInvalidRouteBackend, "backend must be a valid http or https URL", err)
	}
	if backendURL.Scheme != "http" && backendURL.Scheme != "https" {
		return newError(ErrCodeInvalidRouteBackend, "backend must be a valid http or https URL", nil)
	}
	return nil
}

func normalizeBackends(backends []store.Backend) []store.Backend {
	if len(backends) == 0 {
		return backends
	}

	normalized := make([]store.Backend, len(backends))
	for i, backend := range backends {
		backend.URL = strings.TrimSpace(backend.URL)
		normalized[i] = backend
	}
	return normalized
}

func requiresLeadingSlash(pathMatchMode string) bool {
	switch pathMatchMode {
	case "regex", "regex_i":
		return false
	default:
		return true
	}
}

func normalizePathMatchMode(pathMatchMode string) string {
	pathMatchMode = strings.ToLower(strings.TrimSpace(pathMatchMode))
	switch pathMatchMode {
	case "", "prefix":
		return ""
	default:
		return pathMatchMode
	}
}

func normalizeRouteType(routeType string) string {
	routeType = strings.ToLower(strings.TrimSpace(routeType))
	switch routeType {
	case "static":
		return "static"
	default:
		return "" // "proxy" is the default, stored as empty
	}
}

// resolveCertificate populates TLSCert and TLSKey from a managed certificate
// when CertificateID is set. The certificate ID takes precedence over any
// manually supplied paths.
func (s *Service) resolveCertificate(route *store.Route) error {
	if route.CertificateID == "" {
		return nil
	}
	if s.certSvc == nil {
		return newError(ErrCodeCertificateNotFound, "certificate service not available", nil)
	}
	cert, err := s.certSvc.Get(route.CertificateID)
	if err != nil {
		return newError(ErrCodeCertificateNotFound, "certificate not found", err)
	}
	route.TLSCert = cert.CertPath
	route.TLSKey = cert.KeyPath
	route.TLSEnabled = true
	return nil
}

func normalizeHost(host string) string {
	return routehost.Normalize(host)
}

func normalizeStoredRoute(route store.Route) store.Route {
	route.Host = normalizeHost(route.Host)
	route.PathMatchMode = normalizePathMatchMode(route.PathMatchMode)
	route.RewriteTarget = strings.TrimSpace(route.RewriteTarget)
	if !isValidRedirectCode(route.RedirectCode) {
		route.RedirectCode = 0
	}
	return route
}

// normalizeHeaderMap trims whitespace from both keys and values and drops
// entries whose key is empty after trimming.
func normalizeHeaderMap(m map[string]string) map[string]string {
	if len(m) == 0 {
		return nil
	}
	out := make(map[string]string, len(m))
	for k, v := range m {
		k = strings.TrimSpace(k)
		if k == "" {
			continue
		}
		out[k] = v
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// normalizeHeaderNames trims whitespace and drops empty strings.
func normalizeHeaderNames(names []string) []string {
	if len(names) == 0 {
		return nil
	}
	out := make([]string, 0, len(names))
	for _, n := range names {
		n = strings.TrimSpace(n)
		if n != "" {
			out = append(out, n)
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}
