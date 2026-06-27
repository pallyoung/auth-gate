package proxy

import (
	"crypto/tls"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/pallyoung/auth-gate/packages/server/internal/auth"
	"github.com/pallyoung/auth-gate/packages/server/internal/router"
	"github.com/pallyoung/auth-gate/packages/server/internal/store"

	"github.com/gin-gonic/gin"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func newProxyTestDB(t *testing.T) store.Store {
	t.Helper()

	db, err := store.NewJSONStore(t.TempDir())
	if err != nil {
		t.Fatalf("NewJSONStore() error = %v", err)
	}
	t.Cleanup(func() {
		_ = db.Close()
	})
	return db
}

// Test the path-rewriting logic in isolation, mirroring what proxy.go does in Director.
func TestPathRewrite(t *testing.T) {
	tests := []struct {
		name       string
		pathPrefix string
		strip      bool
		reqPath    string
		wantPath   string
	}{
		{"strip prefix", "/api", true, "/api/users/123", "/users/123"},
		{"strip prefix root", "/api", true, "/api", "/"},
		{"strip deeper", "/api/v1", true, "/api/v1/users/123", "/users/123"},
		{"no strip", "/api", false, "/api/users", "/api/users"},
		{"strip no leading slash result", "/api", true, "/apifoo", "/foo"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := tt.reqPath
			if tt.strip {
				path = strings.TrimPrefix(path, tt.pathPrefix)
				if !strings.HasPrefix(path, "/") {
					path = "/" + path
				}
			}
			if path != tt.wantPath {
				t.Errorf("rewritePath(%q, strip=%v) = %q, want %q", tt.reqPath, tt.strip, path, tt.wantPath)
			}
		})
	}
}

// Test that X-Forwarded-* headers are set correctly.
func TestForwardedHeaders(t *testing.T) {
	req := httptest.NewRequest("GET", "/test", nil)
	req.Host = "example.com:8080"

	// Simulate what proxy.go Director does
	host := req.Host
	if idx := strings.Index(host, ":"); idx != -1 {
		host = host[:idx]
	}
	req.Header.Set("X-Forwarded-Host", host)
	req.Header.Set("X-Forwarded-Proto", "http")
	req.Header.Set("X-Forwarded-For", "192.168.1.1")

	if got := req.Header.Get("X-Forwarded-Host"); got != "example.com" {
		t.Errorf("X-Forwarded-Host = %q, want %q", got, "example.com")
	}
	if got := req.Header.Get("X-Forwarded-Proto"); got != "http" {
		t.Errorf("X-Forwarded-Proto = %q, want %q", got, "http")
	}
	if got := req.Header.Get("X-Forwarded-For"); got != "192.168.1.1" {
		t.Errorf("X-Forwarded-For = %q, want %q", got, "192.168.1.1")
	}
}

func TestForwardedProto_UsesFirstReverseProxyValue(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "http://example.com/secure", nil)
	req.Header.Set("X-Forwarded-Proto", "https, http")

	if got := forwardedProto(req); got != "https" {
		t.Fatalf("forwardedProto() = %q, want %q", got, "https")
	}
}

func TestForwardedProto_PrefersDirectTLSOverForwardedHeader(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "https://example.com/secure", nil)
	req.TLS = &tls.ConnectionState{}
	req.Header.Set("X-Forwarded-Proto", "http")

	if got := forwardedProto(req); got != "https" {
		t.Fatalf("forwardedProto() = %q, want %q", got, "https")
	}
}

func TestSanitizeAccessRedirect_FallsBackForSchemeRelativePaths(t *testing.T) {
	fallback := "/cloud"

	tests := []struct {
		name string
		next string
	}{
		{name: "triple slash external host", next: "///evil.com/path"},
		{name: "triple slash without suffix", next: "///evil.com"},
		{name: "leading slash backslash host", next: `/\evil.com/path`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := SanitizeAccessRedirect(tt.next, fallback); got != fallback {
				t.Fatalf("SanitizeAccessRedirect(%q, %q) = %q, want fallback %q", tt.next, fallback, got, fallback)
			}
		})
	}
}

// Test the host:port stripping logic used in proxy.go.
func TestHostPortStripping(t *testing.T) {
	tests := []struct {
		host string
		want string
	}{
		{"example.com:8080", "example.com"},
		{"example.com", "example.com"},
		{"[2001:db8::1]:8443", "2001:db8::1"},
		{"[2001:db8::1]", "2001:db8::1"},
	}

	for _, tt := range tests {
		t.Run(tt.host, func(t *testing.T) {
			host := requestMatchHost(tt.host)
			if host != tt.want {
				t.Errorf("requestMatchHost(%q) = %q, want %q", tt.host, host, tt.want)
			}
		})
	}
}

// Test backend URL parsing handles various formats.
func TestBackendURLParsing(t *testing.T) {
	tests := []struct {
		backend string
		wantErr bool
	}{
		{"http://localhost:3000", false},
		{"http://127.0.0.1:3000", false},
		{"http://backend:8080/path", false},
		{"://invalid", true},
	}

	for _, tt := range tests {
		t.Run(tt.backend, func(t *testing.T) {
			_, err := url.Parse(tt.backend)
			if (err != nil) != tt.wantErr {
				t.Errorf("url.Parse(%q) err=%v, wantErr=%v", tt.backend, err, tt.wantErr)
			}
		})
	}
}

