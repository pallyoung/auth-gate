package middleware

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

// CORS returns a middleware that adds CORS headers to responses.
// It supports both simple CORS and preflight (OPTIONS) requests.
//
// AllowedOrigins: comma-separated list of allowed origins (e.g. "https://app.example.com,https://admin.example.com").
//                 Set to "*" to allow all origins (not recommended for production with credentials).
// AllowedMethods: comma-separated list of allowed HTTP methods.
// AllowedHeaders: comma-separated list of allowed request headers.
// MaxAge: seconds to cache preflight results (default 86400 = 24h).
// AllowCredentials: whether to set Access-Control-Allow-Credentials.
func CORS(
	allowedOrigins string,
	allowedMethods string,
	allowedHeaders string,
	maxAge int,
	allowCredentials bool,
) gin.HandlerFunc {
	if maxAge <= 0 {
		maxAge = 86400
	}
	if allowedMethods == "" {
		allowedMethods = "GET,POST,PUT,DELETE,PATCH,OPTIONS"
	}
	if allowedHeaders == "" {
		allowedHeaders = "Authorization,Content-Type,X-Requested-With"
	}

	methods := strings.Split(allowedMethods, ",")
	methodSet := make(map[string]bool, len(methods))
	for _, m := range methods {
		methodSet[strings.TrimSpace(m)] = true
	}
	allowAll := strings.TrimSpace(allowedOrigins) == "*"

	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")
		if origin == "" {
			c.Next()
			return
		}

		allowed := allowAll || originMatches(origin, allowedOrigins)
		if !allowed {
			c.Next()
			return
		}

		c.Header("Access-Control-Allow-Origin", origin)
		if allowCredentials {
			c.Header("Access-Control-Allow-Credentials", "true")
		}
		c.Header("Access-Control-Allow-Methods", allowedMethods)
		c.Header("Access-Control-Allow-Headers", allowedHeaders)
		c.Header("Access-Control-Max-Age", formatInt(maxAge))

		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

// originMatches checks if the given origin matches any pattern in the allowed list.
// Supports wildcard subdomains: e.g. ".example.com" matches "app.example.com" and "admin.example.com".
func originMatches(origin, allowed string) bool {
	allowedList := strings.Split(allowed, ",")
	for _, pattern := range allowedList {
		pattern = strings.TrimSpace(pattern)
		if pattern == "" {
			continue
		}
		if pattern[0] == '.' {
			domain := strings.ToLower(pattern[1:])
			lowerOrigin := strings.ToLower(origin)
			if strings.HasSuffix(lowerOrigin, domain) {
				return true
			}
		} else if strings.EqualFold(pattern, origin) {
			return true
		}
	}
	return false
}

func formatInt(n int) string {
	return strconv.Itoa(n)
}