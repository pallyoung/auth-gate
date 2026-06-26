package admin

import (
	"context"
	"errors"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/pallyoung/auth-gate/packages/server/internal/api/dto"
	"github.com/pallyoung/auth-gate/packages/server/internal/auth"
	"github.com/pallyoung/auth-gate/packages/server/internal/config"
	"github.com/pallyoung/auth-gate/packages/server/internal/localca"
	"github.com/pallyoung/auth-gate/packages/server/internal/router"
	hostservice "github.com/pallyoung/auth-gate/packages/server/internal/service/hosts"
	accesslogservice "github.com/pallyoung/auth-gate/packages/server/internal/service/accesslog"
	apikeyservice "github.com/pallyoung/auth-gate/packages/server/internal/service/apikeys"
	routeauthservice "github.com/pallyoung/auth-gate/packages/server/internal/service/routeauth"
	httpresponse "github.com/pallyoung/auth-gate/packages/server/internal/http/response"
	authrulesservice "github.com/pallyoung/auth-gate/packages/server/internal/service/authrules"
	certservice "github.com/pallyoung/auth-gate/packages/server/internal/service/certificate"
	routesservice "github.com/pallyoung/auth-gate/packages/server/internal/service/routes"
	sessionservice "github.com/pallyoung/auth-gate/packages/server/internal/service/session"
	systemservice "github.com/pallyoung/auth-gate/packages/server/internal/service/system"
	usersservice "github.com/pallyoung/auth-gate/packages/server/internal/service/users"
	"github.com/pallyoung/auth-gate/packages/server/internal/store"
)

// CertService is the subset of *certservice.Service that the admin HTTP layer
// consumes. Defined here (rather than reusing the concrete type) so handler
// tests can use stubs.
type CertService interface {
	List() ([]store.Certificate, error)
	Get(id string) (*store.Certificate, error)
	ProvisionLocal(ctx context.Context, name, domain string, info *localca.SubjectInfo) (*store.Certificate, error)
	Import(ctx context.Context, name, domain, certPEM, keyPEM string) (*store.Certificate, error)
	Resign(id string) (*store.Certificate, error)
	Delete(id string) error
	GetCAExport() (certPEM, name string, notAfter time.Time, err error)
}

// HostService is the subset of *hostservice.Service that the admin HTTP layer
// consumes. Mirrors the CertService pattern.
type HostService interface {
	ListProfiles() ([]store.HostProfile, error)
	GetProfile(id string) (*store.HostProfile, error)
	CreateProfile(in hostservice.ProfileInput) (*store.HostProfile, error)
	UpdateProfile(id string, in hostservice.ProfileInput) (*store.HostProfile, error)
	DeleteProfile(id string) error
	ActivateProfile(id string) (*store.HostProfile, error)
	ListEntries(profileID string) ([]store.HostEntry, error)
	CreateEntry(profileID string, in hostservice.EntryInput) (*store.HostEntry, error)
	UpdateEntry(profileID, entryID string, in hostservice.EntryInput) (*store.HostEntry, error)
	ReorderEntries(profileID string, orderedIDs []string) error
	DeleteEntry(profileID, entryID string) error
}

