package dto

import (
	"time"

	"github.com/pallyoung/auth-gate/packages/server/internal/store"
)

type Route struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Host        string    `json:"host"`
	PathPrefix  string    `json:"path_prefix"`
	Backend     string    `json:"backend"`
	StripPrefix bool      `json:"strip_prefix"`
	Enabled     bool      `json:"enabled"`
	Priority    int       `json:"priority"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type RouteWriteRequest struct {
	Name        string `json:"name"`
	Host        string `json:"host"`
	PathPrefix  string `json:"path_prefix" binding:"required"`
	Backend     string `json:"backend" binding:"required"`
	StripPrefix bool   `json:"strip_prefix"`
	Enabled     bool   `json:"enabled"`
	Priority    int    `json:"priority"`
}

func RouteResponse(route store.Route) Route {
	return Route{
		ID:          route.ID,
		Name:        route.Name,
		Host:        route.Host,
		PathPrefix:  route.PathPrefix,
		Backend:     route.Backend,
		StripPrefix: route.StripPrefix,
		Enabled:     route.Enabled,
		Priority:    route.Priority,
		CreatedAt:   route.CreatedAt,
		UpdatedAt:   route.UpdatedAt,
	}
}

func RouteListResponse(routes []store.Route) []Route {
	result := make([]Route, 0, len(routes))
	for _, route := range routes {
		result = append(result, RouteResponse(route))
	}
	return result
}
