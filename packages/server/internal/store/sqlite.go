package store

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

type SQLite struct {
	db *sql.DB
}

func NewSQLite(path string) (*SQLite, error) {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}

	db, err := sql.Open("sqlite", fmt.Sprintf("%s?_pragma=foreign_keys(1)", path))
	if err != nil {
		return nil, err
	}

	schema := `
	CREATE TABLE IF NOT EXISTS routes (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		host TEXT DEFAULT '',
		path_prefix TEXT NOT NULL,
		backend TEXT NOT NULL,
		strip_prefix INTEGER DEFAULT 0,
		enabled INTEGER DEFAULT 1,
		priority INTEGER DEFAULT 0,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS auth_rules (
		id TEXT PRIMARY KEY,
		route_id TEXT NOT NULL,
		type TEXT NOT NULL DEFAULT 'none',
		config TEXT DEFAULT '{}',
		whitelist TEXT DEFAULT '[]',
		rate_limit INTEGER DEFAULT 0,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (route_id) REFERENCES routes(id) ON DELETE CASCADE
	);

	CREATE TABLE IF NOT EXISTS users (
		id TEXT PRIMARY KEY,
		username TEXT UNIQUE NOT NULL,
		password_hash TEXT NOT NULL,
		role TEXT DEFAULT 'viewer',
		enabled INTEGER DEFAULT 1,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE INDEX IF NOT EXISTS idx_routes_host ON routes(host);
	CREATE INDEX IF NOT EXISTS idx_routes_enabled ON routes(enabled);
	CREATE UNIQUE INDEX IF NOT EXISTS idx_auth_rules_route_id ON auth_rules(route_id);
	CREATE INDEX IF NOT EXISTS idx_users_username ON users(username);
	`

	if _, err := db.Exec(schema); err != nil {
		return nil, err
	}

	return &SQLite{db: db}, nil
}

func (s *SQLite) DB() *sql.DB {
	return s.db
}

func (s *SQLite) Close() error {
	return s.db.Close()
}
