package static

import (
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
)

func RegisterRoutes(engine *gin.Engine, webRoot string, basePath string) {
	basePath = normalizeBasePath(basePath)
	assetsDir := filepath.Join(webRoot, "assets")
	if info, err := os.Stat(assetsDir); err == nil && info.IsDir() {
		engine.Static(path.Join(basePath, "assets"), assetsDir)
	}

	registerStaticFile(engine, path.Join(basePath, "favicon.svg"), filepath.Join(webRoot, "favicon.svg"))
	registerStaticFile(engine, path.Join(basePath, "favicon.ico"), filepath.Join(webRoot, "favicon.ico"))
	engine.GET(basePath, serveIndex(webRoot))
	// Avoid duplicate registration: Gin treats "" and "/" as the same path.
	if basePath != "" {
		engine.GET(basePath+"/", serveIndex(webRoot))
	}
}

func normalizeBasePath(basePath string) string {
	trimmed := "/" + strings.Trim(basePath, "/")
	if trimmed == "/" {
		return ""
	}
	return trimmed
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
