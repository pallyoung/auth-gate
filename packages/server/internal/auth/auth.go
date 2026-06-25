package auth

import (
	"errors"
	"net/http"

	httpresponse "github.com/pallyoung/auth-gate/packages/server/internal/http/response"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

func RequireAuth(authType string) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("WWW-Authenticate", authType)
		c.AbortWithStatusJSON(http.StatusUnauthorized, httpresponse.ErrorEnvelope{
			Error: httpresponse.ErrorDetail{
				Code:    "unauthorized",
				Message: "unauthorized",
			},
		})
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
