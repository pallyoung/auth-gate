package router

import (
	"net/http"
	"regexp"
	"strings"
	"sync"

	"github.com/pallyoung/auth-gate/packages/server/internal/store"
)

// RouteAuthConfig is the compiled runtime representation of a route's auth configuration.
type RouteAuthConfig struct {
	ApiKeyEnabled    bool
	ApiKeyHeader     string // default "X-API-Key"
	BasicEnabled     bool
	BasicUsername    string
	BasicPassword    string
	GatewayEnabled   bool
	GatewayLoginMode string
	Whitelist        []string
	RateLimit        int
	Burst            int
	CORSAllowedOrigins   string
	CORSAllowedMethods   string
	CORSAllowedHeaders   string
	CORSAllowCredentials bool
	CORSMaxAge           int
}

// HasAuth returns true if any authentication method is enabled.
func (c *RouteAuthConfig) HasAuth() bool {
	return c.ApiKeyEnabled || c.BasicEnabled || c.GatewayEnabled
}

// ApiKeyEntry is the compiled runtime representation of an API key.
type ApiKeyEntry struct {
	ID        string
	Secret    string
	ExpiresAt *string // ISO8601 or nil
	Status    string
}

type Route struct {
	ID            string
	Name          string
	Host          string
	PathPrefix    string
	Backend       string
	Backends      []store.Backend
	StripPrefix   bool
	Enabled       bool
	Priority      int
	TLSCert       string
	TLSKey        string
	TLSEnabled    bool
	HTTPSRedirect bool
	TimeoutMs     int
	RetryAttempts int
	PathMatchMode  string // "prefix"|"exact"|"regex"
	HeaderName     string
	HeaderValue    string
	RewriteTarget  string
	RedirectCode   int
	PathRegex      *regexp.Regexp  // compiled regex (runtime only)
	AuthConfig     *RouteAuthConfig // route-level auth config (nil = no auth)
	ApiKeys        []ApiKeyEntry    // compiled api keys for this route
	// Header manipulation
	SetRequestHeaders     map[string]string
	RemoveRequestHeaders  []string
	AddResponseHeaders    map[string]string
	RemoveResponseHeaders []string
}

func (r *Route) EffectiveBackends() []store.Backend {
	if len(r.Backends) > 0 {
		return r.Backends
	}
	if r.Backend != "" {
		return []store.Backend{{URL: r.Backend, Weight: 1}}
	}
	return nil
}

type Manager struct {
	db     store.Store
	routes []Route
	mu     sync.RWMutex
}

func NewManager(db store.Store) *Manager {
	m := &Manager{db: db}
	m.loadRoutes()
	return m
}

func (m *Manager) DB() store.Store {
	return m.db
}

func (m *Manager) loadRoutes() {
	routes, err := m.db.ListRoutes()
	if err != nil {
		return
	}

	// Load auth configs by route
	authConfigs := make(map[string]store.RouteAuthConfig)
	// We need a ListRouteAuthConfigs method; for now, load per-route
	for _, route := range routes {
		if cfg, err := m.db.GetRouteAuthConfig(route.ID); err == nil {
			authConfigs[route.ID] = *cfg
		}
	}

	// Load all API keys
	apiKeys := make(map[string]store.ApiKey)
	for _, route := range routes {
		keys, err := m.db.ListApiKeysByRoute(route.ID)
		if err != nil {
			continue
		}
		for _, key := range keys {
			apiKeys[key.ID] = key
		}
	}

	result := compileRoutes(routes, authConfigs, apiKeys)

	m.mu.Lock()
	m.routes = result
	m.mu.Unlock()
}

func (m *Manager) Reload() {
	m.loadRoutes()
}

func (m *Manager) Match(host, path string, headers http.Header) *Route {
	m.mu.RLock()
	defer m.mu.RUnlock()

	host = normalizeStoredHost(host)

	for i := range m.routes {
		r := &m.routes[i]
		if !r.Enabled {
			continue
		}
		if r.Host != "" && r.Host != host {
			continue
		}
		// Header match: skip if header_name is set and value doesn't match
		if r.HeaderName != "" {
			if headers.Get(r.HeaderName) != r.HeaderValue {
				continue
			}
		}
		switch r.PathMatchMode {
		case "exact":
			if path == r.PathPrefix {
				return r
			}
		case "regex", "regex_i":
			if r.PathRegex != nil && r.PathRegex.MatchString(path) {
				return r
			}
		case "stop":
			if pathMatchesPrefix(path, r.PathPrefix) {
				return r
			}
		default: // "prefix"
			if pathMatchesPrefix(path, r.PathPrefix) {
				return r
			}
		}
	}
	return nil
}

func (m *Manager) FindByID(id string) *Route {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for i := range m.routes {
		if m.routes[i].ID == id {
			routeCopy := m.routes[i]
			if routeCopy.AuthConfig != nil {
				cfgCopy := *routeCopy.AuthConfig
				routeCopy.AuthConfig = &cfgCopy
			}
			if len(routeCopy.ApiKeys) > 0 {
				routeCopy.ApiKeys = make([]ApiKeyEntry, len(m.routes[i].ApiKeys))
				copy(routeCopy.ApiKeys, m.routes[i].ApiKeys)
			}
			return &routeCopy
		}
	}
	return nil
}

func pathMatchesPrefix(path, prefix string) bool {
	if prefix == "" || prefix == "/" {
		return true
	}
	if path == prefix {
		return true
	}
	return strings.HasPrefix(path, prefix+"/")
}

func (m *Manager) GetRoutes() []Route {
	m.mu.RLock()
	defer m.mu.RUnlock()
	result := make([]Route, len(m.routes))
	for i, route := range m.routes {
		result[i] = route
		if route.AuthConfig != nil {
			cfgCopy := *route.AuthConfig
			result[i].AuthConfig = &cfgCopy
		}
		if len(route.ApiKeys) > 0 {
			result[i].ApiKeys = make([]ApiKeyEntry, len(route.ApiKeys))
			copy(result[i].ApiKeys, route.ApiKeys)
		}
	}
	return result
}
