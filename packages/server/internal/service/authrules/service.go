package authrules

import (
	"database/sql"
	"errors"
	"strings"

	"github.com/pallyoung/auth-gate/packages/server/internal/service/runtime"
	"github.com/pallyoung/auth-gate/packages/server/internal/store"
)

const (
	ErrCodeAuthRuleNotFound         = "auth_rule_not_found"
	ErrCodeRouteNotFound            = "route_not_found"
	ErrCodeRouteIDRequired          = "route_id_required"
	ErrCodeInvalidAuthRuleType      = "invalid_auth_rule_type"
	ErrCodeMissingAPIKeySecret      = "missing_apikey_secret"
	ErrCodeMissingBearerSecret      = "missing_bearer_secret"
	ErrCodeMissingBasicCredentials  = "missing_basic_credentials"
	ErrCodeDuplicateRouteAuthRule   = "duplicate_route_auth_rule"
	ErrCodeAuthRuleStoreFailure     = "auth_rule_store_failure"
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

type AuthConfigInput struct {
	HeaderName string
	Secret     string
	Username   string
	Password   string
	LoginMode  string
}

type CreateInput struct {
	RouteID   string
	Type      string
	Config    AuthConfigInput
}

type UpdateInput = CreateInput

type Service struct {
	db       *store.SQLite
	reloader runtime.Reloader
}

func NewService(db *store.SQLite, reloader runtime.Reloader) *Service {
	return &Service{
		db:       db,
		reloader: reloader,
	}
}

func (s *Service) List() ([]store.AuthRule, error) {
	rules, err := s.db.ListAuthRules()
	if err != nil {
		return nil, newError(ErrCodeAuthRuleStoreFailure, "failed to list auth rules", err)
	}
	return rules, nil
}

func (s *Service) Get(id string) (*store.AuthRule, error) {
	rule, err := s.db.GetAuthRule(id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, newError(ErrCodeAuthRuleNotFound, "auth rule not found", err)
		}
		return nil, newError(ErrCodeAuthRuleStoreFailure, "failed to get auth rule", err)
	}
	return rule, nil
}

func (s *Service) Create(input CreateInput) (*store.AuthRule, error) {
	rule, err := s.build(input, nil)
	if err != nil {
		return nil, err
	}
	if err := s.db.CreateAuthRule(rule); err != nil {
		if isUniqueViolation(err) {
			return nil, newError(ErrCodeDuplicateRouteAuthRule, "route already has an auth rule", err)
		}
		return nil, newError(ErrCodeAuthRuleStoreFailure, "failed to create auth rule", err)
	}
	s.reload()
	return rule, nil
}

func (s *Service) Update(id string, input UpdateInput) (*store.AuthRule, error) {
	existing, err := s.Get(id)
	if err != nil {
		return nil, err
	}
	rule, err := s.build(input, existing)
	if err != nil {
		return nil, err
	}
	rule.ID = id
	if err := s.db.UpdateAuthRule(rule); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, newError(ErrCodeAuthRuleNotFound, "auth rule not found", err)
		}
		if isUniqueViolation(err) {
			return nil, newError(ErrCodeDuplicateRouteAuthRule, "route already has an auth rule", err)
		}
		return nil, newError(ErrCodeAuthRuleStoreFailure, "failed to update auth rule", err)
	}
	s.reload()
	return rule, nil
}

func (s *Service) Delete(id string) error {
	if _, err := s.Get(id); err != nil {
		return err
	}
	if err := s.db.DeleteAuthRule(id); err != nil {
		return newError(ErrCodeAuthRuleStoreFailure, "failed to delete auth rule", err)
	}
	s.reload()
	return nil
}

func (s *Service) build(input CreateInput, existing *store.AuthRule) (*store.AuthRule, error) {
	routeID := strings.TrimSpace(input.RouteID)
	if routeID == "" {
		return nil, newError(ErrCodeRouteIDRequired, "route_id required", nil)
	}
	if _, err := s.db.GetRoute(routeID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, newError(ErrCodeRouteNotFound, "route not found", err)
		}
		return nil, newError(ErrCodeAuthRuleStoreFailure, "failed to load route", err)
	}

	ruleType := strings.ToLower(strings.TrimSpace(input.Type))
	if ruleType == "" {
		ruleType = "none"
	}
	config := store.AuthConfig{
		HeaderName: strings.TrimSpace(input.Config.HeaderName),
		Secret:     strings.TrimSpace(input.Config.Secret),
		Username:   strings.TrimSpace(input.Config.Username),
		Password:   input.Config.Password,
		LoginMode:  strings.TrimSpace(input.Config.LoginMode),
	}

	if existing != nil {
		config = mergeProtectedAuthConfig(existing.Config, config, ruleType)
	}
	if ruleType == "gateway" && config.LoginMode == "" {
		config.LoginMode = "form"
	}
	if err := validate(ruleType, config); err != nil {
		return nil, err
	}

	return &store.AuthRule{
		RouteID:   routeID,
		Type:      ruleType,
		Config:    config,
	}, nil
}

func (s *Service) reload() {
	if s.reloader != nil {
		s.reloader.Reload()
	}
}

func validate(ruleType string, config store.AuthConfig) error {
	switch ruleType {
	case "none":
		return nil
	case "apikey":
		if config.Secret == "" {
			return newError(ErrCodeMissingAPIKeySecret, "apikey secret required", nil)
		}
		return nil
	case "bearer":
		if config.Secret == "" {
			return newError(ErrCodeMissingBearerSecret, "bearer secret required", nil)
		}
		return nil
	case "basic":
		if config.Username == "" || config.Password == "" {
			return newError(ErrCodeMissingBasicCredentials, "basic username and password required", nil)
		}
		return nil
	case "gateway":
		return nil
	default:
		return newError(ErrCodeInvalidAuthRuleType, "invalid auth rule type", nil)
	}
}

func mergeProtectedAuthConfig(existing, incoming store.AuthConfig, ruleType string) store.AuthConfig {
	merged := incoming

	switch ruleType {
	case "apikey", "bearer":
		if merged.Secret == "" {
			merged.Secret = existing.Secret
		}
	case "basic":
		if merged.Password == "" {
			merged.Password = existing.Password
		}
		if merged.Username == "" {
			merged.Username = existing.Username
		}
	case "gateway":
		if merged.LoginMode == "" {
			merged.LoginMode = existing.LoginMode
		}
	}

	return merged
}

func isUniqueViolation(err error) bool {
	return strings.Contains(strings.ToLower(err.Error()), "unique")
}
