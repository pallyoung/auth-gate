package router

import (
	"sort"
	"strings"
	"sync"

	"github.com/pallyoung/auth-gate/packages/server/internal/store"
)

type Route struct {
	ID          string
	Name        string
	Host        string
	PathPrefix  string
	Backend     string
	StripPrefix bool
	Enabled     bool
	Priority    int
	AuthRule    *store.AuthRule
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

func (m *Manager) loadRoutes() {
	routes, err := m.db.ListRoutes()
	if err != nil {
		return
	}

	authRules, _ := m.db.ListAuthRules()
	authMap := make(map[string]*store.AuthRule)
	for i := range authRules {
		authMap[authRules[i].RouteID] = &authRules[i]
	}

	var result []Route
	for _, r := range routes {
		route := Route{
			ID:          r.ID,
			Name:        r.Name,
			Host:        r.Host,
			PathPrefix:  r.PathPrefix,
			Backend:     r.Backend,
			StripPrefix: r.StripPrefix,
			Enabled:     r.Enabled,
			Priority:    r.Priority,
		}
		if auth, ok := authMap[r.ID]; ok {
			route.AuthRule = auth
		}
		result = append(result, route)
	}

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
		// 路径前缀匹配
		if strings.HasPrefix(path, r.PathPrefix) {
			return r
		}
	}
	return nil
}

func (m *Manager) GetRoutes() []Route {
	m.mu.RLock()
	defer m.mu.RUnlock()
	result := make([]Route, len(m.routes))
	copy(result, m.routes)
	return result
}
