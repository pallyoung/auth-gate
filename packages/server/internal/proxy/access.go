package proxy

import (
	"database/sql"
	"net/url"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/pallyoung/auth-gate/packages/server/internal/auth"
	"github.com/pallyoung/auth-gate/packages/server/internal/router"
	"github.com/pallyoung/auth-gate/packages/server/internal/store"
)

const (
	routeAccessCookieName = "auth_gate_route_token"
	routeLoginHashPath    = "/access-login"
)

func routeAccessClaims(c *gin.Context, db *store.SQLite) (*auth.Claims, bool) {
	token, err := c.Cookie(routeAccessCookieName)
	if err != nil || strings.TrimSpace(token) == "" {
		return nil, false
	}

	claims, err := auth.ValidateToken(token)
	if err != nil || claims.Scope != auth.ScopeRouteAccess {
		return nil, false
	}

	if db == nil {
		return claims, true
	}

	user, err := db.GetUserByID(claims.UserID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, false
		}
		return nil, false
	}
	if !user.Enabled {
		return nil, false
	}

	claims.Username = user.Username
	claims.Role = user.Role
	claims.RouteIDs = append([]string(nil), user.RouteIDs...)
	return claims, true
}

func SetRouteAccessCookie(c *gin.Context, token string) {
	secure := forwardedProto(c.Request) == "https"
	c.SetCookie(routeAccessCookieName, token, 86400, "/", "", secure, true)
}

func ClearRouteAccessCookie(c *gin.Context) {
	secure := forwardedProto(c.Request) == "https"
	c.SetCookie(routeAccessCookieName, "", -1, "/", "", secure, true)
}

func SanitizeAccessRedirect(next string, fallback string) string {
	next = strings.TrimSpace(next)
	if next == "" {
		return fallback
	}
	if strings.HasPrefix(next, "//") {
		return fallback
	}
	if strings.Contains(next, `\`) {
		return fallback
	}

	parsed, err := url.Parse(next)
	if err != nil || parsed.IsAbs() {
		return fallback
	}
	if !strings.HasPrefix(parsed.Path, "/") {
		return fallback
	}
	return parsed.RequestURI()
}

func buildAccessLoginURL(route *router.Route, requestURI string) string {
	values := url.Values{}
	values.Set("route_id", route.ID)
	values.Set("next", SanitizeAccessRedirect(requestURI, route.PathPrefix))
	values.Set("route_name", route.Name)
	values.Set("path_prefix", route.PathPrefix)
	return "/_authgate/#" + routeLoginHashPath + "?" + values.Encode()
}
