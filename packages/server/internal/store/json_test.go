package store

import (
	"database/sql"
	"errors"
	"testing"
	"time"
)

func newTestStore(t *testing.T) Store {
	t.Helper()

	s, err := NewJSONStore(t.TempDir())
	if err != nil {
		t.Fatalf("NewJSONStore() error = %v", err)
	}
	return s
}

func createTestRoute(t *testing.T, db Store, id string) *Route {
	t.Helper()

	route := &Route{
		ID:         id,
		Name:       "test-route",
		PathPrefix: "/test",
		Backend:    "http://example.com",
		Enabled:    true,
	}
	if err := db.CreateRoute(route); err != nil {
		t.Fatalf("CreateRoute() error = %v", err)
	}
	return route
}

func TestGetAuthRule_NormalizesLegacyStoredConfigWhitespace(t *testing.T) {
	db := newTestStore(t)
	route := createTestRoute(t, db, "route-1")

	if err := db.CreateAuthRule(&AuthRule{
		ID:      "rule-1",
		RouteID: route.ID,
		Type:    " apikey ",
		Config: AuthConfig{
			HeaderName: " X-Route-Key ",
			Secret:     " shared-secret ",
			Username:   " service-user ",
			Password:   "  keep-password  ",
			LoginMode:  " form ",
		},
	}); err != nil {
		t.Fatalf("CreateAuthRule() error = %v", err)
	}

	rule, err := db.GetAuthRule("rule-1")
	if err != nil {
		t.Fatalf("GetAuthRule() error = %v", err)
	}
	if rule.Config.HeaderName != "X-Route-Key" {
		t.Fatalf("HeaderName = %q, want %q", rule.Config.HeaderName, "X-Route-Key")
	}
	if rule.Config.Secret != "shared-secret" {
		t.Fatalf("Secret = %q, want %q", rule.Config.Secret, "shared-secret")
	}
	if rule.Config.Username != "service-user" {
		t.Fatalf("Username = %q, want %q", rule.Config.Username, "service-user")
	}
	if rule.Config.Password != "  keep-password  " {
		t.Fatalf("Password = %q, want preserved whitespace-sensitive password", rule.Config.Password)
	}
	if rule.Config.LoginMode != "form" {
		t.Fatalf("LoginMode = %q, want %q", rule.Config.LoginMode, "form")
	}
}

func TestAuthRule_PersistsRuntimePolicyFields(t *testing.T) {
	db := newTestStore(t)
	route := createTestRoute(t, db, "route-1")

	if err := db.CreateAuthRule(&AuthRule{
		ID:                   "rule-1",
		RouteID:              route.ID,
		Type:                 "bearer",
		Config:               AuthConfig{Secret: "shared-secret"},
		Whitelist:            []string{"127.0.0.1/32", "10.0.0.0/8"},
		RateLimit:            12,
		Burst:                24,
		CORSAllowedOrigins:   "https://app.example.com,.example.com",
		CORSAllowedMethods:   "GET,POST,OPTIONS",
		CORSAllowedHeaders:   "Authorization,Content-Type",
		CORSAllowCredentials: true,
		CORSMaxAge:           7200,
	}); err != nil {
		t.Fatalf("CreateAuthRule() error = %v", err)
	}

	rule, err := db.GetAuthRule("rule-1")
	if err != nil {
		t.Fatalf("GetAuthRule() error = %v", err)
	}

	if len(rule.Whitelist) != 2 || rule.Whitelist[0] != "127.0.0.1/32" || rule.Whitelist[1] != "10.0.0.0/8" {
		t.Fatalf("Whitelist = %#v, want %#v", rule.Whitelist, []string{"127.0.0.1/32", "10.0.0.0/8"})
	}
	if rule.RateLimit != 12 {
		t.Fatalf("RateLimit = %d, want %d", rule.RateLimit, 12)
	}
	if rule.Burst != 24 {
		t.Fatalf("Burst = %d, want %d", rule.Burst, 24)
	}
	if rule.CORSAllowedOrigins != "https://app.example.com,.example.com" {
		t.Fatalf("CORSAllowedOrigins = %q, want %q", rule.CORSAllowedOrigins, "https://app.example.com,.example.com")
	}
	if rule.CORSAllowedMethods != "GET,POST,OPTIONS" {
		t.Fatalf("CORSAllowedMethods = %q, want %q", rule.CORSAllowedMethods, "GET,POST,OPTIONS")
	}
	if rule.CORSAllowedHeaders != "Authorization,Content-Type" {
		t.Fatalf("CORSAllowedHeaders = %q, want %q", rule.CORSAllowedHeaders, "Authorization,Content-Type")
	}
	if !rule.CORSAllowCredentials {
		t.Fatal("CORSAllowCredentials = false, want true")
	}
	if rule.CORSMaxAge != 7200 {
		t.Fatalf("CORSMaxAge = %d, want %d", rule.CORSMaxAge, 7200)
	}
}

