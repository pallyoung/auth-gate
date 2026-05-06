package dto

import (
	"time"

	"github.com/pallyoung/auth-gate/packages/server/internal/store"
)

type AuthRuleConfig struct {
	HeaderName string `json:"header_name,omitempty"`
	Username   string `json:"username,omitempty"`
}

type AuthRuleConfigWriteRequest struct {
	HeaderName string `json:"header_name,omitempty"`
	Secret     string `json:"secret,omitempty"`
	Username   string `json:"username,omitempty"`
	Password   string `json:"password,omitempty"`
}

type AuthRule struct {
	ID        string         `json:"id"`
	RouteID   string         `json:"route_id"`
	Type      string         `json:"type"`
	Config    AuthRuleConfig `json:"config"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
}

type AuthRuleWriteRequest struct {
	RouteID   string                     `json:"route_id" binding:"required"`
	Type      string                     `json:"type"`
	Config    AuthRuleConfigWriteRequest `json:"config"`
}

func AuthRuleResponse(rule store.AuthRule) AuthRule {
	return AuthRule{
		ID:      rule.ID,
		RouteID: rule.RouteID,
		Type:    rule.Type,
		Config: AuthRuleConfig{
			HeaderName: rule.Config.HeaderName,
			Username:   rule.Config.Username,
		},
		CreatedAt: rule.CreatedAt,
		UpdatedAt: rule.UpdatedAt,
	}
}

func AuthRuleListResponse(rules []store.AuthRule) []AuthRule {
	result := make([]AuthRule, 0, len(rules))
	for _, rule := range rules {
		result = append(result, AuthRuleResponse(rule))
	}
	return result
}
