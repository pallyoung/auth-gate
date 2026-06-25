package proxy

import (
	"bufio"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/pallyoung/auth-gate/packages/server/internal/router"
	"github.com/pallyoung/auth-gate/packages/server/internal/store"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// TestRateLimit_429 verifies that requests exceeding the token bucket burst
// return HTTP 429 with a Retry-After header.
//
// Use a low steady-state rate so backend roundtrip latency does not refill
// the bucket fast enough to hide the burst limit during the test.
func TestRateLimit_429(t *testing.T) {
	gin.SetMode(gin.TestMode)

	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	}))
	defer backend.Close()

	db := newProxyTestDB(t)

	if err := db.CreateRoute(&store.Route{
		ID:         "rl-route",
		Name:       "rl-test",
		PathPrefix: "/api/limited",
		Backend:    backend.URL, // full URL with scheme so url.Parse works correctly
		Enabled:    true,
	}); err != nil {
		t.Fatalf("CreateRoute error: %v", err)
	}
	if err := db.CreateAuthRule(&store.AuthRule{
		RouteID:   "rl-route",
		Type:      "none",
		RateLimit: 1, // one request per second after the initial burst
		Burst:     2,
	}); err != nil {
		t.Fatalf("CreateAuthRule error: %v", err)
	}

	mgr := router.NewManager(db)
	engine := gin.New()
	engine.Any("/*path", Handler(mgr, nil))

	// Send 5 requests as fast as possible from the same client IP.
	// With a 1 req/sec refill rate, only the first two should fit in the
	// initial burst window before 429 responses start.
	allowed := 0
	rejected := 0
	for i := 0; i < 5; i++ {
		req := httptest.NewRequest(http.MethodGet, "/api/limited/resource", nil)
		req.RemoteAddr = "192.168.1.1:12345"
		w := httptest.NewRecorder()
		engine.ServeHTTP(w, req)

		switch w.Code {
		case http.StatusOK:
			allowed++
		case http.StatusTooManyRequests:
			rejected++
			if ra := w.Header().Get("Retry-After"); ra == "" {
				t.Errorf("request %d: 429 response missing Retry-After header", i+1)
			}
		default:
			t.Errorf("request %d: unexpected status %d", i+1, w.Code)
		}
	}

	if allowed == 0 {
		t.Errorf("all requests were rejected; expected at least %d to succeed", 2)
	}
	if rejected == 0 {
		t.Errorf("no 429 received; expected requests after burst=%d to be rejected", 2)
	}
	t.Logf("allowed=%d rejected=%d", allowed, rejected)
}

// accessLogJSON matches the JSON structure written to stdout by proxy.go.
type accessLogJSON struct {
	RequestID        string `json:"request_id"`
	RouteID          string `json:"route_id"`
	Method           string `json:"method"`
	Path             string `json:"path"`
	BackendURL       string `json:"backend_url"`
	BackendLatencyMs int64  `json:"backend_latency_ms"`
	StatusCode       int    `json:"status_code"`
	ClientIP         string `json:"client_ip"`
	UserAgent        string `json:"user_agent"`
}

// TestAccessLog_Output verifies that every proxied request produces a JSON
// access log line on stdout containing all required fields.
func TestAccessLog_Output(t *testing.T) {
	gin.SetMode(gin.TestMode)

	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("hello"))
	}))
	defer backend.Close()

	db := newProxyTestDB(t)

	if err := db.CreateRoute(&store.Route{
		ID:         "log-route",
		Name:       "log-test",
		PathPrefix: "/api/logged",
		Backend:    backend.URL, // full URL with scheme so url.Parse works correctly
		Enabled:    true,
	}); err != nil {
		t.Fatalf("CreateRoute error: %v", err)
	}
	// No auth rule → rate limiting is disabled, request goes straight to backend.

	mgr := router.NewManager(db)
	engine := gin.New()
	engine.Any("/*path", Handler(mgr, nil))

	// Capture stdout so we can inspect the access log output.
	origStdout := os.Stdout
	r, wPipe, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe error: %v", err)
	}
	os.Stdout = wPipe

	req := httptest.NewRequest(http.MethodGet, "/api/logged/users", nil)
	req.Header.Set("User-Agent", "e2e-test-agent")
	req.RemoteAddr = "10.20.30.40:5555"
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)

	// Restore stdout and close the write end so ReadAll returns.
	wPipe.Close()
	os.Stdout = origStdout

	if rec.Code != http.StatusOK {
		t.Fatalf("response status = %d, want %d", rec.Code, http.StatusOK)
	}

	captured, _ := io.ReadAll(r)
	r.Close()

	// Find the access log line (starts with "access ").
	var found bool
	var entry accessLogJSON
	scanner := bufio.NewScanner(strings.NewReader(string(captured)))
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.Contains(line, `"request_id":`) {
			continue
		}
		// The log line format is: <timestamp> access <json>
		idx := strings.Index(line, `{"request_id":`)
		if idx < 0 {
			continue
		}
		if err := json.Unmarshal([]byte(line[idx:]), &entry); err != nil {
			t.Fatalf("failed to parse access log JSON: %v\nline: %s", err, line)
		}
		found = true
		break
	}
	if !found {
		t.Fatalf("no access log JSON found in stdout output:\n%s", string(captured))
	}

	assertField := func(name, got, wantPrefix string) {
		t.Helper()
		if got == "" {
			t.Errorf("access log missing field %q", name)
		}
		if wantPrefix != "" && !strings.HasPrefix(got, wantPrefix) {
			t.Errorf("access log %q = %q, want prefix %q", name, got, wantPrefix)
		}
	}
	assertField("request_id", entry.RequestID, "")
	assertField("route_id", entry.RouteID, "log-route")
	assertField("method", entry.Method, "GET")
	assertField("path", entry.Path, "/api/logged/users")
	assertField("client_ip", entry.ClientIP, "10.20.30.40")
	assertField("user_agent", entry.UserAgent, "e2e-test-agent")

	if entry.StatusCode != http.StatusOK {
		t.Errorf("status_code = %d, want %d", entry.StatusCode, http.StatusOK)
	}
	if entry.BackendLatencyMs < 0 {
		t.Errorf("backend_latency_ms = %d, want >= 0", entry.BackendLatencyMs)
	}
	if entry.BackendURL == "" {
		t.Errorf("backend_url is empty")
	}
}

