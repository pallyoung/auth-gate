package store

import (
	"database/sql"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type User struct {
	ID           string    `json:"id"`
	Username     string    `json:"username"`
	PasswordHash string    `json:"-"`
	Role         string    `json:"role"`
	Enabled      bool      `json:"enabled"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

const (
	RoleAdmin  = "admin"
	RoleEditor = "editor"
	RoleViewer = "viewer"
)

type Permissions struct {
	CanManageRoutes bool
	CanManageAuth   bool
	CanManageUsers  bool
	CanViewLogs     bool
}

func GetPermissions(role string) Permissions {
	switch role {
	case RoleAdmin:
		return Permissions{CanManageRoutes: true, CanManageAuth: true, CanManageUsers: true, CanViewLogs: true}
	case RoleEditor:
		return Permissions{CanManageRoutes: true, CanManageAuth: true, CanManageUsers: false, CanViewLogs: true}
	default:
		return Permissions{CanManageRoutes: false, CanManageAuth: false, CanManageUsers: false, CanViewLogs: false}
	}
}

func (s *SQLite) ListUsers() ([]User, error) {
	rows, err := s.db.Query(`SELECT id, username, password_hash, role, enabled, created_at, updated_at FROM users ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	users := make([]User, 0)
	for rows.Next() {
		var u User
		var enabled int
		if err := rows.Scan(&u.ID, &u.Username, &u.PasswordHash, &u.Role, &enabled, &u.CreatedAt, &u.UpdatedAt); err != nil {
			return nil, err
		}
		u.Enabled = enabled == 1
		users = append(users, u)
	}
	return users, nil
}

func (s *SQLite) GetUserByUsername(username string) (*User, error) {
	var u User
	var enabled int
	err := s.db.QueryRow(`SELECT id, username, password_hash, role, enabled, created_at, updated_at FROM users WHERE username = ?`, username).
		Scan(&u.ID, &u.Username, &u.PasswordHash, &u.Role, &enabled, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		return nil, err
	}
	u.Enabled = enabled == 1
	return &u, nil
}

func (s *SQLite) GetUserByID(id string) (*User, error) {
	var u User
	var enabled int
	err := s.db.QueryRow(`SELECT id, username, password_hash, role, enabled, created_at, updated_at FROM users WHERE id = ?`, id).
		Scan(&u.ID, &u.Username, &u.PasswordHash, &u.Role, &enabled, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		return nil, err
	}
	u.Enabled = enabled == 1
	return &u, nil
}

func (s *SQLite) CreateUser(u *User) error {
	if u.ID == "" {
		u.ID = uuid.New().String()
	}
	now := time.Now()
	u.CreatedAt = now
	u.UpdatedAt = now
	enabled := 0
	if u.Enabled {
		enabled = 1
	}
	_, err := s.db.Exec(`INSERT INTO users (id, username, password_hash, role, enabled, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		u.ID, u.Username, u.PasswordHash, u.Role, enabled, u.CreatedAt, u.UpdatedAt)
	return err
}

func (s *SQLite) UpdateUser(u *User) error {
	u.UpdatedAt = time.Now()
	enabled := 0
	if u.Enabled {
		enabled = 1
	}
	result, err := s.db.Exec(`UPDATE users SET username = ?, role = ?, enabled = ?, updated_at = ? WHERE id = ?`,
		u.Username, u.Role, enabled, u.UpdatedAt, u.ID)
	if err != nil {
		return err
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (s *SQLite) DeleteUser(id string) error {
	_, err := s.db.Exec("DELETE FROM users WHERE id = ?", id)
	return err
}

func (s *SQLite) VerifyPassword(user *User, password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password))
	return err == nil
}

func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(bytes), err
}

func (s *SQLite) EnsureAdmin(username, password string) (bool, error) {
	if strings.TrimSpace(username) == "" {
		username = "admin"
	}
	if password == "" {
		return false, errors.New("bootstrap admin password required")
	}

	users, err := s.ListUsers()
	if err != nil {
		return false, err
	}
	if len(users) == 0 {
		hash, err := HashPassword(password)
		if err != nil {
			return false, err
		}
		if err := s.CreateUser(&User{
			Username:     "admin",
			PasswordHash: hash,
			Role:         RoleAdmin,
			Enabled:      true,
		}); err != nil {
			return false, err
		}
		return true, nil
	}
	return false, nil
}
