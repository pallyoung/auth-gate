package proxy

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/pallyoung/auth-gate/packages/server/internal/auth"
	"github.com/pallyoung/auth-gate/packages/server/internal/router"
	"github.com/pallyoung/auth-gate/packages/server/internal/store"
)

func TestAccessLogCapturesOriginalHostname(t *testing.T) {
	for _, mode := range []string{"success", "api_key_failure", "gateway_redirect", "gateway_success", "gateway_path_denied"} {
		t.Run(mode, func(t *testing.T) {
			upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) }))
			defer upstream.Close()
			db := newProxyTestDB(t)
			route := &store.Route{ID: "log-host", PathPrefix: "/api", Backend: upstream.URL, Enabled: true, StripPrefix: true}
			if err := db.CreateRoute(route); err != nil {
				t.Fatal(err)
			}
			if mode != "success" {
				cfg := &store.RouteAuthConfig{RouteID: route.ID, ApiKeyEnabled: mode == "api_key_failure", ApiKeyHeader: "X-API-Key", GatewayEnabled: mode != "api_key_failure"}
				if err := db.CreateOrUpdateRouteAuthConfig(cfg); err != nil {
					t.Fatal(err)
				}
			}
			logs, err := store.NewAccessLogStore(t.TempDir(), 10)
			if err != nil {
				t.Fatal(err)
			}
			engine := gin.New()
			engine.Any("/*path", Handler(router.NewManager(db), logs))
			req := httptest.NewRequest(http.MethodGet, "http://Client.Example.:8443/api/resource", nil)
			if mode == "gateway_success" || mode == "gateway_path_denied" {
				auth.ConfigureJWTSecret("log-gateway-test")
				user := &store.User{Username: "alice", Role: store.RoleViewer, Enabled: true, RouteIDs: []string{route.ID}}
				if err := db.CreateUser(user); err != nil {
					t.Fatal(err)
				}
				var paths map[string][]string
				if mode == "gateway_path_denied" {
					paths = map[string][]string{route.ID: {"/allowed"}}
				}
				token, err := auth.GenerateRouteAccessToken(user.ID, user.Username, user.Role, user.RouteIDs, paths)
				if err != nil {
					t.Fatal(err)
				}
				req.AddCookie(&http.Cookie{Name: routeAccessCookieName, Value: token})
			}
			req.Header.Set("X-Forwarded-Host", "untrusted.example")
			resp := httptest.NewRecorder()
			engine.ServeHTTP(resp, req)
			entries, total := logs.Query(store.AccessLogFilter{}, 0, 10)
			if total != 1 {
				t.Fatalf("expected one log for %s, got %d (status %d)", mode, total, resp.Code)
			}
			if entries[0].Host != "client.example" {
				t.Fatalf("host = %q, want original hostname", entries[0].Host)
			}
			if entries[0].Path != "/api/resource" {
				t.Fatalf("path = %q, want incoming path", entries[0].Path)
			}
			if mode != "success" && mode != "gateway_success" && entries[0].AuthResult != "fail" {
				t.Fatalf("auth_result = %q", entries[0].AuthResult)
			}
			if mode == "gateway_success" || mode == "gateway_path_denied" {
				if entries[0].Username != "alice" {
					t.Fatalf("gateway username = %q", entries[0].Username)
				}
			}
		})
	}
}
