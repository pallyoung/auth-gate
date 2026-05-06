package main

import (
	"log"
	"os"
	"path/filepath"

	"github.com/pallyoung/auth-gate/packages/server/internal/api"
	"github.com/pallyoung/auth-gate/packages/server/internal/auth"
	"github.com/pallyoung/auth-gate/packages/server/internal/config"
	"github.com/pallyoung/auth-gate/packages/server/internal/proxy"
	"github.com/pallyoung/auth-gate/packages/server/internal/router"
	"github.com/pallyoung/auth-gate/packages/server/internal/store"

	"github.com/gin-gonic/gin"
)

func getWebRoot() string {
	// Try executable directory first
	if exePath, err := os.Executable(); err == nil {
		exeDir := filepath.Dir(exePath)
		webDist := filepath.Join(exeDir, "web", "dist")
		if _, err := os.Stat(webDist); err == nil {
			return webDist
		}
	}

	// Fallback to current working directory
	cwd, _ := os.Getwd()
	return filepath.Join(cwd, "web", "dist")
}

func main() {
    ensureDataDir()
	cfg, err := config.Load("config.yaml")
	if err != nil {
		log.Printf("Warning: config load failed: %v, using defaults", err)
		cfg = config.DefaultConfig()
	}

	db, err := store.NewSQLite(cfg.Database.Path)
	if err != nil {
		log.Fatalf("Failed to init database: %v", err)
	}
	defer db.Close()

	if err := db.EnsureAdmin(); err != nil {
		log.Printf("Warning: failed to ensure admin: %v", err)
	}

	routerMgr := router.NewManager(db)

	gin.SetMode(gin.ReleaseMode)
	if os.Getenv("DEBUG") == "true" {
		gin.SetMode(gin.DebugMode)
	}

	engine := gin.New()
	engine.Use(gin.Recovery())

	webRoot := getWebRoot()
	log.Printf("Serving web from: %s", webRoot)

	engine.Static("/assets", filepath.Join(webRoot, "assets"))
	engine.StaticFile("/", filepath.Join(webRoot, "index.html"))
	engine.StaticFile("/favicon.ico", filepath.Join(webRoot, "favicon.ico"))

	// Public routes
	engine.POST("/api/auth/login", api.LoginHandler(db))

	// Protected API routes
	apiGroup := engine.Group("/api")
	apiGroup.Use(auth.AuthMiddleware())
	api.RegisterHandlers(apiGroup, routerMgr, db)

	engine.NoRoute(proxy.Handler(routerMgr))

	addr := cfg.Server.Addr
	if addr == "" {
		addr = ":8080"
	}
	log.Printf("Auth Gate starting on %s", addr)
	if err := engine.Run(addr); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}

func ensureDataDir() {
    os.MkdirAll("data", 0755)
}
