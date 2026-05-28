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

type Features struct {
	Certificates bool `json:"certificates"`
}

type User struct {
	ID          string       `json:"id"`
	Username    string       `json:"username"`
	Role        string       `json:"role"`
	Enabled     *bool        `json:"enabled,omitempty"`
	RouteIDs    []string     `json:"route_ids,omitempty"`
	CreatedAt   *time.Time   `json:"created_at,omitempty"`
	UpdatedAt   *time.Time   `json:"updated_at,omitempty"`
	Permissions *Permissions `json:"permissions,omitempty"`
	Features    *Features    `json:"features,omitempty"`
}

type UserCreateRequest struct {
	Username string   `json:"username" binding:"required"`
	Password string   `json:"password" binding:"required"`
	Role     string   `json:"role"`
	Enabled  *bool    `json:"enabled,omitempty"`
	RouteIDs []string `json:"route_ids,omitempty"`
}

type UserUpdateRequest struct {
	Username *string   `json:"username,omitempty"`
	Password string    `json:"password,omitempty"`
	Role     *string   `json:"role,omitempty"`
	Enabled  *bool     `json:"enabled,omitempty"`
	RouteIDs *[]string `json:"route_ids,omitempty"`
}

func UserResponse(user store.User) User {
	createdAt := user.CreatedAt
	updatedAt := user.UpdatedAt
	enabled := user.Enabled
	return User{
		ID:        user.ID,
		Username:  user.Username,
		Role:      user.Role,
		Enabled:   &enabled,
		RouteIDs:  user.RouteIDs,
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

func PermissionsResponse(permissions store.Permissions) *Permissions {
	return &Permissions{
		CanManageRoutes: permissions.CanManageRoutes,
		CanManageAuth:   permissions.CanManageAuth,
		CanManageUsers:  permissions.CanManageUsers,
		CanViewLogs:     permissions.CanViewLogs,
	}
}

func FeaturesResponse(certificatesEnabled bool) *Features {
	return &Features{
		Certificates: certificatesEnabled,
	}
}
