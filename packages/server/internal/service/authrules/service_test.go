package authrules

import (
	"path/filepath"
	"testing"

	"github.com/pallyoung/auth-gate/packages/server/internal/store"
)

func newTestDB(t *testing.T) *store.SQLite {
	t.Helper()

	db, err := store.NewSQLite(filepath.Join(t.TempDir(), "auth-gate.db"))
	if err != nil {
		t.Fatalf("NewSQLite() error = %v", err)
	}
	t.Cleanup(func() {
		_ = db.Close()
	})
	return db
}

func stringPtr(v string) *string {
	return &v
}

func createRoute(t *testing.T, db *store.SQLite, id string) {
	t.Helper()

	if err := db.CreateRoute(&store.Route{
		ID:         id,
		Name:       "svc",
		PathPrefix: "/svc",
		Backend:    "http://example.com",
		Enabled:    true,
	}); err != nil {
		t.Fatalf("CreateRoute() error = %v", err)
	}
}

func TestServiceCreateAuthRule_RejectsMissingRoute(t *testing.T) {
	svc := NewService(newTestDB(t), nil)

	_, err := svc.Create(CreateInput{
		RouteID: "missing",
		Type:    "apikey",
		Config: AuthConfigInput{
			Secret: "secret-1",
		},
	})
	if err == nil {
		t.Fatal("Create() error = nil, want route not found")
	}
	if Code(err) != ErrCodeRouteNotFound {
		t.Fatalf("Code(err) = %q, want %q", Code(err), ErrCodeRouteNotFound)
	}
}

func TestServiceCreateAuthRule_RejectsMissingSecret(t *testing.T) {
	db := newTestDB(t)
	createRoute(t, db, "route-1")
	svc := NewService(db, nil)

	_, err := svc.Create(CreateInput{
		RouteID: "route-1",
		Type:    "bearer",
		Config:  AuthConfigInput{},
	})
	if err == nil {
		t.Fatal("Create() error = nil, want validation error")
	}
	if Code(err) != ErrCodeMissingBearerSecret {
		t.Fatalf("Code(err) = %q, want %q", Code(err), ErrCodeMissingBearerSecret)
	}
}

func TestServiceCreateAuthRule_RejectsWhitespaceOnlyBasicPassword(t *testing.T) {
	db := newTestDB(t)
	createRoute(t, db, "route-1")
	svc := NewService(db, nil)

	_, err := svc.Create(CreateInput{
		RouteID: "route-1",
		Type:    "basic",
		Config: AuthConfigInput{
			Username: "service-user",
			Password: "   ",
		},
	})
	if err == nil {
		t.Fatal("Create() error = nil, want validation error")
	}
	if Code(err) != ErrCodeMissingBasicCredentials {
		t.Fatalf("Code(err) = %q, want %q", Code(err), ErrCodeMissingBasicCredentials)
	}
}

