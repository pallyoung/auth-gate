package main

import (
	"context"
	"crypto/rand"
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/pallyoung/auth-gate/packages/server/internal/auth"
	"github.com/pallyoung/auth-gate/packages/server/internal/config"
	adminhttp "github.com/pallyoung/auth-gate/packages/server/internal/http/admin"
	proxyhttp "github.com/pallyoung/auth-gate/packages/server/internal/http/proxy"
	statichttp "github.com/pallyoung/auth-gate/packages/server/internal/http/static"
	"github.com/pallyoung/auth-gate/packages/server/internal/localca"
	"github.com/pallyoung/auth-gate/packages/server/internal/routehost"
	"github.com/pallyoung/auth-gate/packages/server/internal/router"
	certservice "github.com/pallyoung/auth-gate/packages/server/internal/service/certificate"
	hostservice "github.com/pallyoung/auth-gate/packages/server/internal/service/hosts"
	"github.com/pallyoung/auth-gate/packages/server/internal/store"
	"github.com/pallyoung/auth-gate/packages/server/internal/syshosts"

	"github.com/gin-gonic/gin"
)

const controlPlaneBasePath = "/_authgate"
const controlPlaneAPIBasePath = controlPlaneBasePath + "/api"

// TLSHost groups routes that share the same TLS certificate (same host).
type TLSHost struct {
	Host       string // ":443" or "example.com:443"
	CertPath   string
	KeyPath    string
	RouteCount int
	RouteNames []string
}

// tlsHostKey returns a unique key for grouping routes by TLS config.
// Routes sharing the same (host, cert, key) tuple go into the same TLS server.
func tlsHostKey(host, cert, key string) string {
	return fmt.Sprintf("%s|%s|%s", routehost.TLSListenHost(host), cert, key)
}

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

// buildEngine constructs a single unified engine (compatibility mode).
// Used when admin.addr is not configured.
func buildEngine(routerMgr *router.Manager, webRoot string, db store.Store, certSvc adminhttp.CertService, hostSvc adminhttp.HostService, accessLogStore *store.AccessLogStore, cfg *config.Config) *gin.Engine {
	engine := gin.New()
	engine.Use(gin.Recovery())

	statichttp.RegisterRoutes(engine, webRoot, controlPlaneBasePath)

	// Runtime config for frontend (tells it the API base path)
	engine.GET(controlPlaneAPIBasePath+"/config/app", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"api_base": controlPlaneAPIBasePath})
	})

	// Public routes
	engine.POST(controlPlaneAPIBasePath+"/auth/login", adminhttp.LoginRoute(db, certSvc))
	engine.GET(controlPlaneAPIBasePath+"/auth/setup-status", adminhttp.SetupStatusRoute(db))
	engine.POST(controlPlaneAPIBasePath+"/auth/setup", adminhttp.SetupRoute(db, certSvc))
	engine.POST(controlPlaneAPIBasePath+"/access/login", proxyhttp.AccessLoginRoute(routerMgr, db))
	engine.POST(controlPlaneAPIBasePath+"/access/logout", proxyhttp.AccessLogoutRoute())

	// Protected API routes
	apiGroup := engine.Group(controlPlaneAPIBasePath)
	apiGroup.Use(auth.AuthMiddleware(db))
	adminhttp.RegisterRoutes(apiGroup, routerMgr, db, certSvc, hostSvc, accessLogStore, cfg)

	// Proxy for unmatched routes
	proxyhttp.RegisterRoutes(engine, routerMgr, accessLogStore)

	return engine
}

// buildAdminEngine constructs the admin/control-plane engine.
// Serves the management UI (SPA) and admin API only — no proxy, no NoRoute.
// In dual-engine mode the admin has its own port, so no /_authgate prefix is needed.
func buildAdminEngine(routerMgr *router.Manager, webRoot string, db store.Store, certSvc adminhttp.CertService, hostSvc adminhttp.HostService, accessLogStore *store.AccessLogStore, cfg *config.Config) *gin.Engine {
	engine := gin.New()
	engine.Use(gin.Recovery())

	// No prefix — admin engine owns its port, no conflict with proxy routes.
	statichttp.RegisterRoutes(engine, webRoot, "")

	// Runtime config for frontend
	engine.GET("/api/config/app", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"api_base": "/api"})
	})

	// Public admin routes
	engine.POST("/api/auth/login", adminhttp.LoginRoute(db, certSvc))
	engine.GET("/api/auth/setup-status", adminhttp.SetupStatusRoute(db))
	engine.POST("/api/auth/setup", adminhttp.SetupRoute(db, certSvc))

	// Protected admin API routes
	apiGroup := engine.Group("/api")
	apiGroup.Use(auth.AuthMiddleware(db))
	adminhttp.RegisterRoutes(apiGroup, routerMgr, db, certSvc, hostSvc, accessLogStore, cfg)

	return engine
}