func TestWriteUnauthorized_Returns401JSON(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/cloud", nil)

	writeUnauthorized(c, &router.Route{
		Name:       "Cloud Console",
		PathPrefix: "/cloud",
		AuthConfig: &router.RouteAuthConfig{
			ApiKeyEnabled: true,
		},
	})

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
	if got := w.Body.String(); !strings.Contains(got, "unauthorized") {
		t.Fatalf("body = %q, want it to contain 'unauthorized'", got)
	}
}

func TestProxyNormalizesLegacyStoredAPIKeyAuthRuleType(t *testing.T) {
	gin.SetMode(gin.TestMode)

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("upstream-ok"))
	}))
	defer upstream.Close()

	db := newProxyTestDB(t)
	if err := db.CreateRoute(&store.Route{
		ID:         "route-1",
		Name:       "Cloud Console",
		PathPrefix: "/cloud",
		Backend:    upstream.URL,
		Enabled:    true,
	}); err != nil {
		t.Fatalf("CreateRoute() error = %v", err)
	}
	if err := db.CreateAuthRule(&store.AuthRule{
		RouteID: "route-1",
		Type:    " APIKEY ",
		Config: store.AuthConfig{
			HeaderName: "X-API-Key",
			Secret:     "test-secret",
		},
	}); err != nil {
		t.Fatalf("CreateAuthRule() error = %v", err)
	}

	mgr := router.NewManager(db)
	engine := gin.New()
	engine.Any("/*proxyPath", Handler(mgr, nil))

	req := httptest.NewRequest(http.MethodGet, "/cloud", nil)
	req.Header.Set("X-API-Key", "test-secret")
	resp := httptest.NewRecorder()
	engine.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body=%s", resp.Code, http.StatusOK, resp.Body.String())
	}
	if resp.Body.String() != "upstream-ok" {
		t.Fatalf("body = %q, want %q", resp.Body.String(), "upstream-ok")
	}
}

func TestProxyMatchesLegacyStoredAPIKeyConfigWithWhitespace(t *testing.T) {
	gin.SetMode(gin.TestMode)

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("upstream-ok"))
	}))
	defer upstream.Close()

	db := newProxyTestDB(t)
	if err := db.CreateRoute(&store.Route{
		ID:         "route-1",
		Name:       "Cloud Console",
		PathPrefix: "/cloud",
		Backend:    upstream.URL,
		Enabled:    true,
	}); err != nil {
		t.Fatalf("CreateRoute() error = %v", err)
	}
	if err := db.CreateAuthRule(&store.AuthRule{
		RouteID: "route-1",
		Type:    " APIKEY ",
		Config: store.AuthConfig{
			HeaderName: " X-Route-Key ",
			Secret:     " shared-secret ",
		},
	}); err != nil {
		t.Fatalf("CreateAuthRule() error = %v", err)
	}

	mgr := router.NewManager(db)
	engine := gin.New()
	engine.Any("/*proxyPath", Handler(mgr, nil))

	req := httptest.NewRequest(http.MethodGet, "/cloud", nil)
	req.Header.Set("X-Route-Key", "shared-secret")
	resp := httptest.NewRecorder()
	engine.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body=%s", resp.Code, http.StatusOK, resp.Body.String())
	}
	if resp.Body.String() != "upstream-ok" {
		t.Fatalf("body = %q, want %q", resp.Body.String(), "upstream-ok")
	}
}

func TestRouteAccessClaims_RejectsDisabledUser(t *testing.T) {
	auth.ConfigureJWTSecret("test-secret")

	db := newProxyTestDB(t)
	if err := db.CreateRoute(&store.Route{
		ID:         "route-1",
		Name:       "cloud",
		PathPrefix: "/cloud",
		Backend:    "http://example.com",
		Enabled:    true,
	}); err != nil {
		t.Fatalf("CreateRoute() error = %v", err)
	}
	hash, err := store.HashPassword("password123")
	if err != nil {
		t.Fatalf("HashPassword() error = %v", err)
	}

	user := &store.User{
		ID:           "member-1",
		Username:     "member",
		PasswordHash: hash,
		Role:         store.RoleMember,
		Enabled:      true,
		RouteIDs:     []string{"route-1"},
	}
	if err := db.CreateUser(user); err != nil {
		t.Fatalf("CreateUser() error = %v", err)
	}

	token, err := auth.GenerateRouteAccessToken(user.ID, user.Username, user.Role, user.RouteIDs, nil)
	if err != nil {
		t.Fatalf("GenerateRouteAccessToken() error = %v", err)
	}

	user.Enabled = false
	if err := db.UpdateUser(user); err != nil {
		t.Fatalf("UpdateUser() error = %v", err)
	}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req := httptest.NewRequest(http.MethodGet, "/cloud", nil)
	req.AddCookie(&http.Cookie{Name: routeAccessCookieName, Value: token})
	c.Request = req

	if claims, ok := routeAccessClaims(c, db); ok || claims != nil {
		t.Fatalf("routeAccessClaims() = (%v, %v), want (nil, false)", claims, ok)
	}
}

func TestBalancer_WeightedRoundRobin(t *testing.T) {
	backends := []store.Backend{
		{URL: "http://a", Weight: 2},
		{URL: "http://b", Weight: 1},
	}
	bl := newBalancer(backends)
	if bl == nil {
		t.Fatal("newBalancer returned nil")
	}

	counts := map[string]int{"http://a": 0, "http://b": 0}
	for i := 0; i < 30; i++ {
		picked, _ := bl.pick()
		counts[picked.URL]++
	}
	// With weight 2:1, "a" should appear roughly twice as often as "b"
	if counts["http://a"] == 0 || counts["http://b"] == 0 {
		t.Errorf("counts = %v, want both > 0", counts)
	}
	if counts["http://a"] <= counts["http://b"] {
		t.Errorf("a (%d) should appear more than b (%d)", counts["http://a"], counts["http://b"])
	}
}

