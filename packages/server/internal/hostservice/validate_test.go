package hostservice

import (
	"strings"
	"testing"
)

func TestValidateIP(t *testing.T) {
	cases := []struct {
		in      string
		wantErr bool
	}{
		{"127.0.0.1", false},
		{"::1", false},
		{"2001:db8::1", false},
		{"10.0.0.1", false},
		{"", true},
		{"   ", true},
		{"not-an-ip", true},
		{"999.0.0.1", true},
		{"127.0.0", true},
	}
	for _, c := range cases {
		if err := validateIP(c.in); (err != nil) != c.wantErr {
			t.Errorf("validateIP(%q) error = %v, wantErr = %v", c.in, err, c.wantErr)
		}
	}
}

func TestValidateHostname(t *testing.T) {
	cases := []struct {
		in      string
		wantErr bool
	}{
		{"api.local", false},
		{"a", false},
		{"my-host.example.com", false},
		{"", true},
		{"   ", true},
		{".local", true},
		{"-bad.example.com", true},
		{"bad-.example.com", true},
		{"under_score.example.com", true},
		{"api..local", true},
		{"api local", true},
	}
	for _, c := range cases {
		if err := validateHostname(c.in); (err != nil) != c.wantErr {
			t.Errorf("validateHostname(%q) error = %v, wantErr = %v", c.in, err, c.wantErr)
		}
	}
}

func TestValidateProfileName(t *testing.T) {
	cases := []struct {
		in      string
		wantErr bool
	}{
		{"dev", false},
		{"staging-cluster", false},
		{"prod_01", false},
		{"My Host", false},
		{"", true},
		{"   ", true},
		{strings.Repeat("a", 33), true},
		{"bad/name", true},
		{"bad:name", true},
	}
	for _, c := range cases {
		if err := validateProfileName(c.in); (err != nil) != c.wantErr {
			t.Errorf("validateProfileName(%q) error = %v, wantErr = %v", c.in, err, c.wantErr)
		}
	}
}

func TestValidateComment(t *testing.T) {
	if err := validateComment(""); err != nil {
		t.Errorf("validateComment(\"\") error = %v, want nil", err)
	}
	if err := validateComment(strings.Repeat("x", 200)); err != nil {
		t.Errorf("validateComment(200) error = %v, want nil", err)
	}
	if err := validateComment(strings.Repeat("x", 201)); err == nil {
		t.Error("validateComment(201) error = nil, want error")
	}
}
