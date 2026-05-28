package store

import (
	"database/sql"
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

func (s *SQLite) ListAuthRules() ([]AuthRule, error) {
	rows, err := s.db.Query(`
		SELECT id, route_id, type, config, whitelist, rate_limit, burst, cors_allowed_origins, cors_allowed_methods, cors_allowed_headers, cors_allow_credentials, cors_max_age, created_at, updated_at
		FROM auth_rules
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	rules := make([]AuthRule, 0)
	for rows.Next() {
		var r AuthRule
		var configStr string
		var whitelistStr string
		if err := rows.Scan(
			&r.ID,
			&r.RouteID,
			&r.Type,
			&configStr,
			&whitelistStr,
			&r.RateLimit,
			&r.Burst,
			&r.CORSAllowedOrigins,
			&r.CORSAllowedMethods,
			&r.CORSAllowedHeaders,
			&r.CORSAllowCredentials,
			&r.CORSMaxAge,
			&r.CreatedAt,
			&r.UpdatedAt,
		); err != nil {
			return nil, err
		}
		r.Config = ParseAuthConfig(configStr)
		r.Whitelist = parseWhitelist(whitelistStr)
		rules = append(rules, r)
	}
	return rules, nil
}

func (s *SQLite) GetAuthRule(id string) (*AuthRule, error) {
	var r AuthRule
	var configStr string
	var whitelistStr string
	err := s.db.QueryRow(`
		SELECT id, route_id, type, config, whitelist, rate_limit, burst, cors_allowed_origins, cors_allowed_methods, cors_allowed_headers, cors_allow_credentials, cors_max_age, created_at, updated_at
		FROM auth_rules WHERE id = ?
	`, id).Scan(
		&r.ID,
		&r.RouteID,
		&r.Type,
		&configStr,
		&whitelistStr,
		&r.RateLimit,
		&r.Burst,
		&r.CORSAllowedOrigins,
		&r.CORSAllowedMethods,
		&r.CORSAllowedHeaders,
		&r.CORSAllowCredentials,
		&r.CORSMaxAge,
		&r.CreatedAt,
		&r.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	r.Config = ParseAuthConfig(configStr)
	r.Whitelist = parseWhitelist(whitelistStr)
	return &r, nil
}

func (s *SQLite) GetAuthRuleByRouteID(routeID string) (*AuthRule, error) {
	var r AuthRule
	var configStr string
	var whitelistStr string
	err := s.db.QueryRow(`
		SELECT id, route_id, type, config, whitelist, rate_limit, burst, cors_allowed_origins, cors_allowed_methods, cors_allowed_headers, cors_allow_credentials, cors_max_age, created_at, updated_at
		FROM auth_rules WHERE route_id = ?
	`, routeID).Scan(
		&r.ID,
		&r.RouteID,
		&r.Type,
		&configStr,
		&whitelistStr,
		&r.RateLimit,
		&r.Burst,
		&r.CORSAllowedOrigins,
		&r.CORSAllowedMethods,
		&r.CORSAllowedHeaders,
		&r.CORSAllowCredentials,
		&r.CORSMaxAge,
		&r.CreatedAt,
		&r.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	r.Config = ParseAuthConfig(configStr)
	r.Whitelist = parseWhitelist(whitelistStr)
	return &r, nil
}

func (s *SQLite) CreateAuthRule(r *AuthRule) error {
	if r.ID == "" {
		r.ID = uuid.New().String()
	}
	now := time.Now()
	r.CreatedAt = now
	r.UpdatedAt = now

	wl, _ := json.Marshal(r.Whitelist)
	_, err := s.db.Exec(`
		INSERT INTO auth_rules (id, route_id, type, config, whitelist, rate_limit, burst, cors_allowed_origins, cors_allowed_methods, cors_allowed_headers, cors_allow_credentials, cors_max_age, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, r.ID, r.RouteID, r.Type, r.Config.ToJSON(), string(wl), r.RateLimit, r.Burst, r.CORSAllowedOrigins, r.CORSAllowedMethods, r.CORSAllowedHeaders, r.CORSAllowCredentials, r.CORSMaxAge, r.CreatedAt, r.UpdatedAt)
	return err
}

func (s *SQLite) UpdateAuthRule(r *AuthRule) error {
	r.UpdatedAt = time.Now()

	wl, _ := json.Marshal(r.Whitelist)
	result, err := s.db.Exec(`
		UPDATE auth_rules SET route_id = ?, type = ?, config = ?, whitelist = ?, rate_limit = ?, burst = ?, cors_allowed_origins = ?, cors_allowed_methods = ?, cors_allowed_headers = ?, cors_allow_credentials = ?, cors_max_age = ?, updated_at = ?
		WHERE id = ?
	`, r.RouteID, r.Type, r.Config.ToJSON(), string(wl), r.RateLimit, r.Burst, r.CORSAllowedOrigins, r.CORSAllowedMethods, r.CORSAllowedHeaders, r.CORSAllowCredentials, r.CORSMaxAge, r.UpdatedAt, r.ID)
	if err != nil {
		return err
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (s *SQLite) DeleteAuthRule(id string) error {
	_, err := s.db.Exec("DELETE FROM auth_rules WHERE id = ?", id)
	return err
}

func parseWhitelist(s string) []string {
	if s == "" || s == "[]" {
		return nil
	}
	var out []string
	if err := json.Unmarshal([]byte(s), &out); err != nil {
		return nil
	}
	return out
}
