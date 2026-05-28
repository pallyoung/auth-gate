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
	engine.Any("/*path", Handler(mgr))

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
	engine.Any("/*path", Handler(mgr))

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
