package api

import (
	"net/http"

	"auth-gate/internal/auth"
	"auth-gate/internal/router"
	"auth-gate/internal/store"

	"github.com/gin-gonic/gin"
)

func RegisterHandlers(r *gin.RouterGroup, routerMgr *router.Manager, db *store.SQLite) {
	// Auth
	r.POST("/auth/logout", logoutHandler())
	r.GET("/auth/me", meHandler())

	// Routes (viewer, editor, admin)
	r.GET("/routes", listRoutes(db))
	r.GET("/routes/:id", getRoute(db))

	// Routes (editor, admin only)
	editorRoutes := r.Group("")
	editorRoutes.Use(auth.RequireRole("admin", "editor"))
	{
		editorRoutes.POST("/routes", createRoute(db, routerMgr))
		editorRoutes.PUT("/routes/:id", updateRoute(db, routerMgr))
		editorRoutes.DELETE("/routes/:id", deleteRoute(db, routerMgr))
	}

	// Auth Rules (viewer, editor, admin)
	r.GET("/auth-rules", listAuthRules(db))
	r.GET("/auth-rules/:id", getAuthRule(db))

	// Auth Rules (editor, admin only)
	editorAuth := r.Group("")
	editorAuth.Use(auth.RequireRole("admin", "editor"))
	{
		editorAuth.POST("/auth-rules", createAuthRule(db, routerMgr))
		editorAuth.PUT("/auth-rules/:id", updateAuthRule(db, routerMgr))
		editorAuth.DELETE("/auth-rules/:id", deleteAuthRule(db, routerMgr))
	}

	// Users (admin only)
	admin := r.Group("")
	admin.Use(auth.RequireRole("admin"))
	{
		admin.GET("/users", listUsers(db))
		admin.POST("/users", createUser(db))
		admin.PUT("/users/:id", updateUser(db))
		admin.DELETE("/users/:id", deleteUser(db))
	}

	// Config
	r.GET("/config/reload", reloadConfig(routerMgr))
}

// === Auth ===

func logoutHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "logged out"})
	}
}

func meHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		user := auth.GetCurrentUser(c)
		c.JSON(http.StatusOK, gin.H{
			"id":       user.UserID,
			"username": user.Username,
			"role":     user.Role,
		})
	}
}

// === Routes ===

func listRoutes(db *store.SQLite) gin.HandlerFunc {
	return func(c *gin.Context) {
		routes, err := db.ListRoutes()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, routes)
	}
}

func getRoute(db *store.SQLite) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		route, err := db.GetRoute(id)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "route not found"})
			return
		}
		c.JSON(http.StatusOK, route)
	}
}

func createRoute(db *store.SQLite, routerMgr *router.Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		var route store.Route
		if err := c.ShouldBindJSON(&route); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if route.PathPrefix == "" || route.Backend == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "path_prefix and backend required"})
			return
		}
		if err := db.CreateRoute(&route); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		routerMgr.Reload()
		c.JSON(http.StatusCreated, route)
	}
}

func updateRoute(db *store.SQLite, routerMgr *router.Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		var route store.Route
		if err := c.ShouldBindJSON(&route); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		route.ID = id
		if err := db.UpdateRoute(&route); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		routerMgr.Reload()
		c.JSON(http.StatusOK, route)
	}
}

func deleteRoute(db *store.SQLite, routerMgr *router.Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		if err := db.DeleteRoute(id); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		routerMgr.Reload()
		c.JSON(http.StatusOK, gin.H{"message": "deleted"})
	}
}

// === Auth Rules ===

func listAuthRules(db *store.SQLite) gin.HandlerFunc {
	return func(c *gin.Context) {
		rules, err := db.ListAuthRules()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, rules)
	}
}

func getAuthRule(db *store.SQLite) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		rule, err := db.GetAuthRule(id)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "auth rule not found"})
			return
		}
		c.JSON(http.StatusOK, rule)
	}
}

func createAuthRule(db *store.SQLite, routerMgr *router.Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		var rule store.AuthRule
		if err := c.ShouldBindJSON(&rule); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if rule.RouteID == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "route_id required"})
			return
		}
		if _, err := db.GetRoute(rule.RouteID); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "route not found"})
			return
		}
		if err := db.CreateAuthRule(&rule); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		routerMgr.Reload()
		c.JSON(http.StatusCreated, rule)
	}
}

func updateAuthRule(db *store.SQLite, routerMgr *router.Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		var rule store.AuthRule
		if err := c.ShouldBindJSON(&rule); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		rule.ID = id
		if err := db.UpdateAuthRule(&rule); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		routerMgr.Reload()
		c.JSON(http.StatusOK, rule)
	}
}

func deleteAuthRule(db *store.SQLite, routerMgr *router.Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		if err := db.DeleteAuthRule(id); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		routerMgr.Reload()
		c.JSON(http.StatusOK, gin.H{"message": "deleted"})
	}
}

// === Users ===

func listUsers(db *store.SQLite) gin.HandlerFunc {
	return func(c *gin.Context) {
		users, err := db.ListUsers()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, users)
	}
}

func createUser(db *store.SQLite) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			Username string `json:"username" binding:"required"`
			Password string `json:"password" binding:"required"`
			Role     string `json:"role"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		role := req.Role
		if role == "" {
			role = store.RoleViewer
		}

		hash, err := store.HashPassword(req.Password)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to hash password"})
			return
		}

		user := &store.User{
			Username:     req.Username,
			PasswordHash: hash,
			Role:         role,
			Enabled:      true,
		}

		if err := db.CreateUser(user); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "username already exists"})
			return
		}

		c.JSON(http.StatusCreated, user)
	}
}

func updateUser(db *store.SQLite) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		var req struct {
			Username string `json:"username" binding:"required"`
			Role     string `json:"role"`
			Enabled  bool   `json:"enabled"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		user, err := db.GetUserByID(id)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
			return
		}

		user.Username = req.Username
		user.Role = req.Role
		user.Enabled = req.Enabled

		if err := db.UpdateUser(user); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, user)
	}
}

func deleteUser(db *store.SQLite) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		if err := db.DeleteUser(id); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "deleted"})
	}
}

// === Config ===

func reloadConfig(routerMgr *router.Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		routerMgr.Reload()
		c.JSON(http.StatusOK, gin.H{"message": "reloaded"})
	}
}
