package router

import (
	"testing"

	"github.com/pallyoung/auth-gate/packages/server/internal/store"
)

func TestCompileRoutes_AttachesCompiledAuthRule(t *testing.T) {
	routes := []store.Route{{
		ID:         "route-1",
		Name:       "svc",
		PathPrefix: "/svc",
		Backend:    "http://example.com",
		Enabled:    true,
	}}
	authRules := []store.AuthRule{{
		ID:      "rule-1",
		RouteID: "route-1",
		Type:    "basic",
		Config: store.AuthConfig{
			Username: "service",
			Password: "secret",
		},
	}}

	compiled := compileRoutes(routes, authRules)
	if len(compiled) != 1 {
		t.Fatalf("len(compiled) = %d, want 1", len(compiled))
	}
	if compiled[0].AuthRule == nil {
		t.Fatal("compiled[0].AuthRule = nil, want compiled auth rule")
	}
	if compiled[0].AuthRule.Config.Username != "service" {
		t.Fatalf("compiled auth username = %q, want %q", compiled[0].AuthRule.Config.Username, "service")
	}
}
