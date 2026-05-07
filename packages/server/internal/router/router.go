package router

import (
	"sort"
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
	ID        string
	RouteID   string
	Type      string
	Config    AuthConfig
}

type Route struct {
	ID          string
	Name        string
	Host        string
	PathPrefix  string
	Backend     string
	StripPrefix bool
	Enabled     bool
	Priority    int
	AuthRule    *AuthRule
}

type Manager struct {
	db     *store.SQLite
	routes []Route
	mu     sync.RWMutex
}

func NewManager(db *store.SQLite) *Manager {
	m := &Manager{db: db}
	m.loadRoutes()
	return m
}

func (m *Manager) DB() *store.SQLite {
	return m.db
}

func (m *Manager) loadRoutes() {
	routes, err := m.db.ListRoutes()
	if err != nil {
		return
	}

	authRules, _ := m.db.ListAuthRules()
	result := compileRoutes(routes, authRules)

	// 按 priority 降序，path_prefix 长度降序排序
	sort.Slice(result, func(i, j int) bool {
		if result[i].Priority != result[j].Priority {
			return result[i].Priority > result[j].Priority
		}
		return len(result[i].PathPrefix) > len(result[j].PathPrefix)
	})

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

	for i := range m.routes {
		r := &m.routes[i]
		if !r.Enabled {
			continue
		}
		// host 匹配 (空 host 匹配所有)
		if r.Host != "" && r.Host != host {
			continue
		}
		// 路径边界匹配: "/api" 只匹配 "/api" 或 "/api/..."
		if pathMatchesPrefix(path, r.PathPrefix) {
			return r
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
