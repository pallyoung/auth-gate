package store

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

// JSONStore implements Store backed by JSON files on disk.
type JSONStore struct {
	dir string
	mu  sync.RWMutex
	data storeData
}

type storeData struct {
	Routes           map[string]Route           `json:"routes"`
	AuthRules        map[string]AuthRule        `json:"auth_rules"`
	RouteAuthConfigs map[string]RouteAuthConfig `json:"route_auth_configs"`
	ApiKeys          map[string]ApiKey          `json:"api_keys"`
	Users            map[string]User            `json:"users"`
	PermissionGroups map[string]PermissionGroup `json:"permission_groups"`
	Certificates     map[string]Certificate     `json:"certificates"`
	CACertificates   map[string]CACertificate   `json:"ca_certificates"`
	HostProfiles     map[string]HostProfile     `json:"host_profiles"`
	HostEntries      map[string]HostEntry       `json:"host_entries"`
	Settings         map[string]string          `json:"settings"`
}

// NewJSONStore creates or loads a JSON-backed store from the given directory.
func NewJSONStore(dir string) (*JSONStore, error) {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("store: create dir: %w", err)
	}
	s := &JSONStore{dir: dir}
	if err := s.load(); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *JSONStore) load() error {
	path := filepath.Join(s.dir, "store.json")
	raw, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		s.data = storeData{
			Routes:           make(map[string]Route),
			AuthRules:        make(map[string]AuthRule),
			RouteAuthConfigs: make(map[string]RouteAuthConfig),
			ApiKeys:          make(map[string]ApiKey),
			Users:            make(map[string]User),
			PermissionGroups: make(map[string]PermissionGroup),
			Certificates:     make(map[string]Certificate),
			CACertificates:   make(map[string]CACertificate),
			HostProfiles:     make(map[string]HostProfile),
			HostEntries:      make(map[string]HostEntry),
			Settings:         make(map[string]string),
		}
		return nil
	}
	if err != nil {
		return fmt.Errorf("store: read data: %w", err)
	}
	if err := json.Unmarshal(raw, &s.data); err != nil {
		return fmt.Errorf("store: parse data: %w", err)
	}
	// Initialize new maps if missing (first load after upgrade)
	if s.data.RouteAuthConfigs == nil {
		s.data.RouteAuthConfigs = make(map[string]RouteAuthConfig)
	}
	if s.data.ApiKeys == nil {
		s.data.ApiKeys = make(map[string]ApiKey)
	}
	if s.data.Settings == nil {
		s.data.Settings = make(map[string]string)
	}
	if s.data.PermissionGroups == nil {
		s.data.PermissionGroups = make(map[string]PermissionGroup)
	}
	// Migrate old AuthRules to RouteAuthConfigs + ApiKeys
	if len(s.data.AuthRules) > 0 {
		s.migrateAuthRules()
	}
	return nil
}

func (s *JSONStore) save() error {
	raw, err := json.MarshalIndent(s.data, "", "  ")
	if err != nil {
		return err
	}
	path := filepath.Join(s.dir, "store.json")
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, raw, 0644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func (s *JSONStore) Close() error { return nil }

// ---- Routes ----

func (s *JSONStore) ListRoutes() ([]Route, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	routes := make([]Route, 0, len(s.data.Routes))
	for _, r := range s.data.Routes {
		routes = append(routes, r)
	}
	sort.Slice(routes, func(i, j int) bool {
		if routes[i].Priority != routes[j].Priority {
			return routes[i].Priority > routes[j].Priority
		}
		return routes[i].PathPrefix > routes[j].PathPrefix
	})
	return routes, nil
}

func (s *JSONStore) GetRoute(id string) (*Route, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	r, ok := s.data.Routes[id]
	if !ok {
		return nil, sql.ErrNoRows
	}
	return &r, nil
}

func (s *JSONStore) CreateRoute(r *Route) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if r.ID == "" {
		r.ID = uuid.New().String()
	}
	now := time.Now()
	r.CreatedAt = now
	r.UpdatedAt = now
	s.data.Routes[r.ID] = *r
	return s.save()
}

func (s *JSONStore) UpdateRoute(r *Route) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.data.Routes[r.ID]; !ok {
		return sql.ErrNoRows
	}
	r.UpdatedAt = time.Now()
	s.data.Routes[r.ID] = *r
	return s.save()
}

