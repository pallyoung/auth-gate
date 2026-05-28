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
	db *store.SQLite
}

func NewService(db *store.SQLite) *Service {
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
		return nil, newError(ErrCodeRouteAccessDenied, "route access denied", nil)
	}

	token, err := auth.GenerateRouteAccessToken(user.ID, user.Username, user.Role, user.RouteIDs)
	if err != nil {
		return nil, newError(ErrCodeTokenGeneration, "failed to generate token", err)
	}

	return &RouteSession{
		Token: token,
		User:  *user,
	}, nil
}