func TestServiceUpdateAuthRule_PreservesSecretWhenOmitted(t *testing.T) {
	db := newTestDB(t)
	createRoute(t, db, "route-1")
	svc := NewService(db, nil)

	created, err := svc.Create(CreateInput{
		RouteID: "route-1",
		Type:    "bearer",
		Config: AuthConfigInput{
			Secret: "shared-secret",
		},
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	updated, err := svc.Update(created.ID, UpdateInput{
		RouteID: stringPtr("route-1"),
		Type:    stringPtr("bearer"),
	})
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	if updated.Config.Secret != "shared-secret" {
		t.Fatalf("updated.Config.Secret = %q, want preserved secret", updated.Config.Secret)
	}
}

func TestServiceUpdateAuthRule_PreservesBasicPasswordWhenWhitespaceOnly(t *testing.T) {
	db := newTestDB(t)
	createRoute(t, db, "route-1")
	svc := NewService(db, nil)

	created, err := svc.Create(CreateInput{
		RouteID: "route-1",
		Type:    "basic",
		Config: AuthConfigInput{
			Username: "service-user",
			Password: "shared-password",
		},
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	updated, err := svc.Update(created.ID, UpdateInput{
		Config: UpdateAuthConfigInput{
			Password: stringPtr("   "),
		},
	})
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	if updated.Config.Password != "shared-password" {
		t.Fatalf("updated.Config.Password = %q, want preserved password", updated.Config.Password)
	}
}

func TestServiceUpdateAuthRule_PreservesOmittedTypeRouteAndHeaderState(t *testing.T) {
	db := newTestDB(t)
	createRoute(t, db, "route-1")
	svc := NewService(db, nil)

	created, err := svc.Create(CreateInput{
		RouteID: "route-1",
		Type:    "apikey",
		Config: AuthConfigInput{
			HeaderName: "X-Original-Key",
			Secret:     "shared-secret",
		},
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	updated, err := svc.Update(created.ID, UpdateInput{
		Config: UpdateAuthConfigInput{
			HeaderName: stringPtr("X-Updated-Key"),
		},
	})
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	if updated.RouteID != "route-1" {
		t.Fatalf("updated.RouteID = %q, want %q", updated.RouteID, "route-1")
	}
	if updated.Type != "apikey" {
		t.Fatalf("updated.Type = %q, want %q", updated.Type, "apikey")
	}
	if updated.Config.HeaderName != "X-Updated-Key" {
		t.Fatalf("updated.Config.HeaderName = %q, want %q", updated.Config.HeaderName, "X-Updated-Key")
	}
	if updated.Config.Secret != "shared-secret" {
		t.Fatalf("updated.Config.Secret = %q, want preserved secret", updated.Config.Secret)
	}
}

func TestServiceUpdateAuthRule_PreservesOmittedRuntimePolicyFields(t *testing.T) {
	db := newTestDB(t)
	createRoute(t, db, "route-1")
	if err := db.CreateAuthRule(&store.AuthRule{
		ID:                   "rule-1",
		RouteID:              "route-1",
		Type:                 "apikey",
		Config:               store.AuthConfig{HeaderName: "X-Original-Key", Secret: "shared-secret"},
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

	svc := NewService(db, nil)
	updated, err := svc.Update("rule-1", UpdateInput{
		Config: UpdateAuthConfigInput{
			HeaderName: stringPtr("X-Updated-Key"),
		},
	})
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	if updated.Config.HeaderName != "X-Updated-Key" {
		t.Fatalf("updated.Config.HeaderName = %q, want %q", updated.Config.HeaderName, "X-Updated-Key")
	}
	if len(updated.Whitelist) != 2 || updated.Whitelist[0] != "127.0.0.1/32" || updated.Whitelist[1] != "10.0.0.0/8" {
		t.Fatalf("updated.Whitelist = %#v, want %#v", updated.Whitelist, []string{"127.0.0.1/32", "10.0.0.0/8"})
	}
	if updated.RateLimit != 12 {
		t.Fatalf("updated.RateLimit = %d, want %d", updated.RateLimit, 12)
	}
	if updated.Burst != 24 {
		t.Fatalf("updated.Burst = %d, want %d", updated.Burst, 24)
	}
	if updated.CORSAllowedOrigins != "https://app.example.com,.example.com" {
		t.Fatalf("updated.CORSAllowedOrigins = %q, want %q", updated.CORSAllowedOrigins, "https://app.example.com,.example.com")
	}
	if updated.CORSAllowedMethods != "GET,POST,OPTIONS" {
		t.Fatalf("updated.CORSAllowedMethods = %q, want %q", updated.CORSAllowedMethods, "GET,POST,OPTIONS")
	}
	if updated.CORSAllowedHeaders != "Authorization,Content-Type" {
		t.Fatalf("updated.CORSAllowedHeaders = %q, want %q", updated.CORSAllowedHeaders, "Authorization,Content-Type")
	}
	if !updated.CORSAllowCredentials {
		t.Fatal("updated.CORSAllowCredentials = false, want true")
	}
	if updated.CORSMaxAge != 7200 {
		t.Fatalf("updated.CORSMaxAge = %d, want %d", updated.CORSMaxAge, 7200)
	}
}

func TestServiceCreateAuthRule_AllowsGatewayType(t *testing.T) {
	db := newTestDB(t)
	createRoute(t, db, "route-1")
	svc := NewService(db, nil)

	rule, err := svc.Create(CreateInput{
		RouteID: "route-1",
		Type:    "gateway",
		Config:  AuthConfigInput{},
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if rule.Type != "gateway" {
		t.Fatalf("rule.Type = %q, want gateway", rule.Type)
	}
	if rule.Config.LoginMode != "form" {
		t.Fatalf("rule.Config.LoginMode = %q, want form", rule.Config.LoginMode)
	}
}

func TestServiceGetAuthRule_NormalizesLegacyStoredType(t *testing.T) {
	db := newTestDB(t)
	createRoute(t, db, "route-1")
	if err := db.CreateAuthRule(&store.AuthRule{
		ID:      "rule-1",
		RouteID: "route-1",
		Type:    " BASIC ",
		Config: store.AuthConfig{
			Username: "service-user",
			Password: "shared-password",
		},
	}); err != nil {
		t.Fatalf("CreateAuthRule() error = %v", err)
	}

	svc := NewService(db, nil)
	rule, err := svc.Get("rule-1")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if rule.Type != "basic" {
		t.Fatalf("rule.Type = %q, want %q", rule.Type, "basic")
	}
}

func TestServiceUpdateAuthRule_PreservesCredentialsForLegacyStoredType(t *testing.T) {
	db := newTestDB(t)
	createRoute(t, db, "route-1")
	if err := db.CreateAuthRule(&store.AuthRule{
		ID:      "rule-1",
		RouteID: "route-1",
		Type:    " BASIC ",
		Config: store.AuthConfig{
			Username: "service-user",
			Password: "shared-password",
		},
	}); err != nil {
		t.Fatalf("CreateAuthRule() error = %v", err)
	}

	svc := NewService(db, nil)
	updated, err := svc.Update("rule-1", UpdateInput{
		Type: stringPtr("basic"),
		Config: UpdateAuthConfigInput{
			Username: stringPtr("service-user-updated"),
		},
	})
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	if updated.Type != "basic" {
		t.Fatalf("updated.Type = %q, want %q", updated.Type, "basic")
	}
	if updated.Config.Username != "service-user-updated" {
		t.Fatalf("updated.Config.Username = %q, want %q", updated.Config.Username, "service-user-updated")
	}
	if updated.Config.Password != "shared-password" {
		t.Fatalf("updated.Config.Password = %q, want preserved password", updated.Config.Password)
	}
}