func RegisterRoutes(group *gin.RouterGroup, routerMgr *router.Manager, db store.Store, certSvc CertService, hostSvc HostService, accessLogStore *store.AccessLogStore, cfg *config.Config, systemSvc *systemservice.Service) {
	group.Use(requestLogger())

	sessionSvc := sessionservice.NewService(db)
	routeSvc := routesservice.NewService(db, routerMgr, certSvc)
	userSvc := usersservice.NewService(db)
	accessLogSvc := accesslogservice.NewService(accessLogStore)

	group.POST("/auth/logout", logoutHandler())
	group.GET("/auth/me", meHandler(db, certSvc))

	// System stats - available to any authenticated user
	if systemSvc != nil {
		group.GET("/system/stats", getSystemStats(systemSvc))
	}

	group.GET("/routes", listRoutes(routeSvc))
	group.GET("/routes/:id", getRoute(routeSvc))
	group.GET("/route-auth-config/:routeId", getRouteAuthConfig(db))
	group.GET("/route-api-keys/:routeId", listApiKeys(db))

	// Access log endpoints - available to any authenticated user
	if accessLogStore != nil {
		group.GET("/access-logs", listAccessLogs(accessLogSvc, routerMgr))
		group.GET("/access-logs/stats", getAccessLogStats(accessLogSvc))
	}

	// Certificate endpoints
	if certSvc != nil {
		group.GET("/certificates", listCertificates(certSvc))
		group.GET("/certificates/:id", getCertificate(certSvc))
		group.GET("/ca", caExportHandler(certSvc))

		editor := group.Group("")
		editor.Use(auth.RequireRole(store.RoleAdmin, store.RoleEditor))
		{
			editor.POST("/certificates", createCertificate(certSvc))
			editor.DELETE("/certificates/:id", deleteCertificate(certSvc))
			editor.POST("/certificates/:id/resign", resignCertificate(certSvc))
		}
	}

	editor := group.Group("")
	editor.Use(auth.RequireRole(store.RoleAdmin, store.RoleEditor))
	{
		editor.POST("/routes", createRoute(routeSvc))
		editor.PUT("/routes/:id", updateRoute(routeSvc))
		editor.DELETE("/routes/:id", deleteRoute(routeSvc))

		editor.PUT("/route-auth-config/:routeId", updateRouteAuthConfig(db, routerMgr))
		editor.DELETE("/route-auth-config/:routeId", deleteRouteAuthConfig(db, routerMgr))

		editor.POST("/route-api-keys/:routeId", createApiKey(db, routerMgr))
		editor.PUT("/api-keys/:id", updateApiKey(db))
		editor.POST("/api-keys/:id/rotate", rotateApiKey(db))
		editor.POST("/api-keys/:id/expire", expireApiKey(db))
		editor.DELETE("/api-keys/:id", deleteApiKey(db))

		editor.POST("/config/reload", reloadConfig(routerMgr))
	}

	adminOnly := group.Group("")
	adminOnly.Use(auth.RequireRole(store.RoleAdmin))
	{
		adminOnly.GET("/config", getConfig(cfg))
		adminOnly.PUT("/config", updateConfig(cfg))
		adminOnly.GET("/settings/log-retention", getLogRetention(db))
		adminOnly.PUT("/settings/log-retention", updateLogRetention(db))
		adminOnly.POST("/settings/log-retention/purge", purgeLogs(db, accessLogStore))
		adminOnly.GET("/users", listUsers(userSvc))
		adminOnly.POST("/users", createUser(userSvc))
		adminOnly.PUT("/users/:id", updateUser(userSvc))
		adminOnly.DELETE("/users/:id", deleteUser(userSvc))
		adminOnly.GET("/metrics", metricsHandler())

		if hostSvc != nil {
			adminOnly.POST("/host-profiles", createHostProfile(hostSvc))
			adminOnly.PUT("/host-profiles/:id", updateHostProfile(hostSvc))
			adminOnly.DELETE("/host-profiles/:id", deleteHostProfile(hostSvc))
			adminOnly.POST("/host-profiles/:id/activate", activateHostProfile(hostSvc))
			adminOnly.POST("/host-profiles/:id/entries", createHostEntry(hostSvc))
			adminOnly.PUT("/host-profiles/:id/entries/reorder", reorderHostEntries(hostSvc))
			adminOnly.PUT("/host-profiles/:id/entries/:eid", updateHostEntry(hostSvc))
			adminOnly.DELETE("/host-profiles/:id/entries/:eid", deleteHostEntry(hostSvc))
		}
	}

	// Host profile read endpoints — available to any authenticated user.
	if hostSvc != nil {
		group.GET("/host-profiles", listHostProfiles(hostSvc))
		group.GET("/host-profiles/:id", getHostProfile(hostSvc))
		group.GET("/host-profiles/:id/entries", listHostEntries(hostSvc))
	}

	_ = sessionSvc
}

func LoginRoute(db store.Store, certSvc CertService) gin.HandlerFunc {
	sessionSvc := sessionservice.NewService(db)
	certificatesEnabled := certSvc != nil

	return func(c *gin.Context) {
		var req dto.LoginRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			writeError(c, http.StatusBadRequest, "invalid_request", "invalid request")
			return
		}

		session, err := sessionSvc.Login(req.Username, req.Password)
		if err != nil {
			writeServiceError(c, err)
			return
		}

		c.JSON(http.StatusOK, dto.LoginResponseFromStore(session.Token, session.User, session.Permissions, certificatesEnabled))
	}
}

// SetupStatusRoute returns whether the initial admin setup is required.
func SetupStatusRoute(db store.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		hasAdmin, err := db.HasAdminUsers()
		if err != nil {
			writeError(c, http.StatusInternalServerError, "internal_error", "failed to check setup status")
			return
		}
		c.JSON(http.StatusOK, gin.H{"setup_required": !hasAdmin})
	}
}

// SetupRoute creates the first admin user when no admin exists yet.
func SetupRoute(db store.Store, certSvc CertService) gin.HandlerFunc {
	certificatesEnabled := certSvc != nil

	return func(c *gin.Context) {
		hasAdmin, err := db.HasAdminUsers()
		if err != nil {
			writeError(c, http.StatusInternalServerError, "internal_error", "failed to check setup status")
			return
		}
		if hasAdmin {
			writeError(c, http.StatusConflict, "setup_already_completed", "admin user already exists")
			return
		}

		var req dto.SetupRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			writeError(c, http.StatusBadRequest, "invalid_request", "invalid request")
			return
		}

		user := &store.User{
			Username: strings.TrimSpace(req.Username),
			Role:     store.RoleAdmin,
			Enabled:  true,
		}
		if user.Username == "" {
			writeError(c, http.StatusBadRequest, "invalid_request", "username is required")
			return
		}
		hash, err := store.HashPassword(req.Password)
		if err != nil {
			writeError(c, http.StatusInternalServerError, "internal_error", "failed to hash password")
			return
		}
		user.PasswordHash = hash

		if err := db.CreateUser(user); err != nil {
			writeServiceError(c, err)
			return
		}

		token, err := auth.GenerateControlPlaneToken(user.ID, user.Username, user.Role)
		if err != nil {
			writeError(c, http.StatusInternalServerError, "token_generation_failed", "failed to generate token")
			return
		}

		permissions := store.GetPermissions(user.Role)
		c.JSON(http.StatusOK, dto.LoginResponseFromStore(token, *user, permissions, certificatesEnabled))
	}
}

func logoutHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, httpresponse.Message{Message: "logged out"})
	}
}

func meHandler(db store.Store, certSvc CertService) gin.HandlerFunc {
	certificatesEnabled := certSvc != nil

	return func(c *gin.Context) {
		current := auth.GetCurrentUser(c)
		user, err := db.GetUserByID(current.UserID)
		if err != nil {
			writeError(c, http.StatusUnauthorized, "unauthorized", "unauthorized")
			return
		}

		c.JSON(http.StatusOK, dto.CurrentUserResponse(*user, store.GetPermissions(user.Role), certificatesEnabled))
	}
}

func listRoutes(routeSvc *routesservice.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		routes, err := routeSvc.List()
		if err != nil {
			writeServiceError(c, err)
			return
		}
		c.JSON(http.StatusOK, dto.RouteListResponse(routes))
	}
}

func getRoute(routeSvc *routesservice.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		route, err := routeSvc.Get(c.Param("id"))
		if err != nil {
			writeServiceError(c, err)
			return
		}
		c.JSON(http.StatusOK, dto.RouteResponse(*route))
	}
}

func createRoute(routeSvc *routesservice.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req dto.RouteCreateRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			writeError(c, http.StatusBadRequest, "invalid_request", err.Error())
			return
		}

		route, err := routeSvc.Create(routesservice.CreateInput{
			Name:          req.Name,
			Host:          req.Host,
			PathPrefix:    req.PathPrefix,
			Backend:       req.Backend,
			StripPrefix:   req.StripPrefix,
			Enabled:       req.Enabled,
			Priority:      req.Priority,
			Type:          req.Type,
			StaticRoot:    req.StaticRoot,
			StaticSPA:     req.StaticSPA,
			TLSCert:       req.TLSCert,
			TLSKey:        req.TLSKey,
			TLSEnabled:    req.TLSEnabled,
			HTTPSRedirect: req.HTTPSRedirect,
			CertificateID: req.CertificateID,
			TimeoutMs:     req.TimeoutMs,
			RetryAttempts: req.RetryAttempts,
			Backends:      req.Backends,
			PathMatchMode: req.PathMatchMode,
			HeaderName:    req.HeaderName,
			HeaderValue:   req.HeaderValue,
			RewriteTarget: req.RewriteTarget,
			RedirectCode:  req.RedirectCode,
			// Header manipulation
			SetRequestHeaders:     req.SetRequestHeaders,
			RemoveRequestHeaders:  req.RemoveRequestHeaders,
			AddResponseHeaders:    req.AddResponseHeaders,
			RemoveResponseHeaders: req.RemoveResponseHeaders,
		})
		if err != nil {
			writeServiceError(c, err)
			return
		}
		c.JSON(http.StatusCreated, dto.RouteResponse(*route))
	}
}

func updateRoute(routeSvc *routesservice.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req dto.RouteUpdateRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			writeError(c, http.StatusBadRequest, "invalid_request", err.Error())
			return
		}

		route, err := routeSvc.Update(c.Param("id"), routesservice.UpdateInput{
			Name:          req.Name,
			Host:          req.Host,
			PathPrefix:    req.PathPrefix,
			Backend:       req.Backend,
			StripPrefix:   req.StripPrefix,
			Enabled:       req.Enabled,
			Priority:      req.Priority,
			Type:          req.Type,
			StaticRoot:    req.StaticRoot,
			StaticSPA:     req.StaticSPA,
			TLSCert:       req.TLSCert,
			TLSKey:        req.TLSKey,
			TLSEnabled:    req.TLSEnabled,
			HTTPSRedirect: req.HTTPSRedirect,
			CertificateID: req.CertificateID,
			TimeoutMs:     req.TimeoutMs,
			RetryAttempts: req.RetryAttempts,
			Backends:      req.Backends,
			PathMatchMode: req.PathMatchMode,
			HeaderName:    req.HeaderName,
			HeaderValue:   req.HeaderValue,
			RewriteTarget: req.RewriteTarget,
			RedirectCode:  req.RedirectCode,
			// Header manipulation
			SetRequestHeaders:     req.SetRequestHeaders,
			RemoveRequestHeaders:  req.RemoveRequestHeaders,
			AddResponseHeaders:    req.AddResponseHeaders,
			RemoveResponseHeaders: req.RemoveResponseHeaders,
		})
		if err != nil {
			writeServiceError(c, err)
			return
		}
		c.JSON(http.StatusOK, dto.RouteResponse(*route))
	}
}

func deleteRoute(routeSvc *routesservice.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		if err := routeSvc.Delete(c.Param("id")); err != nil {
			writeServiceError(c, err)
			return
		}
		c.JSON(http.StatusOK, httpresponse.Message{Message: "deleted"})
	}
}

func listAuthRules(authRuleSvc *authrulesservice.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		rules, err := authRuleSvc.List()
		if err != nil {
			writeServiceError(c, err)
			return
		}
		c.JSON(http.StatusOK, dto.AuthRuleListResponse(rules))
	}
}