// buildProxyEngine constructs the proxy engine.
// Serves gateway access login, self-contained login page, and the catch-all
// reverse proxy — no admin UI, no management API.
func buildProxyEngine(routerMgr *router.Manager, db store.Store, accessLogStore *store.AccessLogStore) *gin.Engine {
	engine := gin.New()
	engine.Use(gin.Recovery())

	proxyhttp.RegisterProxyRoutes(engine, routerMgr, db, accessLogStore)

	return engine
}

func buildTLSHostGroups(routes []router.Route, httpsPort int) []TLSHost {
	groupMap := make(map[string]*TLSHost)

	for _, r := range routes {
		if !r.Enabled || !r.TLSEnabled {
			continue
		}
		if r.TLSCert == "" || r.TLSKey == "" {
			continue
		}

		key := tlsHostKey(r.Host, r.TLSCert, r.TLSKey)
		if g, ok := groupMap[key]; ok {
			g.RouteCount++
			g.RouteNames = append(g.RouteNames, r.Name)
		} else {
			listenHost := routehost.TLSListenHostPort(r.Host, httpsPort)
			groupMap[key] = &TLSHost{
				Host:       listenHost,
				CertPath:   r.TLSCert,
				KeyPath:    r.TLSKey,
				RouteCount: 1,
				RouteNames: []string{r.Name},
			}
		}
	}

	result := make([]TLSHost, 0, len(groupMap))
	for _, g := range groupMap {
		result = append(result, *g)
	}
	return result
}

func startHTTPServers(ctx context.Context, engine *gin.Engine, routerMgr *router.Manager, cfg *config.Config) []*http.Server {
	var servers []*http.Server

	// Start HTTP servers for each non-TLS listen address
	for _, addr := range cfg.EffectiveListenAddrs() {
		srv := &http.Server{
			Addr:    addr,
			Handler: engine,
		}
		go func(a string) {
			log.Printf("HTTP server listening on %s", a)
			if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				log.Printf("HTTP server error on %s: %v", a, err)
			}
		}(addr)
		servers = append(servers, srv)
	}

	// Start HTTPS servers for each TLS listen address.
	// Uses GetConfigForClient so certificates added via the admin UI are picked
	// up on the next TLS handshake without restarting the server.
	httpsAddrs := cfg.EffectiveHTTPSAddrs()
	if len(httpsAddrs) > 0 {
		dynamicTLS := &tls.Config{
			GetConfigForClient: func(hello *tls.ClientHelloInfo) (*tls.Config, error) {
				allRoutes := routerMgr.GetRoutes()
				certCfg := &tls.Config{}
				seen := make(map[string]bool)
				for _, r := range allRoutes {
					if !r.Enabled || !r.TLSEnabled || r.TLSCert == "" || r.TLSKey == "" {
						continue
					}
					key := r.TLSCert + "|" + r.TLSKey
					if seen[key] {
						continue
					}
					seen[key] = true
					cert, err := tls.LoadX509KeyPair(r.TLSCert, r.TLSKey)
					if err != nil {
						log.Printf("TLS: failed to load cert (cert=%s): %v", r.TLSCert, err)
						continue
					}
					certCfg.Certificates = append(certCfg.Certificates, cert)
				}
				if len(certCfg.Certificates) == 0 {
					return nil, fmt.Errorf("no TLS certificates available")
				}
				return certCfg, nil
			},
		}

		for _, addr := range httpsAddrs {
			ln, err := tls.Listen("tcp", addr, dynamicTLS)
			if err != nil {
				log.Printf("Failed to listen on TLS %s: %v", addr, err)
				continue
			}
			httpsSrv := &http.Server{
				Addr:    addr,
				Handler: engine,
			}
			go func(a string, l net.Listener) {
				log.Printf("HTTPS server listening on %s (dynamic TLS)", a)
				if err := httpsSrv.Serve(l); err != nil && err != http.ErrServerClosed {
					log.Printf("HTTPS server error on %s: %v", a, err)
				}
			}(addr, ln)
			servers = append(servers, httpsSrv)
		}
	}

	return servers
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