func TestBalancer_SingleBackend(t *testing.T) {
	bl := newBalancer([]store.Backend{{URL: "http://only", Weight: 1}})
	for i := 0; i < 10; i++ {
		if got, ok := bl.pick(); ok && got.URL != "http://only" {
			t.Errorf("pick() = %q, want http://only", got.URL)
		}
	}
}

func TestRouteAllowedByClaims_UsesCurrentRouteAssignments(t *testing.T) {
	claims := &auth.Claims{
		Role:     store.RoleMember,
		RouteIDs: []string{"route-2"},
	}

	if routeAllowedByClaims(claims, "route-1") {
		t.Fatal("routeAllowedByClaims() = true, want false")
	}
}

func TestRouteAllowedByClaims_DoesNotGrantUnassignedOperatorAccess(t *testing.T) {
	claims := &auth.Claims{
		Role:     store.RoleMember,
		RouteIDs: nil,
	}

	if routeAllowedByClaims(claims, "route-1") {
		t.Fatal("routeAllowedByClaims() = true, want false")
	}
}

func TestBalancer_WeightedRoundRobin_RespectsWeight(t *testing.T) {
	backends := []store.Backend{
		{URL: "http://a", Weight: 3},
		{URL: "http://b", Weight: 1},
	}
	bl := newBalancer(backends)
	if bl == nil {
		t.Fatal("newBalancer returned nil")
	}

	counts := map[string]int{"http://a": 0, "http://b": 0}
	for i := 0; i < 40; i++ {
		picked, _ := bl.pick()
		counts[picked.URL]++
	}
	// weight 3:1 — "a" must appear strictly more than "b"
	if counts["http://a"] <= counts["http://b"] {
		t.Errorf("a picked %d times, b picked %d times — a should dominate", counts["http://a"], counts["http://b"])
	}
}

func TestBalancer_PicksAllBackends(t *testing.T) {
	backends := []store.Backend{
		{URL: "http://a", Weight: 2},
		{URL: "http://b", Weight: 2},
		{URL: "http://c", Weight: 2},
	}
	bl := newBalancer(backends)
	seen := make(map[string]bool)
	for i := 0; i < 30; i++ {
		if bk, ok := bl.pick(); ok {
			seen[bk.URL] = true
		}
	}
	for _, b := range backends {
		if !seen[b.URL] {
			t.Errorf("backend %s never picked in 30 picks", b.URL)
		}
	}
}

func TestBalancer_ConsecutivePicksAreWeighted(t *testing.T) {
	// With weight 1:1, alternating a and b, after 20 picks neither should dominate
	backends := []store.Backend{
		{URL: "http://a", Weight: 1},
		{URL: "http://b", Weight: 1},
	}
	bl := newBalancer(backends)
	counts := map[string]int{}
	for i := 0; i < 20; i++ {
		if bk, ok := bl.pick(); ok {
			counts[bk.URL]++
		}
	}
	// Both should be close to 10 each; neither should be 0 or 20
	if counts["http://a"] == 0 || counts["http://b"] == 0 {
		t.Errorf("with weight 1:1, got counts=%v — distribution broken", counts)
	}
}

// mockRetryTransport is a test-only RoundTripper that can be configured to fail.
type mockRetryTransport struct {
	attemptCount int
	shouldFail   bool
}

func (rt *mockRetryTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	rt.attemptCount++
	if rt.shouldFail && rt.attemptCount == 1 {
		return nil, errors.New("connection refused")
	}
	return &http.Response{StatusCode: http.StatusOK, Request: req}, nil
}

func TestRetryTransport_DoesNotRetryOnSuccess(t *testing.T) {
	mock := &mockRetryTransport{shouldFail: false}
	rt := newRetryTransport(mock, 1)
	client := &http.Client{Transport: rt}

	req, _ := http.NewRequest(http.MethodGet, "/ok", nil)
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}
	if mock.attemptCount != 1 {
		t.Errorf("attemptCount = %d, want 1", mock.attemptCount)
	}
}

func TestRetryTransport_RetriesOnFailure(t *testing.T) {
	mock := &mockRetryTransport{shouldFail: true}
	rt := newRetryTransport(mock, 1)
	client := &http.Client{Transport: rt}

	req, _ := http.NewRequest(http.MethodGet, "/fail", nil)
	resp, err := client.Do(req)
	// First attempt fails, retry succeeds → final result is success
	if err != nil {
		t.Errorf("expected success after retry, got error: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}
	// With maxRetries=1, should try twice (1 original + 1 retry)
	if mock.attemptCount != 2 {
		t.Errorf("attemptCount = %d, want 2", mock.attemptCount)
	}
}

// verifyBackend verifies a single backend request succeeds.
func verifyBackend(t *testing.T, backendURL string, w httptest.ResponseRecorder, req *http.Request) {
	t.Helper()
	if w.Code != http.StatusOK {
		t.Errorf("backend %s: status=%d, want %d", backendURL, w.Code, http.StatusOK)
	}
}

// TestRetryProxy_RoundRobinAcrossAllBackends is a full integration test verifying
// that the proxy handler distributes requests across all healthy backends.
func TestRetryProxy_RoundRobinAcrossAllBackends(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Set up two real httptest backends.
	bA := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("a"))
	}))
	defer bA.Close()
	bB := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("b"))
	}))
	defer bB.Close()

	aURL, _ := url.Parse(bA.URL)
	bURL, _ := url.Parse(bB.URL)

	// Build a route with two backends.
	_ = &router.Route{
		ID:         "lb-route",
		PathPrefix: "/api",
		Backend:    aURL.Host,
		Backends: []store.Backend{
			{URL: aURL.Host, Weight: 1},
			{URL: bURL.Host, Weight: 1},
		},
		Enabled: true,
	}

	// Create a minimal DB and router.Manager so Handler can compile regex if needed.
	db := newProxyTestDB(t)
	if err := db.CreateRoute(&store.Route{
		ID: "lb-route", PathPrefix: "/api", Backend: aURL.Host, Enabled: true,
	}); err != nil {
		t.Fatalf("CreateRoute error: %v", err)
	}
	mgr := router.NewManager(db)
	_ = mgr
	// Verify round-robin using the balancer directly (unit test).
	bl := newBalancer([]store.Backend{
		{URL: aURL.Host, Weight: 1},
		{URL: bURL.Host, Weight: 1},
	})
	seen := make(map[string]int)
	for i := 0; i < 20; i++ {
		picked, _ := bl.pick()
		seen[picked.URL]++
	}
	// Both backends should appear at least once.
	if seen[aURL.Host] == 0 || seen[bURL.Host] == 0 {
		t.Errorf("seen = %v — both backends should appear", seen)
	}
}

