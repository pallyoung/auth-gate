package main

import (
	"log"
	"os"

	"auth-gate/internal/api"
	"auth-gate/internal/auth"
	"auth-gate/internal/config"
	"auth-gate/internal/proxy"
	"auth-gate/internal/router"
	"auth-gate/internal/store"

	"github.com/gin-gonic/gin"
)

func main() {
	cfg, err := config.Load("configs/config.yaml")
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

	engine.Static("/assets", "./web/dist/assets")
	engine.StaticFile("/", "./web/dist/index.html")
	engine.StaticFile("/favicon.ico", "./web/dist/favicon.ico")

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
