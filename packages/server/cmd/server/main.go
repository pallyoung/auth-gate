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

const bootstrapAdminUsername = "admin"
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

func buildEngine(routerMgr *router.Manager, webRoot string, db *store.SQLite, certSvc adminhttp.CertService, hostSvc adminhttp.HostService) *gin.Engine {
	engine := gin.New()
	engine.Use(gin.Recovery())

	statichttp.RegisterRoutes(engine, webRoot, controlPlaneBasePath)

	// Public routes
	engine.POST(controlPlaneAPIBasePath+"/auth/login", adminhttp.LoginRoute(db, certSvc))
	engine.POST(controlPlaneAPIBasePath+"/access/login", proxyhttp.AccessLoginRoute(routerMgr, db))
	engine.POST(controlPlaneAPIBasePath+"/access/logout", proxyhttp.AccessLogoutRoute())

	// Protected API routes
	apiGroup := engine.Group(controlPlaneAPIBasePath)
	apiGroup.Use(auth.AuthMiddleware(db))
	adminhttp.RegisterRoutes(apiGroup, routerMgr, db, certSvc, hostSvc)

	// Proxy for unmatched routes
	proxyhttp.RegisterRoutes(engine, routerMgr)

	return engine
}

func buildTLSHostGroups(routes []router.Route) []TLSHost {
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
			listenHost := routehost.TLSListenHost(r.Host)
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

func startHTTPServers(ctx context.Context, engine *gin.Engine, routerMgr *router.Manager, cfg config.ServerConfig) []*http.Server {
	// Collect routes and group by TLS config
	allRoutes := routerMgr.GetRoutes()
	tlsGroups := buildTLSHostGroups(allRoutes)

	var servers []*http.Server

	// Start HTTP server (existing behavior)
	httpAddr := cfg.Addr
	if httpAddr == "" {
		httpAddr = ":8080"
	}
	srv := &http.Server{
		Addr:    httpAddr,
		Handler: engine,
	}
	go func() {
		log.Printf("HTTP server listening on %s", httpAddr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("HTTP server error: %v", err)
		}
	}()
	servers = append(servers, srv)

	// Start per-route HTTPS servers
	for _, g := range tlsGroups {
		tlsCfg := &tls.Config{
			Certificates: make([]tls.Certificate, 0, 1),
		}
		cert, err := tls.LoadX509KeyPair(g.CertPath, g.KeyPath)
		if err != nil {
			log.Printf("Failed to load TLS certificate for %s (cert=%s): %v", g.Host, g.CertPath, err)
			continue
		}
		tlsCfg.Certificates = append(tlsCfg.Certificates, cert)

		httpsSrv := &http.Server{
			Addr:    g.Host,
			Handler: engine,
		}
		ln, err := tls.Listen("tcp", g.Host, tlsCfg)
		if err != nil {
			log.Printf("Failed to listen on %s: %v", g.Host, err)
			continue
		}

		go func(srv *http.Server, ln net.Listener, grp TLSHost) {
			routeNames := strings.Join(grp.RouteNames, ", ")
			log.Printf("HTTPS server listening on %s (cert=%s, key=%s, routes=%d: %s)",
				grp.Host, grp.CertPath, grp.KeyPath, grp.RouteCount, routeNames)
			if err := srv.Serve(ln); err != nil && err != http.ErrServerClosed {
				log.Printf("HTTPS server error on %s: %v", grp.Host, err)
			}
		}(httpsSrv, ln, g)
		servers = append(servers, httpsSrv)
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
		log.Printf("Bootstrap admin created: username=%s (using configured password)", bootstrapAdminUsername)
		return nil
	}

	log.Printf("")
	log.Printf("========================================")
	log.Printf("  Auth Gate Admin Console")
	log.Printf("========================================")
	log.Printf("  URL:   http://localhost:8080/_authgate")
	log.Printf("  Username: %s", bootstrapAdminUsername)
	log.Printf("  Password: %s", password)
	log.Printf("========================================")
	log.Printf("")
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

	webRoot := getWebRoot()
	log.Printf("Serving web from: %s", webRoot)
	log.Printf("Control plane available at: %s", controlPlaneBasePath)

	engine := buildEngine(routerMgr, webRoot, db, certSvc, hostSvc)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	log.Printf("Auth Gate starting...")
	servers := startHTTPServers(ctx, engine, routerMgr, cfg.Server)

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

func ensureDataDir() {
	os.MkdirAll("data", 0755)
}

// persistLocalCA stores the CA cert/key in the database on first run. Existing
// CA rows are left untouched, so re-running the service is a no-op.
func persistLocalCA(db *store.SQLite, ca *localca.CA) error {
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
