package store

import (
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
