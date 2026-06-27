package groups

import (
	"database/sql"
	"errors"
	"strings"

	"github.com/pallyoung/auth-gate/packages/server/internal/store"
)

const (
	ErrCodeGroupNotFound     = "group_not_found"
	ErrCodeInvalidName       = "invalid_group_name"
	ErrCodeDuplicateName     = "duplicate_group_name"
	ErrCodeGroupStoreFailure = "group_store_failure"
)

type Error struct {
	code    string
	message string
	cause   error
}

func (e *Error) Error() string {
	return e.message
}

func (e *Error) Unwrap() error {
	return e.cause
}

func Code(err error) string {
	var target *Error
	if errors.As(err, &target) {
		return target.code
	}
	return ""
}

func newError(code, message string, cause error) error {
	return &Error{
		code:    code,
		message: message,
		cause:   cause,
	}
}

type CreateInput struct {
	Name       string
	RouteIDs   []string
	RoutePaths map[string][]string
}

type UpdateInput struct {
	Name       *string
	RouteIDs   *[]string
	RoutePaths *map[string][]string
}

type Service struct {
	db store.Store
}

func NewService(db store.Store) *Service {
	return &Service{db: db}
}

func (s *Service) List() ([]store.PermissionGroup, error) {
	groups, err := s.db.ListPermissionGroups()
	if err != nil {
		return nil, newError(ErrCodeGroupStoreFailure, "failed to list groups", err)
	}
	return groups, nil
}

func (s *Service) Get(id string) (*store.PermissionGroup, error) {
	g, err := s.db.GetPermissionGroup(id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, newError(ErrCodeGroupNotFound, "group not found", err)
		}
		return nil, newError(ErrCodeGroupStoreFailure, "failed to get group", err)
	}
	return g, nil
}

func (s *Service) Create(input CreateInput) (*store.PermissionGroup, error) {
	name := strings.TrimSpace(input.Name)
	if name == "" {
		return nil, newError(ErrCodeInvalidName, "group name required", nil)
	}

	g := &store.PermissionGroup{
		Name:       name,
		RouteIDs:   input.RouteIDs,
		RoutePaths: input.RoutePaths,
	}
	if err := s.db.CreatePermissionGroup(g); err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "unique") {
			return nil, newError(ErrCodeDuplicateName, "group name already exists", err)
		}
		return nil, newError(ErrCodeGroupStoreFailure, "failed to create group", err)
	}
	return g, nil
}

func (s *Service) Update(id string, input UpdateInput) (*store.PermissionGroup, error) {
	g, err := s.Get(id)
	if err != nil {
		return nil, err
	}

	if input.Name != nil {
		name := strings.TrimSpace(*input.Name)
		if name == "" {
			return nil, newError(ErrCodeInvalidName, "group name required", nil)
		}
		g.Name = name
	}
	if input.RoutePaths != nil {
		g.RoutePaths = *input.RoutePaths
	}
	if input.RouteIDs != nil {
		g.RouteIDs = *input.RouteIDs
	}

	if err := s.db.UpdatePermissionGroup(g); err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "unique") {
			return nil, newError(ErrCodeDuplicateName, "group name already exists", err)
		}
		return nil, newError(ErrCodeGroupStoreFailure, "failed to update group", err)
	}
	return g, nil
}

func (s *Service) Delete(id string) error {
	if _, err := s.Get(id); err != nil {
		return err
	}
	if err := s.db.DeletePermissionGroup(id); err != nil {
		return newError(ErrCodeGroupStoreFailure, "failed to delete group", err)
	}
	return nil
}
