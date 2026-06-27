package session

import (
	"database/sql"
	"errors"
	"strings"

	"github.com/pallyoung/auth-gate/packages/server/internal/auth"
	"github.com/pallyoung/auth-gate/packages/server/internal/store"
)

const (
	ErrCodeInvalidCredentials = "invalid_credentials"
	ErrCodeUserDisabled       = "user_disabled"
	ErrCodeControlPlaneAccessDenied = "control_plane_access_denied"
	ErrCodeRouteAccessDenied  = "route_access_denied"
	ErrCodeRouteNotFound      = "route_not_found"
	ErrCodeTokenGeneration    = "token_generation_failed"
	ErrCodeSessionStoreFailure = "session_store_failure"
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

type Session struct {
	Token       string
	User        store.User
	Permissions store.Permissions
}

type RouteSession struct {
	Token string
	User  store.User
}

type Service struct {
	db store.Store
}

func NewService(db store.Store) *Service {
	return &Service{db: db}
}

func (s *Service) Login(username, password string) (*Session, error) {
	username = strings.TrimSpace(username)
	user, err := s.db.GetUserByUsername(username)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, newError(ErrCodeInvalidCredentials, "invalid credentials", err)
		}
		return nil, newError(ErrCodeSessionStoreFailure, "failed to load user", err)
	}
	if !user.Enabled {
		return nil, newError(ErrCodeUserDisabled, "user disabled", nil)
	}
	if !s.db.VerifyPassword(user, password) {
		return nil, newError(ErrCodeInvalidCredentials, "invalid credentials", nil)
	}
	if !store.CanAccessControlPlane(user.Role) {
		return nil, newError(ErrCodeControlPlaneAccessDenied, "control plane access denied", nil)
	}

	token, err := auth.GenerateControlPlaneToken(user.ID, user.Username, user.Role)
	if err != nil {
		return nil, newError(ErrCodeTokenGeneration, "failed to generate token", err)
	}

	return &Session{
		Token:       token,
		User:        *user,
		Permissions: store.GetPermissions(user.Role),
	}, nil
}

func (s *Service) LoginForRoute(routeID, username, password string) (*RouteSession, error) {
	username = strings.TrimSpace(username)
	if _, err := s.db.GetRoute(routeID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, newError(ErrCodeRouteNotFound, "route not found", err)
		}
		return nil, newError(ErrCodeSessionStoreFailure, "failed to load route", err)
	}

	user, err := s.db.GetUserByUsername(username)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, newError(ErrCodeInvalidCredentials, "invalid credentials", err)
		}
		return nil, newError(ErrCodeSessionStoreFailure, "failed to load user", err)
	}
	if !user.Enabled {
		return nil, newError(ErrCodeUserDisabled, "user disabled", nil)
	}
	if !s.db.VerifyPassword(user, password) {
		return nil, newError(ErrCodeInvalidCredentials, "invalid credentials", nil)
	}
	if !store.UserHasRouteAccess(user, routeID) {
		// Also check if any of the user's groups grant access to this route
		groups, _ := s.db.GetPermissionGroupsByIDs(user.GroupIDs)
		groupAllowed := false
		for _, g := range groups {
			for _, rid := range g.RouteIDs {
				if rid == routeID {
					groupAllowed = true
					break
				}
			}
			if groupAllowed {
				break
			}
		}
		if !groupAllowed {
			return nil, newError(ErrCodeRouteAccessDenied, "route access denied", nil)
		}
	}

	effectiveRouteIDs := buildEffectiveRouteIDs(s.db, user)
	effectivePaths := buildEffectivePaths(s.db, user)

	token, err := auth.GenerateRouteAccessToken(user.ID, user.Username, user.Role, effectiveRouteIDs, effectivePaths)
	if err != nil {
		return nil, newError(ErrCodeTokenGeneration, "failed to generate token", err)
	}

	return &RouteSession{
		Token: token,
		User:  *user,
	}, nil
}

// buildEffectiveRouteIDs merges user's personal RouteIDs with all assigned
// permission groups' RouteIDs into a deduplicated slice.
func buildEffectiveRouteIDs(db store.Store, user *store.User) []string {
	if user.Role == store.RoleAdmin {
		return user.RouteIDs
	}

	seen := make(map[string]struct{})
	var result []string
	for _, id := range user.RouteIDs {
		if _, ok := seen[id]; !ok {
			seen[id] = struct{}{}
			result = append(result, id)
		}
	}

	if len(user.GroupIDs) > 0 {
		groups, err := db.GetPermissionGroupsByIDs(user.GroupIDs)
		if err == nil {
			for _, g := range groups {
				for _, id := range g.RouteIDs {
					if _, ok := seen[id]; !ok {
						seen[id] = struct{}{}
						result = append(result, id)
					}
				}
			}
		}
	}

	return result
}

// buildEffectivePaths merges user's personal RoutePaths with all assigned
// permission groups' RoutePaths into a single map (union).
func buildEffectivePaths(db store.Store, user *store.User) map[string][]string {
	if user.Role == store.RoleAdmin {
		return nil
	}

	merged := make(map[string][]string)

	// Merge group paths first
	if len(user.GroupIDs) > 0 {
		groups, err := db.GetPermissionGroupsByIDs(user.GroupIDs)
		if err == nil {
			for _, g := range groups {
				for routeID, paths := range g.RoutePaths {
					merged[routeID] = append(merged[routeID], paths...)
				}
			}
		}
	}

	// Merge user's personal paths
	for routeID, paths := range user.RoutePaths {
		merged[routeID] = append(merged[routeID], paths...)
	}

	// Dedup each route's paths
	for routeID, paths := range merged {
		merged[routeID] = dedup(paths)
	}

	if len(merged) == 0 {
		return nil
	}
	return merged
}

func dedup(paths []string) []string {
	seen := make(map[string]struct{}, len(paths))
	result := make([]string, 0, len(paths))
	for _, p := range paths {
		if _, ok := seen[p]; !ok {
			seen[p] = struct{}{}
			result = append(result, p)
		}
	}
	return result
}
