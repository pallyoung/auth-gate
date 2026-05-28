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
	binaryPath := filepath.Join(t.TempDir(), "auth-gate")

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
	bootstrapPassword := "bootstrap-admin-password"
	configPath := filepath.Join(dataDir, "config.yaml")
	configBody := fmt.Sprintf("server:\n  addr: \":%d\"\ndatabase:\n  path: %q\nauth:\n  bootstrap_admin_password: %q\n", port, filepath.Join(dataDir, "auth-gate.db"), bootstrapPassword)
	if err := os.WriteFile(configPath, []byte(configBody), 0644); err != nil {
		t.Fatalf("WriteFile(config.yaml) error = %v", err)
	}

	serverCmd := exec.Command(binaryPath)
	serverCmd.Dir = dataDir
	serverCmd.Env = append(os.Environ(),
		"WEB_ROOT="+webRoot,
		"GIN_MODE=release",
	)
	var serverLogs bytes.Buffer
	serverCmd.Stdout = &serverLogs
	serverCmd.Stderr = &serverLogs

	if err := serverCmd.Start(); err != nil {
		t.Fatalf("server start error = %v", err)
	}
	defer stopProcess(t, serverCmd)

	waitForHTTPReady(t, baseURL+controlPlaneBasePath, &serverLogs)

	rootBody := getBody(t, http.MethodGet, baseURL+"/", "", nil, http.StatusNotFound)
	if !strings.Contains(rootBody, `"code":"route_not_found"`) {
		t.Fatalf("GET / returned unexpected body: %q", rootBody)
	}

	indexBody := getBody(t, http.MethodGet, baseURL+controlPlaneBasePath, "", nil, http.StatusOK)
	if !strings.Contains(indexBody, "<div id=\"root\"></div>") {
		t.Fatalf("GET %s returned unexpected body: %q", controlPlaneBasePath, indexBody)
	}

	loginRaw := getBody(t, http.MethodPost, baseURL+controlPlaneAPIBasePath+"/auth/login", fmt.Sprintf(`{"username":"admin","password":%q}`, bootstrapPassword), map[string]string{
		"Content-Type": "application/json",
	}, http.StatusOK)

	var login loginResponse
	if err := json.Unmarshal([]byte(loginRaw), &login); err != nil {
		t.Fatalf("login response json error = %v", err)
	}
	if login.Token == "" {
		t.Fatalf("login token is empty")
	}

	routeRaw := getBody(t, http.MethodPost, baseURL+controlPlaneAPIBasePath+"/routes", fmt.Sprintf(`{"name":"backend","path_prefix":"/backend","backend":%q,"strip_prefix":true,"enabled":true}`, backend.URL), map[string]string{
		"Content-Type":  "application/json",
		"Authorization": "Bearer " + login.Token,
	}, http.StatusCreated)

	var route routeResponse
	if err := json.Unmarshal([]byte(routeRaw), &route); err != nil {
		t.Fatalf("route response json error = %v", err)
	}
	if route.ID == "" {
		t.Fatalf("route ID is empty")
	}

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

	authRuleBody := fmt.Sprintf(`{"route_id":%q,"type":"apikey","config":{"secret":"secret-123","header_name":"X-API-Key"}}`, route.ID)
	_ = getBody(t, http.MethodPost, baseURL+controlPlaneAPIBasePath+"/auth-rules", authRuleBody, map[string]string{
		"Content-Type":  "application/json",
		"Authorization": "Bearer " + login.Token,
	}, http.StatusCreated)

	getBody(t, http.MethodGet, baseURL+"/backend/protected", "", nil, http.StatusUnauthorized)
	protectedBody := getBody(t, http.MethodGet, baseURL+"/backend/protected", "", map[string]string{
		"X-API-Key": "secret-123",
	}, http.StatusOK)
	if !strings.Contains(protectedBody, `"path":"/protected"`) {
		t.Fatalf("protected proxy body = %q, want backend response", protectedBody)
	}
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
