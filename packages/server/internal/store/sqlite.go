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

	db, err := sql.Open("sqlite", fmt.Sprintf("%s?_pragma=busy_timeout(5000)&_pragma=foreign_keys(1)", path))
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
		cert_path TEXT DEFAULT '',
		key_path TEXT DEFAULT '',
		tls_enabled INTEGER DEFAULT 0,
		timeout_ms INTEGER DEFAULT 0,
		retry_attempts INTEGER DEFAULT 0,
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
		burst INTEGER DEFAULT 0,
		cors_allowed_origins TEXT DEFAULT '',
		cors_allowed_methods TEXT DEFAULT '',
		cors_allowed_headers TEXT DEFAULT '',
		cors_allow_credentials INTEGER DEFAULT 0,
		cors_max_age INTEGER DEFAULT 0,
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

	CREATE TABLE IF NOT EXISTS user_route_access (
		user_id TEXT NOT NULL,
		route_id TEXT NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		PRIMARY KEY (user_id, route_id),
		FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
		FOREIGN KEY (route_id) REFERENCES routes(id) ON DELETE CASCADE
	);

	CREATE INDEX IF NOT EXISTS idx_routes_host ON routes(host);
	CREATE INDEX IF NOT EXISTS idx_routes_enabled ON routes(enabled);
	CREATE UNIQUE INDEX IF NOT EXISTS idx_auth_rules_route_id ON auth_rules(route_id);
	CREATE INDEX IF NOT EXISTS idx_users_username ON users(username);
	CREATE INDEX IF NOT EXISTS idx_user_route_access_route_id ON user_route_access(route_id);

	CREATE TABLE IF NOT EXISTS certificates (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		domain TEXT NOT NULL,
		cert_path TEXT NOT NULL DEFAULT '',
		key_path TEXT NOT NULL DEFAULT '',
		dns_provider TEXT NOT NULL DEFAULT '',
		dns_provider_config TEXT NOT NULL DEFAULT '',
		source TEXT NOT NULL DEFAULT 'local_ca',
		ca_id TEXT NOT NULL DEFAULT '',
		status TEXT NOT NULL DEFAULT 'active',
		not_before DATETIME DEFAULT '',
		not_after DATETIME DEFAULT '',
		renew_at DATETIME DEFAULT '',
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS ca_certificates (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		cert_pem TEXT NOT NULL,
		key_pem TEXT NOT NULL,
		not_before DATETIME DEFAULT '',
		not_after DATETIME DEFAULT '',
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE INDEX IF NOT EXISTS idx_certificates_domain ON certificates(domain);
	CREATE INDEX IF NOT EXISTS idx_certificates_status ON certificates(status);
	CREATE INDEX IF NOT EXISTS idx_certificates_source ON certificates(source);

	CREATE TABLE IF NOT EXISTS host_profiles (
		id          TEXT PRIMARY KEY,
		name        TEXT UNIQUE NOT NULL,
		description TEXT NOT NULL DEFAULT '',
		is_active   INTEGER NOT NULL DEFAULT 0,
		created_at  DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at  DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS host_entries (
		id          TEXT PRIMARY KEY,
		profile_id  TEXT NOT NULL,
		position    INTEGER NOT NULL,
		ip          TEXT NOT NULL,
		hostnames   TEXT NOT NULL,
		comment     TEXT NOT NULL DEFAULT '',
		enabled     INTEGER NOT NULL DEFAULT 1,
		created_at  DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at  DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (profile_id) REFERENCES host_profiles(id) ON DELETE CASCADE
	);

	CREATE INDEX IF NOT EXISTS idx_host_entries_profile_position ON host_entries(profile_id, position);
	`

	if _, err := db.Exec(schema); err != nil {
		return nil, err
	}

	// Migration: add TLS columns, backends JSON, and rewrite/redirect fields
	migrations := []string{
		`ALTER TABLE routes ADD COLUMN cert_path TEXT DEFAULT ''`,
		`ALTER TABLE routes ADD COLUMN key_path TEXT DEFAULT ''`,
		`ALTER TABLE routes ADD COLUMN tls_enabled INTEGER DEFAULT 0`,
		`ALTER TABLE routes ADD COLUMN backends TEXT DEFAULT '[]'`,
		`ALTER TABLE routes ADD COLUMN path_match_mode TEXT DEFAULT ''`,
		`ALTER TABLE routes ADD COLUMN rewrite_target TEXT DEFAULT ''`,
		`ALTER TABLE routes ADD COLUMN redirect_code INTEGER DEFAULT 0`,
		`ALTER TABLE routes ADD COLUMN timeout_ms INTEGER DEFAULT 0`,
		`ALTER TABLE routes ADD COLUMN retry_attempts INTEGER DEFAULT 0`,
		`ALTER TABLE auth_rules ADD COLUMN burst INTEGER DEFAULT 0`,
		`ALTER TABLE auth_rules ADD COLUMN cors_allowed_origins TEXT DEFAULT ''`,
		`ALTER TABLE auth_rules ADD COLUMN cors_allowed_methods TEXT DEFAULT ''`,
		`ALTER TABLE auth_rules ADD COLUMN cors_allowed_headers TEXT DEFAULT ''`,
		`ALTER TABLE auth_rules ADD COLUMN cors_allow_credentials INTEGER DEFAULT 0`,
		`ALTER TABLE auth_rules ADD COLUMN cors_max_age INTEGER DEFAULT 0`,
		// Certificate table migrations (add columns if upgrading from older version)
		`ALTER TABLE certificates ADD COLUMN name TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE certificates ADD COLUMN source TEXT NOT NULL DEFAULT 'local_ca'`,
		`ALTER TABLE certificates ADD COLUMN ca_id TEXT NOT NULL DEFAULT ''`,
		// Host profile/entry table migrations (defensive no-ops for older DBs)
		`ALTER TABLE host_profiles ADD COLUMN description TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE host_entries ADD COLUMN comment TEXT NOT NULL DEFAULT ''`,
	}
	for _, m := range migrations {
		db.Exec(m) // ignore errors - column may already exist in older DBs
	}

	return &SQLite{db: db}, nil
}

func (s *SQLite) DB() *sql.DB {
	return s.db
}

func (s *SQLite) Close() error {
	return s.db.Close()
}
