package session

import (
	"database/sql"
	"errors"

	"github.com/pallyoung/auth-gate/packages/server/internal/auth"
	"github.com/pallyoung/auth-gate/packages/server/internal/store"
)

const (
	ErrCodeInvalidCredentials = "invalid_credentials"
	ErrCodeUserDisabled       = "user_disabled"
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

type Service struct {
	db *store.SQLite
}

func NewService(db *store.SQLite) *Service {
	return &Service{db: db}
}

func (s *Service) Login(username, password string) (*Session, error) {
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

	token, err := auth.GenerateToken(user.ID, user.Username, user.Role)
	if err != nil {
		return nil, newError(ErrCodeTokenGeneration, "failed to generate token", err)
	}

	return &Session{
		Token:       token,
		User:        *user,
		Permissions: store.GetPermissions(user.Role),
	}, nil
}
