package store

import (
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
)

type User struct {
	ID           string    `json:"id"`
	Username     string    `json:"username"`
	PasswordHash string    `json:"password_hash"`
	Role         string    `json:"role"`
	Enabled      bool      `json:"enabled"`
	RouteIDs     []string  `json:"route_ids,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

const (
	RoleAdmin  = "admin"
	RoleEditor = "editor"
	RoleViewer = "viewer"
	RoleMember = "member"
)

type Permissions struct {
	CanManageRoutes bool
	CanManageAuth   bool
	CanManageUsers  bool
	CanViewLogs     bool
	CanManageHosts  bool
}

func GetPermissions(role string) Permissions {
	role = normalizeStoredUserRole(role)
	switch role {
	case RoleAdmin:
		return Permissions{CanManageRoutes: true, CanManageAuth: true, CanManageUsers: true, CanViewLogs: true, CanManageHosts: true}
	case RoleEditor:
		return Permissions{CanManageRoutes: true, CanManageAuth: true, CanManageUsers: false, CanViewLogs: true}
	default:
		return Permissions{CanManageRoutes: false, CanManageAuth: false, CanManageUsers: false, CanViewLogs: false}
	}
}

func CanAccessControlPlane(role string) bool {
	role = normalizeStoredUserRole(role)
	switch role {
	case RoleAdmin, RoleEditor, RoleViewer:
		return true
	default:
		return false
	}
}

func UserHasRouteAccess(user *User, routeID string) bool {
	if user == nil || !user.Enabled {
		return false
	}
	// 管理员默认拥有所有项目的访问权限
	if user.Role == RoleAdmin {
		return true
	}
	for _, allowedRouteID := range user.RouteIDs {
		if allowedRouteID == routeID {
			return true
		}
	}
	return false
}

func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(bytes), err
}

func normalizeStoredUserRole(role string) string {
	return strings.ToLower(strings.TrimSpace(role))
}
