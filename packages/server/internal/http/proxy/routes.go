package proxy

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"

	proxyhandler "github.com/pallyoung/auth-gate/packages/server/internal/proxy"
	"github.com/pallyoung/auth-gate/packages/server/internal/router"
	"github.com/pallyoung/auth-gate/packages/server/internal/store"
)

// RegisterRoutes registers the proxy catch-all handler (used in single-engine mode).
func RegisterRoutes(engine *gin.Engine, routerMgr *router.Manager, accessLogStore *store.AccessLogStore) {
	engine.NoRoute(func(c *gin.Context) {
		log.Printf("proxy request method=%s host=%s path=%s", c.Request.Method, c.Request.Host, c.Request.URL.Path)
		proxyhandler.Handler(routerMgr, accessLogStore)(c)
	})
}

// RegisterProxyRoutes registers the proxy engine's NoRoute handler.
//
// All routing is dispatched inside a single NoRoute handler because Gin's
// radix tree does not allow a catch-all to coexist with any other routes
// on the same engine. Internal dispatch replaces explicit route registration.
func RegisterProxyRoutes(engine *gin.Engine, routerMgr *router.Manager, db store.Store, accessLogStore *store.AccessLogStore) {
	loginHandler := AccessLoginRoute(routerMgr, db)
	logoutHandler := AccessLogoutRoute()

	engine.NoRoute(func(c *gin.Context) {
		p := c.Request.URL.Path
		m := c.Request.Method

		// Gateway access login/logout API
		if p == "/api/access/login" && m == "POST" {
			loginHandler(c)
			return
		}
		if p == "/api/access/logout" && m == "POST" {
			logoutHandler(c)
			return
		}

		// Self-contained login page for gateway authentication
		if p == "/auth/access-login" && m == "GET" {
			c.Header("Content-Type", "text/html; charset=utf-8")
			c.String(http.StatusOK, loginPageHTML)
			return
		}

		// Everything else → reverse proxy
		log.Printf("proxy request method=%s host=%s path=%s", m, c.Request.Host, p)
		proxyhandler.Handler(routerMgr, accessLogStore)(c)
	})
}