func getAuthRule(authRuleSvc *authrulesservice.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		rule, err := authRuleSvc.Get(c.Param("id"))
		if err != nil {
			writeServiceError(c, err)
			return
		}
		c.JSON(http.StatusOK, dto.AuthRuleResponse(*rule))
	}
}

func createAuthRule(authRuleSvc *authrulesservice.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req dto.AuthRuleCreateRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			writeError(c, http.StatusBadRequest, "invalid_request", err.Error())
			return
		}

		rule, err := authRuleSvc.Create(authrulesservice.CreateInput{
			RouteID: req.RouteID,
			Type:    req.Type,
			Config: authrulesservice.AuthConfigInput{
				HeaderName: req.Config.HeaderName,
				Secret:     req.Config.Secret,
				Username:   req.Config.Username,
				Password:   req.Config.Password,
				LoginMode:  req.Config.LoginMode,
			},
			Whitelist:            req.Whitelist,
			RateLimit:            req.RateLimit,
			Burst:                req.Burst,
			CORSAllowedOrigins:   req.CORSAllowedOrigins,
			CORSAllowedMethods:   req.CORSAllowedMethods,
			CORSAllowedHeaders:   req.CORSAllowedHeaders,
			CORSAllowCredentials: req.CORSAllowCredentials,
			CORSMaxAge:           req.CORSMaxAge,
		})
		if err != nil {
			writeServiceError(c, err)
			return
		}
		c.JSON(http.StatusCreated, dto.AuthRuleResponse(*rule))
	}
}

func updateAuthRule(authRuleSvc *authrulesservice.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req dto.AuthRuleUpdateRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			writeError(c, http.StatusBadRequest, "invalid_request", err.Error())
			return
		}

		rule, err := authRuleSvc.Update(c.Param("id"), authrulesservice.UpdateInput{
			RouteID: req.RouteID,
			Type:    req.Type,
			Config: authrulesservice.UpdateAuthConfigInput{
				HeaderName: req.Config.HeaderName,
				Secret:     req.Config.Secret,
				Username:   req.Config.Username,
				Password:   req.Config.Password,
				LoginMode:  req.Config.LoginMode,
			},
			Whitelist:            req.Whitelist,
			RateLimit:            req.RateLimit,
			Burst:                req.Burst,
			CORSAllowedOrigins:   req.CORSAllowedOrigins,
			CORSAllowedMethods:   req.CORSAllowedMethods,
			CORSAllowedHeaders:   req.CORSAllowedHeaders,
			CORSAllowCredentials: req.CORSAllowCredentials,
			CORSMaxAge:           req.CORSMaxAge,
		})
		if err != nil {
			writeServiceError(c, err)
			return
		}
		c.JSON(http.StatusOK, dto.AuthRuleResponse(*rule))
	}
}

func deleteAuthRule(authRuleSvc *authrulesservice.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		if err := authRuleSvc.Delete(c.Param("id")); err != nil {
			writeServiceError(c, err)
			return
		}
		c.JSON(http.StatusOK, httpresponse.Message{Message: "deleted"})
	}
}

func listUsers(userSvc *usersservice.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		users, err := userSvc.List()
		if err != nil {
			writeServiceError(c, err)
			return
		}
		c.JSON(http.StatusOK, dto.UserListResponse(users))
	}
}

func createUser(userSvc *usersservice.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req dto.UserCreateRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			writeError(c, http.StatusBadRequest, "invalid_request", err.Error())
			return
		}

		user, err := userSvc.Create(usersservice.CreateInput{
			Username: req.Username,
			Password: req.Password,
			Role:     req.Role,
			Enabled:  req.Enabled == nil || *req.Enabled,
			RouteIDs: req.RouteIDs,
		})
		if err != nil {
			writeServiceError(c, err)
			return
		}
		c.JSON(http.StatusCreated, dto.UserResponse(*user))
	}
}

func updateUser(userSvc *usersservice.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req dto.UserUpdateRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			writeError(c, http.StatusBadRequest, "invalid_request", err.Error())
			return
		}

		user, err := userSvc.Update(c.Param("id"), usersservice.UpdateInput{
			Username: req.Username,
			Password: req.Password,
			Role:     req.Role,
			Enabled:  req.Enabled,
			RouteIDs: req.RouteIDs,
		})
		if err != nil {
			writeServiceError(c, err)
			return
		}
		c.JSON(http.StatusOK, dto.UserResponse(*user))
	}
}

func deleteUser(userSvc *usersservice.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		if err := userSvc.Delete(c.Param("id")); err != nil {
			writeServiceError(c, err)
			return
		}
		c.JSON(http.StatusOK, httpresponse.Message{Message: "deleted"})
	}
}

func reloadConfig(reloader interface{ Reload() }) gin.HandlerFunc {
	return func(c *gin.Context) {
		reloader.Reload()
		c.JSON(http.StatusOK, httpresponse.Message{Message: "reloaded"})
	}
}

func metricsHandler() gin.HandlerFunc {
	return gin.WrapH(promhttp.Handler())
}

// Certificate handlers

func listCertificates(certSvc CertService) gin.HandlerFunc {
	return func(c *gin.Context) {
		certs, err := certSvc.List()
		if err != nil {
			writeServiceError(c, err)
			return
		}
		c.JSON(http.StatusOK, dto.CertificateListResponseFromStore(certs))
	}
}

