package store

import (
	"database/sql"
	"time"

	"github.com/google/uuid"
)

func (s *SQLite) ListAuthRules() ([]AuthRule, error) {
	rows, err := s.db.Query(`
		SELECT id, route_id, type, config, created_at, updated_at
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
		if err := rows.Scan(&r.ID, &r.RouteID, &r.Type, &configStr, &r.CreatedAt, &r.UpdatedAt); err != nil {
			return nil, err
		}
		r.Config = ParseAuthConfig(configStr)
		rules = append(rules, r)
	}
	return rules, nil
}

func (s *SQLite) GetAuthRule(id string) (*AuthRule, error) {
	var r AuthRule
	var configStr string
	err := s.db.QueryRow(`
		SELECT id, route_id, type, config, created_at, updated_at
		FROM auth_rules WHERE id = ?
	`, id).Scan(&r.ID, &r.RouteID, &r.Type, &configStr, &r.CreatedAt, &r.UpdatedAt)
	if err != nil {
		return nil, err
	}
	r.Config = ParseAuthConfig(configStr)
	return &r, nil
}

func (s *SQLite) GetAuthRuleByRouteID(routeID string) (*AuthRule, error) {
	var r AuthRule
	var configStr string
	err := s.db.QueryRow(`
		SELECT id, route_id, type, config, created_at, updated_at
		FROM auth_rules WHERE route_id = ?
	`, routeID).Scan(&r.ID, &r.RouteID, &r.Type, &configStr, &r.CreatedAt, &r.UpdatedAt)
	if err != nil {
		return nil, err
	}
	r.Config = ParseAuthConfig(configStr)
	return &r, nil
}

func (s *SQLite) CreateAuthRule(r *AuthRule) error {
	if r.ID == "" {
		r.ID = uuid.New().String()
	}
	now := time.Now()
	r.CreatedAt = now
	r.UpdatedAt = now

	_, err := s.db.Exec(`
		INSERT INTO auth_rules (id, route_id, type, config, whitelist, rate_limit, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, r.ID, r.RouteID, r.Type, r.Config.ToJSON(), "[]", 0, r.CreatedAt, r.UpdatedAt)
	return err
}

func (s *SQLite) UpdateAuthRule(r *AuthRule) error {
	r.UpdatedAt = time.Now()

	result, err := s.db.Exec(`
		UPDATE auth_rules SET route_id = ?, type = ?, config = ?, whitelist = ?, rate_limit = ?, updated_at = ?
		WHERE id = ?
	`, r.RouteID, r.Type, r.Config.ToJSON(), "[]", 0, r.UpdatedAt, r.ID)
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