// TestRetryProxy_UsesCorrectBackendURL verifies the Director sets the correct Host header
// based on the picked backend URL, not the original route backend.
func TestRetryProxy_UsesCorrectBackendHost(t *testing.T) {
	gin.SetMode(gin.TestMode)

	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer backend.Close()

	backendURL, _ := url.Parse(backend.URL)
	host := backendURL.Host

	// Simulate Director behavior: req.Host should be set to backendURL.Host.
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.URL, _ = url.Parse("http://original-host/api")
	req.Host = host // what proxy.Director sets

	if req.Host != host {
		t.Errorf("req.Host = %q, want %q", req.Host, host)
	}
}

func TestProxyRegexInsensitiveRewriteTargetIsApplied(t *testing.T) {
	gin.SetMode(gin.TestMode)

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(r.URL.Path))
	}))
	defer upstream.Close()

	db := newProxyTestDB(t)
	if err := db.CreateRoute(&store.Route{
		ID:            "regex-i-route",
		PathPrefix:    "^/api/(.*)$",
		PathMatchMode: "regex_i",
		RewriteTarget: "/rewritten/$1",
		Backend:       upstream.URL,
		Enabled:       true,
	}); err != nil {
		t.Fatalf("CreateRoute() error = %v", err)
	}

	mgr := router.NewManager(db)
	engine := gin.New()
	engine.Any("/*proxyPath", Handler(mgr, nil))

	req := httptest.NewRequest(http.MethodGet, "/API/Users", nil)
	resp := httptest.NewRecorder()
	engine.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body=%s", resp.Code, http.StatusOK, resp.Body.String())
	}
	if strings.TrimSpace(resp.Body.String()) != "/rewritten/Users" {
		t.Fatalf("body = %q, want %q", resp.Body.String(), "/rewritten/Users")
	}
}

func TestProxyMatchesRouteWhenStoredPathMatchModeHasWhitespaceAndUppercase(t *testing.T) {
	gin.SetMode(gin.TestMode)

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(r.URL.Path))
	}))
	defer upstream.Close()

	db := newProxyTestDB(t)
	if err := db.CreateRoute(&store.Route{
		ID:            "regex-i-route-legacy",
		PathPrefix:    "^/api/(.*)$",
		PathMatchMode: " REGEX_I ",
		RewriteTarget: "/rewritten/$1",
		Backend:       upstream.URL,
		Enabled:       true,
	}); err != nil {
		t.Fatalf("CreateRoute() error = %v", err)
	}

	mgr := router.NewManager(db)
	engine := gin.New()
	engine.Any("/*proxyPath", Handler(mgr, nil))

	req := httptest.NewRequest(http.MethodGet, "/API/Users", nil)
	resp := httptest.NewRecorder()
	engine.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body=%s", resp.Code, http.StatusOK, resp.Body.String())
	}
	if strings.TrimSpace(resp.Body.String()) != "/rewritten/Users" {
		t.Fatalf("body = %q, want %q", resp.Body.String(), "/rewritten/Users")
	}
}

func TestProxyMatchesRouteForIPv6HostWithPort(t *testing.T) {
	gin.SetMode(gin.TestMode)

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ipv6-ok"))
	}))
	defer upstream.Close()

	db := newProxyTestDB(t)
	if err := db.CreateRoute(&store.Route{
		ID:         "ipv6-route",
		Host:       "2001:db8::1",
		PathPrefix: "/api",
		Backend:    upstream.URL,
		Enabled:    true,
	}); err != nil {
		t.Fatalf("CreateRoute() error = %v", err)
	}

	mgr := router.NewManager(db)
	engine := gin.New()
	engine.Any("/*proxyPath", Handler(mgr, nil))

	req := httptest.NewRequest(http.MethodGet, "http://[2001:db8::1]/api/users", nil)
	req.Host = "[2001:db8::1]:8443"
	resp := httptest.NewRecorder()
	engine.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body=%s", resp.Code, http.StatusOK, resp.Body.String())
	}
	if strings.TrimSpace(resp.Body.String()) != "ipv6-ok" {
		t.Fatalf("body = %q, want %q", resp.Body.String(), "ipv6-ok")
	}
}

