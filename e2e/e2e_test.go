package e2e

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

type loginResponse struct {
	Token string `json:"token"`
	User  struct {
		ID       string `json:"id"`
		Username string `json:"username"`
		Role     string `json:"role"`
	} `json:"user"`
}

type routeResponse struct {
	ID         string `json:"id"`
	PathPrefix string `json:"path_prefix"`
}

const (
	controlPlaneBasePath    = "/_authgate"
	controlPlaneAPIBasePath = controlPlaneBasePath + "/api"
)

func TestAuthGateRealE2E(t *testing.T) {
	t.Parallel()

	repoRoot := repoRootFromTestFile(t)
	serverDir := filepath.Join(repoRoot, "packages", "server")
	binaryName := "auth-gate"
	if runtime.GOOS == "windows" {
		binaryName = "auth-gate.exe"
	}
	binaryPath := filepath.Join(t.TempDir(), binaryName)

	buildServerBinary(t, serverDir, binaryPath)

	backendHits := make(chan *http.Request, 4)
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cloned := r.Clone(context.Background())
		backendHits <- cloned
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true,"path":"` + r.URL.Path + `"}`))
	}))
	defer backend.Close()

	webRoot := filepath.Join(repoRoot, "packages", "web", "dist")
	if _, err := os.Stat(filepath.Join(webRoot, "index.html")); err != nil {
		t.Fatalf("web build output missing at %s: %v", webRoot, err)
	}

	port := freePort(t)
	baseURL := fmt.Sprintf("http://127.0.0.1:%d", port)
	dataDir := t.TempDir()
	configPath := filepath.Join(dataDir, "config.yaml")
	configBody := fmt.Sprintf("server:\n  addr: \":%d\"\ndatabase:\n  path: %q\n", port, filepath.Join(dataDir, "store.json"))
	if err := os.WriteFile(configPath, []byte(configBody), 0644); err != nil {
		t.Fatalf("WriteFile(config.yaml) error = %v", err)
	}

	serverCmd := exec.Command(binaryPath, "start", "-f")
	serverCmd.Dir = dataDir
	serverCmd.Env = append(os.Environ(),
		"WEB_ROOT="+webRoot,
		"GIN_MODE=release",
		"AUTH_GATE_DATA_DIR="+dataDir,
	)
	var serverLogs bytes.Buffer
	serverCmd.Stdout = &serverLogs
	serverCmd.Stderr = &serverLogs

	if err := serverCmd.Start(); err != nil {
		t.Fatalf("server start error = %v", err)
	}
	defer stopProcess(t, serverCmd)

	waitForHTTPReady(t, baseURL+controlPlaneBasePath, &serverLogs)

	// --- Verify root returns 404 (no routes configured yet) ---
	rootBody := getBody(t, http.MethodGet, baseURL+"/", "", nil, http.StatusNotFound)
	if !strings.Contains(rootBody, `"code":"route_not_found"`) {
		t.Fatalf("GET / returned unexpected body: %q", rootBody)
	}

	// --- Verify SPA index page ---
	indexBody := getBody(t, http.MethodGet, baseURL+controlPlaneBasePath, "", nil, http.StatusOK)
	if !strings.Contains(indexBody, "<div id=\"root\"></div>") {
		t.Fatalf("GET %s returned unexpected body: %q", controlPlaneBasePath, indexBody)
	}

	// --- Verify setup-status reports setup is required ---
	setupStatusRaw := getBody(t, http.MethodGet, baseURL+controlPlaneAPIBasePath+"/auth/setup-status", "", nil, http.StatusOK)
	if !strings.Contains(setupStatusRaw, `"setup_required":true`) {
		t.Fatalf("setup-status should report setup_required=true, got: %q", setupStatusRaw)
	}

	// --- Create admin via setup endpoint ---
	setupPassword := "e2e-test-password-123"
	setupRaw := getBody(t, http.MethodPost, baseURL+controlPlaneAPIBasePath+"/auth/setup",
		fmt.Sprintf(`{"username":"admin","password":%q}`, setupPassword),
		map[string]string{"Content-Type": "application/json"},
		http.StatusOK)

	var login loginResponse
	if err := json.Unmarshal([]byte(setupRaw), &login); err != nil {
		t.Fatalf("setup response json error = %v", err)
	}
	if login.Token == "" {
		t.Fatalf("setup token is empty")
	}

	// --- Verify setup is now idempotent (second call returns 409) ---
	getBody(t, http.MethodPost, baseURL+controlPlaneAPIBasePath+"/auth/setup",
		`{"username":"admin2","password":"another-password-123"}`,
		map[string]string{"Content-Type": "application/json"},
		http.StatusConflict)

	// --- Verify setup-status reports setup is no longer required ---
	setupStatusAfterRaw := getBody(t, http.MethodGet, baseURL+controlPlaneAPIBasePath+"/auth/setup-status", "", nil, http.StatusOK)
	if !strings.Contains(setupStatusAfterRaw, `"setup_required":false`) {
		t.Fatalf("setup-status should report setup_required=false after setup, got: %q", setupStatusAfterRaw)
	}

	// --- Login with the created admin ---
	loginRaw := getBody(t, http.MethodPost, baseURL+controlPlaneAPIBasePath+"/auth/login",
		fmt.Sprintf(`{"username":"admin","password":%q}`, setupPassword),
		map[string]string{"Content-Type": "application/json"},
		http.StatusOK)

	var loginResp loginResponse
	if err := json.Unmarshal([]byte(loginRaw), &loginResp); err != nil {
		t.Fatalf("login response json error = %v", err)
	}
	if loginResp.Token == "" {
		t.Fatalf("login token is empty")
	}
	token := loginResp.Token

	// --- Create a proxy route ---
	routeRaw := getBody(t, http.MethodPost, baseURL+controlPlaneAPIBasePath+"/routes",
		fmt.Sprintf(`{"name":"backend","path_prefix":"/backend","backend":%q,"strip_prefix":true,"enabled":true}`, backend.URL),
		map[string]string{
			"Content-Type":  "application/json",
			"Authorization": "Bearer " + token,
		}, http.StatusCreated)

	var route routeResponse
	if err := json.Unmarshal([]byte(routeRaw), &route); err != nil {
		t.Fatalf("route response json error = %v", err)
	}
	if route.ID == "" {
		t.Fatalf("route ID is empty")
	}

	// --- Verify proxy forwarding + path rewriting ---
	proxyBody := getBody(t, http.MethodGet, baseURL+"/backend/hello", "", nil, http.StatusOK)
	if !strings.Contains(proxyBody, `"path":"/hello"`) {
		t.Fatalf("proxy body = %q, want rewritten backend path", proxyBody)
	}

	select {
	case req := <-backendHits:
		if req.URL.Path != "/hello" {
			t.Fatalf("backend path = %q, want %q", req.URL.Path, "/hello")
		}
		if req.Header.Get("X-Forwarded-Host") == "" {
			t.Fatalf("missing X-Forwarded-Host header")
		}
	case <-time.After(3 * time.Second):
		t.Fatal("backend did not receive proxied request")
	}

	// --- Create an auth rule (API key) ---
	authRuleBody := fmt.Sprintf(`{"route_id":%q,"type":"apikey","config":{"secret":"secret-123","header_name":"X-API-Key"}}`, route.ID)
	_ = getBody(t, http.MethodPost, baseURL+controlPlaneAPIBasePath+"/auth-rules", authRuleBody, map[string]string{
		"Content-Type":  "application/json",
		"Authorization": "Bearer " + token,
	}, http.StatusCreated)

	// --- Verify auth rule enforcement ---
	getBody(t, http.MethodGet, baseURL+"/backend/protected", "", nil, http.StatusUnauthorized)
	protectedBody := getBody(t, http.MethodGet, baseURL+"/backend/protected", "", map[string]string{
		"X-API-Key": "secret-123",
	}, http.StatusOK)
	if !strings.Contains(protectedBody, `"path":"/protected"`) {
		t.Fatalf("protected proxy body = %q, want backend response", protectedBody)
	}

	// ================================================================
	// Users management
	// ================================================================

	// List users — should have the admin we created via setup
	usersRaw := getBody(t, http.MethodGet, baseURL+controlPlaneAPIBasePath+"/users", "", map[string]string{
		"Authorization": "Bearer " + token,
	}, http.StatusOK)
	if !strings.Contains(usersRaw, `"admin"`) {
		t.Fatalf("users list should contain admin, got: %q", usersRaw)
	}

	// Create a new user (editor role)
	editorRaw := getBody(t, http.MethodPost, baseURL+controlPlaneAPIBasePath+"/users",
		`{"username":"editor1","password":"editor-password-123","role":"editor","enabled":true}`,
		map[string]string{
			"Content-Type":  "application/json",
			"Authorization": "Bearer " + token,
		}, http.StatusCreated)
	if !strings.Contains(editorRaw, `"editor1"`) {
		t.Fatalf("create user response should contain username, got: %q", editorRaw)
	}

	// Login as the editor
	editorLoginRaw := getBody(t, http.MethodPost, baseURL+controlPlaneAPIBasePath+"/auth/login",
		`{"username":"editor1","password":"editor-password-123"}`,
		map[string]string{"Content-Type": "application/json"},
		http.StatusOK)
	var editorLogin loginResponse
	if err := json.Unmarshal([]byte(editorLoginRaw), &editorLogin); err != nil {
		t.Fatalf("editor login json error = %v", err)
	}
	editorToken := editorLogin.Token

	// Editor should NOT be able to list users (admin-only)
getBody(t, http.MethodGet, baseURL+controlPlaneAPIBasePath+"/users", "", map[string]string{
		"Authorization": "Bearer " + editorToken,
	}, http.StatusForbidden)

	// Editor SHOULD be able to list routes
getBody(t, http.MethodGet, baseURL+controlPlaneAPIBasePath+"/routes", "", map[string]string{
		"Authorization": "Bearer " + editorToken,
	}, http.StatusOK)

	// ================================================================
	// Auth rules management
	// ================================================================

	// List auth rules
authRulesRaw := getBody(t, http.MethodGet, baseURL+controlPlaneAPIBasePath+"/auth-rules", "", map[string]string{
		"Authorization": "Bearer " + token,
	}, http.StatusOK)
	if !strings.Contains(authRulesRaw, route.ID) {
		t.Fatalf("auth rules should contain the route, got: %q", authRulesRaw)
	}

	// ================================================================
	// Settings / config reload
	// ================================================================

getBody(t, http.MethodPost, baseURL+controlPlaneAPIBasePath+"/config/reload", "", map[string]string{
		"Authorization": "Bearer " + token,
	}, http.StatusOK)

	// ================================================================
	// Access logs
	// ================================================================

getBody(t, http.MethodGet, baseURL+controlPlaneAPIBasePath+"/access-logs?page=1&per_page=10", "", map[string]string{
		"Authorization": "Bearer " + token,
	}, http.StatusOK)

getBody(t, http.MethodGet, baseURL+controlPlaneAPIBasePath+"/access-logs/stats?duration=1h", "", map[string]string{
		"Authorization": "Bearer " + token,
	}, http.StatusOK)

	// ================================================================
	// Error scenarios
	// ================================================================

	// No auth header → 401
getBody(t, http.MethodGet, baseURL+controlPlaneAPIBasePath+"/users", "", nil, http.StatusUnauthorized)

	// Invalid token → 401
getBody(t, http.MethodGet, baseURL+controlPlaneAPIBasePath+"/users", "", map[string]string{
		"Authorization": "Bearer invalid-token",
	}, http.StatusUnauthorized)

	// Duplicate setup → 409
getBody(t, http.MethodPost, baseURL+controlPlaneAPIBasePath+"/auth/setup",
		`{"username":"admin","password":"any-password-12345"}`,
		map[string]string{"Content-Type": "application/json"},
		http.StatusConflict)

	// Login with wrong password → 401
getBody(t, http.MethodPost, baseURL+controlPlaneAPIBasePath+"/auth/login",
		`{"username":"admin","password":"wrong-password"}`,
		map[string]string{"Content-Type": "application/json"},
		http.StatusUnauthorized)

	// Create user with duplicate username → 400
getBody(t, http.MethodPost, baseURL+controlPlaneAPIBasePath+"/users",
		`{"username":"admin","password":"dup-password-123","role":"viewer","enabled":true}`,
		map[string]string{
			"Content-Type":  "application/json",
			"Authorization": "Bearer " + token,
		}, http.StatusBadRequest)

	// Create route with missing fields → 400
getBody(t, http.MethodPost, baseURL+controlPlaneAPIBasePath+"/routes",
		`{"name":"bad-route"}`,
		map[string]string{
			"Content-Type":  "application/json",
			"Authorization": "Bearer " + token,
		}, http.StatusBadRequest)
}

func repoRootFromTestFile(t *testing.T) string {
	t.Helper()

	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd() error = %v", err)
	}
	return filepath.Clean(filepath.Join(wd, ".."))
}

func buildServerBinary(t *testing.T, serverDir, binaryPath string) {
	t.Helper()

	cmd := exec.Command("go", "build", "-o", binaryPath, "./cmd/server")
	cmd.Dir = serverDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("go build error = %v\n%s", err, output)
	}
}

func freePort(t *testing.T) int {
	t.Helper()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Listen() error = %v", err)
	}
	defer listener.Close()

	return listener.Addr().(*net.TCPAddr).Port
}

func waitForHTTPReady(t *testing.T, url string, logs *bytes.Buffer) {
	t.Helper()

	client := &http.Client{Timeout: 500 * time.Millisecond}
	deadline := time.Now().Add(10 * time.Second)

	for time.Now().Before(deadline) {
		resp, err := client.Get(url)
		if err == nil {
			_ = resp.Body.Close()
			return
		}
		time.Sleep(150 * time.Millisecond)
	}

	t.Fatalf("server did not become ready\nlogs:\n%s", logs.String())
}

func getBody(t *testing.T, method, url, body string, headers map[string]string, wantStatus int) string {
	t.Helper()

	var reader io.Reader
	if body != "" {
		reader = strings.NewReader(body)
	}

	req, err := http.NewRequest(method, url, reader)
	if err != nil {
		t.Fatalf("NewRequest() error = %v", err)
	}
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	resp, err := (&http.Client{Timeout: 5 * time.Second}).Do(req)
	if err != nil {
		t.Fatalf("%s %s error = %v", method, url, err)
	}
	defer resp.Body.Close()

	payload, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("ReadAll() error = %v", err)
	}

	if resp.StatusCode != wantStatus {
		t.Fatalf("%s %s status = %d, want %d, body=%s", method, url, resp.StatusCode, wantStatus, string(payload))
	}

	return string(payload)
}

func stopProcess(t *testing.T, cmd *exec.Cmd) {
	t.Helper()

	if cmd.Process == nil {
		return
	}

	_ = cmd.Process.Signal(os.Interrupt)

	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()

	select {
	case <-time.After(3 * time.Second):
		_ = cmd.Process.Kill()
		<-done
	case <-done:
	}
}
