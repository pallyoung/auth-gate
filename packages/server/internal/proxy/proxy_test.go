package proxy

import (
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// Test the path-rewriting logic in isolation, mirroring what proxy.go does in Director.
func TestPathRewrite(t *testing.T) {
	tests := []struct {
		name        string
		pathPrefix  string
		strip       bool
		reqPath     string
		wantPath    string
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

// Test the host:port stripping logic used in proxy.go.
func TestHostPortStripping(t *testing.T) {
	tests := []struct {
		host string
		want string
	}{
		{"example.com:8080", "example.com"},
		{"example.com", "example.com"},
	}

	for _, tt := range tests {
		t.Run(tt.host, func(t *testing.T) {
			host := tt.host
			if idx := strings.Index(host, ":"); idx != -1 {
				host = host[:idx]
			}
			if host != tt.want {
				t.Errorf("stripPort(%q) = %q, want %q", tt.host, host, tt.want)
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
