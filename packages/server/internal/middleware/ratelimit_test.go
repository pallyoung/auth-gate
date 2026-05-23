package middleware

import (
	"testing"
)

func TestCheck_ZeroLimit_AllowsAll(t *testing.T) {
	reset()
	allowed, retry := Check("route-a", "10.0.0.1", 0, 0, nil)
	if !allowed {
		t.Errorf("allowed = false, want true when rateLimit=0")
	}
	if retry != 0 {
		t.Errorf("retry = %v, want 0", retry)
	}
}

func TestCheck_WhitelistedBypassesLimiter(t *testing.T) {
	reset()
	whitelist := []string{"10.0.0.1", "192.168.0.0/16"}
	// Exhaust the bucket first: rateLimit=1, burst=1
	Check("route-wl", "10.0.0.2", 1, 1, nil) // consumes the single token
	allowed, _ := Check("route-wl", "10.0.0.1", 1, 1, whitelist)
	if !allowed {
		t.Errorf("whitelisted IP 10.0.0.1 should be allowed")
	}
}

func TestCheck_WhitelistedCIDR(t *testing.T) {
	reset()
	whitelist := []string{"192.168.0.0/16"}
	allowed, _ := Check("route-cidr", "192.168.1.50", 1, 1, whitelist)
	if !allowed {
		t.Errorf("CIDR-whitelisted IP 192.168.1.50 should be allowed")
	}
}

func TestCheck_ExceedsBurst(t *testing.T) {
	reset()
	routeID := "route-burst"
	// burst=2: first two requests allowed, third rejected
	for i := 0; i < 2; i++ {
		allowed, _ := Check(routeID, "10.0.0.5", 1000, 2, nil)
		if !allowed {
			t.Fatalf("request %d: allowed = false, want true", i+1)
		}
	}
	allowed, retry := Check(routeID, "10.0.0.5", 1000, 2, nil)
	if allowed {
		t.Errorf("3rd request: allowed = true, want false")
	}
	if retry <= 0 {
		t.Errorf("retry = %v, want > 0", retry)
	}
}

func TestCheck_DifferentRoutesAreIndependent(t *testing.T) {
	reset()
	// Exhaust route-x
	Check("route-x", "10.0.0.1", 1000, 1, nil)
	// route-y should still allow
	allowed, _ := Check("route-y", "10.0.0.1", 1000, 1, nil)
	if !allowed {
		t.Errorf("different route should have independent limiter")
	}
}

func TestIsWhitelisted(t *testing.T) {
	tests := []struct {
		name      string
		ip        string
		whitelist []string
		want      bool
	}{
		{"empty whitelist", "10.0.0.1", nil, false},
		{"exact match", "10.0.0.1", []string{"10.0.0.1"}, true},
		{"no match", "10.0.0.2", []string{"10.0.0.1"}, false},
		{"cidr match", "192.168.5.10", []string{"192.168.0.0/16"}, true},
		{"cidr no match", "10.0.0.1", []string{"192.168.0.0/16"}, false},
		{"empty IP", "", []string{"10.0.0.1"}, false},
		{"whitespace entry", "10.0.0.1", []string{"  ", "10.0.0.1"}, true},
		{"invalid IP", "not-an-ip", []string{"10.0.0.1"}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isWhitelisted(tt.ip, tt.whitelist)
			if got != tt.want {
				t.Errorf("isWhitelisted(%q, %v) = %v, want %v", tt.ip, tt.whitelist, got, tt.want)
			}
		})
	}
}
