package store

import (
	"database/sql"
	"time"

	"github.com/google/uuid"
)

func (s *SQLite) ListHostProfiles() ([]HostProfile, error) {
	rows, err := s.db.Query(`
		SELECT id, name, description, is_active, created_at, updated_at
		FROM host_profiles
		ORDER BY name
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	profiles := make([]HostProfile, 0)
	for rows.Next() {
		var p HostProfile
		var isActive int
		if err := rows.Scan(&p.ID, &p.Name, &p.Description, &isActive, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, err
		}
		p.IsActive = isActive == 1
		profiles = append(profiles, p)
	}
	return profiles, nil
}

func (s *SQLite) CreateHostProfile(p *HostProfile) error {
	if p.ID == "" {
		p.ID = uuid.New().String()
	}
	now := time.Now()
	p.CreatedAt = now
	p.UpdatedAt = now

	_, err := s.db.Exec(`
		INSERT INTO host_profiles (id, name, description, is_active, created_at, updated_at)
		VALUES (?, ?, ?, 0, ?, ?)
	`, p.ID, p.Name, p.Description, p.CreatedAt, p.UpdatedAt)
	return err
}

func (s *SQLite) UpdateHostProfile(p *HostProfile) error {
	p.UpdatedAt = time.Now()
	result, err := s.db.Exec(`
		UPDATE host_profiles SET name = ?, description = ?, updated_at = ? WHERE id = ?
	`, p.Name, p.Description, p.UpdatedAt, p.ID)
	if err != nil {
		return err
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (s *SQLite) DeleteHostProfile(id string) error {
	_, err := s.db.Exec(`DELETE FROM host_profiles WHERE id = ?`, id)
	return err
}

func (s *SQLite) CreateHostEntry(e *HostEntry) error {
	if e.ID == "" {
		e.ID = uuid.New().String()
	}

	// Auto-assign position = max(position) + 1 for the profile, unless the
	// caller already supplied a positive position (used by ReorderHostEntries).
	if e.Position <= 0 {
		var maxPos sql.NullInt64
		if err := s.db.QueryRow(`
			SELECT MAX(position) FROM host_entries WHERE profile_id = ?
		`, e.ProfileID).Scan(&maxPos); err != nil {
			return err
		}
		e.Position = int(maxPos.Int64) + 1
	}

	now := time.Now()
	e.CreatedAt = now
	e.UpdatedAt = now

	enabled := 0
	if e.Enabled {
		enabled = 1
	}

	_, err := s.db.Exec(`
		INSERT INTO host_entries (id, profile_id, position, ip, hostnames, comment, enabled, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, e.ID, e.ProfileID, e.Position, e.IP, e.Hostnames, e.Comment, enabled, e.CreatedAt, e.UpdatedAt)
	return err
}

func (s *SQLite) ListHostEntries(profileID string) ([]HostEntry, error) {
	rows, err := s.db.Query(`
		SELECT id, profile_id, position, ip, hostnames, comment, enabled, created_at, updated_at
		FROM host_entries
		WHERE profile_id = ?
		ORDER BY position, id
	`, profileID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	entries := make([]HostEntry, 0)
	for rows.Next() {
		var e HostEntry
		var enabled int
		if err := rows.Scan(&e.ID, &e.ProfileID, &e.Position, &e.IP, &e.Hostnames, &e.Comment, &enabled, &e.CreatedAt, &e.UpdatedAt); err != nil {
			return nil, err
		}
		e.Enabled = enabled == 1
		entries = append(entries, e)
	}
	return entries, nil
}

func (s *SQLite) GetHostProfile(id string) (*HostProfile, error) {
	var p HostProfile
	var isActive int
	err := s.db.QueryRow(`
		SELECT id, name, description, is_active, created_at, updated_at
		FROM host_profiles WHERE id = ?
	`, id).Scan(&p.ID, &p.Name, &p.Description, &isActive, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		return nil, err
	}
	p.IsActive = isActive == 1
	return &p, nil
}

func (s *SQLite) GetHostEntry(id string) (*HostEntry, error) {
	var e HostEntry
	var enabled int
	err := s.db.QueryRow(`
		SELECT id, profile_id, position, ip, hostnames, comment, enabled, created_at, updated_at
		FROM host_entries WHERE id = ?
	`, id).Scan(&e.ID, &e.ProfileID, &e.Position, &e.IP, &e.Hostnames, &e.Comment, &enabled, &e.CreatedAt, &e.UpdatedAt)
	if err != nil {
		return nil, err
	}
	e.Enabled = enabled == 1
	return &e, nil
}

func (s *SQLite) UpdateHostEntry(e *HostEntry) error {
	e.UpdatedAt = time.Now()
	enabled := 0
	if e.Enabled {
		enabled = 1
	}
	result, err := s.db.Exec(`
		UPDATE host_entries SET position = ?, ip = ?, hostnames = ?, comment = ?, enabled = ?, updated_at = ?
		WHERE id = ?
	`, e.Position, e.IP, e.Hostnames, e.Comment, enabled, e.UpdatedAt, e.ID)
	if err != nil {
		return err
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (s *SQLite) DeleteHostEntry(id string) error {
	_, err := s.db.Exec(`DELETE FROM host_entries WHERE id = ?`, id)
	return err
}

// SetActiveHostProfile clears is_active on all profiles and then sets
// is_active = 1 on the given profile. It runs inside the caller-provided
// transaction so the caller can wrap it together with other writes (e.g.
// the syshosts file rewrite) for true atomicity. The "only one active" rule
// is enforced by the caller via the transaction, not by a partial unique
// index, because SQLite doesn't support partial unique indexes.
func (s *SQLite) SetActiveHostProfile(tx *sql.Tx, profileID string) error {
	if _, err := tx.Exec(`UPDATE host_profiles SET is_active = 0, updated_at = ?`, time.Now()); err != nil {
		return err
	}
	result, err := tx.Exec(`UPDATE host_profiles SET is_active = 1, updated_at = ? WHERE id = ?`, time.Now(), profileID)
	if err != nil {
		return err
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return sql.ErrNoRows
	}
	return nil
}

// ReorderHostEntries atomically rewrites the position of every entry in
// the profile to match the new order. The caller (service layer) is
// responsible for ensuring the IDs all belong to the given profile and
// that no IDs are missing. If any UPDATE affects 0 rows the whole
// transaction is rolled back, so partial reorders are impossible.
func (s *SQLite) ReorderHostEntries(profileID string, orderedIDs []string) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	now := time.Now()
	for i, id := range orderedIDs {
		result, err := tx.Exec(`
			UPDATE host_entries SET position = ?, updated_at = ?
			WHERE id = ? AND profile_id = ?
		`, i, now, id, profileID)
		if err != nil {
			return err
		}
		rows, _ := result.RowsAffected()
		if rows == 0 {
			return sql.ErrNoRows
		}
	}
	return tx.Commit()
}

// ListEnabledHostEntries returns only the enabled entries in the profile,
// ordered by position. Returns an empty slice (not nil) when there are no
// matching entries.
func (s *SQLite) ListEnabledHostEntries(profileID string) ([]HostEntry, error) {
	rows, err := s.db.Query(`
		SELECT id, profile_id, position, ip, hostnames, comment, enabled, created_at, updated_at
		FROM host_entries
		WHERE profile_id = ? AND enabled = 1
		ORDER BY position, id
	`, profileID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	entries := make([]HostEntry, 0)
	for rows.Next() {
		var e HostEntry
		var enabled int
		if err := rows.Scan(&e.ID, &e.ProfileID, &e.Position, &e.IP, &e.Hostnames, &e.Comment, &enabled, &e.CreatedAt, &e.UpdatedAt); err != nil {
			return nil, err
		}
		e.Enabled = enabled == 1
		entries = append(entries, e)
	}
	return entries, nil
}
