package proxy

import (
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"

	"github.com/pallyoung/auth-gate/packages/server/internal/auth"
	httpresponse "github.com/pallyoung/auth-gate/packages/server/internal/http/response"
	"github.com/pallyoung/auth-gate/packages/server/internal/router"

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
			log.Printf("proxy match miss host=%s path=%s", host, path)
			c.JSON(http.StatusNotFound, httpresponse.ErrorEnvelope{
				Error: httpresponse.ErrorDetail{
					Code:    "route_not_found",
					Message: "no route found",
				},
			})
			return
		}
		log.Printf("proxy match route_id=%s host=%s path=%s backend=%s", route.ID, host, path, route.Backend)

		// 鉴权
		if route.AuthRule != nil && route.AuthRule.Type != "none" {
			if !auth.Check(c, route.AuthRule) {
				c.JSON(http.StatusUnauthorized, httpresponse.ErrorEnvelope{
					Error: httpresponse.ErrorDetail{
						Code:    "unauthorized",
						Message: "unauthorized",
					},
				})
				return
			}
		}

		// 反向代理
		backend, err := url.Parse(route.Backend)
		if err != nil {
			c.JSON(http.StatusInternalServerError, httpresponse.ErrorEnvelope{
				Error: httpresponse.ErrorDetail{
					Code:    "invalid_backend",
					Message: "invalid backend",
				},
			})
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
			log.Printf("proxy upstream error route_id=%s backend=%s err=%v", route.ID, route.Backend, err)
			c.JSON(http.StatusBadGateway, httpresponse.ErrorEnvelope{
				Error: httpresponse.ErrorDetail{
					Code:    "backend_error",
					Message: fmt.Sprintf("backend error: %v", err),
				},
			})
		}

		proxy.ServeHTTP(c.Writer, c.Request)
		c.Abort()
	}
}
