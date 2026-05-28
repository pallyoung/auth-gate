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

func SessionUserResponse(user store.User, certificatesEnabled bool) User {
	return User{
		ID:       user.ID,
		Username: user.Username,
		Role:     user.Role,
		RouteIDs: user.RouteIDs,
		Features: FeaturesResponse(certificatesEnabled),
	}
}

func LoginResponseFromStore(token string, user store.User, permissions store.Permissions, certificatesEnabled bool) LoginResponse {
	return LoginResponse{
		Token:       token,
		User:        SessionUserResponse(user, certificatesEnabled),
		Permissions: *PermissionsResponse(permissions),
	}
}

func CurrentUserResponse(user store.User, permissions store.Permissions, certificatesEnabled bool) User {
	return User{
		ID:          user.ID,
		Username:    user.Username,
		Role:        user.Role,
		RouteIDs:    user.RouteIDs,
		Permissions: PermissionsResponse(permissions),
		Features:    FeaturesResponse(certificatesEnabled),
	}
}
