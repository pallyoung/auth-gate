package dto

import (
	"time"

	"github.com/pallyoung/auth-gate/packages/server/internal/store"
)

type PermissionGroup struct {
	ID         string              `json:"id"`
	Name       string              `json:"name"`
	RouteIDs   []string            `json:"route_ids"`
	RoutePaths map[string][]string `json:"route_paths"`
	CreatedAt  *time.Time          `json:"created_at,omitempty"`
	UpdatedAt  *time.Time          `json:"updated_at,omitempty"`
}

type PermissionGroupCreateRequest struct {
	Name       string              `json:"name" binding:"required"`
	RouteIDs   []string            `json:"route_ids"`
	RoutePaths map[string][]string `json:"route_paths"`
}

type PermissionGroupUpdateRequest struct {
	Name       *string             `json:"name,omitempty"`
	RouteIDs   *[]string           `json:"route_ids,omitempty"`
	RoutePaths *map[string][]string `json:"route_paths,omitempty"`
}

func PermissionGroupResponse(g store.PermissionGroup) PermissionGroup {
	createdAt := g.CreatedAt
	updatedAt := g.UpdatedAt
	return PermissionGroup{
		ID:         g.ID,
		Name:       g.Name,
		RouteIDs:   g.RouteIDs,
		RoutePaths: g.RoutePaths,
		CreatedAt:  &createdAt,
		UpdatedAt:  &updatedAt,
	}
}

func PermissionGroupListResponse(groups []store.PermissionGroup) []PermissionGroup {
	result := make([]PermissionGroup, 0, len(groups))
	for _, g := range groups {
		result = append(result, PermissionGroupResponse(g))
	}
	return result
}
