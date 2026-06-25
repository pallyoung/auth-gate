package apikeys

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"strings"
	"time"

	"github.com/pallyoung/auth-gate/packages/server/internal/store"
)

const (
	ErrCodeAPIKeyNotFound    = "api_key_not_found"
	ErrCodeRouteNotFound     = "route_not_found"
	ErrCodeNameRequired      = "name_required"
	ErrCodeAPIKeyStoreFailure = "api_key_store_failure"
)

type Error struct {
	code    string
	message string
	cause   error
}

func (e *Error) Error() string   { return e.message }
func (e *Error) Unwrap() error   { return e.cause }
func Code(err error) string {
	var target *Error
	if errors.As(err, &target) {
		return target.code
	}
	return ""
}

func newError(code, message string, cause error) error {
	return &Error{code: code, message: message, cause: cause}
}

type Service struct {
	db store.Store
}

func NewService(db store.Store) *Service {
	return &Service{db: db}
}

func (s *Service) ListByRoute(routeID string) ([]store.ApiKey, error) {
	keys, err := s.db.ListApiKeysByRoute(routeID)
	if err != nil {
		return nil, newError(ErrCodeAPIKeyStoreFailure, "failed to list api keys", err)
	}
	return keys, nil
}

func (s *Service) Create(routeID, name string, expiresAt *time.Time) (*store.ApiKey, string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, "", newError(ErrCodeNameRequired, "api key name required", nil)
	}
	// Verify route exists
	if _, err := s.db.GetRoute(routeID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, "", newError(ErrCodeRouteNotFound, "route not found", err)
		}
		return nil, "", newError(ErrCodeAPIKeyStoreFailure, "failed to verify route", err)
	}

	secret, err := generateSecret()
	if err != nil {
		return nil, "", newError(ErrCodeAPIKeyStoreFailure, "failed to generate api key", err)
	}

	prefix := secret
	if len(prefix) > 8 {
		prefix = prefix[:8]
	}

	key := &store.ApiKey{
		RouteID:   routeID,
		Name:      name,
		KeyPrefix: prefix,
		Secret:    secret,
		ExpiresAt: expiresAt,
		Status:    "active",
	}

	if err := s.db.CreateApiKey(key); err != nil {
		return nil, "", newError(ErrCodeAPIKeyStoreFailure, "failed to create api key", err)
	}

	return key, secret, nil
}

func (s *Service) Rotate(id string) (*store.ApiKey, string, error) {
	key, err := s.db.GetApiKey(id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, "", newError(ErrCodeAPIKeyNotFound, "api key not found", err)
		}
		return nil, "", newError(ErrCodeAPIKeyStoreFailure, "failed to get api key", err)
	}

	secret, err := generateSecret()
	if err != nil {
		return nil, "", newError(ErrCodeAPIKeyStoreFailure, "failed to generate api key", err)
	}

	prefix := secret
	if len(prefix) > 8 {
		prefix = prefix[:8]
	}

	key.Secret = secret
	key.KeyPrefix = prefix
	key.Status = "active"

	if err := s.db.UpdateApiKey(key); err != nil {
		return nil, "", newError(ErrCodeAPIKeyStoreFailure, "failed to rotate api key", err)
	}

	return key, secret, nil
}

func (s *Service) Expire(id string) error {
	key, err := s.db.GetApiKey(id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return newError(ErrCodeAPIKeyNotFound, "api key not found", err)
		}
		return newError(ErrCodeAPIKeyStoreFailure, "failed to get api key", err)
	}

	key.Status = "revoked"
	if err := s.db.UpdateApiKey(key); err != nil {
		return newError(ErrCodeAPIKeyStoreFailure, "failed to expire api key", err)
	}
	return nil
}

func (s *Service) UpdateName(id, name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return newError(ErrCodeNameRequired, "api key name required", nil)
	}

	key, err := s.db.GetApiKey(id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return newError(ErrCodeAPIKeyNotFound, "api key not found", err)
		}
		return newError(ErrCodeAPIKeyStoreFailure, "failed to get api key", err)
	}

	key.Name = name
	if err := s.db.UpdateApiKey(key); err != nil {
		return newError(ErrCodeAPIKeyStoreFailure, "failed to update api key", err)
	}
	return nil
}

func (s *Service) Delete(id string) error {
	if err := s.db.DeleteApiKey(id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return newError(ErrCodeAPIKeyNotFound, "api key not found", err)
		}
		return newError(ErrCodeAPIKeyStoreFailure, "failed to delete api key", err)
	}
	return nil
}

func generateSecret() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return "ag_" + hex.EncodeToString(bytes), nil
}
