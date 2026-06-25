package store

import "time"

// Store is the persistence interface for all domain entities.
type Store interface {
	// Routes
	ListRoutes() ([]Route, error)
	GetRoute(id string) (*Route, error)
	CreateRoute(r *Route) error
	UpdateRoute(r *Route) error
	DeleteRoute(id string) error

	// Auth rules (legacy, kept for migration)
	ListAuthRules() ([]AuthRule, error)
	GetAuthRule(id string) (*AuthRule, error)
	GetAuthRuleByRouteID(routeID string) (*AuthRule, error)
	CreateAuthRule(r *AuthRule) error
	UpdateAuthRule(r *AuthRule) error
	DeleteAuthRule(id string) error

	// Route Auth Config
	GetRouteAuthConfig(routeID string) (*RouteAuthConfig, error)
	CreateOrUpdateRouteAuthConfig(c *RouteAuthConfig) error
	DeleteRouteAuthConfig(routeID string) error

	// API Keys
	ListApiKeysByRoute(routeID string) ([]ApiKey, error)
	GetApiKey(id string) (*ApiKey, error)
	FindApiKeyBySecret(secret string) (*ApiKey, error)
	CreateApiKey(k *ApiKey) error
	UpdateApiKey(k *ApiKey) error
	DeleteApiKey(id string) error

	// Users
	ListUsers() ([]User, error)
	GetUserByUsername(username string) (*User, error)
	GetUserByID(id string) (*User, error)
	CreateUser(u *User) error
	UpdateUser(u *User) error
	DeleteUser(id string) error
	VerifyPassword(user *User, password string) bool
	EnsureAdmin(username, password string) (bool, error)
	HasAdminUsers() (bool, error)

	// Certificates
	ListCertificates() ([]Certificate, error)
	GetCertificate(id string) (*Certificate, error)
	GetCertificateByDomain(domain string) (*Certificate, error)
	CreateCertificate(c *Certificate) error
	UpdateCertificate(c *Certificate) error
	DeleteCertificate(id string) error
	ListExpiringLocalCertificates(before time.Time) ([]Certificate, error)

	// CA certificates
	ListCACertificates() ([]CACertificate, error)
	GetCACertificate(id string) (*CACertificate, error)
	GetFirstCACertificate() (*CACertificate, error)
	CreateCACertificate(c *CACertificate) error

	// Host profiles
	ListHostProfiles() ([]HostProfile, error)
	GetHostProfile(id string) (*HostProfile, error)
	CreateHostProfile(p *HostProfile) error
	UpdateHostProfile(p *HostProfile) error
	DeleteHostProfile(id string) error
	SetActiveHostProfile(profileID string) error

	// Host entries
	CreateHostEntry(e *HostEntry) error
	ListHostEntries(profileID string) ([]HostEntry, error)
	GetHostEntry(id string) (*HostEntry, error)
	UpdateHostEntry(e *HostEntry) error
	DeleteHostEntry(id string) error
	ReorderHostEntries(profileID string, orderedIDs []string) error
	ListEnabledHostEntries(profileID string) ([]HostEntry, error)

	// Lifecycle
	Close() error
}
