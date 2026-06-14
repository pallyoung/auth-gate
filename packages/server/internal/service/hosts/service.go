package hostservice

import (
	"database/sql"
	"errors"
	"strings"

	"github.com/pallyoung/auth-gate/packages/server/internal/store"
	"github.com/pallyoung/auth-gate/packages/server/internal/syshosts"
)

const (
	ErrCodeProfileNotFound     = "host_profile_not_found"
	ErrCodeEntryNotFound       = "host_entry_not_found"
	ErrCodeDuplicateProfileName = "duplicate_host_profile_name"
	ErrCodeInvalidProfileName   = "invalid_host_profile_name"
	ErrCodeInvalidIP            = "invalid_host_ip"
	ErrCodeInvalidHostname      = "invalid_host_hostname"
	ErrCodeDuplicateHostname    = "duplicate_host_hostname"
	ErrCodeMarkerMissing        = "host_marker_missing"
	ErrCodePermissionDenied     = "host_permission_denied"
	ErrCodeStoreFailure         = "host_store_failure"
	ErrCodeRenderFailure        = "host_render_failure"
)

type Error struct {
	code    string
	message string
	cause   error
}

func (e *Error) Error() string { return e.message }

func (e *Error) Unwrap() error { return e.cause }

func Code(err error) string {
	var target *Error
	if errors.As(err, &target) {
		return target.code
	}
	return ""
}

func newError(code, message string, cause error) error {
	return &Error{code: code, message: message, cause: cause}
}

type ProfileInput struct {
	Name        string
	Description string
}

type Service struct {
	db       *store.SQLite
	renderer *syshosts.Renderer
}

func NewService(db *store.SQLite, renderer *syshosts.Renderer) *Service {
	return &Service{db: db, renderer: renderer}
}

func (s *Service) ListProfiles() ([]store.HostProfile, error) {
	profiles, err := s.db.ListHostProfiles()
	if err != nil {
		return nil, newError(ErrCodeStoreFailure, "failed to list host profiles", err)
	}
	return profiles, nil
}

func (s *Service) GetProfile(id string) (*store.HostProfile, error) {
	p, err := s.db.GetHostProfile(id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, newError(ErrCodeProfileNotFound, "host profile not found", err)
		}
		return nil, newError(ErrCodeStoreFailure, "failed to get host profile", err)
	}
	return p, nil
}

func (s *Service) CreateProfile(in ProfileInput) (*store.HostProfile, error) {
	name := strings.TrimSpace(in.Name)
	description := strings.TrimSpace(in.Description)
	if err := validateProfileName(name); err != nil {
		return nil, err
	}
	p := &store.HostProfile{Name: name, Description: description}
	if err := s.db.CreateHostProfile(p); err != nil {
		if isUniqueViolation(err, "host_profiles.name") {
			return nil, newError(ErrCodeDuplicateProfileName, "a profile with this name already exists", err)
		}
		return nil, newError(ErrCodeStoreFailure, "failed to create host profile", err)
	}
	return p, nil
}

func (s *Service) UpdateProfile(id string, in ProfileInput) (*store.HostProfile, error) {
	p, err := s.GetProfile(id)
	if err != nil {
		return nil, err
	}
	if in.Name != "" {
		name := strings.TrimSpace(in.Name)
		if err := validateProfileName(name); err != nil {
			return nil, err
		}
		p.Name = name
	}
	p.Description = strings.TrimSpace(in.Description)
	if err := s.db.UpdateHostProfile(p); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, newError(ErrCodeProfileNotFound, "host profile not found", err)
		}
		if isUniqueViolation(err, "host_profiles.name") {
			return nil, newError(ErrCodeDuplicateProfileName, "a profile with this name already exists", err)
		}
		return nil, newError(ErrCodeStoreFailure, "failed to update host profile", err)
	}
	return p, nil
}

func (s *Service) DeleteProfile(id string) error {
	if _, err := s.GetProfile(id); err != nil {
		return err
	}
	if err := s.db.DeleteHostProfile(id); err != nil {
		return newError(ErrCodeStoreFailure, "failed to delete host profile", err)
	}
	return nil
}

func isUniqueViolation(err error, column string) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "UNIQUE") && strings.Contains(msg, column)
}
