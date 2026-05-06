package auth

import (
	"errors"
	"net/http"
	"strings"

	"github.com/pallyoung/auth-gate/packages/server/internal/store"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

func Check(c *gin.Context, rule *store.AuthRule) bool {
	switch rule.Type {
	case "apikey":
		return checkAPIKey(c, rule)
	case "bearer":
		return checkBearer(c, rule)
	case "basic":
		return checkBasic(c, rule)
	default:
		return true
	}
}

func checkAPIKey(c *gin.Context, rule *store.AuthRule) bool {
	headerName := rule.Config.HeaderName
	if headerName == "" {
		headerName = "X-API-Key"
	}

	key := c.GetHeader(headerName)
	if key == "" {
		// 也支持 query 参数
		key = c.Query("api_key")
	}

	return key != "" && key == rule.Config.Secret
}

func checkBearer(c *gin.Context, rule *store.AuthRule) bool {
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		return false
	}

	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
		return false
	}

	token := parts[1]
	if token == "" {
		return false
	}

	// Use the rule's secret as the JWT signing key for validation.
	// If no secret is configured, reject all bearer tokens.
	if rule.Config.Secret == "" {
		return false
	}

	claims, err := ValidateTokenWithSecret(token, []byte(rule.Config.Secret))
	if err != nil {
		return false
	}

	// Store validated claims in context for downstream use.
	c.Set("jwt_subject", claims.UserID)
	c.Set("jwt_username", claims.Username)
	c.Set("jwt_role", claims.Role)

	return true
}

func checkBasic(c *gin.Context, rule *store.AuthRule) bool {
	username, password, ok := c.Request.BasicAuth()
	if !ok {
		return false
	}

	return username == rule.Config.Username && password == rule.Config.Password
}

func RequireAuth(authType string) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("WWW-Authenticate", authType)
		c.AbortWithStatus(http.StatusUnauthorized)
	}
}

// ValidateTokenWithSecret validates a JWT token using a provided secret key.
// This allows route-specific JWT validation with different secrets per auth rule.
func ValidateTokenWithSecret(tokenString string, secret []byte) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		// Ensure the signing method is HMAC (HS256, HS384, HS512)
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return secret, nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}

	return nil, jwt.ErrSignatureInvalid
}