func (s *JSONStore) DeleteRoute(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.data.Routes, id)
	for rid, r := range s.data.AuthRules {
		if r.RouteID == id {
			delete(s.data.AuthRules, rid)
		}
	}
	delete(s.data.RouteAuthConfigs, id)
	for kid, k := range s.data.ApiKeys {
		if k.RouteID == id {
			delete(s.data.ApiKeys, kid)
		}
	}
	return s.save()
}

// ---- Auth Rules ----

func (s *JSONStore) ListAuthRules() ([]AuthRule, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	rules := make([]AuthRule, 0, len(s.data.AuthRules))
	for _, r := range s.data.AuthRules {
		rules = append(rules, r)
	}
	return rules, nil
}

func (s *JSONStore) GetAuthRule(id string) (*AuthRule, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	r, ok := s.data.AuthRules[id]
	if !ok {
		return nil, sql.ErrNoRows
	}
	return &r, nil
}

func (s *JSONStore) GetAuthRuleByRouteID(routeID string) (*AuthRule, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, r := range s.data.AuthRules {
		if r.RouteID == routeID {
			return &r, nil
		}
	}
	return nil, sql.ErrNoRows
}

func (s *JSONStore) CreateAuthRule(r *AuthRule) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if r.ID == "" {
		r.ID = uuid.New().String()
	}
	// Enforce foreign key: route must exist
	if _, ok := s.data.Routes[r.RouteID]; !ok {
		return sql.ErrNoRows
	}
	now := time.Now()
	r.CreatedAt = now
	r.UpdatedAt = now
	normalizeAuthRule(r)
	s.data.AuthRules[r.ID] = *r
	return s.save()
}

func (s *JSONStore) UpdateAuthRule(r *AuthRule) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.data.AuthRules[r.ID]; !ok {
		return sql.ErrNoRows
	}
	r.UpdatedAt = time.Now()
	normalizeAuthRule(r)
	s.data.AuthRules[r.ID] = *r
	return s.save()
}

func (s *JSONStore) DeleteAuthRule(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.data.AuthRules, id)
	return s.save()
}

// ---- Migration: AuthRules → RouteAuthConfigs + ApiKeys ----

func (s *JSONStore) migrateAuthRules() {
	log.Println("store: migrating auth_rules to route_auth_configs + api_keys")

	// Group rules by route_id
	type ruleGroup struct {
		config  RouteAuthConfig
		apiKeys []ApiKey
	}
	groups := make(map[string]*ruleGroup)

	for _, rule := range s.data.AuthRules {
		g, ok := groups[rule.RouteID]
		if !ok {
			g = &ruleGroup{
				config: RouteAuthConfig{
					ID:       uuid.New().String(),
					RouteID:  rule.RouteID,
					Whitelist: rule.Whitelist,
					RateLimit: rule.RateLimit,
					Burst:     rule.Burst,
					CORSAllowedOrigins:   rule.CORSAllowedOrigins,
					CORSAllowedMethods:   rule.CORSAllowedMethods,
					CORSAllowedHeaders:   rule.CORSAllowedHeaders,
					CORSAllowCredentials: rule.CORSAllowCredentials,
					CORSMaxAge:           rule.CORSMaxAge,
				},
			}
			groups[rule.RouteID] = g
		}

		switch rule.Type {
		case "apikey":
			g.config.ApiKeyEnabled = true
			if rule.Config.HeaderName != "" {
				g.config.ApiKeyHeader = rule.Config.HeaderName
			}
			if rule.Config.Secret != "" {
				now := time.Now()
				secret := rule.Config.Secret
				prefix := secret
				if len(prefix) > 8 {
					prefix = prefix[:8]
				}
				g.apiKeys = append(g.apiKeys, ApiKey{
					ID:        uuid.New().String(),
					RouteID:   rule.RouteID,
					Name:      "Migrated Key",
					KeyPrefix: prefix,
					Secret:    secret,
					Status:    "active",
					CreatedAt: now,
					UpdatedAt: now,
				})
			}
		case "gateway":
			g.config.GatewayEnabled = true
			if rule.Config.LoginMode != "" {
				g.config.GatewayLoginMode = rule.Config.LoginMode
			} else {
				g.config.GatewayLoginMode = "form"
			}
		}
	}

	now := time.Now()
	for _, g := range groups {
		g.config.CreatedAt = now
		g.config.UpdatedAt = now
		s.data.RouteAuthConfigs[g.config.RouteID] = g.config
		for _, key := range g.apiKeys {
			s.data.ApiKeys[key.ID] = key
		}
	}

	// Clear old auth rules
	s.data.AuthRules = make(map[string]AuthRule)
	if err := s.save(); err != nil {
		log.Printf("store: warning: failed to save migration: %v", err)
	} else {
		log.Printf("store: migrated %d route auth configs, %d api keys", len(groups), len(s.data.ApiKeys))
	}
}

