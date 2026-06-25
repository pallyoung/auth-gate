package router

import (
	"regexp"
	"strings"
	"sync"

	"github.com/pallyoung/auth-gate/packages/server/internal/store"
)

type AuthConfig struct {
	HeaderName string
	Secret     string
	Username   string
	Password   string
	LoginMode  string
}

type AuthRule struct {
	ID                    string
	RouteID               string
	Type                  string
	Config                AuthConfig
	Whitelist             []string
	RateLimit             int
	Burst                 int
	CORSAllowedOrigins    string
	CORSAllowedMethods    string
	CORSAllowedHeaders    string
	CORSAllowCredentials  bool
	CORSMaxAge            int
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
	RewriteTarget  string
	RedirectCode   int
	PathRegex      *regexp.Regexp // compiled regex (runtime only)
	AuthRule       *AuthRule
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

	authRules, _ := m.db.ListAuthRules()
	result := compileRoutes(routes, authRules)

	m.mu.Lock()
	m.routes = result
	m.mu.Unlock()
}

func (m *Manager) Reload() {
	m.loadRoutes()
}

func (m *Manager) Match(host, path string) *Route {
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
			if routeCopy.AuthRule != nil {
				authRuleCopy := *routeCopy.AuthRule
				routeCopy.AuthRule = &authRuleCopy
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
		if route.AuthRule != nil {
			authRuleCopy := *route.AuthRule
			result[i].AuthRule = &authRuleCopy
		}
	}
	return result
}
