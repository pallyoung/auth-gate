package users

import (
	"database/sql"
	"errors"
	"strings"

	"github.com/pallyoung/auth-gate/packages/server/internal/store"
)

const (
	ErrCodeUserNotFound         = "user_not_found"
	ErrCodeInvalidUsername      = "invalid_username"
	ErrCodeInvalidRole          = "invalid_role"
	ErrCodeDuplicateUser        = "duplicate_user"
	ErrCodeDuplicateRouteAccess = "duplicate_route_access"
	ErrCodeRouteNotFound        = "route_not_found"
	ErrCodePasswordHashing      = "password_hash_failed"
	ErrCodeMissingPassword      = "missing_password"
	ErrCodeUserStoreFailure     = "user_store_failure"
)

type Error struct {
	code    string
	message string
	cause   error
}

func (e *Error) Error() string {
	return e.message
}

func (e *Error) Unwrap() error {
	return e.cause
}

func Code(err error) string {
	var target *Error
	if errors.As(err, &target) {
		return target.code
	}
	return ""
}

func newError(code, message string, cause error) error {
	return &Error{
		code:    code,
		message: message,
		cause:   cause,
	}
}

type CreateInput struct {
	Username   string
	Password   string
	Role       string
	Enabled    bool
	RouteIDs   []string
	GroupIDs   []string
	RoutePaths map[string][]string
}

type UpdateInput struct {
	Username   *string
	Password   string
	Role       *string
	Enabled    *bool
	RouteIDs   *[]string
	GroupIDs   *[]string
	RoutePaths *map[string][]string
}

type Service struct {
	db store.Store
}

func NewService(db store.Store) *Service {
	return &Service{db: db}
}

func (s *Service) List() ([]store.User, error) {
	users, err := s.db.ListUsers()
	if err != nil {
		return nil, newError(ErrCodeUserStoreFailure, "failed to list users", err)
	}
	return users, nil
}

func (s *Service) Get(id string) (*store.User, error) {
	user, err := s.db.GetUserByID(id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, newError(ErrCodeUserNotFound, "user not found", err)
		}
		return nil, newError(ErrCodeUserStoreFailure, "failed to get user", err)
	}
	return user, nil
}

func (s *Service) Create(input CreateInput) (*store.User, error) {
	username, err := normalizeUsername(input.Username)
	if err != nil {
		return nil, err
	}
	role, ok := normalizeRole(input.Role)
	if !ok {
		return nil, newError(ErrCodeInvalidRole, "invalid role", nil)
	}
	if strings.TrimSpace(input.Password) == "" {
		return nil, newError(ErrCodeMissingPassword, "password required", nil)
	}
	routeIDs, err := s.validateRouteAccess(role, input.RouteIDs)
	if err != nil {
		return nil, err
	}

	hash, err := store.HashPassword(input.Password)
	if err != nil {
		return nil, newError(ErrCodePasswordHashing, "failed to hash password", err)
	}

	user := &store.User{
		Username:     username,
		PasswordHash: hash,
		Role:         role,
		Enabled:      input.Enabled,
		RouteIDs:     routeIDs,
		GroupIDs:     input.GroupIDs,
		RoutePaths:   input.RoutePaths,
	}
	if err := s.db.CreateUser(user); err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "unique") {
			return nil, newError(ErrCodeDuplicateUser, "username already exists", err)
		}
		return nil, newError(ErrCodeUserStoreFailure, "failed to create user", err)
	}
	return user, nil
}

func (s *Service) Update(id string, input UpdateInput) (*store.User, error) {
	user, err := s.Get(id)
	if err != nil {
		return nil, err
	}

	username := user.Username
	if input.Username != nil {
		normalizedUsername, err := normalizeUsername(*input.Username)
		if err != nil {
			return nil, err
		}
		username = normalizedUsername
	}

	role := user.Role
	if input.Role != nil {
		normalizedRole, ok := normalizeRole(*input.Role)
		if !ok {
			return nil, newError(ErrCodeInvalidRole, "invalid role", nil)
		}
		role = normalizedRole
	}

	routeIDs := user.RouteIDs
	if input.RouteIDs != nil {
		normalizedRouteIDs, err := s.validateRouteAccess(role, *input.RouteIDs)
		if err != nil {
			return nil, err
		}
		routeIDs = normalizedRouteIDs
	}

	if input.Role != nil && input.RouteIDs == nil {
		normalizedRouteIDs, err := s.validateRouteAccess(role, routeIDs)
		if err != nil {
			return nil, err
		}
		routeIDs = normalizedRouteIDs
	}

	user.Username = username
	user.Role = role
	if input.Enabled != nil {
		user.Enabled = *input.Enabled
	}
	user.RouteIDs = routeIDs
	if input.GroupIDs != nil {
		user.GroupIDs = *input.GroupIDs
	}
	if input.RoutePaths != nil {
		user.RoutePaths = *input.RoutePaths
	}

	if strings.TrimSpace(input.Password) != "" {
		hash, err := store.HashPassword(input.Password)
		if err != nil {
			return nil, newError(ErrCodePasswordHashing, "failed to hash password", err)
		}
		user.PasswordHash = hash
	}

	if err := s.db.UpdateUser(user); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, newError(ErrCodeUserNotFound, "user not found", err)
		}
		if strings.Contains(strings.ToLower(err.Error()), "unique") {
			return nil, newError(ErrCodeDuplicateUser, "username already exists", err)
		}
		return nil, newError(ErrCodeUserStoreFailure, "failed to update user", err)
	}
	return user, nil
}

func (s *Service) Delete(id string) error {
	if _, err := s.Get(id); err != nil {
		return err
	}
	if err := s.db.DeleteUser(id); err != nil {
		return newError(ErrCodeUserStoreFailure, "failed to delete user", err)
	}
	return nil
}

func normalizeRole(role string) (string, bool) {
	role = strings.ToLower(strings.TrimSpace(role))
	if role == "" {
		role = store.RoleMember
	}
	switch role {
	case store.RoleAdmin, store.RoleEditor, store.RoleViewer, store.RoleMember:
		return role, true
	default:
		return "", false
	}
}

func normalizeUsername(username string) (string, error) {
	username = strings.TrimSpace(username)
	if username == "" {
		return "", newError(ErrCodeInvalidUsername, "username required", nil)
	}
	return username, nil
}

func (s *Service) validateRouteAccess(role string, routeIDs []string) ([]string, error) {
	normalized := make([]string, 0, len(routeIDs))
	seen := make(map[string]struct{}, len(routeIDs))

	for _, routeID := range routeIDs {
		routeID = strings.TrimSpace(routeID)
		if routeID == "" {
			continue
		}
		if _, exists := seen[routeID]; exists {
			return nil, newError(ErrCodeDuplicateRouteAccess, "duplicate route assignment", nil)
		}
		seen[routeID] = struct{}{}

		if _, err := s.db.GetRoute(routeID); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return nil, newError(ErrCodeRouteNotFound, "route not found", err)
			}
			return nil, newError(ErrCodeUserStoreFailure, "failed to validate route access", err)
		}

		normalized = append(normalized, routeID)
	}

	if role == store.RoleAdmin || role == store.RoleEditor {
		return normalized, nil
	}
	return normalized, nil
}
