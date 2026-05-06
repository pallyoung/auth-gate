package static

import (
	"os"
	"path/filepath"

	"github.com/gin-gonic/gin"
)

func RegisterRoutes(engine *gin.Engine, webRoot string) {
	assetsDir := filepath.Join(webRoot, "assets")
	if info, err := os.Stat(assetsDir); err == nil && info.IsDir() {
		engine.Static("/assets", assetsDir)
	}

	registerStaticFile(engine, "/favicon.svg", filepath.Join(webRoot, "favicon.svg"))
	registerStaticFile(engine, "/favicon.ico", filepath.Join(webRoot, "favicon.ico"))
	engine.GET("/", serveIndex(webRoot))
}

func registerStaticFile(engine *gin.Engine, route, path string) {
	info, err := os.Stat(path)
	if err != nil || info.IsDir() {
		return
	}
	engine.StaticFile(route, path)
}

func serveIndex(webRoot string) gin.HandlerFunc {
	indexPath := filepath.Join(webRoot, "index.html")
	return func(c *gin.Context) {
		c.File(indexPath)
	}
}
