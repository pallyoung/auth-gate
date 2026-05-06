package users

import (
	"database/sql"
	"errors"
	"strings"

	"github.com/pallyoung/auth-gate/packages/server/internal/store"
)

const (
	ErrCodeUserNotFound    = "user_not_found"
	ErrCodeInvalidRole     = "invalid_role"
	ErrCodeDuplicateUser   = "duplicate_user"
	ErrCodePasswordHashing = "password_hash_failed"
	ErrCodeUserStoreFailure = "user_store_failure"
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
	Username string
	Password string
	Role     string
}

type UpdateInput struct {
	Username string
	Role     string
	Enabled  bool
}

type Service struct {
	db *store.SQLite
}

func NewService(db *store.SQLite) *Service {
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
	role, ok := normalizeRole(input.Role)
	if !ok {
		return nil, newError(ErrCodeInvalidRole, "invalid role", nil)
	}

	hash, err := store.HashPassword(input.Password)
	if err != nil {
		return nil, newError(ErrCodePasswordHashing, "failed to hash password", err)
	}

	user := &store.User{
		Username:     strings.TrimSpace(input.Username),
		PasswordHash: hash,
		Role:         role,
		Enabled:      true,
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

	role, ok := normalizeRole(input.Role)
	if !ok {
		return nil, newError(ErrCodeInvalidRole, "invalid role", nil)
	}

	user.Username = strings.TrimSpace(input.Username)
	user.Role = role
	user.Enabled = input.Enabled

	if err := s.db.UpdateUser(user); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, newError(ErrCodeUserNotFound, "user not found", err)
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
		role = store.RoleViewer
	}
	switch role {
	case store.RoleAdmin, store.RoleEditor, store.RoleViewer:
		return role, true
	default:
		return "", false
	}
}
