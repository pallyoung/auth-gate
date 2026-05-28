package admin

import (
	"context"
	"errors"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/pallyoung/auth-gate/packages/server/internal/api/dto"
	"github.com/pallyoung/auth-gate/packages/server/internal/auth"
	httpresponse "github.com/pallyoung/auth-gate/packages/server/internal/http/response"
	"github.com/pallyoung/auth-gate/packages/server/internal/router"
	authrulesservice "github.com/pallyoung/auth-gate/packages/server/internal/service/authrules"
	certservice "github.com/pallyoung/auth-gate/packages/server/internal/service/certificate"
	routesservice "github.com/pallyoung/auth-gate/packages/server/internal/service/routes"
	sessionservice "github.com/pallyoung/auth-gate/packages/server/internal/service/session"
	usersservice "github.com/pallyoung/auth-gate/packages/server/internal/service/users"
	"github.com/pallyoung/auth-gate/packages/server/internal/store"
)

func RegisterRoutes(group *gin.RouterGroup, routerMgr *router.Manager, db *store.SQLite, certSvc *certservice.Service) {
	group.Use(requestLogger())

	sessionSvc := sessionservice.NewService(db)
	routeSvc := routesservice.NewService(db, routerMgr)
	authRuleSvc := authrulesservice.NewService(db, routerMgr)
	userSvc := usersservice.NewService(db)

	group.POST("/auth/logout", logoutHandler())
	group.GET("/auth/me", meHandler(db, certSvc))

	group.GET("/routes", listRoutes(routeSvc))
	group.GET("/routes/:id", getRoute(routeSvc))
	group.GET("/auth-rules", listAuthRules(authRuleSvc))
	group.GET("/auth-rules/:id", getAuthRule(authRuleSvc))

	// Certificate endpoints
	if certSvc != nil {
		group.GET("/certificates", listCertificates(certSvc))
		group.GET("/certificates/:id", getCertificate(certSvc))

		editor := group.Group("")
		editor.Use(auth.RequireRole(store.RoleAdmin, store.RoleEditor))
		{
			editor.POST("/certificates", createCertificate(certSvc))
			editor.DELETE("/certificates/:id", deleteCertificate(certSvc))
			editor.POST("/certificates/:id/renew", renewCertificate(certSvc))
		}
	}

	editor := group.Group("")
	editor.Use(auth.RequireRole(store.RoleAdmin, store.RoleEditor))
	{
		editor.POST("/routes", createRoute(routeSvc))
		editor.PUT("/routes/:id", updateRoute(routeSvc))
		editor.DELETE("/routes/:id", deleteRoute(routeSvc))

		editor.POST("/auth-rules", createAuthRule(authRuleSvc))
		editor.PUT("/auth-rules/:id", updateAuthRule(authRuleSvc))
		editor.DELETE("/auth-rules/:id", deleteAuthRule(authRuleSvc))

		editor.POST("/config/reload", reloadConfig(routerMgr))
	}

	adminOnly := group.Group("")
	adminOnly.Use(auth.RequireRole(store.RoleAdmin))
	{
		adminOnly.GET("/users", listUsers(userSvc))
		adminOnly.POST("/users", createUser(userSvc))
		adminOnly.PUT("/users/:id", updateUser(userSvc))
		adminOnly.DELETE("/users/:id", deleteUser(userSvc))
		adminOnly.GET("/metrics", metricsHandler())
	}

	_ = sessionSvc
}

func LoginRoute(db *store.SQLite, certSvc *certservice.Service) gin.HandlerFunc {
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

func logoutHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, httpresponse.Message{Message: "logged out"})
	}
}

func meHandler(db *store.SQLite, certSvc *certservice.Service) gin.HandlerFunc {
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
			TLSCert:       req.TLSCert,
			TLSKey:        req.TLSKey,
			TLSEnabled:    req.TLSEnabled,
			TimeoutMs:     req.TimeoutMs,
			RetryAttempts: req.RetryAttempts,
			Backends:      req.Backends,
			PathMatchMode: req.PathMatchMode,
			RewriteTarget: req.RewriteTarget,
			RedirectCode:  req.RedirectCode,
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
			TLSCert:       req.TLSCert,
			TLSKey:        req.TLSKey,
			TLSEnabled:    req.TLSEnabled,
			TimeoutMs:     req.TimeoutMs,
			RetryAttempts: req.RetryAttempts,
			Backends:      req.Backends,
			PathMatchMode: req.PathMatchMode,
			RewriteTarget: req.RewriteTarget,
			RedirectCode:  req.RedirectCode,
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

func listCertificates(certSvc *certservice.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		certs, err := certSvc.List()
		if err != nil {
			writeServiceError(c, err)
			return
		}
		c.JSON(http.StatusOK, dto.CertificateListResponseFromStore(certs))
	}
}

func getCertificate(certSvc *certservice.Service) gin.HandlerFunc {
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

func createCertificate(certSvc *certservice.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req dto.CertificateWriteRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			writeError(c, http.StatusBadRequest, "invalid_request", err.Error())
			return
		}

		// Parse DNS provider config
		dnsConfig, err := certservice.ParseDNSProviderConfig(req.DNSProvider, req.ProviderConfig)
		if err != nil {
			writeError(c, http.StatusBadRequest, "invalid_provider", err.Error())
			return
		}

		cert, err := certSvc.Provision(context.Background(), certservice.ProvisionInput{
			Name:        req.Name,
			Domain:      req.Domain,
			DNSProvider: dnsConfig,
		})
		if err != nil {
			writeServiceError(c, err)
			return
		}
		c.JSON(http.StatusCreated, dto.CertificateResponseFromStore(*cert))
	}
}

func deleteCertificate(certSvc *certservice.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		if err := certSvc.Delete(c.Param("id")); err != nil {
			writeServiceError(c, err)
			return
		}
		c.JSON(http.StatusOK, httpresponse.Message{Message: "deleted"})
	}
}

func renewCertificate(certSvc *certservice.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		if err := certSvc.Renew(c.Param("id")); err != nil {
			writeServiceError(c, err)
			return
		}
		c.JSON(http.StatusOK, httpresponse.Message{Message: "renewal started"})
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
		certservice.ErrCodeInvalidProvider, certservice.ErrCodeCertNotActive:
		writeError(c, http.StatusBadRequest, targetCode, message)
	case certservice.ErrCodeDNSProvider:
		writeError(c, http.StatusBadRequest, targetCode, message)
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
	case routesservice.ErrCodeMissingRouteFields, routesservice.ErrCodeInvalidRoutePathPrefix, routesservice.ErrCodeInvalidRoutePathMatchMode, routesservice.ErrCodeInvalidRoutePathRegex, routesservice.ErrCodeReservedRoutePathPrefix, routesservice.ErrCodeInvalidRouteHost, routesservice.ErrCodeInvalidRouteBackend, routesservice.ErrCodeInvalidRouteBackendWeight, routesservice.ErrCodeInvalidRouteRedirectCode:
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
		authrulesservice.ErrCodeMissingAPIKeySecret,
		authrulesservice.ErrCodeMissingBearerSecret,
		authrulesservice.ErrCodeMissingBasicCredentials,
		authrulesservice.ErrCodeDuplicateRouteAuthRule:
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