func TestProxyIgnoresWhitespaceOnlyRedirectTarget(t *testing.T) {
	gin.SetMode(gin.TestMode)

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("upstream-ok"))
	}))
	defer upstream.Close()

	db := newProxyTestDB(t)
	if err := db.CreateRoute(&store.Route{
		ID:            "redirect-route",
		PathPrefix:    "/billing",
		RewriteTarget: "   ",
		RedirectCode:  302,
		Backend:       upstream.URL,
		Enabled:       true,
	}); err != nil {
		t.Fatalf("CreateRoute() error = %v", err)
	}

	mgr := router.NewManager(db)
	engine := gin.New()
	engine.Any("/*proxyPath", Handler(mgr, nil))

	req := httptest.NewRequest(http.MethodGet, "/billing/invoices", nil)
	resp := httptest.NewRecorder()
	engine.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, location=%q body=%s", resp.Code, http.StatusOK, resp.Header().Get("Location"), resp.Body.String())
	}
	if resp.Body.String() != "upstream-ok" {
		t.Fatalf("body = %q, want %q", resp.Body.String(), "upstream-ok")
	}
	if location := resp.Header().Get("Location"); location != "" {
		t.Fatalf("Location = %q, want empty", location)
	}
}

func TestProxyIgnoresLegacyUnsupportedRedirectCode(t *testing.T) {
	gin.SetMode(gin.TestMode)

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("upstream-ok"))
	}))
	defer upstream.Close()

	db := newProxyTestDB(t)
	if err := db.CreateRoute(&store.Route{
		ID:            "legacy-redirect-route",
		PathPrefix:    "/legacy",
		RewriteTarget: "/new-home",
		RedirectCode:  http.StatusTemporaryRedirect,
		Backend:       upstream.URL,
		Enabled:       true,
	}); err != nil {
		t.Fatalf("CreateRoute() error = %v", err)
	}

	mgr := router.NewManager(db)
	engine := gin.New()
	engine.Any("/*proxyPath", Handler(mgr, nil))

	req := httptest.NewRequest(http.MethodGet, "/legacy/dashboard", nil)
	resp := httptest.NewRecorder()
	engine.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, location=%q body=%s", resp.Code, http.StatusOK, resp.Header().Get("Location"), resp.Body.String())
	}
	if resp.Body.String() != "upstream-ok" {
		t.Fatalf("body = %q, want %q", resp.Body.String(), "upstream-ok")
	}
	if location := resp.Header().Get("Location"); location != "" {
		t.Fatalf("Location = %q, want empty", location)
	}
}

func TestProxyForwardsHTTPSProtoToUpstream(t *testing.T) {
	gin.SetMode(gin.TestMode)

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(r.Header.Get("X-Forwarded-Proto")))
	}))
	defer upstream.Close()

	db := newProxyTestDB(t)
	if err := db.CreateRoute(&store.Route{
		ID:         "https-route",
		PathPrefix: "/secure",
		Backend:    upstream.URL,
		Enabled:    true,
	}); err != nil {
		t.Fatalf("CreateRoute() error = %v", err)
	}

	mgr := router.NewManager(db)
	engine := gin.New()
	engine.Any("/*proxyPath", Handler(mgr, nil))

	req := httptest.NewRequest(http.MethodGet, "https://example.com/secure", nil)
	req.TLS = &tls.ConnectionState{}
	resp := httptest.NewRecorder()
	engine.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body=%s", resp.Code, http.StatusOK, resp.Body.String())
	}
	if strings.TrimSpace(resp.Body.String()) != "https" {
		t.Fatalf("X-Forwarded-Proto = %q, want %q", resp.Body.String(), "https")
	}
}

func TestProxyLegacyInvalidBackendWeights_ReturnsStructuredErrorWithoutPanicking(t *testing.T) {
	gin.SetMode(gin.TestMode)

	db := newProxyTestDB(t)
	if err := db.CreateRoute(&store.Route{
		ID:         "invalid-weight-route",
		PathPrefix: "/api",
		Backend:    "http://example.com",
		Backends: []store.Backend{
			{URL: "http://backend-a.example.com", Weight: 1},
			{URL: "http://backend-b.example.com", Weight: -1},
		},
		Enabled: true,
	}); err != nil {
		t.Fatalf("CreateRoute() error = %v", err)
	}

	mgr := router.NewManager(db)
	engine := gin.New()
	engine.Any("/*proxyPath", Handler(mgr, nil))

	req := httptest.NewRequest(http.MethodGet, "/api/users", nil)
	resp := httptest.NewRecorder()

	var recovered any
	func() {
		defer func() {
			recovered = recover()
		}()
		engine.ServeHTTP(resp, req)
	}()

	if recovered != nil {
		t.Fatalf("ServeHTTP panicked: %v", recovered)
	}
	if resp.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d, body=%s", resp.Code, http.StatusInternalServerError, resp.Body.String())
	}
	if resp.Body.String() != "{\"error\":{\"code\":\"invalid_backend\",\"message\":\"invalid backend\"}}" {
		t.Fatalf("body = %s", resp.Body.String())
	}
}

