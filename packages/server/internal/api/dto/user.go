package dto

import (
	"time"

	"github.com/pallyoung/auth-gate/packages/server/internal/store"
)

type Permissions struct {
	CanManageRoutes bool `json:"can_manage_routes"`
	CanManageAuth   bool `json:"can_manage_auth"`
	CanManageUsers  bool `json:"can_manage_users"`
	CanViewLogs     bool `json:"can_view_logs"`
}

type User struct {
	ID          string      `json:"id"`
	Username    string      `json:"username"`
	Role        string      `json:"role"`
	Enabled     bool        `json:"enabled,omitempty"`
	CreatedAt   *time.Time  `json:"created_at,omitempty"`
	UpdatedAt   *time.Time  `json:"updated_at,omitempty"`
	Permissions Permissions `json:"permissions,omitempty"`
}

type UserCreateRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
	Role     string `json:"role"`
}

type UserUpdateRequest struct {
	Username string `json:"username" binding:"required"`
	Role     string `json:"role"`
	Enabled  bool   `json:"enabled"`
}

func UserResponse(user store.User) User {
	createdAt := user.CreatedAt
	updatedAt := user.UpdatedAt
	return User{
		ID:        user.ID,
		Username:  user.Username,
		Role:      user.Role,
		Enabled:   user.Enabled,
		CreatedAt: &createdAt,
		UpdatedAt: &updatedAt,
	}
}

func UserListResponse(users []store.User) []User {
	result := make([]User, 0, len(users))
	for _, user := range users {
		result = append(result, UserResponse(user))
	}
	return result
}

func PermissionsResponse(permissions store.Permissions) Permissions {
	return Permissions{
		CanManageRoutes: permissions.CanManageRoutes,
		CanManageAuth:   permissions.CanManageAuth,
		CanManageUsers:  permissions.CanManageUsers,
		CanViewLogs:     permissions.CanViewLogs,
	}
}