// ---- Route Auth Config ----

func (s *JSONStore) GetRouteAuthConfig(routeID string) (*RouteAuthConfig, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	cfg, ok := s.data.RouteAuthConfigs[routeID]
	if !ok {
		return nil, sql.ErrNoRows
	}
	return &cfg, nil
}

func (s *JSONStore) CreateOrUpdateRouteAuthConfig(c *RouteAuthConfig) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if c.ID == "" {
		c.ID = uuid.New().String()
	}
	// Verify route exists
	if _, ok := s.data.Routes[c.RouteID]; !ok {
		return sql.ErrNoRows
	}
	now := time.Now()
	if existing, ok := s.data.RouteAuthConfigs[c.RouteID]; ok {
		c.ID = existing.ID
		c.CreatedAt = existing.CreatedAt
	} else {
		c.CreatedAt = now
	}
	c.UpdatedAt = now
	s.data.RouteAuthConfigs[c.RouteID] = *c
	return s.save()
}

func (s *JSONStore) DeleteRouteAuthConfig(routeID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.data.RouteAuthConfigs, routeID)
	// Also delete all API keys for this route
	for id, key := range s.data.ApiKeys {
		if key.RouteID == routeID {
			delete(s.data.ApiKeys, id)
		}
	}
	return s.save()
}

// ---- API Keys ----

func (s *JSONStore) ListApiKeysByRoute(routeID string) ([]ApiKey, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var keys []ApiKey
	for _, k := range s.data.ApiKeys {
		if k.RouteID == routeID {
			keys = append(keys, k)
		}
	}
	sort.Slice(keys, func(i, j int) bool {
		return keys[i].CreatedAt.After(keys[j].CreatedAt)
	})
	return keys, nil
}

func (s *JSONStore) GetApiKey(id string) (*ApiKey, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	k, ok := s.data.ApiKeys[id]
	if !ok {
		return nil, sql.ErrNoRows
	}
	return &k, nil
}

func (s *JSONStore) FindApiKeyBySecret(secret string) (*ApiKey, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, k := range s.data.ApiKeys {
		if k.Secret == secret {
			return &k, nil
		}
	}
	return nil, sql.ErrNoRows
}

func (s *JSONStore) CreateApiKey(k *ApiKey) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if k.ID == "" {
		k.ID = uuid.New().String()
	}
	// Verify route exists
	if _, ok := s.data.Routes[k.RouteID]; !ok {
		return sql.ErrNoRows
	}
	now := time.Now()
	k.CreatedAt = now
	k.UpdatedAt = now
	if k.Status == "" {
		k.Status = "active"
	}
	s.data.ApiKeys[k.ID] = *k
	return s.save()
}

func (s *JSONStore) UpdateApiKey(k *ApiKey) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.data.ApiKeys[k.ID]; !ok {
		return sql.ErrNoRows
	}
	k.UpdatedAt = time.Now()
	s.data.ApiKeys[k.ID] = *k
	return s.save()
}

func (s *JSONStore) DeleteApiKey(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.data.ApiKeys[id]; !ok {
		return sql.ErrNoRows
	}
	delete(s.data.ApiKeys, id)
	return s.save()
}

// ---- Users ----

func (s *JSONStore) ListUsers() ([]User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	users := make([]User, 0, len(s.data.Users))
	for _, u := range s.data.Users {
		u.Role = normalizeStoredUserRole(u.Role)
		users = append(users, u)
	}
	sort.Slice(users, func(i, j int) bool {
		return users[i].CreatedAt.After(users[j].CreatedAt)
	})
	return users, nil
}

func (s *JSONStore) GetUserByUsername(username string) (*User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, u := range s.data.Users {
		if u.Username == username {
			u.Role = normalizeStoredUserRole(u.Role)
			return &u, nil
		}
	}
	return nil, sql.ErrNoRows
}

func (s *JSONStore) GetUserByID(id string) (*User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	u, ok := s.data.Users[id]
	if !ok {
		return nil, sql.ErrNoRows
	}
	u.Role = normalizeStoredUserRole(u.Role)
	return &u, nil
}

