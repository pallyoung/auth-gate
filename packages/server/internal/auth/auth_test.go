package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/pallyoung/auth-gate/packages/server/internal/store"

	"github.com/gin-gonic/gin"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func makeTestContext(method, path string, fns ...func(*http.Request)) (*gin.Context, *httptest.ResponseRecorder) {
	req := httptest.NewRequest(method, path, nil)
	for _, fn := range fns {
		fn(req)
	}
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req
	return c, w
}

func withHeader(name, value string) func(*http.Request) {
	return func(r *http.Request) { r.Header.Set(name, value) }
}

func withBasicAuth(user, pass string) func(*http.Request) {
	return func(r *http.Request) { r.SetBasicAuth(user, pass) }
}

func withQuery(query string) func(*http.Request) {
	return func(r *http.Request) { r.URL.RawQuery = query }
}

func TestCheck_APIKey(t *testing.T) {
	rule := &store.AuthRule{Type: "apikey", Config: store.AuthConfig{Secret: "secret-abc"}}

	tests := []struct {
		name  string
		fns   []func(*http.Request)
		want  bool
	}{
		{"Valid header key", []func(*http.Request){withHeader("X-API-Key", "secret-abc")}, true},
		{"Wrong key", []func(*http.Request){withHeader("X-API-Key", "wrong-key")}, false},
		{"Missing key", nil, false},
		{"Valid query key", []func(*http.Request){withQuery("api_key=secret-abc")}, true},
		{"Wrong query key", []func(*http.Request){withQuery("api_key=wrong")}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, _ := makeTestContext("GET", "/api/test", tt.fns...)
			if got := checkAPIKey(c, rule); got != tt.want {
				t.Errorf("checkAPIKey() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCheck_APIKey_CustomHeader(t *testing.T) {
	rule := &store.AuthRule{Type: "apikey", Config: store.AuthConfig{HeaderName: "X-Custom-Auth", Secret: "my-secret"}}

	c, _ := makeTestContext("GET", "/api/test", withHeader("X-Custom-Auth", "my-secret"))
	if got := checkAPIKey(c, rule); !got {
		t.Errorf("checkAPIKey() with custom header = %v, want true", got)
	}

	c2, _ := makeTestContext("GET", "/api/test", withHeader("X-API-Key", "my-secret"))
	if got := checkAPIKey(c2, rule); got {
		t.Errorf("checkAPIKey() with wrong header = %v, want false", got)
	}
}

func TestCheck_Bearer_JWT(t *testing.T) {
	// Generate tokens using the module's JWTSecret (used by GenerateToken).
	validToken, _ := GenerateToken("user-bearer", "bearertest", "viewer")

	tests := []struct {
		name      string
		authHdr   string
		secret    []byte
		want      bool
	}{
		{"Valid bearer token", "Bearer " + validToken, JWTSecret, true},
		{"lowercase bearer", "bearer " + validToken, JWTSecret, true},
		{"Wrong secret", "Bearer " + validToken, []byte("wrong-secret"), false},
		{"Missing token", "Bearer", JWTSecret, false},
		{"Empty header", "", JWTSecret, false},
		{"No bearer prefix", "Basic dXNlcjpwYXNz", JWTSecret, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rule := &store.AuthRule{Type: "bearer", Config: store.AuthConfig{Secret: string(tt.secret)}}
			c, _ := makeTestContext("GET", "/api/test", withHeader("Authorization", tt.authHdr))
			if got := checkBearer(c, rule); got != tt.want {
				t.Errorf("checkBearer() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCheck_Bearer_NoSecretConfigured(t *testing.T) {
	rule := &store.AuthRule{Type: "bearer", Config: store.AuthConfig{Secret: ""}}
	c, _ := makeTestContext("GET", "/api/test", withHeader("Authorization", "Bearer somerandomtoken"))
	if got := checkBearer(c, rule); got {
		t.Errorf("checkBearer() = %v, want false when no secret is configured", got)
	}
}

func TestCheck_Bearer_StoresClaims(t *testing.T) {
	// Generate token with module JWTSecret so it validates with the rule's secret.
	token, _ := GenerateToken("claim-user", "claimtest", "editor")
	rule := &store.AuthRule{Type: "bearer", Config: store.AuthConfig{Secret: string(JWTSecret)}}
	c, _ := makeTestContext("GET", "/api/test", withHeader("Authorization", "Bearer "+token))

	checkBearer(c, rule)

	if v, exists := c.Get("jwt_subject"); !exists || v != "claim-user" {
		t.Errorf("jwt_subject = %v, want %q", v, "claim-user")
	}
	if v, exists := c.Get("jwt_username"); !exists || v != "claimtest" {
		t.Errorf("jwt_username = %v, want %q", v, "claimtest")
	}
	if v, exists := c.Get("jwt_role"); !exists || v != "editor" {
		t.Errorf("jwt_role = %v, want %q", v, "editor")
	}
}

func TestCheck_Basic(t *testing.T) {
	rule := &store.AuthRule{Type: "basic", Config: store.AuthConfig{Username: "admin", Password: "supersecret"}}

	tests := []struct {
		name  string
		fn    func(*http.Request)
		want  bool
	}{
		{"Valid credentials", withBasicAuth("admin", "supersecret"), true},
		{"Wrong password", withBasicAuth("admin", "wrongpass"), false},
		{"Wrong username", withBasicAuth("user", "supersecret"), false},
		{"No credentials", func(r *http.Request) {}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, _ := makeTestContext("GET", "/api/test", tt.fn)
			if got := checkBasic(c, rule); got != tt.want {
				t.Errorf("checkBasic() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCheck_TypeNone(t *testing.T) {
	rule := &store.AuthRule{Type: "none"}
	c, _ := makeTestContext("GET", "/api/test")
	if got := Check(c, rule); !got {
		t.Errorf("Check(type=none) = %v, want true", got)
	}
}

func TestCheck_UnknownType(t *testing.T) {
	rule := &store.AuthRule{Type: "unknown"}
	c, _ := makeTestContext("GET", "/api/test")
	if got := Check(c, rule); !got {
		t.Errorf("Check(type=unknown) = %v, want true (fallback)", got)
	}
}

func TestRequireAuth(t *testing.T) {
	handler := RequireAuth(`Basic realm="test"`)
	c, w := makeTestContext("GET", "/")
	handler(c)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
	if h := w.Header().Get("WWW-Authenticate"); h != `Basic realm="test"` {
		t.Errorf("WWW-Authenticate = %q, want %q", h, `Basic realm="test"`)
	}
}
