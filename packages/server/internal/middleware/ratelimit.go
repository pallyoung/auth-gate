package middleware

import (
	"fmt"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/pallyoung/auth-gate/packages/server/internal/metrics"
	"golang.org/x/time/rate"
)

type routeLimiter struct {
	lim  *rate.Limiter
	mu   sync.Mutex
	seen map[string]struct{} // tracks unique clients per route (approximate)
}

type Limiter struct {
	limiters map[string]*routeLimiter
	mu       sync.RWMutex
}

var globalLimiter = &Limiter{limiters: make(map[string]*routeLimiter)}

// reset clears all route limiters. Only for use in tests.
func reset() {
	globalLimiter = &Limiter{limiters: make(map[string]*routeLimiter)}
}

// Allow checks whether a request is allowed under token bucket rules.
// Returns (allowed bool, retryAfter time.Duration).
// When allowed is false, retryAfter is the minimum time the client should wait.
func (l *Limiter) Allow(routeID, clientIP string, rateLimit, burst int) (bool, time.Duration) {
	if rateLimit <= 0 || burst <= 0 {
		return true, 0
	}

	lim := l.getLimiter(routeID, rateLimit, burst)

	// Use deadline to get retry info
	now := time.Now()
	if lim.allowWithDeadline(now) {
		return true, 0
	}
	return false, 100 * time.Millisecond
}

func (l *Limiter) getLimiter(routeID string, rateLimit, burst int) *routeLimiter {
	l.mu.RLock()
	rl, ok := l.limiters[routeID]
	l.mu.RUnlock()
	if ok {
		return rl
	}

	l.mu.Lock()
	defer l.mu.Unlock()
	if rl, ok = l.limiters[routeID]; ok {
		return rl
	}

	rl = &routeLimiter{
		lim:  rate.NewLimiter(rate.Limit(rateLimit), burst),
		seen: make(map[string]struct{}),
	}
	l.limiters[routeID] = rl
	return rl
}

func (rl *routeLimiter) allowWithDeadline(now time.Time) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	return rl.lim.AllowN(now, 1)
}

// Check enforces token bucket rate limiting for the given route.
// Returns (allowed bool, retryAfter time.Duration). When allowed is false,
// retryAfter is the minimum time the client should wait before retrying.
// If rateLimit or burst is 0, rate limiting is disabled and all requests are allowed.
func Check(routeID, clientIP string, rateLimit, burst int, whitelist []string) (bool, time.Duration) {
	if rateLimit <= 0 || burst <= 0 {
		return true, 0
	}
	if isWhitelisted(clientIP, whitelist) {
		return true, 0
	}

	allowed, retryAfter := globalLimiter.Allow(routeID, clientIP, rateLimit, burst)
	return allowed, retryAfter
}

// IsWhitelisted returns true if the given clientIP matches any entry in the whitelist.
// Supports plain IP addresses and CIDR notation (e.g. "192.168.0.0/16").
func isWhitelisted(clientIP string, whitelist []string) bool {
	if len(whitelist) == 0 || clientIP == "" {
		return false
	}
	// Treat X-Forwarded-For as single IP for now
	ip := net.ParseIP(clientIP)
	if ip == nil {
		return false
	}
	for _, entry := range whitelist {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}
		if strings.Contains(entry, "/") {
			_, cidr, err := net.ParseCIDR(entry)
			if err == nil && cidr.Contains(ip) {
				return true
			}
		} else {
			whitelisted := net.ParseIP(entry)
			if whitelisted != nil && whitelisted.Equal(ip) {
				return true
			}
		}
	}
	return false
}

// CheckMiddleware builds a Gin middleware that enforces rate limiting.
// rateLimit = max requests per second, burst = token bucket size.
// Set both to 0 to disable rate limiting for the route.
func CheckMiddleware(routeID string, rateLimit, burst int, whitelist []string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Whitelist check — skip rate limiting for whitelisted IPs
		clientIP := c.ClientIP()
		if isWhitelisted(clientIP, whitelist) {
			c.Next()
			return
		}

		if rateLimit <= 0 || burst <= 0 {
			c.Next()
			return
		}

		allowed, retryAfter := globalLimiter.Allow(routeID, clientIP, rateLimit, burst)
		if !allowed {
			metrics.RecordRateLimitExceeded(routeID)
			c.Header("Retry-After", fmt.Sprintf("%.0f", retryAfter.Seconds()))
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error": gin.H{
					"code":    "rate_limit_exceeded",
					"message": "too many requests",
				},
			})
			return
		}

		c.Next()
	}
}