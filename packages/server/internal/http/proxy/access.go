package proxy

import (
	"net/http"

	"github.com/gin-gonic/gin"

	proxyhandler "github.com/pallyoung/auth-gate/packages/server/internal/proxy"
	"github.com/pallyoung/auth-gate/packages/server/internal/router"
	"github.com/pallyoung/auth-gate/packages/server/internal/service/session"
	"github.com/pallyoung/auth-gate/packages/server/internal/store"
)

func AccessLoginRoute(routerMgr *router.Manager, db *store.SQLite) gin.HandlerFunc {
	sessionSvc := session.NewService(db)
	return accessLoginHandler(routerMgr, sessionSvc)
}

func AccessLogoutRoute() gin.HandlerFunc {
	return accessLogoutHandler()
}

func accessLoginHandler(routerMgr *router.Manager, sessionSvc *session.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			RouteID  string `json:"route_id" binding:"required"`
			Username string `json:"username" binding:"required"`
			Password string `json:"password" binding:"required"`
			Next     string `json:"next"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": gin.H{"code": "invalid_request", "message": "invalid request"},
			})
			return
		}

		activeRoute := routerMgr.FindByID(req.RouteID)
		if activeRoute == nil {
			c.JSON(http.StatusNotFound, gin.H{
				"error": gin.H{"code": "route_not_found", "message": "route not found"},
			})
			return
		}

		routeSession, err := sessionSvc.LoginForRoute(req.RouteID, req.Username, req.Password)
		if err != nil {
			c.JSON(routeSessionStatus(err), gin.H{
				"error": gin.H{"code": session.Code(err), "message": err.Error()},
			})
			return
		}

		proxyhandler.SetRouteAccessCookie(c, routeSession.Token)

		next := sanitizeAccessRedirect(req.Next, activeRoute.PathPrefix)
		c.JSON(http.StatusOK, gin.H{
			"next": next,
			"user": gin.H{
				"id":       routeSession.User.ID,
				"username": routeSession.User.Username,
				"role":     routeSession.User.Role,
			},
		})
	}
}

func accessLogoutHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		proxyhandler.ClearRouteAccessCookie(c)
		c.JSON(http.StatusOK, gin.H{"message": "logged out"})
	}
}

func routeSessionStatus(err error) int {
	switch session.Code(err) {
	case session.ErrCodeInvalidCredentials, session.ErrCodeUserDisabled, session.ErrCodeRouteAccessDenied, session.ErrCodeControlPlaneAccessDenied:
		return http.StatusUnauthorized
	case session.ErrCodeRouteNotFound:
		return http.StatusNotFound
	default:
		return http.StatusInternalServerError
	}
}

func sanitizeAccessRedirect(next string, fallback string) string { return proxyhandler.SanitizeAccessRedirect(next, fallback) }