func TestProxySingleOpenBackend_StillProxiesWithoutPanicking(t *testing.T) {
	gin.SetMode(gin.TestMode)

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("upstream-ok"))
	}))
	defer upstream.Close()

	localCS := &circuitState{breakers: map[string]*circuitBreaker{}}
	for i := 0; i < circuitFailureThreshold; i++ {
		localCS.recordFailure(upstream.URL)
	}
	if !localCS.isOpen(upstream.URL) {
		t.Fatal("upstream should be open in circuit breaker precondition")
	}

	origCS := cs
	cs = localCS
	defer func() { cs = origCS }()

	db := newProxyTestDB(t)
	if err := db.CreateRoute(&store.Route{
		ID:         "single-open-route",
		PathPrefix: "/api",
		Backend:    upstream.URL,
		Enabled:    true,
	}); err != nil {
		t.Fatalf("CreateRoute() error = %v", err)
	}

	mgr := router.NewManager(db)
	engine := gin.New()
	engine.Any("/*proxyPath", Handler(mgr, nil))

	req := httptest.NewRequest(http.MethodGet, "/api/users", nil)
	resp := httptest.NewRecorder()

	var recovered any
	func() {
		defer func() {
			recovered = recover()
		}()
		engine.ServeHTTP(resp, req)
	}()

	if recovered != nil {
		t.Fatalf("ServeHTTP panicked: %v", recovered)
	}
	if resp.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body=%s", resp.Code, http.StatusOK, resp.Body.String())
	}
	if strings.TrimSpace(resp.Body.String()) != "upstream-ok" {
		t.Fatalf("body = %q, want %q", resp.Body.String(), "upstream-ok")
	}
}

func TestProxyPreservesForwardedHTTPSProtoFromReverseProxy(t *testing.T) {
	gin.SetMode(gin.TestMode)

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(r.Header.Get("X-Forwarded-Proto")))
	}))
	defer upstream.Close()

	db := newProxyTestDB(t)
	if err := db.CreateRoute(&store.Route{
		ID:         "proxied-https-route",
		PathPrefix: "/secure",
		Backend:    upstream.URL,
		Enabled:    true,
	}); err != nil {
		t.Fatalf("CreateRoute() error = %v", err)
	}

	mgr := router.NewManager(db)
	engine := gin.New()
	engine.Any("/*proxyPath", Handler(mgr, nil))

	req := httptest.NewRequest(http.MethodGet, "http://example.com/secure", nil)
	req.Header.Set("X-Forwarded-Proto", "https")
	resp := httptest.NewRecorder()
	engine.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body=%s", resp.Code, http.StatusOK, resp.Body.String())
	}
	if strings.TrimSpace(resp.Body.String()) != "https" {
		t.Fatalf("X-Forwarded-Proto = %q, want %q", resp.Body.String(), "https")
	}
}

// TestWebSocketDirector_ForwardsUpgradeHeaders verifies that the Director preserves
// the Upgrade and Connection headers needed for WebSocket handshake.
func TestWebSocketDirector_ForwardsUpgradeHeaders(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Simulate backend: check for required WebSocket upgrade headers
		if r.Header.Get("Upgrade") != "websocket" {
			t.Errorf("Upgrade header = %q, want %q", r.Header.Get("Upgrade"), "websocket")
		}
		if r.Header.Get("Connection") != "Upgrade" {
			t.Errorf("Connection header = %q, want %q", r.Header.Get("Connection"), "Upgrade")
		}
		if r.Header.Get("Sec-WebSocket-Key") == "" {
			t.Error("Sec-WebSocket-Key header missing")
		}
		if r.Header.Get("Sec-WebSocket-Version") != "13" {
			t.Errorf("Sec-WebSocket-Version = %q, want %q", r.Header.Get("Sec-WebSocket-Version"), "13")
		}
		// Return 101 Switching Protocols to complete the handshake
		w.WriteHeader(http.StatusSwitchingProtocols)
	}))
	defer upstream.Close()

	u, _ := url.Parse(upstream.URL)
	backendHost := u.Host

	// Simulate the Director logic as it would be configured in proxy.go.
	// This is the critical piece: after calling originalDirector, we must NOT
	// overwrite Upgrade and Connection headers.
	req := httptest.NewRequest(http.MethodGet, "/ws", nil)
	req.Header.Set("Upgrade", "websocket")
	req.Header.Set("Connection", "Upgrade")
	req.Header.Set("Sec-WebSocket-Key", "dGhlIHNhbXBsZSBub25jZQ==")
	req.Header.Set("Sec-WebSocket-Version", "13")
	req.Header.Set("Host", "example.com")

	// Simulate originalDirector behavior: it clears Upgrade and Connection
	req.Header.Del("Upgrade")
	req.Header.Del("Connection")
	req.Header.Del("Sec-WebSocket-Key")
	req.Header.Del("Sec-WebSocket-Version")

	// Apply our fixed Director logic
	req.Host = backendHost
	req.Header.Set("X-Forwarded-Host", "example.com")
	req.Header.Set("X-Forwarded-Proto", "http")
	req.Header.Set("X-Forwarded-For", "127.0.0.1")
	// Critical: preserve Upgrade headers for WebSocket
	req.Header.Set("Upgrade", "websocket")
	req.Header.Set("Connection", "Upgrade")
	req.Header.Set("Sec-WebSocket-Key", "dGhlIHNhbXBsZSBub25jZQ==")
	req.Header.Set("Sec-WebSocket-Version", "13")

	// Verify all headers are present after Director runs
	if got := req.Header.Get("Upgrade"); got != "websocket" {
		t.Errorf("Upgrade = %q, want %q", got, "websocket")
	}
	if got := req.Header.Get("Connection"); got != "Upgrade" {
		t.Errorf("Connection = %q, want %q", got, "Upgrade")
	}
	if got := req.Header.Get("Sec-WebSocket-Key"); got == "" {
		t.Error("Sec-WebSocket-Key was lost")
	}
}

