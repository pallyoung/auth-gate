package store

import (
	"database/sql"
	"time"

	"github.com/google/uuid"
)

func (s *SQLite) ListRoutes() ([]Route, error) {
	rows, err := s.db.Query(`
		SELECT id, name, host, path_prefix, backend, strip_prefix, enabled, priority, created_at, updated_at
		FROM routes
		ORDER BY priority DESC, path_prefix DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var routes []Route
	for rows.Next() {
		var r Route
		var stripPrefix, enabled int
		if err := rows.Scan(&r.ID, &r.Name, &r.Host, &r.PathPrefix, &r.Backend, &stripPrefix, &enabled, &r.Priority, &r.CreatedAt, &r.UpdatedAt); err != nil {
			return nil, err
		}
		r.StripPrefix = stripPrefix == 1
		r.Enabled = enabled == 1
		routes = append(routes, r)
	}
	return routes, nil
}

func (s *SQLite) GetRoute(id string) (*Route, error) {
	var r Route
	var stripPrefix, enabled int
	err := s.db.QueryRow(`
		SELECT id, name, host, path_prefix, backend, strip_prefix, enabled, priority, created_at, updated_at
		FROM routes WHERE id = ?
	`, id).Scan(&r.ID, &r.Name, &r.Host, &r.PathPrefix, &r.Backend, &stripPrefix, &enabled, &r.Priority, &r.CreatedAt, &r.UpdatedAt)
	if err != nil {
		return nil, err
	}
	r.StripPrefix = stripPrefix == 1
	r.Enabled = enabled == 1
	return &r, nil
}

func (s *SQLite) CreateRoute(r *Route) error {
	if r.ID == "" {
		r.ID = uuid.New().String()
	}
	now := time.Now()
	r.CreatedAt = now
	r.UpdatedAt = now

	stripPrefix := 0
	if r.StripPrefix {
		stripPrefix = 1
	}
	enabled := 0
	if r.Enabled {
		enabled = 1
	}

	_, err := s.db.Exec(`
		INSERT INTO routes (id, name, host, path_prefix, backend, strip_prefix, enabled, priority, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, r.ID, r.Name, r.Host, r.PathPrefix, r.Backend, stripPrefix, enabled, r.Priority, r.CreatedAt, r.UpdatedAt)
	return err
}

func (s *SQLite) UpdateRoute(r *Route) error {
	r.UpdatedAt = time.Now()

	stripPrefix := 0
	if r.StripPrefix {
		stripPrefix = 1
	}
	enabled := 0
	if r.Enabled {
		enabled = 1
	}

	result, err := s.db.Exec(`
		UPDATE routes SET name = ?, host = ?, path_prefix = ?, backend = ?, strip_prefix = ?, enabled = ?, priority = ?, updated_at = ?
		WHERE id = ?
	`, r.Name, r.Host, r.PathPrefix, r.Backend, stripPrefix, enabled, r.Priority, r.UpdatedAt, r.ID)
	if err != nil {
		return err
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (s *SQLite) DeleteRoute(id string) error {
	_, err := s.db.Exec("DELETE FROM routes WHERE id = ?", id)
	return err
}