func (s *JSONStore) CreateUser(u *User) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if u.ID == "" {
		u.ID = uuid.New().String()
	}
	now := time.Now()
	u.CreatedAt = now
	u.UpdatedAt = now
	// Check unique username
	for _, existing := range s.data.Users {
		if existing.Username == u.Username {
			return fmt.Errorf("UNIQUE constraint failed: users.username")
		}
	}
	s.data.Users[u.ID] = *u
	return s.save()
}

func (s *JSONStore) UpdateUser(u *User) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.data.Users[u.ID]; !ok {
		return sql.ErrNoRows
	}
	// Check unique username (excluding self)
	for id, existing := range s.data.Users {
		if id != u.ID && existing.Username == u.Username {
			return fmt.Errorf("UNIQUE constraint failed: users.username")
		}
	}
	u.UpdatedAt = time.Now()
	s.data.Users[u.ID] = *u
	return s.save()
}

func (s *JSONStore) DeleteUser(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.data.Users, id)
	return s.save()
}

func (s *JSONStore) VerifyPassword(user *User, password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password))
	return err == nil
}

// ---- Certificates ----

func (s *JSONStore) ListCertificates() ([]Certificate, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	certs := make([]Certificate, 0, len(s.data.Certificates))
	for _, c := range s.data.Certificates {
		certs = append(certs, c)
	}
	sort.Slice(certs, func(i, j int) bool {
		return certs[i].CreatedAt.After(certs[j].CreatedAt)
	})
	return certs, nil
}

func (s *JSONStore) GetCertificate(id string) (*Certificate, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	c, ok := s.data.Certificates[id]
	if !ok {
		return nil, nil
	}
	return &c, nil
}

func (s *JSONStore) GetCertificateByDomain(domain string) (*Certificate, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, c := range s.data.Certificates {
		if c.Domain == domain {
			return &c, nil
		}
	}
	return nil, nil
}

func (s *JSONStore) CreateCertificate(c *Certificate) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data.Certificates[c.ID] = *c
	return s.save()
}

func (s *JSONStore) UpdateCertificate(c *Certificate) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.data.Certificates[c.ID]; !ok {
		return sql.ErrNoRows
	}
	c.UpdatedAt = time.Now()
	s.data.Certificates[c.ID] = *c
	return s.save()
}

func (s *JSONStore) DeleteCertificate(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.data.Certificates, id)
	return s.save()
}

func (s *JSONStore) ListExpiringLocalCertificates(before time.Time) ([]Certificate, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var certs []Certificate
	for _, c := range s.data.Certificates {
		if c.Source == SourceLocalCA && c.Status == CertStatusActive &&
			!c.RenewAt.IsZero() && !c.RenewAt.After(before) {
			certs = append(certs, c)
		}
	}
	return certs, nil
}

// ---- CA Certificates ----

func (s *JSONStore) ListCACertificates() ([]CACertificate, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	cas := make([]CACertificate, 0, len(s.data.CACertificates))
	for _, c := range s.data.CACertificates {
		cas = append(cas, c)
	}
	sort.Slice(cas, func(i, j int) bool {
		return cas[i].CreatedAt.Before(cas[j].CreatedAt)
	})
	return cas, nil
}

func (s *JSONStore) GetCACertificate(id string) (*CACertificate, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	c, ok := s.data.CACertificates[id]
	if !ok {
		return nil, nil
	}
	return &c, nil
}

func (s *JSONStore) GetFirstCACertificate() (*CACertificate, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if len(s.data.CACertificates) == 0 {
		return nil, nil
	}
	var first *CACertificate
	for _, c := range s.data.CACertificates {
		if first == nil || c.CreatedAt.Before(first.CreatedAt) {
			cc := c
			first = &cc
		}
	}
	return first, nil
}

func (s *JSONStore) CreateCACertificate(c *CACertificate) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data.CACertificates[c.ID] = *c
	return s.save()
}

// ---- Host Profiles ----

func (s *JSONStore) ListHostProfiles() ([]HostProfile, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	profiles := make([]HostProfile, 0, len(s.data.HostProfiles))
	for _, p := range s.data.HostProfiles {
		profiles = append(profiles, p)
	}
	sort.Slice(profiles, func(i, j int) bool {
		return profiles[i].Name < profiles[j].Name
	})
	return profiles, nil
}

