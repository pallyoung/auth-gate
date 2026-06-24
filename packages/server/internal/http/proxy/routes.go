package proxy

import (
	"log"

	"github.com/gin-gonic/gin"

	proxyhandler "github.com/pallyoung/auth-gate/packages/server/internal/proxy"
	"github.com/pallyoung/auth-gate/packages/server/internal/router"
	"github.com/pallyoung/auth-gate/packages/server/internal/store"
)

func RegisterRoutes(engine *gin.Engine, routerMgr *router.Manager, accessLogStore *store.AccessLogStore) {
	engine.NoRoute(func(c *gin.Context) {
		log.Printf("proxy request method=%s host=%s path=%s", c.Request.Method, c.Request.Host, c.Request.URL.Path)
		proxyhandler.Handler(routerMgr, accessLogStore)(c)
	})
}
