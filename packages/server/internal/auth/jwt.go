package auth

import (
	"crypto/rand"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"

	httpresponse "github.com/pallyoung/auth-gate/packages/server/internal/http/response"
)

type Claims struct {
	UserID   string `json:"user_id"`
	Username string `json:"username"`
	Role     string `json:"role"`
	jwt.RegisteredClaims
}

var (
	jwtSecretMu             sync.RWMutex
	jwtSecret               []byte
	usingGeneratedJWTSecret bool
)

func init() {
	secret := os.Getenv("JWT_SECRET")
	if strings.TrimSpace(secret) != "" {
		setJWTSecret([]byte(secret), false)
		return
	}

	generatedSecret := make([]byte, 32)
	if _, err := rand.Read(generatedSecret); err != nil {
		panic("failed to generate JWT secret")
	}
	setJWTSecret(generatedSecret, true)
}

func setJWTSecret(secret []byte, generated bool) {
	jwtSecretMu.Lock()
	defer jwtSecretMu.Unlock()

	jwtSecret = append([]byte(nil), secret...)
	usingGeneratedJWTSecret = generated
}

func currentJWTSecret() []byte {
	jwtSecretMu.RLock()
	defer jwtSecretMu.RUnlock()

	return append([]byte(nil), jwtSecret...)
}

func ConfigureJWTSecret(secret string) {
	if strings.TrimSpace(secret) == "" {
		return
	}
	setJWTSecret([]byte(secret), false)
}

func UsingGeneratedJWTSecret() bool {
	jwtSecretMu.RLock()
	defer jwtSecretMu.RUnlock()

	return usingGeneratedJWTSecret
}

func GenerateToken(userID, username, role string) (string, error) {
	claims := &Claims{
		UserID:   userID,
		Username: username,
		Role:     role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(currentJWTSecret())
}

func ValidateToken(tokenString string) (*Claims, error) {
	secret := currentJWTSecret()
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
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

func GetTokenFromRequest(c *gin.Context) string {
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		return ""
	}

	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
		return ""
	}

	return parts[1]
}

func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenString := GetTokenFromRequest(c)
		if tokenString == "" {
			c.JSON(http.StatusUnauthorized, httpresponse.ErrorEnvelope{
				Error: httpresponse.ErrorDetail{
					Code:    "unauthorized",
					Message: "unauthorized",
				},
			})
			c.Abort()
			return
		}

		claims, err := ValidateToken(tokenString)
		if err != nil {
			c.JSON(http.StatusUnauthorized, httpresponse.ErrorEnvelope{
				Error: httpresponse.ErrorDetail{
					Code:    "invalid_token",
					Message: "invalid token",
				},
			})
			c.Abort()
			return
		}

		c.Set("user_id", claims.UserID)
		c.Set("username", claims.Username)
		c.Set("role", claims.Role)
		c.Next()
	}
}

func RequireRole(roles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		role, exists := c.Get("role")
		if !exists {
			c.JSON(http.StatusUnauthorized, httpresponse.ErrorEnvelope{
				Error: httpresponse.ErrorDetail{
					Code:    "unauthorized",
					Message: "unauthorized",
				},
			})
			c.Abort()
			return
		}

		userRole := role.(string)
		for _, r := range roles {
			if userRole == r {
				c.Next()
				return
			}
		}

		c.JSON(http.StatusForbidden, httpresponse.ErrorEnvelope{
			Error: httpresponse.ErrorDetail{
				Code:    "insufficient_permissions",
				Message: "insufficient permissions",
			},
		})
		c.Abort()
	}
}

func GetCurrentUser(c *gin.Context) *Claims {
	userID, _ := c.Get("user_id")
	username, _ := c.Get("username")
	role, _ := c.Get("role")

	return &Claims{
		UserID:   userID.(string),
		Username: username.(string),
		Role:     role.(string),
	}
}
