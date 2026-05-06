package api

import (
	"net/http"

	"auth-gate/internal/auth"
	"auth-gate/internal/store"

	"github.com/gin-gonic/gin"
)

type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type LoginResponse struct {
	Token       string             `json:"token"`
	User        UserResponse       `json:"user"`
	Permissions store.Permissions `json:"permissions"`
}

type UserResponse struct {
	ID       string `json:"id"`
	Username string `json:"username"`
	Role     string `json:"role"`
}

func LoginHandler(db *store.SQLite) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req LoginRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
			return
		}

		user, err := db.GetUserByUsername(req.Username)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
			return
		}

		if !user.Enabled {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "user disabled"})
			return
		}

		if !db.VerifyPassword(user, req.Password) {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
			return
		}

		token, err := auth.GenerateToken(user.ID, user.Username, user.Role)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate token"})
			return
		}

		c.JSON(http.StatusOK, LoginResponse{
			Token:       token,
			User:        UserResponse{ID: user.ID, Username: user.Username, Role: user.Role},
			Permissions: store.GetPermissions(user.Role),
		})
	}
}

func LogoutHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "logged out"})
	}
}

func MeHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		user := auth.GetCurrentUser(c)
		c.JSON(http.StatusOK, UserResponse{
			ID:       user.UserID,
			Username: user.Username,
			Role:     user.Role,
		})
	}
}