func getCertificate(certSvc CertService) gin.HandlerFunc {
	return func(c *gin.Context) {
		cert, err := certSvc.Get(c.Param("id"))
		if err != nil {
			writeServiceError(c, err)
			return
		}
		if cert == nil {
			writeError(c, http.StatusNotFound, "cert_not_found", "certificate not found")
			return
		}
		c.JSON(http.StatusOK, dto.CertificateResponseFromStore(*cert))
	}
}

func createCertificate(certSvc CertService) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req dto.CertificateWriteRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			writeError(c, http.StatusBadRequest, "invalid_request", err.Error())
			return
		}

		var cert *store.Certificate
		var err error
		switch req.Source {
		case "", dto.CertificateSourceLocalCA:
			var info *localca.SubjectInfo
			if req.Organization != "" || req.OrganizationalUnit != "" || req.Country != "" || req.Province != "" || req.Locality != "" {
				info = &localca.SubjectInfo{
					Organization:       req.Organization,
					OrganizationalUnit: req.OrganizationalUnit,
					Country:            req.Country,
					Province:           req.Province,
					Locality:           req.Locality,
				}
			}
			cert, err = certSvc.ProvisionLocal(context.Background(), req.Name, req.Domain, info)
		case dto.CertificateSourceImported:
			cert, err = certSvc.Import(context.Background(), req.Name, req.Domain, req.CertPEM, req.KeyPEM)
		default:
			writeError(c, http.StatusBadRequest, "invalid_source", "unknown certificate source: "+req.Source)
			return
		}
		if err != nil {
			writeServiceError(c, err)
			return
		}
		c.JSON(http.StatusCreated, dto.CertificateResponseFromStore(*cert))
	}
}

func deleteCertificate(certSvc CertService) gin.HandlerFunc {
	return func(c *gin.Context) {
		if err := certSvc.Delete(c.Param("id")); err != nil {
			writeServiceError(c, err)
			return
		}
		c.JSON(http.StatusOK, httpresponse.Message{Message: "deleted"})
	}
}

func resignCertificate(certSvc CertService) gin.HandlerFunc {
	return func(c *gin.Context) {
		if _, err := certSvc.Resign(c.Param("id")); err != nil {
			writeServiceError(c, err)
			return
		}
		c.JSON(http.StatusOK, httpresponse.Message{Message: "resigned"})
	}
}

func caExportHandler(certSvc CertService) gin.HandlerFunc {
	return func(c *gin.Context) {
		certPEM, name, notAfter, err := certSvc.GetCAExport()
		if err != nil {
			writeServiceError(c, err)
			return
		}
		c.JSON(http.StatusOK, dto.CAExportResponse{
			CertPEM: certPEM,
			Name:    name,
			NotAfter: notAfter.Format(time.RFC3339),
		})
	}
}

func certServiceError(c *gin.Context, err error) bool {
	var target *certservice.Error
	if !errors.As(err, &target) {
		return false
	}

	message := certservice.Message(err)

	switch targetCode := certservice.Code(err); targetCode {
	case certservice.ErrCodeCertNotFound:
		writeError(c, http.StatusNotFound, targetCode, message)
	case certservice.ErrCodeInvalidName, certservice.ErrCodeInvalidDomain, certservice.ErrCodeDomainExists,
		certservice.ErrCodeInvalidPEM, certservice.ErrCodeDomainMismatch,
		certservice.ErrCodeImportedCannotResign:
		writeError(c, http.StatusBadRequest, targetCode, message)
	case certservice.ErrCodeLocalCA, certservice.ErrCodeFilesystem:
		writeError(c, http.StatusInternalServerError, targetCode, message)
	default:
		writeError(c, http.StatusInternalServerError, targetCode, message)
	}
	return true
}

func writeServiceError(c *gin.Context, err error) {
	switch {
	case routeServiceError(c, err):
	case authRuleServiceError(c, err):
	case userServiceError(c, err):
	case sessionServiceError(c, err):
	case certServiceError(c, err):
	case hostServiceError(c, err):
	case apiKeyServiceError(c, err):
	case routeAuthServiceError(c, err):
	default:
		writeError(c, http.StatusInternalServerError, "internal_error", "internal server error")
	}
}

func routeServiceError(c *gin.Context, err error) bool {
	var target *routesservice.Error
	if !errors.As(err, &target) {
		return false
	}

	switch targetCode := routesservice.Code(err); targetCode {
	case routesservice.ErrCodeRouteNotFound:
		writeError(c, http.StatusNotFound, targetCode, target.Error())
	case routesservice.ErrCodeMissingRouteFields, routesservice.ErrCodeInvalidRoutePathPrefix, routesservice.ErrCodeInvalidRoutePathMatchMode, routesservice.ErrCodeInvalidRoutePathRegex, routesservice.ErrCodeReservedRoutePathPrefix, routesservice.ErrCodeInvalidRouteHost, routesservice.ErrCodeInvalidRouteBackend, routesservice.ErrCodeInvalidRouteBackendWeight, routesservice.ErrCodeInvalidRouteRedirectCode, routesservice.ErrCodeCertificateNotFound:
		writeError(c, http.StatusBadRequest, targetCode, target.Error())
	default:
		writeError(c, http.StatusInternalServerError, targetCode, target.Error())
	}
	return true
}

