package config

import (
	"os"
	"testing"
)

func TestAllowEphemeralJWT_DefaultsToAllowedInDevLikeModes(t *testing.T) {
	previous := os.Getenv("APP_ENV")
	t.Cleanup(func() {
		_ = os.Setenv("APP_ENV", previous)
	})

	_ = os.Unsetenv("APP_ENV")
	if !(AuthConfig{}).AllowEphemeralJWT() {
		t.Fatal("AllowEphemeralJWT() = false, want true when APP_ENV is unset")
	}

	_ = os.Setenv("APP_ENV", "development")
	if !(AuthConfig{}).AllowEphemeralJWT() {
		t.Fatal("AllowEphemeralJWT() = false, want true in development")
	}
}

func TestAllowEphemeralJWT_DisallowsProductionByDefault(t *testing.T) {
	previous := os.Getenv("APP_ENV")
	t.Cleanup(func() {
		_ = os.Setenv("APP_ENV", previous)
	})

	_ = os.Setenv("APP_ENV", "production")
	if (AuthConfig{}).AllowEphemeralJWT() {
		t.Fatal("AllowEphemeralJWT() = true, want false in production without override")
	}

	if !(AuthConfig{AllowEphemeralSecret: true}).AllowEphemeralJWT() {
		t.Fatal("AllowEphemeralJWT() = false, want true with explicit override")
	}
}