func (s *JSONStore) GetHostProfile(id string) (*HostProfile, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	p, ok := s.data.HostProfiles[id]
	if !ok {
		return nil, sql.ErrNoRows
	}
	return &p, nil
}

func (s *JSONStore) CreateHostProfile(p *HostProfile) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if p.ID == "" {
		p.ID = uuid.New().String()
	}
	// Check unique name
	for _, existing := range s.data.HostProfiles {
		if existing.Name == p.Name {
			return fmt.Errorf("UNIQUE constraint failed: host_profiles.name")
		}
	}
	now := time.Now()
	p.CreatedAt = now
	p.UpdatedAt = now
	p.IsActive = false
	s.data.HostProfiles[p.ID] = *p
	return s.save()
}

func (s *JSONStore) UpdateHostProfile(p *HostProfile) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.data.HostProfiles[p.ID]; !ok {
		return sql.ErrNoRows
	}
	// Check unique name (excluding self)
	for id, existing := range s.data.HostProfiles {
		if id != p.ID && existing.Name == p.Name {
			return fmt.Errorf("UNIQUE constraint failed: host_profiles.name")
		}
	}
	p.UpdatedAt = time.Now()
	s.data.HostProfiles[p.ID] = *p
	return s.save()
}

func (s *JSONStore) DeleteHostProfile(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.data.HostProfiles, id)
	for eid, e := range s.data.HostEntries {
		if e.ProfileID == id {
			delete(s.data.HostEntries, eid)
		}
	}
	return s.save()
}

func (s *JSONStore) SetActiveHostProfile(profileID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.data.HostProfiles[profileID]; !ok {
		return sql.ErrNoRows
	}
	now := time.Now()
	for id := range s.data.HostProfiles {
		p := s.data.HostProfiles[id]
		p.IsActive = (id == profileID)
		p.UpdatedAt = now
		s.data.HostProfiles[id] = p
	}
	return s.save()
}

// ---- Host Entries ----

func (s *JSONStore) CreateHostEntry(e *HostEntry) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if e.ID == "" {
		e.ID = uuid.New().String()
	}
	if e.Position <= 0 {
		maxPos := 0
		for _, existing := range s.data.HostEntries {
			if existing.ProfileID == e.ProfileID && existing.Position >= maxPos {
				maxPos = existing.Position
			}
		}
		e.Position = maxPos + 1
	}
	now := time.Now()
	e.CreatedAt = now
	e.UpdatedAt = now
	s.data.HostEntries[e.ID] = *e
	return s.save()
}

func (s *JSONStore) ListHostEntries(profileID string) ([]HostEntry, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var entries []HostEntry
	for _, e := range s.data.HostEntries {
		if e.ProfileID == profileID {
			entries = append(entries, e)
		}
	}
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].Position != entries[j].Position {
			return entries[i].Position < entries[j].Position
		}
		return entries[i].ID < entries[j].ID
	})
	return entries, nil
}

func (s *JSONStore) GetHostEntry(id string) (*HostEntry, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	e, ok := s.data.HostEntries[id]
	if !ok {
		return nil, sql.ErrNoRows
	}
	return &e, nil
}

func (s *JSONStore) UpdateHostEntry(e *HostEntry) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.data.HostEntries[e.ID]; !ok {
		return sql.ErrNoRows
	}
	e.UpdatedAt = time.Now()
	s.data.HostEntries[e.ID] = *e
	return s.save()
}

func (s *JSONStore) DeleteHostEntry(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.data.HostEntries, id)
	return s.save()
}

func (s *JSONStore) ReorderHostEntries(profileID string, orderedIDs []string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now()
	for i, id := range orderedIDs {
		e, ok := s.data.HostEntries[id]
		if !ok || e.ProfileID != profileID {
			return sql.ErrNoRows
		}
		e.Position = i
		e.UpdatedAt = now
		s.data.HostEntries[id] = e
	}
	return s.save()
}

func (s *JSONStore) ListEnabledHostEntries(profileID string) ([]HostEntry, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	entries := make([]HostEntry, 0)
	for _, e := range s.data.HostEntries {
		if e.ProfileID == profileID && e.Enabled {
			entries = append(entries, e)
		}
	}
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].Position != entries[j].Position {
			return entries[i].Position < entries[j].Position
		}
		return entries[i].ID < entries[j].ID
	})
	return entries, nil
}