func authRuleServiceError(c *gin.Context, err error) bool {
	var target *authrulesservice.Error
	if !errors.As(err, &target) {
		return false
	}

	switch targetCode := authrulesservice.Code(err); targetCode {
	case authrulesservice.ErrCodeAuthRuleNotFound:
		writeError(c, http.StatusNotFound, targetCode, target.Error())
	case authrulesservice.ErrCodeRouteNotFound,
		authrulesservice.ErrCodeRouteIDRequired,
		authrulesservice.ErrCodeInvalidAuthRuleType,
		authrulesservice.ErrCodeMissingAPIKeySecret:
		writeError(c, http.StatusBadRequest, targetCode, target.Error())
	default:
		writeError(c, http.StatusInternalServerError, targetCode, target.Error())
	}
	return true
}

func userServiceError(c *gin.Context, err error) bool {
	var target *usersservice.Error
	if !errors.As(err, &target) {
		return false
	}

	switch targetCode := usersservice.Code(err); targetCode {
	case usersservice.ErrCodeUserNotFound:
		writeError(c, http.StatusNotFound, targetCode, target.Error())
	case usersservice.ErrCodeInvalidUsername, usersservice.ErrCodeInvalidRole, usersservice.ErrCodeDuplicateUser, usersservice.ErrCodeMissingPassword, usersservice.ErrCodeDuplicateRouteAccess, usersservice.ErrCodeRouteNotFound:
		writeError(c, http.StatusBadRequest, targetCode, target.Error())
	default:
		writeError(c, http.StatusInternalServerError, targetCode, target.Error())
	}
	return true
}

func sessionServiceError(c *gin.Context, err error) bool {
	var target *sessionservice.Error
	if !errors.As(err, &target) {
		return false
	}

	switch targetCode := sessionservice.Code(err); targetCode {
	case sessionservice.ErrCodeInvalidCredentials, sessionservice.ErrCodeUserDisabled:
		writeError(c, http.StatusUnauthorized, targetCode, target.Error())
	case sessionservice.ErrCodeControlPlaneAccessDenied:
		writeError(c, http.StatusForbidden, targetCode, target.Error())
	default:
		writeError(c, http.StatusInternalServerError, targetCode, target.Error())
	}
	return true
}

func hostServiceError(c *gin.Context, err error) bool {
	var target *hostservice.Error
	if !errors.As(err, &target) {
		return false
	}

	switch code := hostservice.Code(err); code {
	case hostservice.ErrCodeProfileNotFound, hostservice.ErrCodeEntryNotFound:
		writeError(c, http.StatusNotFound, code, target.Error())
	case hostservice.ErrCodeInvalidProfileName, hostservice.ErrCodeInvalidIP, hostservice.ErrCodeInvalidHostname,
		hostservice.ErrCodeInvalidComment, hostservice.ErrCodeDuplicateProfileName,
		hostservice.ErrCodeDuplicateHostname:
		writeError(c, http.StatusBadRequest, code, target.Error())
	case hostservice.ErrCodeMarkerMissing:
		writeError(c, http.StatusConflict, code, target.Error())
	case hostservice.ErrCodePermissionDenied:
		writeError(c, http.StatusForbidden, code, target.Error())
	case hostservice.ErrCodeRenderFailure, hostservice.ErrCodeStoreFailure:
		writeError(c, http.StatusInternalServerError, code, target.Error())
	default:
		writeError(c, http.StatusInternalServerError, code, target.Error())
	}
	return true
}

func writeError(c *gin.Context, status int, code, message string) {
	c.JSON(status, httpresponse.ErrorEnvelope{
		Error: httpresponse.ErrorDetail{
			Code:    code,
			Message: message,
		},
	})
}

func requestLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		startedAt := time.Now()
		c.Next()

		username, _ := c.Get("username")
		log.Printf(
			"admin request method=%s path=%s status=%d duration=%s user=%v",
			c.Request.Method,
			c.FullPath(),
			c.Writer.Status(),
			time.Since(startedAt).Round(time.Millisecond),
			username,
		)

		switch c.Request.Method {
		case http.MethodPost, http.MethodPut, http.MethodDelete:
			log.Printf(
				"admin audit method=%s path=%s status=%d user=%v",
				c.Request.Method,
				c.FullPath(),
				c.Writer.Status(),
				username,
			)
		}
	}
}

