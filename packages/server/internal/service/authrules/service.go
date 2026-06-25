package authrules

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"strings"

	"github.com/pallyoung/auth-gate/packages/server/internal/service/runtime"
	"github.com/pallyoung/auth-gate/packages/server/internal/store"
)

const (
	ErrCodeAuthRuleNotFound        = "auth_rule_not_found"
	ErrCodeRouteNotFound           = "route_not_found"
	ErrCodeRouteIDRequired         = "route_id_required"
	ErrCodeInvalidAuthRuleType     = "invalid_auth_rule_type"
	ErrCodeMissingAPIKeySecret     = "missing_apikey_secret"
	ErrCodeMissingBasicCredentials = "missing_basic_credentials"
	ErrCodeAuthRuleStoreFailure    = "auth_rule_store_failure"
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

type UpdateAuthConfigInput struct {
	HeaderName *string
	Secret     *string
	Username   *string
	Password   *string
	LoginMode  *string
}

type CreateInput struct {
	RouteID              string
	Type                 string
	Config               AuthConfigInput
	Whitelist            []string
	RateLimit            int
	Burst                int
	CORSAllowedOrigins   string
	CORSAllowedMethods   string
	CORSAllowedHeaders   string
	CORSAllowCredentials bool
	CORSMaxAge           int
}

type UpdateInput struct {
	RouteID              *string
	Type                 *string
	Config               UpdateAuthConfigInput
	Whitelist            *[]string
	RateLimit            *int
	Burst                *int
	CORSAllowedOrigins   *string
	CORSAllowedMethods   *string
	CORSAllowedHeaders   *string
	CORSAllowCredentials *bool
	CORSMaxAge           *int
}

type Service struct {
	db       store.Store
	reloader runtime.Reloader
}

func NewService(db store.Store, reloader runtime.Reloader) *Service {
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
	for i := range rules {
		rules[i] = normalizeStoredAuthRule(rules[i])
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
	normalized := normalizeStoredAuthRule(*rule)
	return &normalized, nil
}

func (s *Service) Create(input CreateInput) (*store.AuthRule, error) {
	rule, err := s.buildCreate(input)
	if err != nil {
		return nil, err
	}
	if err := s.db.CreateAuthRule(rule); err != nil {
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
	rule, err := s.buildUpdate(input, existing)
	if err != nil {
		return nil, err
	}
	rule.ID = id
	if err := s.db.UpdateAuthRule(rule); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, newError(ErrCodeAuthRuleNotFound, "auth rule not found", err)
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

func (s *Service) buildCreate(input CreateInput) (*store.AuthRule, error) {
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
	if ruleType == "gateway" && config.LoginMode == "" {
		config.LoginMode = "form"
	}
	// apikey 类型自动生成密钥，忽略用户传入值
	if ruleType == "apikey" {
		key, err := generateAPIKey()
		if err != nil {
			return nil, newError(ErrCodeAuthRuleStoreFailure, "failed to generate api key", err)
		}
		config.Secret = key
	}
	if err := validate(ruleType, config); err != nil {
		return nil, err
	}

	return &store.AuthRule{
		RouteID:              routeID,
		Type:                 ruleType,
		Config:               config,
		Whitelist:            normalizeStringSlice(input.Whitelist),
		RateLimit:            input.RateLimit,
		Burst:                input.Burst,
		CORSAllowedOrigins:   normalizeCommaSeparated(input.CORSAllowedOrigins),
		CORSAllowedMethods:   normalizeCommaSeparated(input.CORSAllowedMethods),
		CORSAllowedHeaders:   normalizeCommaSeparated(input.CORSAllowedHeaders),
		CORSAllowCredentials: input.CORSAllowCredentials,
		CORSMaxAge:           input.CORSMaxAge,
	}, nil
}

func (s *Service) buildUpdate(input UpdateInput, existing *store.AuthRule) (*store.AuthRule, error) {
	routeID := existing.RouteID
	if input.RouteID != nil {
		routeID = strings.TrimSpace(*input.RouteID)
	}
	if routeID == "" {
		return nil, newError(ErrCodeRouteIDRequired, "route_id required", nil)
	}
	if _, err := s.db.GetRoute(routeID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, newError(ErrCodeRouteNotFound, "route not found", err)
		}
		return nil, newError(ErrCodeAuthRuleStoreFailure, "failed to load route", err)
	}

	ruleType := existing.Type
	if input.Type != nil {
		ruleType = strings.ToLower(strings.TrimSpace(*input.Type))
		if ruleType == "" {
			ruleType = "none"
		}
	}
	sameType := ruleType == existing.Type

	config := store.AuthConfig{}
	if sameType {
		config = existing.Config
	}

	if input.Config.HeaderName != nil {
		config.HeaderName = strings.TrimSpace(*input.Config.HeaderName)
	}
	// apikey 的密钥由后端自动生成，忽略用户传入值
	if input.Config.Secret != nil && ruleType != "apikey" {
		config.Secret = strings.TrimSpace(*input.Config.Secret)
	}
	if input.Config.Username != nil {
		config.Username = strings.TrimSpace(*input.Config.Username)
	}
	if input.Config.Password != nil {
		if strings.TrimSpace(*input.Config.Password) != "" {
			config.Password = *input.Config.Password
		} else if !sameType {
			config.Password = ""
		}
	}
	if input.Config.LoginMode != nil {
		config.LoginMode = strings.TrimSpace(*input.Config.LoginMode)
	}

	whitelist := existing.Whitelist
	if input.Whitelist != nil {
		whitelist = normalizeStringSlice(*input.Whitelist)
	}

	rateLimit := existing.RateLimit
	if input.RateLimit != nil {
		rateLimit = *input.RateLimit
	}

	burst := existing.Burst
	if input.Burst != nil {
		burst = *input.Burst
	}

	corsAllowedOrigins := existing.CORSAllowedOrigins
	if input.CORSAllowedOrigins != nil {
		corsAllowedOrigins = normalizeCommaSeparated(*input.CORSAllowedOrigins)
	}

	corsAllowedMethods := existing.CORSAllowedMethods
	if input.CORSAllowedMethods != nil {
		corsAllowedMethods = normalizeCommaSeparated(*input.CORSAllowedMethods)
	}

	corsAllowedHeaders := existing.CORSAllowedHeaders
	if input.CORSAllowedHeaders != nil {
		corsAllowedHeaders = normalizeCommaSeparated(*input.CORSAllowedHeaders)
	}

	corsAllowCredentials := existing.CORSAllowCredentials
	if input.CORSAllowCredentials != nil {
		corsAllowCredentials = *input.CORSAllowCredentials
	}

	corsMaxAge := existing.CORSMaxAge
	if input.CORSMaxAge != nil {
		corsMaxAge = *input.CORSMaxAge
	}

	if ruleType == "gateway" && config.LoginMode == "" {
		config.LoginMode = "form"
	}
	// 切换到 apikey 类型时自动生成密钥
	if ruleType == "apikey" && config.Secret == "" {
		key, err := generateAPIKey()
		if err != nil {
			return nil, newError(ErrCodeAuthRuleStoreFailure, "failed to generate api key", err)
		}
		config.Secret = key
	}
	if err := validate(ruleType, config); err != nil {
		return nil, err
	}

	return &store.AuthRule{
		RouteID:              routeID,
		Type:                 ruleType,
		Config:               config,
		Whitelist:            whitelist,
		RateLimit:            rateLimit,
		Burst:                burst,
		CORSAllowedOrigins:   corsAllowedOrigins,
		CORSAllowedMethods:   corsAllowedMethods,
		CORSAllowedHeaders:   corsAllowedHeaders,
		CORSAllowCredentials: corsAllowCredentials,
		CORSMaxAge:           corsMaxAge,
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
	case "basic":
		if config.Username == "" || strings.TrimSpace(config.Password) == "" {
			return newError(ErrCodeMissingBasicCredentials, "basic username and password required", nil)
		}
		return nil
	case "gateway":
		return nil
	default:
		return newError(ErrCodeInvalidAuthRuleType, "invalid auth rule type", nil)
	}
}

func normalizeStoredAuthRule(rule store.AuthRule) store.AuthRule {
	rule.Type = strings.ToLower(strings.TrimSpace(rule.Type))
	rule.Whitelist = normalizeStringSlice(rule.Whitelist)
	rule.CORSAllowedOrigins = normalizeCommaSeparated(rule.CORSAllowedOrigins)
	rule.CORSAllowedMethods = normalizeCommaSeparated(rule.CORSAllowedMethods)
	rule.CORSAllowedHeaders = normalizeCommaSeparated(rule.CORSAllowedHeaders)
	return rule
}

func normalizeStringSlice(values []string) []string {
	if len(values) == 0 {
		return nil
	}

	normalized := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		normalized = append(normalized, trimmed)
	}
	if len(normalized) == 0 {
		return nil
	}
	return normalized
}

func normalizeCommaSeparated(value string) string {
	parts := strings.Split(value, ",")
	normalized := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed == "" {
			continue
		}
		normalized = append(normalized, trimmed)
	}
	return strings.Join(normalized, ",")
}

func generateAPIKey() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}
