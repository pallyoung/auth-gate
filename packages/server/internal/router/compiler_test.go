package router

import (
	"testing"

	"github.com/pallyoung/auth-gate/packages/server/internal/store"
)

func TestCompileRoutes_AttachesAuthConfig(t *testing.T) {
	routes := []store.Route{{
		ID:         "route-1",
		Name:       "svc",
		PathPrefix: "/svc",
		Backend:    "http://example.com",
		Enabled:    true,
	}}
	authConfigs := map[string]store.RouteAuthConfig{
		"route-1": {
			RouteID:      "route-1",
			BasicEnabled: true,
			BasicUsername: "service",
			BasicPassword: "secret",
		},
	}

	compiled := compileRoutes(routes, authConfigs, nil)
	if len(compiled) != 1 {
		t.Fatalf("len(compiled) = %d, want 1", len(compiled))
	}
	if compiled[0].AuthConfig == nil {
		t.Fatal("compiled[0].AuthConfig = nil, want auth config")
	}
	if !compiled[0].AuthConfig.BasicEnabled {
		t.Fatal("compiled[0].AuthConfig.BasicEnabled = false, want true")
	}
	if compiled[0].AuthConfig.BasicUsername != "service" {
		t.Fatalf("compiled auth username = %q, want %q", compiled[0].AuthConfig.BasicUsername, "service")
	}
}

func TestCompileRoutes_PropagatesHeaderManipulation(t *testing.T) {
	routes := []store.Route{{
		ID:         "route-h",
		Name:       "headers",
		PathPrefix: "/api",
		Backend:    "http://backend:8080",
		Enabled:    true,
		SetRequestHeaders:    map[string]string{"X-Custom-Token": "abc123", "Authorization": "Bearer injected"},
		RemoveRequestHeaders: []string{"Cookie", "X-Debug"},
		AddResponseHeaders:   map[string]string{"X-Request-Id": "req-42", "X-Response-From": "gateway"},
		RemoveResponseHeaders: []string{"X-Powered-By", "Server"},
	}}

	compiled := compileRoutes(routes, nil, nil)
	if len(compiled) != 1 {
		t.Fatalf("len(compiled) = %d, want 1", len(compiled))
	}
	r := compiled[0]

	// SetRequestHeaders
	if len(r.SetRequestHeaders) != 2 {
		t.Fatalf("len(SetRequestHeaders) = %d, want 2", len(r.SetRequestHeaders))
	}
	if r.SetRequestHeaders["X-Custom-Token"] != "abc123" {
		t.Errorf("SetRequestHeaders[X-Custom-Token] = %q, want %q", r.SetRequestHeaders["X-Custom-Token"], "abc123")
	}

	// RemoveRequestHeaders
	if len(r.RemoveRequestHeaders) != 2 {
		t.Fatalf("len(RemoveRequestHeaders) = %d, want 2", len(r.RemoveRequestHeaders))
	}
	if r.RemoveRequestHeaders[0] != "Cookie" || r.RemoveRequestHeaders[1] != "X-Debug" {
		t.Errorf("RemoveRequestHeaders = %v, want [Cookie X-Debug]", r.RemoveRequestHeaders)
	}

	// AddResponseHeaders
	if len(r.AddResponseHeaders) != 2 {
		t.Fatalf("len(AddResponseHeaders) = %d, want 2", len(r.AddResponseHeaders))
	}
	if r.AddResponseHeaders["X-Request-Id"] != "req-42" {
		t.Errorf("AddResponseHeaders[X-Request-Id] = %q, want %q", r.AddResponseHeaders["X-Request-Id"], "req-42")
	}

	// RemoveResponseHeaders
	if len(r.RemoveResponseHeaders) != 2 {
		t.Fatalf("len(RemoveResponseHeaders) = %d, want 2", len(r.RemoveResponseHeaders))
	}
	if r.RemoveResponseHeaders[0] != "X-Powered-By" || r.RemoveResponseHeaders[1] != "Server" {
		t.Errorf("RemoveResponseHeaders = %v, want [X-Powered-By Server]", r.RemoveResponseHeaders)
	}
}

func TestCompileRoutes_HeaderFieldsNilWhenEmpty(t *testing.T) {
	routes := []store.Route{{
		ID:         "route-no-h",
		Name:       "no-headers",
		PathPrefix: "/plain",
		Backend:    "http://backend:8080",
		Enabled:    true,
	}}

	compiled := compileRoutes(routes, nil, nil)
	if len(compiled) != 1 {
		t.Fatalf("len(compiled) = %d, want 1", len(compiled))
	}
	r := compiled[0]
	if len(r.SetRequestHeaders) != 0 {
		t.Errorf("SetRequestHeaders should be nil/empty, got %v", r.SetRequestHeaders)
	}
	if len(r.RemoveRequestHeaders) != 0 {
		t.Errorf("RemoveRequestHeaders should be nil/empty, got %v", r.RemoveRequestHeaders)
	}
	if len(r.AddResponseHeaders) != 0 {
		t.Errorf("AddResponseHeaders should be nil/empty, got %v", r.AddResponseHeaders)
	}
	if len(r.RemoveResponseHeaders) != 0 {
		t.Errorf("RemoveResponseHeaders should be nil/empty, got %v", r.RemoveResponseHeaders)
	}
}