func listAccessLogs(svc *accesslogservice.Service, routerMgr *router.Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		var params dto.AccessLogQueryParams
		if err := c.ShouldBindQuery(&params); err != nil {
			writeError(c, http.StatusBadRequest, "invalid_params", "invalid query parameters")
			return
		}

		filter := store.AccessLogFilter{
			ClientIP:   params.ClientIP,
			Path:       params.Path,
			Username:   params.Username,
			AuthResult: params.AuthResult,
			RouteID:    params.RouteID,
			StatusCode: params.StatusCode,
		}

		if params.StartTime != "" {
			t, err := time.Parse(time.RFC3339, params.StartTime)
			if err != nil {
				writeError(c, http.StatusBadRequest, "invalid_start_time", "invalid start_time format")
				return
			}
			filter.StartTime = &t
		}
		if params.EndTime != "" {
			t, err := time.Parse(time.RFC3339, params.EndTime)
			if err != nil {
				writeError(c, http.StatusBadRequest, "invalid_end_time", "invalid end_time format")
				return
			}
			filter.EndTime = &t
		}

		page := params.Page
		if page < 1 {
			page = 1
		}
		perPage := params.PerPage
		if perPage < 1 {
			perPage = 20
		}
		if perPage > 100 {
			perPage = 100
		}

		result, err := svc.List(filter, page, perPage)
		if err != nil {
			writeError(c, http.StatusInternalServerError, "list_failed", "failed to list access logs")
			return
		}

		// Build route name lookup from router manager
		routeNameMap := make(map[string]string)
		if routerMgr != nil {
			for _, r := range routerMgr.GetRoutes() {
				if r.Name != "" {
					routeNameMap[r.ID] = r.Name
				} else if r.Host != "" {
					routeNameMap[r.ID] = r.Host + r.PathPrefix
				} else {
					routeNameMap[r.ID] = r.PathPrefix
				}
			}
		}

		entries := make([]dto.AccessLogEntry, len(result.Entries))
		for i, entry := range result.Entries {
			routeName := entry.RouteName
			if routeName == "" {
				routeName = routeNameMap[entry.RouteID]
			}
			entries[i] = dto.AccessLogEntry{
				RequestID:        entry.RequestID,
				RouteID:          entry.RouteID,
				RouteName:        routeName,
				Method:           entry.Method,
				Path:             entry.Path,
				BackendURL:       entry.BackendURL,
				BackendLatencyMs: entry.BackendLatencyMs,
				StatusCode:       entry.StatusCode,
				ClientIP:         entry.ClientIP,
				UserAgent:        entry.UserAgent,
				Username:         entry.Username,
				AuthResult:       entry.AuthResult,
				Timestamp:        entry.Timestamp.Format(time.RFC3339),
			}
		}

		c.JSON(http.StatusOK, dto.AccessLogListResponse{
			Entries:    entries,
			Total:      result.Total,
			Page:       result.Page,
			PerPage:    result.PerPage,
			TotalPages: result.TotalPages,
		})
	}
}

func getAccessLogStats(svc *accesslogservice.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		durationStr := c.DefaultQuery("duration", "1h")
		duration, err := time.ParseDuration(durationStr)
		if err != nil {
			writeError(c, http.StatusBadRequest, "invalid_duration", "invalid duration format")
			return
		}

		stats, err := svc.Stats(duration)
		if err != nil {
			writeError(c, http.StatusInternalServerError, "stats_failed", "failed to get access log stats")
			return
		}

		// Convert to DTO
		requestsPerMinute := make([]dto.TimeBucket, len(stats.RequestsPerMinute))
		for i, bucket := range stats.RequestsPerMinute {
			requestsPerMinute[i] = dto.TimeBucket{
				Time:  bucket.Time.Format(time.RFC3339),
				Count: bucket.Count,
			}
		}

		errorRatePerHour := make([]dto.TimeBucket, len(stats.ErrorRatePerHour))
		for i, bucket := range stats.ErrorRatePerHour {
			errorRatePerHour[i] = dto.TimeBucket{
				Time:  bucket.Time.Format(time.RFC3339),
				Count: bucket.Count,
			}
		}

		latencyPerHour := make([]dto.LatencyBucket, len(stats.LatencyPerHour))
		for i, bucket := range stats.LatencyPerHour {
			latencyPerHour[i] = dto.LatencyBucket{
				Time:  bucket.Time.Format(time.RFC3339),
				AvgMs: bucket.AvgMs,
				P95Ms: bucket.P95Ms,
			}
		}

		topPaths := make([]dto.PathCount, len(stats.TopPaths))
		for i, pc := range stats.TopPaths {
			topPaths[i] = dto.PathCount{
				Path:  pc.Path,
				Count: pc.Count,
			}
		}

		topIPs := make([]dto.IPCount, len(stats.TopIPs))
		for i, ip := range stats.TopIPs {
			topIPs[i] = dto.IPCount{
				IP:    ip.IP,
				Count: ip.Count,
			}
		}

		c.JSON(http.StatusOK, dto.AccessLogStatsResponse{
			TotalRequests:     stats.TotalRequests,
			SuccessCount:      stats.SuccessCount,
			ErrorCount:        stats.ErrorCount,
			AvgLatencyMs:      stats.AvgLatencyMs,
			P95LatencyMs:      stats.P95LatencyMs,
			RequestsPerMinute: requestsPerMinute,
			ErrorRatePerHour:  errorRatePerHour,
			LatencyPerHour:    latencyPerHour,
			TopPaths:          topPaths,
			TopIPs:            topIPs,
		})
	}
}

// ---- Route Auth Config handlers ----

func getRouteAuthConfig(db store.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		routeID := c.Param("routeId")
		cfg, err := db.GetRouteAuthConfig(routeID)
		if err != nil {
			c.JSON(http.StatusOK, dto.RouteAuthConfigResponseFromStore(store.RouteAuthConfig{RouteID: routeID}))
			return
		}
		c.JSON(http.StatusOK, dto.RouteAuthConfigResponseFromStore(*cfg))
	}
}

