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
	ErrCodeInvalidComment       = "invalid_host_comment"
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

type EntryInput struct {
	IP        string
	Comment   string
	Hostnames []string
	Enabled   bool
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

func splitHostnames(in []string) []string {
	out := make([]string, 0, len(in))
	for _, h := range in {
		h = strings.TrimSpace(h)
		if h == "" {
			continue
		}
		out = append(out, h)
	}
	return out
}

func (s *Service) ListEntries(profileID string) ([]store.HostEntry, error) {
	if _, err := s.GetProfile(profileID); err != nil {
		return nil, err
	}
	entries, err := s.db.ListHostEntries(profileID)
	if err != nil {
		return nil, newError(ErrCodeStoreFailure, "failed to list host entries", err)
	}
	return entries, nil
}

func (s *Service) CreateEntry(profileID string, in EntryInput) (*store.HostEntry, error) {
	if _, err := s.GetProfile(profileID); err != nil {
		return nil, err
	}
	hostnames := splitHostnames(in.Hostnames)
	if err := s.validateEntryFields(in.IP, hostnames, in.Comment); err != nil {
		return nil, err
	}
	if err := s.assertNoDuplicateHostname(profileID, hostnames, ""); err != nil {
		return nil, err
	}

	nextPos, err := s.nextEntryPosition(profileID)
	if err != nil {
		return nil, err
	}

	e := &store.HostEntry{
		ProfileID: profileID,
		Position:  nextPos,
		IP:        strings.TrimSpace(in.IP),
		Hostnames: strings.Join(hostnames, " "),
		Comment:   in.Comment,
		Enabled:   in.Enabled,
	}
	if err := s.db.CreateHostEntry(e); err != nil {
		return nil, newError(ErrCodeStoreFailure, "failed to create host entry", err)
	}
	return e, nil
}

func (s *Service) GetEntry(profileID, entryID string) (*store.HostEntry, error) {
	e, err := s.db.GetHostEntry(entryID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, newError(ErrCodeEntryNotFound, "host entry not found", err)
		}
		return nil, newError(ErrCodeStoreFailure, "failed to get host entry", err)
	}
	if e.ProfileID != profileID {
		return nil, newError(ErrCodeEntryNotFound, "host entry not found", nil)
	}
	return e, nil
}

func (s *Service) UpdateEntry(profileID, entryID string, in EntryInput) (*store.HostEntry, error) {
	existing, err := s.GetEntry(profileID, entryID)
	if err != nil {
		return nil, err
	}
	hostnames := splitHostnames(in.Hostnames)
	if err := s.validateEntryFields(in.IP, hostnames, in.Comment); err != nil {
		return nil, err
	}
	if err := s.assertNoDuplicateHostname(profileID, hostnames, entryID); err != nil {
		return nil, err
	}
	existing.IP = strings.TrimSpace(in.IP)
	existing.Hostnames = strings.Join(hostnames, " ")
	existing.Comment = in.Comment
	existing.Enabled = in.Enabled
	if err := s.db.UpdateHostEntry(existing); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, newError(ErrCodeEntryNotFound, "host entry not found", err)
		}
		return nil, newError(ErrCodeStoreFailure, "failed to update host entry", err)
	}
	return existing, nil
}

func (s *Service) DeleteEntry(profileID, entryID string) error {
	if _, err := s.GetEntry(profileID, entryID); err != nil {
		return err
	}
	if err := s.db.DeleteHostEntry(entryID); err != nil {
		return newError(ErrCodeStoreFailure, "failed to delete host entry", err)
	}
	return nil
}

func (s *Service) ReorderEntries(profileID string, orderedIDs []string) error {
	if _, err := s.GetProfile(profileID); err != nil {
		return err
	}
	for _, id := range orderedIDs {
		if _, err := s.GetEntry(profileID, id); err != nil {
			return err
		}
	}
	if err := s.db.ReorderHostEntries(profileID, orderedIDs); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return newError(ErrCodeEntryNotFound, "host entry not found", err)
		}
		return newError(ErrCodeStoreFailure, "failed to reorder host entries", err)
	}
	return nil
}

func (s *Service) validateEntryFields(ip string, hostnames []string, comment string) error {
	if err := validateIP(ip); err != nil {
		return err
	}
	if len(hostnames) == 0 {
		return newError(ErrCodeInvalidHostname, "at least one hostname is required", nil)
	}
	for _, h := range hostnames {
		if err := validateHostname(h); err != nil {
			return err
		}
	}
	return validateComment(comment)
}

func (s *Service) assertNoDuplicateHostname(profileID string, hostnames []string, excludeEntryID string) error {
	entries, err := s.db.ListHostEntries(profileID)
	if err != nil {
		return newError(ErrCodeStoreFailure, "failed to list host entries", err)
	}
	seen := make(map[string]struct{}, len(hostnames))
	for _, h := range hostnames {
		seen[strings.ToLower(h)] = struct{}{}
	}
	for _, e := range entries {
		if excludeEntryID != "" && e.ID == excludeEntryID {
			continue
		}
		for _, h := range strings.Fields(e.Hostnames) {
			if _, dup := seen[strings.ToLower(h)]; dup {
				return newError(ErrCodeDuplicateHostname, "duplicate hostname in profile: "+h, nil)
			}
		}
	}
	return nil
}

func (s *Service) nextEntryPosition(profileID string) (int, error) {
	entries, err := s.db.ListHostEntries(profileID)
	if err != nil {
		return 0, newError(ErrCodeStoreFailure, "failed to list host entries", err)
	}
	next := 0
	for _, e := range entries {
		if e.Position >= next {
			next = e.Position + 1
		}
	}
	return next, nil
}

func (s *Service) ActivateProfile(id string) (*store.HostProfile, error) {
	if _, err := s.GetProfile(id); err != nil {
		return nil, err
	}
	entries, err := s.db.ListEnabledHostEntries(id)
	if err != nil {
		return nil, newError(ErrCodeStoreFailure, "failed to list host entries", err)
	}
	content := renderEntries(entries)

	tx, err := s.db.DB().Begin()
	if err != nil {
		return nil, newError(ErrCodeStoreFailure, "failed to begin transaction", err)
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	if err := s.db.SetActiveHostProfile(tx, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, newError(ErrCodeProfileNotFound, "host profile not found", err)
		}
		return nil, newError(ErrCodeStoreFailure, "failed to set active host profile", err)
	}

	if s.renderer == nil {
		return nil, newError(ErrCodeRenderFailure, "renderer not configured", nil)
	}
	if err := s.renderer.Apply(content); err != nil {
		if errors.Is(err, syshosts.ErrMarkerMissing) {
			return nil, newError(ErrCodeMarkerMissing, "managed marker block is missing in /etc/hosts; append the markers manually before activating", err)
		}
		return nil, newError(ErrCodeRenderFailure, "failed to apply hosts file change", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, newError(ErrCodeStoreFailure, "failed to commit transaction", err)
	}
	committed = true

	p, err := s.GetProfile(id)
	if err != nil {
		return nil, err
	}
	return p, nil
}

func renderEntries(entries []store.HostEntry) string {
	var b strings.Builder
	for _, e := range entries {
		h := strings.TrimSpace(e.Hostnames)
		if h == "" {
			continue
		}
		b.WriteString(strings.TrimSpace(e.IP))
		b.WriteByte(' ')
		b.WriteString(h)
		if e.Comment != "" {
			b.WriteString("  # ")
			b.WriteString(e.Comment)
		}
		b.WriteByte('\n')
	}
	return b.String()
}