func generateCredential(size int) (string, error) {
	buf := make([]byte, size)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

// startForeground runs the Auth Gate server in the foreground (blocking).
// This is the original main() logic, now callable from the CLI "start --foreground" command.
func startForeground() {
	os.MkdirAll(dataDir, 0755)

	// Write PID file so stop/status can find us.
	if err := writePIDFile(); err != nil {
		log.Printf("Warning: failed to write PID file: %v", err)
	}
	defer removePIDFile()

	cfg, err := config.Load("config.yaml")
	if err != nil {
		log.Printf("Warning: config load failed: %v, using defaults", err)
		cfg = config.DefaultConfig()
	}
	configureJWTSecret(cfg.Auth)

	db, err := store.NewJSONStore(cfg.Database.Path)
	if err != nil {
		log.Fatalf("Failed to init database: %v", err)
	}
	defer db.Close()

	// Initialize access log store
	accessLogStore, err := store.NewAccessLogStore(cfg.Database.Path, 10000)
	if err != nil {
		log.Fatalf("Failed to init access log store: %v", err)
	}
	accessLogStore.StartFlusher(30 * time.Second)
	accessLogStore.StartCleanup(func() int {
		v, _ := db.GetSetting("log_retention_days")
		days, _ := strconv.Atoi(v)
		return days
	})
	defer accessLogStore.StopFlusher()

	routerMgr := router.NewManager(db)

	// Initialize certificate service: local CA + sign-on-demand
	var certSvc *certservice.Service
	certDataDir := os.Getenv("CERT_DATA_DIR")
	if certDataDir == "" {
		certDataDir = "data"
	}
	ca, err := localca.LoadOrCreate(certDataDir)
	if err != nil {
		log.Fatalf("Failed to initialize local CA: %v", err)
	}
	// Persist the CA so the cert service can stamp ca_id on each leaf.
	if err := persistLocalCA(db, ca); err != nil {
		log.Fatalf("Failed to persist local CA: %v", err)
	}
	certSvc, err = certservice.NewService(db, certservice.Config{
		DataDir: certDataDir,
		CA:      ca,
	}, routerMgr)
	if err != nil {
		log.Fatalf("Failed to initialize certificate service: %v", err)
	}
	certSvc.StartRenewer(time.Hour)
	log.Printf("Certificate service initialized (local CA, auto re-sign 30 days before expiry)")

	hostsDataDir := certDataDir
	renderer := syshosts.NewRenderer(hostsDataDir)
	hostSvc := hostservice.NewService(db, renderer)

	gin.SetMode(gin.ReleaseMode)
	if os.Getenv("DEBUG") == "true" {
		gin.SetMode(gin.DebugMode)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	var servers []*http.Server

	adminAddr := cfg.AdminListenAddr()
	if adminAddr != "" {
		// Dual-engine mode: admin and proxy on separate engines/ports.
		webRoot := getWebRoot()
		log.Printf("Serving web from: %s", webRoot)
		log.Printf("Admin server: %s", adminAddr)

		adminEngine := buildAdminEngine(routerMgr, webRoot, db, certSvc, hostSvc, accessLogStore, cfg)
		proxyEngine := buildProxyEngine(routerMgr, db, accessLogStore)

		// Start admin server (HTTP only, typically localhost)
		adminSrv := &http.Server{Addr: adminAddr, Handler: adminEngine}
		go func() {
			log.Printf("Admin server listening on %s", adminAddr)
			if err := adminSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				log.Printf("Admin server error on %s: %v", adminAddr, err)
			}
		}()
		servers = append(servers, adminSrv)

		// Start proxy servers (external-facing, with TLS support)
		log.Printf("Auth Gate starting (dual-engine mode)...")
		proxyServers := startHTTPServers(ctx, proxyEngine, routerMgr, cfg)
		servers = append(servers, proxyServers...)
	} else {
		// Single-engine mode (backward compatible).
		webRoot := getWebRoot()
		log.Printf("Serving web from: %s", webRoot)
		log.Printf("Control plane available at: %s", controlPlaneBasePath)

		engine := buildEngine(routerMgr, webRoot, db, certSvc, hostSvc, accessLogStore, cfg)

		log.Printf("Auth Gate starting...")
		servers = startHTTPServers(ctx, engine, routerMgr, cfg)
	}

	// Wait for shutdown signal
	<-ctx.Done()
	log.Printf("Shutdown signal received, stopping servers...")

	// Stop certificate service renewer
	certSvc.StopRenewer()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), gracefulShutdownTimeout)
	defer cancel()

	for _, srv := range servers {
		if err := srv.Shutdown(shutdownCtx); err != nil {
			log.Printf("Error shutting down server %s: %v", srv.Addr, err)
		}
	}

	log.Printf("Auth Gate stopped gracefully")
}

func main() {
	Execute()
}

func ensureDataDir() {
	os.MkdirAll("data", 0755)
}

// persistLocalCA stores the CA cert/key in the database on first run. Existing
// CA rows are left untouched, so re-running the service is a no-op.
func persistLocalCA(db store.Store, ca *localca.CA) error {
	if existing, err := db.GetFirstCACertificate(); err != nil {
		return err
	} else if existing != nil {
		return nil
	}
	return db.CreateCACertificate(&store.CACertificate{
		ID:        "ca-default",
		Name:      ca.Cert.Subject.CommonName,
		CertPEM:   string(ca.CertPEM),
		KeyPEM:    string(ca.KeyPEM),
		NotBefore: ca.Cert.NotBefore,
		NotAfter:  ca.Cert.NotAfter,
		CreatedAt: time.Now(),
	})
}

const gracefulShutdownTimeout = 10 // seconds