func TestRoute_PersistsRuntimePolicyFields(t *testing.T) {
	db := newTestStore(t)

	if err := db.CreateRoute(&Route{
		ID:            "route-runtime-policy",
		Name:          "runtime-policy-route",
		Host:          "api.example.com",
		PathPrefix:    "/api",
		Backend:       "http://example.com",
		Enabled:       true,
		TimeoutMs:     4500,
		RetryAttempts: 3,
	}); err != nil {
		t.Fatalf("CreateRoute() error = %v", err)
	}

	route, err := db.GetRoute("route-runtime-policy")
	if err != nil {
		t.Fatalf("GetRoute() error = %v", err)
	}

	if route.TimeoutMs != 4500 {
		t.Fatalf("TimeoutMs = %d, want %d", route.TimeoutMs, 4500)
	}
	if route.RetryAttempts != 3 {
		t.Fatalf("RetryAttempts = %d, want %d", route.RetryAttempts, 3)
	}
}

func TestEnsureAdmin_CreatesBootstrapUser(t *testing.T) {
	db := newTestStore(t)

	created, err := db.EnsureAdmin("admin", "bootstrap-secret")
	if err != nil {
		t.Fatalf("EnsureAdmin() error = %v", err)
	}
	if !created {
		t.Fatal("EnsureAdmin() created = false, want true")
	}

	user, err := db.GetUserByUsername("admin")
	if err != nil {
		t.Fatalf("GetUserByUsername() error = %v", err)
	}
	if !db.VerifyPassword(user, "bootstrap-secret") {
		t.Fatal("VerifyPassword() = false, want true")
	}
}

func TestEnsureAdmin_UsesProvidedUsername(t *testing.T) {
	db := newTestStore(t)

	created, err := db.EnsureAdmin("bootstrap-admin", "bootstrap-secret")
	if err != nil {
		t.Fatalf("EnsureAdmin() error = %v", err)
	}
	if !created {
		t.Fatal("EnsureAdmin() created = false, want true")
	}

	user, err := db.GetUserByUsername("bootstrap-admin")
	if err != nil {
		t.Fatalf("GetUserByUsername() error = %v", err)
	}
	if user.Username != "bootstrap-admin" {
		t.Fatalf("user.Username = %q, want %q", user.Username, "bootstrap-admin")
	}
	if !db.VerifyPassword(user, "bootstrap-secret") {
		t.Fatal("VerifyPassword() = false, want true")
	}
}

func TestEnsureAdmin_RequiresPassword(t *testing.T) {
	db := newTestStore(t)

	created, err := db.EnsureAdmin("admin", "")
	if err == nil {
		t.Fatal("EnsureAdmin() error = nil, want error")
	}
	if created {
		t.Fatal("EnsureAdmin() created = true, want false")
	}
}

func TestDeleteRoute_RemovesFromStore(t *testing.T) {
	db := newTestStore(t)
	createTestRoute(t, db, "route-1")

	if err := db.DeleteRoute("route-1"); err != nil {
		t.Fatalf("DeleteRoute() error = %v", err)
	}

	_, err := db.GetRoute("route-1")
	if !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("GetRoute() error = %v, want %v", err, sql.ErrNoRows)
	}
}

