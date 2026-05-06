package auth

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func TestGenerateToken(t *testing.T) {
	token, err := GenerateToken("user-123", "testuser", "admin")
	if err != nil {
		t.Fatalf("GenerateToken failed: %v", err)
	}
	if token == "" {
		t.Fatal("GenerateToken returned empty token")
	}
}

func TestValidateToken(t *testing.T) {
	token, err := GenerateToken("user-123", "testuser", "editor")
	if err != nil {
		t.Fatalf("GenerateToken failed: %v", err)
	}

	claims, err := ValidateToken(token)
	if err != nil {
		t.Fatalf("ValidateToken failed: %v", err)
	}

	if claims.UserID != "user-123" {
		t.Errorf("UserID = %q, want %q", claims.UserID, "user-123")
	}
	if claims.Username != "testuser" {
		t.Errorf("Username = %q, want %q", claims.Username, "testuser")
	}
	if claims.Role != "editor" {
		t.Errorf("Role = %q, want %q", claims.Role, "editor")
	}
}

func TestValidateToken_Invalid(t *testing.T) {
	_, err := ValidateToken("not-a-valid-jwt")
	if err == nil {
		t.Fatal("ValidateToken should fail for invalid token")
	}
}

func TestValidateToken_WrongSecret(t *testing.T) {
	token, err := GenerateToken("user-123", "testuser", "admin")
	if err != nil {
		t.Fatalf("GenerateToken failed: %v", err)
	}

	// Validate with a different secret should fail
	_, err = ValidateTokenWithSecret(token, []byte("wrong-secret"))
	if err == nil {
		t.Fatal("ValidateTokenWithSecret should fail with wrong secret")
	}
}

func TestValidateTokenWithSecret(t *testing.T) {
	token, err := GenerateToken("user-456", "router", "viewer")
	if err != nil {
		t.Fatalf("GenerateToken failed: %v", err)
	}

	claims, err := ValidateTokenWithSecret(token, JWTSecret)
	if err != nil {
		t.Fatalf("ValidateTokenWithSecret failed: %v", err)
	}

	if claims.UserID != "user-456" {
		t.Errorf("UserID = %q, want %q", claims.UserID, "user-456")
	}
}

func TestValidateTokenWithSecret_WrongKey(t *testing.T) {
	token, err := GenerateToken("user-456", "router", "viewer")
	if err != nil {
		t.Fatalf("GenerateToken failed: %v", err)
	}

	_, err = ValidateTokenWithSecret(token, []byte("expected-secret"))
	if err == nil {
		t.Fatal("ValidateTokenWithSecret should reject token signed with different secret")
	}
}

func TestValidateTokenWithSecret_RejectsAlgNone(t *testing.T) {
	// Craft a token with alg=none to simulate algorithm confusion attack
	claims := &Claims{
		UserID:   "attacker",
		Username: "hacker",
		Role:     "admin",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodNone, claims)
	algNoneToken, _ := token.SignedString(jwt.UnsafeAllowNoneSignatureType)

	_, err := ValidateTokenWithSecret(algNoneToken, []byte("any-secret"))
	if err == nil {
		t.Fatal("ValidateTokenWithSecret should reject alg=none tokens")
	}
}

func TestGenerateToken_Expiry(t *testing.T) {
	// Tokens generated now should be valid for 24 hours
	token, _ := GenerateToken("user", "test", "admin")
	claims, _ := ValidateToken(token)

	if claims.ExpiresAt == nil {
		t.Fatal("ExpiresAt should be set")
	}
	expiry := claims.ExpiresAt.Time
	if expiry.Before(time.Now().Add(23 * time.Hour)) {
		t.Errorf("Expiry too soon: %v", expiry)
	}
	if expiry.After(time.Now().Add(25 * time.Hour)) {
		t.Errorf("Expiry too late: %v", expiry)
	}
}
