package dto

import "github.com/pallyoung/auth-gate/packages/server/internal/store"

type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type LoginResponse struct {
	Token       string      `json:"token"`
	User        User        `json:"user"`
	Permissions Permissions `json:"permissions"`
}

func SessionUserResponse(user store.User) User {
	return User{
		ID:       user.ID,
		Username: user.Username,
		Role:     user.Role,
	}
}

func LoginResponseFromStore(token string, user store.User, permissions store.Permissions) LoginResponse {
	return LoginResponse{
		Token:       token,
		User:        SessionUserResponse(user),
		Permissions: PermissionsResponse(permissions),
	}
}

func CurrentUserResponse(user store.User, permissions store.Permissions) User {
	return User{
		ID:          user.ID,
		Username:    user.Username,
		Role:        user.Role,
		Permissions: PermissionsResponse(permissions),
	}
}