func TestCreateAuthRule_RejectsMissingRoute(t *testing.T) {
	db := newTestStore(t)

	err := db.CreateAuthRule(&AuthRule{
		RouteID: "missing-route",
		Type:    "apikey",
		Config:  AuthConfig{Secret: "secret"},
	})
	if err == nil {
		t.Fatal("CreateAuthRule() error = nil, want foreign key failure")
	}
}

func TestCreateAuthRule_AllowsMultipleRulesPerRoute(t *testing.T) {
	db := newTestStore(t)
	route := createTestRoute(t, db, "route-1")

	if err := db.CreateAuthRule(&AuthRule{
		ID:      "rule-1",
		RouteID: route.ID,
		Type:    "apikey",
		Config:  AuthConfig{Secret: "secret-1"},
	}); err != nil {
		t.Fatalf("CreateAuthRule(first) error = %v", err)
	}

	if err := db.CreateAuthRule(&AuthRule{
		ID:      "rule-2",
		RouteID: route.ID,
		Type:    "gateway",
		Config:  AuthConfig{LoginMode: "form"},
	}); err != nil {
		t.Fatalf("CreateAuthRule(second) error = %v, want nil", err)
	}

	rules, err := db.ListAuthRules()
	if err != nil {
		t.Fatalf("ListAuthRules() error = %v", err)
	}
	if len(rules) != 2 {
		t.Fatalf("len(rules) = %d, want 2", len(rules))
	}
}

func TestSetActiveHostProfile(t *testing.T) {
	db := newTestStore(t)

	p1 := &HostProfile{Name: "profile-1"}
	p2 := &HostProfile{Name: "profile-2"}
	if err := db.CreateHostProfile(p1); err != nil {
		t.Fatalf("CreateHostProfile(p1) error = %v", err)
	}
	if err := db.CreateHostProfile(p2); err != nil {
		t.Fatalf("CreateHostProfile(p2) error = %v", err)
	}

	if err := db.SetActiveHostProfile(p1.ID); err != nil {
		t.Fatalf("SetActiveHostProfile(p1) error = %v", err)
	}

	got1, _ := db.GetHostProfile(p1.ID)
	got2, _ := db.GetHostProfile(p2.ID)
	if !got1.IsActive {
		t.Fatal("profile-1 IsActive = false, want true")
	}
	if got2.IsActive {
		t.Fatal("profile-2 IsActive = true, want false")
	}

	if err := db.SetActiveHostProfile(p2.ID); err != nil {
		t.Fatalf("SetActiveHostProfile(p2) error = %v", err)
	}

	got1, _ = db.GetHostProfile(p1.ID)
	got2, _ = db.GetHostProfile(p2.ID)
	if got1.IsActive {
		t.Fatal("profile-1 IsActive = true, want false")
	}
	if !got2.IsActive {
		t.Fatal("profile-2 IsActive = false, want true")
	}
}

func TestListExpiringLocalCertificates(t *testing.T) {
	db := newTestStore(t)

	now := time.Now()
	if err := db.CreateCertificate(&Certificate{
		ID:       "cert-expiring",
		Name:     "Expiring",
		Domain:   "*.example.com",
		Source:   SourceLocalCA,
		Status:   CertStatusActive,
		RenewAt:  now.Add(-time.Hour),
	}); err != nil {
		t.Fatalf("CreateCertificate() error = %v", err)
	}
	if err := db.CreateCertificate(&Certificate{
		ID:       "cert-valid",
		Name:     "Valid",
		Domain:   "*.other.com",
		Source:   SourceLocalCA,
		Status:   CertStatusActive,
		RenewAt:  now.Add(30 * 24 * time.Hour),
	}); err != nil {
		t.Fatalf("CreateCertificate() error = %v", err)
	}

	certs, err := db.ListExpiringLocalCertificates(now)
	if err != nil {
		t.Fatalf("ListExpiringLocalCertificates() error = %v", err)
	}
	if len(certs) != 1 || certs[0].ID != "cert-expiring" {
		t.Fatalf("expected 1 expiring cert, got %d", len(certs))
	}
}