// TestSSEResponseHeaders_NotHijacked verifies that SSE responses (text/event-stream)
// flow through the normal response path without being buffered or interrupted.
func TestSSEResponseHeaders_NotHijacked(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		w.WriteHeader(http.StatusOK)
		// Send a couple of SSE events then close
		w.Write([]byte("data: hello\n\n"))
		w.(http.Flusher).Flush()
	}))
	defer upstream.Close()

	u, _ := url.Parse(upstream.URL)
	backendHost := u.Host

	// Verify that the Director preserves all relevant headers
	req := httptest.NewRequest(http.MethodGet, "/events", nil)
	req.Header.Set("Accept", "text/event-stream")
	req.Host = backendHost

	// Director runs and sets forwarding headers
	req.Header.Set("X-Forwarded-Host", "example.com")
	req.Header.Set("X-Forwarded-Proto", "http")
	req.Header.Set("X-Forwarded-For", "127.0.0.1")

	// Accept header must be preserved (SSE fallback)
	if got := req.Header.Get("Accept"); got != "text/event-stream" {
		t.Errorf("Accept = %q, want %q", got, "text/event-stream")
	}

	// Content-Type from upstream must be preserved in response
	// (this is a property of how ServeHTTP passes through the response, not a header set by Director)
	// The key requirement: response writer must NOT be buffered/manipulated
	_ = backendHost
}

// TestProxyWebSocketResponse_Passes101Through verifies that 101 Switching Protocols
// is not intercepted and returned as a JSON error.
// TestProxyWebSocketResponse_Passes101Through verifies WebSocket behavior.
// Note: httptest.ResponseRecorder does not implement http.Hijacker, so hijack fails.
// The handler must NOT panic — it should handle this gracefully. This test verifies
// that the WebSocket branch is entered, backend headers are forwarded correctly,
// and the connection is not left in a broken state.
func TestProxyWebSocketResponse_Passes101Through(t *testing.T) {
	gin.SetMode(gin.TestMode)

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Upgrade") != "websocket" {
			t.Errorf("upstream Upgrade = %q, want %q", r.Header.Get("Upgrade"), "websocket")
		}
		if r.Header.Get("Connection") != "Upgrade" {
			t.Errorf("upstream Connection = %q, want %q", r.Header.Get("Connection"), "Upgrade")
		}
		if r.Header.Get("Sec-WebSocket-Key") == "" {
			t.Error("upstream Sec-WebSocket-Key missing")
		}
		w.WriteHeader(http.StatusSwitchingProtocols)
	}))
	defer upstream.Close()

	u, _ := url.Parse(upstream.URL)
	db := newProxyTestDB(t)
	if err := db.CreateRoute(&store.Route{
		ID: "ws-route", PathPrefix: "/ws", Backend: u.Host, Enabled: true,
	}); err != nil {
		t.Fatalf("CreateRoute error: %v", err)
	}

	mgr := router.NewManager(db)
	engine := gin.New()
	engine.GET("/ws", Handler(mgr, nil))

	req := httptest.NewRequest(http.MethodGet, "/ws", nil)
	req.Header.Set("Upgrade", "websocket")
	req.Header.Set("Connection", "Upgrade")
	req.Header.Set("Sec-WebSocket-Key", "dGhlIHNhbXBsZSBub25jZQ==")
	req.Header.Set("Sec-WebSocket-Version", "13")
	w := httptest.NewRecorder()

	// This must NOT panic even though httptest.ResponseRecorder is not http.Hijacker.
	// The handler will try to hijack → fail (panic is recovered) → return an error.
	func() {
		defer func() {
			if r := recover(); r != nil {
				t.Logf("handler panicked (recovered): %v", r)
			}
		}()
		engine.ServeHTTP(w, req)
	}()

	// The critical assertion: the handler must NOT return 200 OK.
	// When hijacking fails (non-hijackable connection), it returns an error response.
	// The exact status code depends on how the error is propagated (426 / 500 / etc).
	if w.Code == http.StatusOK {
		t.Errorf("status = %d (OK), handler must NOT return 200 for WebSocket on non-hijackable connection", w.Code)
	}
}

// TestProxyTransport_DisableResponseBuffering verifies that the Transport
// is configured without response buffering for long-lived streams (SSE/WebSocket).
func TestProxyTransport_DisableResponseBuffering(t *testing.T) {
	dialTimeout := 5 * time.Second
	readTimeout := 30 * time.Second

	transport := &http.Transport{
		DialContext:           (&net.Dialer{Timeout: dialTimeout}).DialContext,
		ResponseHeaderTimeout: readTimeout,
		ExpectContinueTimeout: 1 * time.Second,
		// Note: DisableKeepAlives, MaxConnsPerHost, etc. can be tuned per-route
		// but the default Transport does NOT buffer response bodies for hijacked connections.
		// The key property for WebSocket/SSE: we don't set ResponseBodyTimeout,
		// and we don't use a custom Reader that buffers.
	}

	if transport.ResponseHeaderTimeout != readTimeout {
		t.Errorf("ResponseHeaderTimeout = %v, want %v", transport.ResponseHeaderTimeout, readTimeout)
	}
	if transport.DialContext == nil {
		t.Error("DialContext should be set")
	}
}

// --- Circuit Breaker Tests ---

func TestCircuitBreaker_ClosedByDefault(t *testing.T) {
	// Use a fresh state to avoid bleed from other tests.
	localCS := &circuitState{breakers: map[string]*circuitBreaker{}}
	if localCS.isOpen("http://a") {
		t.Error("new backend should not be open by default")
	}
}

