package proxy

import (
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"

	"auth-gate/internal/auth"
	"auth-gate/internal/router"

	"github.com/gin-gonic/gin"
)

func Handler(routerMgr *router.Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		host := c.Request.Host
		// 去掉端口号
		if idx := strings.Index(host, ":"); idx != -1 {
			host = host[:idx]
		}
		path := c.Request.URL.Path

		route := routerMgr.Match(host, path)
		if route == nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "no route found"})
			return
		}

		// 鉴权
		if route.AuthRule != nil && route.AuthRule.Type != "none" {
			if !auth.Check(c, route.AuthRule) {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
				return
			}
		}

		// 反向代理
		backend, err := url.Parse(route.Backend)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "invalid backend"})
			return
		}

		proxy := httputil.NewSingleHostReverseProxy(backend)

		// 修改请求
		originalDirector := proxy.Director
		proxy.Director = func(req *http.Request) {
			originalDirector(req)
			req.Host = backend.Host

			// 传递原始请求信息
			req.Header.Set("X-Forwarded-Host", c.Request.Host)
			req.Header.Set("X-Forwarded-Proto", "http")
			req.Header.Set("X-Forwarded-For", c.ClientIP())

			// 去掉前缀
			if route.StripPrefix {
				req.URL.Path = strings.TrimPrefix(req.URL.Path, route.PathPrefix)
				if !strings.HasPrefix(req.URL.Path, "/") {
					req.URL.Path = "/" + req.URL.Path
				}
			}
		}

		// 错误处理
		proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
			c.JSON(http.StatusBadGateway, gin.H{"error": fmt.Sprintf("backend error: %v", err)})
		}

		proxy.ServeHTTP(c.Writer, c.Request)
		c.Abort()
	}
}