func TestRoute_PersistsHeaderManipulationFields(t *testing.T) {
	db := newTestStore(t)

	route := &Route{
		ID:         "hdr-route",
		Name:       "header-route",
		PathPrefix: "/api",
		Backend:    "http://backend:8080",
		Enabled:    true,
		SetRequestHeaders:     map[string]string{"X-Token": "abc", "Authorization": "Bearer tok"},
		RemoveRequestHeaders:  []string{"Cookie", "X-Debug"},
		AddResponseHeaders:    map[string]string{"X-Request-Id": "req-1"},
		RemoveResponseHeaders: []string{"X-Powered-By"},
	}
	if err := db.CreateRoute(route); err != nil {
		t.Fatalf("CreateRoute() error = %v", err)
	}

	got, err := db.GetRoute("hdr-route")
	if err != nil {
		t.Fatalf("GetRoute() error = %v", err)
	}

	// SetRequestHeaders
	if len(got.SetRequestHeaders) != 2 {
		t.Fatalf("len(SetRequestHeaders) = %d, want 2", len(got.SetRequestHeaders))
	}
	if got.SetRequestHeaders["X-Token"] != "abc" {
		t.Errorf("SetRequestHeaders[X-Token] = %q, want %q", got.SetRequestHeaders["X-Token"], "abc")
	}

	// RemoveRequestHeaders
	if len(got.RemoveRequestHeaders) != 2 {
		t.Fatalf("len(RemoveRequestHeaders) = %d, want 2", len(got.RemoveRequestHeaders))
	}
	if got.RemoveRequestHeaders[0] != "Cookie" {
		t.Errorf("RemoveRequestHeaders[0] = %q, want %q", got.RemoveRequestHeaders[0], "Cookie")
	}

	// AddResponseHeaders
	if len(got.AddResponseHeaders) != 1 {
		t.Fatalf("len(AddResponseHeaders) = %d, want 1", len(got.AddResponseHeaders))
	}
	if got.AddResponseHeaders["X-Request-Id"] != "req-1" {
		t.Errorf("AddResponseHeaders[X-Request-Id] = %q, want %q", got.AddResponseHeaders["X-Request-Id"], "req-1")
	}

	// RemoveResponseHeaders
	if len(got.RemoveResponseHeaders) != 1 {
		t.Fatalf("len(RemoveResponseHeaders) = %d, want 1", len(got.RemoveResponseHeaders))
	}
	if got.RemoveResponseHeaders[0] != "X-Powered-By" {
		t.Errorf("RemoveResponseHeaders[0] = %q, want %q", got.RemoveResponseHeaders[0], "X-Powered-By")
	}

	// Update with new header config
	if err := db.UpdateRoute(&Route{
		ID:                    "hdr-route",
		Name:                  "header-route",
		PathPrefix:            "/api",
		Backend:               "http://backend:8080",
		Enabled:               true,
		SetRequestHeaders:     map[string]string{"X-New": "val"},
		RemoveRequestHeaders:  nil,
		AddResponseHeaders:    nil,
		RemoveResponseHeaders: []string{"Server"},
	}); err != nil {
		t.Fatalf("UpdateRoute() error = %v", err)
	}

	got2, err := db.GetRoute("hdr-route")
	if err != nil {
		t.Fatalf("GetRoute() after update error = %v", err)
	}
	if len(got2.SetRequestHeaders) != 1 || got2.SetRequestHeaders["X-New"] != "val" {
		t.Errorf("updated SetRequestHeaders = %v, want map[X-New:val]", got2.SetRequestHeaders)
	}
	if len(got2.RemoveRequestHeaders) != 0 {
		t.Errorf("updated RemoveRequestHeaders = %v, want empty", got2.RemoveRequestHeaders)
	}
	if len(got2.RemoveResponseHeaders) != 1 || got2.RemoveResponseHeaders[0] != "Server" {
		t.Errorf("updated RemoveResponseHeaders = %v, want [Server]", got2.RemoveResponseHeaders)
	}
}