// TestMetrics_Endpoint verifies that the /metrics endpoint returns Prometheus
// metrics containing at least one auth_gate_ prefixed metric name.
func TestMetrics_Endpoint(t *testing.T) {
	gin.SetMode(gin.TestMode)

	engine := gin.New()
	engine.GET("/metrics", gin.WrapH(promhttp.Handler()))

	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("GET /metrics status = %d, want %d", rec.Code, http.StatusOK)
	}

	body := rec.Body.String()
	if !strings.Contains(body, "auth_gate_") {
		t.Fatalf("metrics response body does not contain any auth_gate_ prefixed metric:\n%s", body[:500])
	}

	if !strings.Contains(body, "auth_gate_requests_total") {
		t.Errorf("metrics body missing expected metric auth_gate_requests_total")
	}
}

// TestCustomRequestHeaders verifies that SetRequestHeaders and
// RemoveRequestHeaders are applied to the request forwarded to the backend.
func TestCustomRequestHeaders(t *testing.T) {
	gin.SetMode(gin.TestMode)

	var receivedHeaders http.Header
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedHeaders = r.Header.Clone()
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	}))
	defer backend.Close()

	db := newProxyTestDB(t)
	if err := db.CreateRoute(&store.Route{
		ID:         "hdr-req",
		Name:       "hdr-req",
		PathPrefix: "/api",
		Backend:    backend.URL,
		Enabled:    true,
		SetRequestHeaders:    map[string]string{"X-Custom-Token": "secret123", "X-Gateway": "auth-gate"},
		RemoveRequestHeaders: []string{"X-Remove-Me", "X-Also-Remove"},
	}); err != nil {
		t.Fatalf("CreateRoute error: %v", err)
	}

	mgr := router.NewManager(db)
	engine := gin.New()
	engine.Any("/*path", Handler(mgr, nil))

	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	req.Header.Set("X-Remove-Me", "should-be-gone")
	req.Header.Set("X-Also-Remove", "also-gone")
	req.Header.Set("X-Keep-Me", "stays")
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	// Custom headers should be set on the forwarded request
	if got := receivedHeaders.Get("X-Custom-Token"); got != "secret123" {
		t.Errorf("backend X-Custom-Token = %q, want %q", got, "secret123")
	}
	if got := receivedHeaders.Get("X-Gateway"); got != "auth-gate" {
		t.Errorf("backend X-Gateway = %q, want %q", got, "auth-gate")
	}

	// Removed headers should not reach the backend
	if got := receivedHeaders.Get("X-Remove-Me"); got != "" {
		t.Errorf("backend X-Remove-Me = %q, want empty", got)
	}
	if got := receivedHeaders.Get("X-Also-Remove"); got != "" {
		t.Errorf("backend X-Also-Remove = %q, want empty", got)
	}

	// Unrelated headers should still be forwarded
	if got := receivedHeaders.Get("X-Keep-Me"); got != "stays" {
		t.Errorf("backend X-Keep-Me = %q, want %q", got, "stays")
	}
}

// TestCustomResponseHeaders verifies that AddResponseHeaders and
// RemoveResponseHeaders are applied to the response returned to the client.
func TestCustomResponseHeaders(t *testing.T) {
	gin.SetMode(gin.TestMode)

	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Powered-By", "Express")
		w.Header().Set("Server", "nginx/1.21")
		w.Header().Set("X-Keep", "backend-value")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	}))
	defer backend.Close()

	db := newProxyTestDB(t)
	if err := db.CreateRoute(&store.Route{
		ID:         "hdr-resp",
		Name:       "hdr-resp",
		PathPrefix: "/api",
		Backend:    backend.URL,
		Enabled:    true,
		AddResponseHeaders:    map[string]string{"X-Request-Id": "req-42", "X-Response-From": "gateway"},
		RemoveResponseHeaders: []string{"X-Powered-By", "Server"},
	}); err != nil {
		t.Fatalf("CreateRoute error: %v", err)
	}

	mgr := router.NewManager(db)
	engine := gin.New()
	engine.Any("/*path", Handler(mgr, nil))

	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	// Added response headers should be present
	if got := rec.Header().Get("X-Request-Id"); got != "req-42" {
		t.Errorf("X-Request-Id = %q, want %q", got, "req-42")
	}
	if got := rec.Header().Get("X-Response-From"); got != "gateway" {
		t.Errorf("X-Response-From = %q, want %q", got, "gateway")
	}

	// Removed response headers should be absent
	if got := rec.Header().Get("X-Powered-By"); got != "" {
		t.Errorf("X-Powered-By = %q, want empty", got)
	}
	if got := rec.Header().Get("Server"); got != "" {
		t.Errorf("Server = %q, want empty", got)
	}

	// Unrelated response headers should still be present
	if got := rec.Header().Get("X-Keep"); got != "backend-value" {
		t.Errorf("X-Keep = %q, want %q", got, "backend-value")
	}
}