// EnsureAdmin creates a bootstrap admin user if no users exist.
func (s *JSONStore) EnsureAdmin(username, password string) (bool, error) {
	username = strings.TrimSpace(username)
	if username == "" {
		username = "admin"
	}
	if password == "" {
		return false, errors.New("bootstrap admin password required")
	}
	users, err := s.ListUsers()
	if err != nil {
		return false, err
	}
	if len(users) == 0 {
		hash, err := HashPassword(password)
		if err != nil {
			return false, err
		}
		if err := s.CreateUser(&User{
			Username:     username,
			PasswordHash: hash,
			Role:         RoleAdmin,
			Enabled:      true,
		}); err != nil {
			return false, err
		}
		return true, nil
	}
	return false, nil
}

// HasAdminUsers reports whether the store contains at least one admin user.
func (s *JSONStore) HasAdminUsers() (bool, error) {
	users, err := s.ListUsers()
	if err != nil {
		return false, err
	}
	for _, u := range users {
		if u.Role == RoleAdmin {
			return true, nil
		}
	}
	return false, nil
}

func normalizeAuthRule(r *AuthRule) {
	r.Type = strings.TrimSpace(r.Type)
	r.Config.HeaderName = strings.TrimSpace(r.Config.HeaderName)
	r.Config.Secret = strings.TrimSpace(r.Config.Secret)
	r.Config.Username = strings.TrimSpace(r.Config.Username)
	// Password is intentionally NOT trimmed (may contain meaningful whitespace).
	r.Config.LoginMode = strings.TrimSpace(r.Config.LoginMode)
}

// ---- Permission Groups ----

func (s *JSONStore) ListPermissionGroups() ([]PermissionGroup, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	groups := make([]PermissionGroup, 0, len(s.data.PermissionGroups))
	for _, g := range s.data.PermissionGroups {
		groups = append(groups, g)
	}
	sort.Slice(groups, func(i, j int) bool {
		return groups[i].Name < groups[j].Name
	})
	return groups, nil
}

func (s *JSONStore) GetPermissionGroup(id string) (*PermissionGroup, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	g, ok := s.data.PermissionGroups[id]
	if !ok {
		return nil, sql.ErrNoRows
	}
	return &g, nil
}

func (s *JSONStore) GetPermissionGroupsByIDs(ids []string) ([]PermissionGroup, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	groups := make([]PermissionGroup, 0, len(ids))
	for _, id := range ids {
		if g, ok := s.data.PermissionGroups[id]; ok {
			groups = append(groups, g)
		}
	}
	return groups, nil
}

func (s *JSONStore) CreatePermissionGroup(g *PermissionGroup) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if g.ID == "" {
		g.ID = uuid.New().String()
	}
	for _, existing := range s.data.PermissionGroups {
		if existing.Name == g.Name {
			return fmt.Errorf("UNIQUE constraint failed: permission_groups.name")
		}
	}
	now := time.Now()
	g.CreatedAt = now
	g.UpdatedAt = now
	s.data.PermissionGroups[g.ID] = *g
	return s.save()
}

func (s *JSONStore) UpdatePermissionGroup(g *PermissionGroup) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.data.PermissionGroups[g.ID]; !ok {
		return sql.ErrNoRows
	}
	for id, existing := range s.data.PermissionGroups {
		if id != g.ID && existing.Name == g.Name {
			return fmt.Errorf("UNIQUE constraint failed: permission_groups.name")
		}
	}
	g.UpdatedAt = time.Now()
	s.data.PermissionGroups[g.ID] = *g
	return s.save()
}

func (s *JSONStore) DeletePermissionGroup(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.data.PermissionGroups[id]; !ok {
		return sql.ErrNoRows
	}
	delete(s.data.PermissionGroups, id)
	// Remove group reference from all users
	for uid, u := range s.data.Users {
		newIDs := make([]string, 0, len(u.GroupIDs))
		for _, gid := range u.GroupIDs {
			if gid != id {
				newIDs = append(newIDs, gid)
			}
		}
		if len(newIDs) != len(u.GroupIDs) {
			u.GroupIDs = newIDs
			s.data.Users[uid] = u
		}
	}
	return s.save()
}

// ---- Settings ----

func (s *JSONStore) GetSetting(key string) (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.data.Settings[key], nil
}

func (s *JSONStore) SetSetting(key, value string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data.Settings[key] = value
	return s.save()
}
