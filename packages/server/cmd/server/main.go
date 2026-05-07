package main

import (
	"crypto/rand"
	"encoding/base64"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/pallyoung/auth-gate/packages/server/internal/auth"
	"github.com/pallyoung/auth-gate/packages/server/internal/config"
	adminhttp "github.com/pallyoung/auth-gate/packages/server/internal/http/admin"
	proxyhttp "github.com/pallyoung/auth-gate/packages/server/internal/http/proxy"
	statichttp "github.com/pallyoung/auth-gate/packages/server/internal/http/static"
	"github.com/pallyoung/auth-gate/packages/server/internal/router"
	"github.com/pallyoung/auth-gate/packages/server/internal/store"

	"github.com/gin-gonic/gin"
)

const bootstrapAdminUsername = "admin"
const controlPlaneBasePath = "/_authgate"
const controlPlaneAPIBasePath = controlPlaneBasePath + "/api"

func getWebRoot() string {
	if webRoot := os.Getenv("WEB_ROOT"); webRoot != "" {
		return webRoot
	}

	for _, candidate := range webRootCandidates() {
		if hasIndexFile(candidate) {
			return candidate
		}
	}

	cwd, _ := os.Getwd()
	return filepath.Join(cwd, "web")
}

func webRootCandidates() []string {
	var candidates []string

	if exePath, err := os.Executable(); err == nil {
		exeDir := filepath.Dir(exePath)
		candidates = append(candidates,
			filepath.Join(exeDir, "web"),
			filepath.Join(exeDir, "dist"),
			filepath.Join(exeDir, "web", "dist"),
		)
	}

	cwd, _ := os.Getwd()
	candidates = append(candidates,
		filepath.Join(cwd, "web"),
		filepath.Join(cwd, "dist"),
		filepath.Join(cwd, "web", "dist"),
		filepath.Join(cwd, "packages", "web", "dist"),
		filepath.Join(cwd, "..", "web", "dist"),
		filepath.Join(cwd, "..", "..", "web", "dist"),
	)

	return dedupePaths(candidates)
}

func dedupePaths(paths []string) []string {
	seen := make(map[string]struct{}, len(paths))
	result := make([]string, 0, len(paths))

	for _, path := range paths {
		cleaned := filepath.Clean(path)
		if _, exists := seen[cleaned]; exists {
			continue
		}
		seen[cleaned] = struct{}{}
		result = append(result, cleaned)
	}

	return result
}

func hasIndexFile(path string) bool {
	info, err := os.Stat(filepath.Join(path, "index.html"))
	return err == nil && !info.IsDir()
}

func buildEngine(routerMgr *router.Manager, webRoot string, db *store.SQLite) *gin.Engine {
	engine := gin.New()
	engine.Use(gin.Recovery())

	statichttp.RegisterRoutes(engine, webRoot, controlPlaneBasePath)

	// Public routes
	engine.POST(controlPlaneAPIBasePath+"/auth/login", adminhttp.LoginRoute(db))
	engine.POST(controlPlaneAPIBasePath+"/access/login", proxyhttp.AccessLoginRoute(routerMgr, db))
	engine.POST(controlPlaneAPIBasePath+"/access/logout", proxyhttp.AccessLogoutRoute())

	// Protected API routes
	apiGroup := engine.Group(controlPlaneAPIBasePath)
	apiGroup.Use(auth.AuthMiddleware(db))
	adminhttp.RegisterRoutes(apiGroup, routerMgr, db)

	// Proxy for unmatched routes
	proxyhttp.RegisterRoutes(engine, routerMgr)

	return engine
}

func configureJWTSecret(cfg config.AuthConfig) {
	if cfg.HasLegacyAdminToken() {
		log.Printf("Warning: auth.admin_token is deprecated and ignored; use JWT_SECRET or auth.jwt_secret")
	}

	if strings.TrimSpace(os.Getenv("JWT_SECRET")) != "" {
		return
	}
	if secret := cfg.JWTSecretValue(); secret != "" {
		auth.ConfigureJWTSecret(secret)
		return
	}
	if !cfg.AllowEphemeralJWT() {
		log.Fatalf("JWT secret not configured; set JWT_SECRET or auth.jwt_secret")
	}
	if auth.UsingGeneratedJWTSecret() {
		log.Printf("Warning: JWT secret not configured; using an ephemeral secret for this process")
	}
}

func ensureBootstrapAdmin(db *store.SQLite, cfg config.AuthConfig) error {
	password := os.Getenv("BOOTSTRAP_ADMIN_PASSWORD")
	configuredPassword := true

	if strings.TrimSpace(password) == "" {
		password = cfg.BootstrapPasswordValue()
	}
	if strings.TrimSpace(password) == "" {
		generatedPassword, err := generateCredential(24)
		if err != nil {
			return err
		}
		password = generatedPassword
		configuredPassword = false
	}

	created, err := db.EnsureAdmin(bootstrapAdminUsername, password)
	if err != nil {
		return err
	}
	if !created {
		return nil
	}

	if configuredPassword {
		log.Printf("Bootstrap admin created: username=%s using configured password", bootstrapAdminUsername)
		return nil
	}

	log.Printf("Bootstrap admin created: username=%s password=%s", bootstrapAdminUsername, password)
	return nil
}

func generateCredential(size int) (string, error) {
	buf := make([]byte, size)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

func main() {
	ensureDataDir()

	cfg, err := config.Load("config.yaml")
	if err != nil {
		log.Printf("Warning: config load failed: %v, using defaults", err)
		cfg = config.DefaultConfig()
	}
	configureJWTSecret(cfg.Auth)

	db, err := store.NewSQLite(cfg.Database.Path)
	if err != nil {
		log.Fatalf("Failed to init database: %v", err)
	}
	defer db.Close()

	if err := ensureBootstrapAdmin(db, cfg.Auth); err != nil {
		log.Printf("Warning: failed to ensure admin: %v", err)
	}

	routerMgr := router.NewManager(db)

	gin.SetMode(gin.ReleaseMode)
	if os.Getenv("DEBUG") == "true" {
		gin.SetMode(gin.DebugMode)
	}

	webRoot := getWebRoot()
	log.Printf("Serving web from: %s", webRoot)
	log.Printf("Control plane available at: %s", controlPlaneBasePath)

	engine := buildEngine(routerMgr, webRoot, db)

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