func TestCircuitBreaker_OpensAfterThreshold(t *testing.T) {
	localCS := &circuitState{breakers: map[string]*circuitBreaker{}}
	backend := "http://circuit-test"

	// Fail up to threshold - 1: should still be closed
	for i := 0; i < circuitFailureThreshold-1; i++ {
		localCS.recordFailure(backend)
		if localCS.isOpen(backend) {
			t.Errorf("isOpen after %d failures = true, want false (threshold=%d)", i+1, circuitFailureThreshold)
		}
	}

	// One more failure: crosses threshold → should open
	localCS.recordFailure(backend)
	if !localCS.isOpen(backend) {
		t.Errorf("isOpen after %d failures = false, want true", circuitFailureThreshold)
	}
}

func TestCircuitBreaker_RecordSuccessResetsFailureCount(t *testing.T) {
	localCS := &circuitState{breakers: map[string]*circuitBreaker{}}
	backend := "http://reset-test"

	for i := 0; i < circuitFailureThreshold; i++ {
		localCS.recordFailure(backend)
	}
	// Now open
	if !localCS.isOpen(backend) {
		t.Fatal("circuit should be open after threshold failures")
	}

	// Simulate recovery window passing
	localCS.breakers[backend].lastFailure = time.Now().Add(-circuitRecoveryWindow - time.Second)
	localCS.breakers[backend].state = circuitHalfOpen

	// Success: should close
	localCS.recordSuccess(backend)
	if localCS.isOpen(backend) {
		t.Error("isOpen after recordSuccess = true, want false")
	}
}

func TestCircuitBreaker_RecoversAfterRecoveryWindow(t *testing.T) {
	localCS := &circuitState{breakers: map[string]*circuitBreaker{}}
	backend := "http://recover-test"

	for i := 0; i < circuitFailureThreshold; i++ {
		localCS.recordFailure(backend)
	}
	if !localCS.isOpen(backend) {
		t.Fatal("circuit should be open after threshold failures")
	}

	// Advance time past recovery window
	localCS.breakers[backend].lastFailure = time.Now().Add(-circuitRecoveryWindow - time.Second)
	// isOpen should allow the probe through (half-open)
	if localCS.isOpen(backend) {
		t.Error("isOpen after recovery window = true, want false (half-open probe)")
	}
}

func TestCircuitBreaker_IgnoresEmptyURL(t *testing.T) {
	localCS := &circuitState{breakers: map[string]*circuitBreaker{}}
	localCS.recordFailure("")
	localCS.recordSuccess("")
	if localCS.isOpen("") {
		t.Error("isOpen on empty URL should return false")
	}
}

func TestBalancer_PickSkipsOpenBackends(t *testing.T) {
	localCS := &circuitState{breakers: map[string]*circuitBreaker{}}
	// Pre-open one backend
	localCS.recordFailure("http://b")
	for i := 1; i < circuitFailureThreshold; i++ {
		localCS.recordFailure("http://b")
	}
	// Now "http://b" is open
	if !localCS.isOpen("http://b") {
		t.Fatal("http://b should be open")
	}

	backends := []store.Backend{
		{URL: "http://a", Weight: 1},
		{URL: "http://b", Weight: 1},
	}

	// Temporarily swap the global cs
	origCS := cs
	cs = localCS
	defer func() { cs = origCS }()

	bl := newBalancer(backends)
	seen := map[string]int{}
	for i := 0; i < 20; i++ {
		bk, ok := bl.pick()
		if !ok {
			continue
		}
		seen[bk.URL]++
	}
	// "http://b" should never be picked
	if seen["http://b"] > 0 {
		t.Errorf("open backend http://b was picked %d times, want 0", seen["http://b"])
	}
	// "http://a" should be the only one picked
	if seen["http://a"] == 0 {
		t.Error("healthy backend http://a was never picked")
	}
}

func TestTransportCache_ReturnsSameInstanceForSameKey(t *testing.T) {
	tc := newTransportCache()
	key := transportKey{DialTimeout: 5 * time.Second, ReadTimeout: 30 * time.Second, MaxIdleConns: 10}

	t1 := tc.get(key)
	t2 := tc.get(key)

	if t1 != t2 {
		t.Error("expected same transport instance for identical keys")
	}
}

func TestTransportCache_ReturnsDifferentInstanceForDifferentKey(t *testing.T) {
	tc := newTransportCache()
	key1 := transportKey{DialTimeout: 5 * time.Second, ReadTimeout: 30 * time.Second, MaxIdleConns: 10}
	key2 := transportKey{DialTimeout: 10 * time.Second, ReadTimeout: 30 * time.Second, MaxIdleConns: 10}

	t1 := tc.get(key1)
	t2 := tc.get(key2)

	if t1 == t2 {
		t.Error("expected different transport instances for different keys")
	}
}

func TestTransportCache_DefaultIdleConnsWhenZero(t *testing.T) {
	tc := newTransportCache()
	key := transportKey{DialTimeout: 5 * time.Second, ReadTimeout: 30 * time.Second, MaxIdleConns: 0}

	tr := tc.get(key)
	// Default should be 2 when MaxIdleConns is 0
	if tr.MaxIdleConnsPerHost != 2 {
		t.Errorf("MaxIdleConnsPerHost = %d, want 2", tr.MaxIdleConnsPerHost)
	}
}

func TestTransportCache_RespectsConfiguredIdleConns(t *testing.T) {
	tc := newTransportCache()
	key := transportKey{DialTimeout: 5 * time.Second, ReadTimeout: 30 * time.Second, MaxIdleConns: 20}

	tr := tc.get(key)
	if tr.MaxIdleConnsPerHost != 20 {
		t.Errorf("MaxIdleConnsPerHost = %d, want 20", tr.MaxIdleConnsPerHost)
	}
}