func updateRouteAuthConfig(db store.Store, routerMgr *router.Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		routeID := c.Param("routeId")
		var req dto.RouteAuthConfigUpdateRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			writeError(c, http.StatusBadRequest, "invalid_request", err.Error())
			return
		}

		svc := routeauthservice.NewService(db, routerMgr)
		cfg, err := svc.Update(routeID, routeauthservice.UpdateInput{
			ApiKeyEnabled:        req.ApiKeyEnabled,
			ApiKeyHeader:         req.ApiKeyHeader,
			GatewayEnabled:       req.GatewayEnabled,
			GatewayLoginMode:     req.GatewayLoginMode,
			Whitelist:            req.Whitelist,
			RateLimit:            req.RateLimit,
			Burst:                req.Burst,
			CORSAllowedOrigins:   req.CORSAllowedOrigins,
			CORSAllowedMethods:   req.CORSAllowedMethods,
			CORSAllowedHeaders:   req.CORSAllowedHeaders,
			CORSAllowCredentials: req.CORSAllowCredentials,
			CORSMaxAge:           req.CORSMaxAge,
		})
		if err != nil {
			writeServiceError(c, err)
			return
		}
		c.JSON(http.StatusOK, dto.RouteAuthConfigResponseFromStore(*cfg))
	}
}

func deleteRouteAuthConfig(db store.Store, routerMgr *router.Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		routeID := c.Param("routeId")
		svc := routeauthservice.NewService(db, routerMgr)
		if err := svc.Delete(routeID); err != nil {
			writeServiceError(c, err)
			return
		}
		c.JSON(http.StatusOK, httpresponse.Message{Message: "deleted"})
	}
}

// ---- API Key handlers ----

func listApiKeys(db store.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		routeID := c.Param("routeId")
		svc := apikeyservice.NewService(db)
		keys, err := svc.ListWithSecrets(routeID)
		if err != nil {
			writeServiceError(c, err)
			return
		}
		c.JSON(http.StatusOK, dto.ApiKeyListWithSecretsResponse(keys))
	}
}

func createApiKey(db store.Store, routerMgr *router.Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		routeID := c.Param("routeId")
		var req dto.ApiKeyCreateRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			writeError(c, http.StatusBadRequest, "invalid_request", err.Error())
			return
		}

		svc := apikeyservice.NewService(db)
		key, secret, err := svc.Create(routeID, req.Name, req.ExpiresAt)
		if err != nil {
			writeServiceError(c, err)
			return
		}
		routerMgr.Reload()

		c.JSON(http.StatusCreated, dto.ApiKeyCreateResponse{
			ApiKeyResponse: dto.ApiKeyResponseFromKey(*key),
			Secret:         secret,
		})
	}
}

func updateApiKey(db store.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		var req dto.ApiKeyUpdateRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			writeError(c, http.StatusBadRequest, "invalid_request", err.Error())
			return
		}

		svc := apikeyservice.NewService(db)
		if req.Name != nil {
			if err := svc.UpdateName(id, *req.Name); err != nil {
				writeServiceError(c, err)
				return
			}
		}
		c.JSON(http.StatusOK, httpresponse.Message{Message: "updated"})
	}
}

func rotateApiKey(db store.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		svc := apikeyservice.NewService(db)
		key, secret, err := svc.Rotate(id)
		if err != nil {
			writeServiceError(c, err)
			return
		}
		c.JSON(http.StatusOK, dto.ApiKeyCreateResponse{
			ApiKeyResponse: dto.ApiKeyResponseFromKey(*key),
			Secret:         secret,
		})
	}
}

func expireApiKey(db store.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		svc := apikeyservice.NewService(db)
		if err := svc.Expire(id); err != nil {
			writeServiceError(c, err)
			return
		}
		c.JSON(http.StatusOK, httpresponse.Message{Message: "expired"})
	}
}

func deleteApiKey(db store.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		svc := apikeyservice.NewService(db)
		if err := svc.Delete(id); err != nil {
			writeServiceError(c, err)
			return
		}
		c.JSON(http.StatusOK, httpresponse.Message{Message: "deleted"})
	}
}

func apiKeyServiceError(c *gin.Context, err error) bool {
	var target *apikeyservice.Error
	if !errors.As(err, &target) {
		return false
	}
	switch targetCode := apikeyservice.Code(err); targetCode {
	case apikeyservice.ErrCodeAPIKeyNotFound:
		writeError(c, http.StatusNotFound, targetCode, target.Error())
	case apikeyservice.ErrCodeRouteNotFound, apikeyservice.ErrCodeNameRequired:
		writeError(c, http.StatusBadRequest, targetCode, target.Error())
	default:
		writeError(c, http.StatusInternalServerError, targetCode, target.Error())
	}
	return true
}

func routeAuthServiceError(c *gin.Context, err error) bool {
	var target *routeauthservice.Error
	if !errors.As(err, &target) {
		return false
	}
	switch targetCode := routeauthservice.Code(err); targetCode {
	case routeauthservice.ErrCodeRouteAuthNotFound:
		writeError(c, http.StatusNotFound, targetCode, target.Error())
	case routeauthservice.ErrCodeRouteNotFound:
		writeError(c, http.StatusBadRequest, targetCode, target.Error())
	default:
		writeError(c, http.StatusInternalServerError, targetCode, target.Error())
	}
	return true
}
