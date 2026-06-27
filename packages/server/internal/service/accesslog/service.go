package accesslog

import (
	"time"

	"github.com/pallyoung/auth-gate/packages/server/internal/store"
)

// Service provides access log query operations.
type Service struct {
	store *store.AccessLogStore
}

// ListResult contains paginated access log entries.
type ListResult struct {
	Entries    []store.AccessLogEntry `json:"entries"`
	Total      int                    `json:"total"`
	Page       int                    `json:"page"`
	PerPage    int                    `json:"per_page"`
	TotalPages int                    `json:"total_pages"`
}

// NewService creates a new access log service.
func NewService(s *store.AccessLogStore) *Service {
	return &Service{store: s}
}

// List retrieves access log entries with filtering and pagination.
func (s *Service) List(filter store.AccessLogFilter, page, perPage int) (*ListResult, error) {
	if page < 1 {
		page = 1
	}
	if perPage < 1 {
		perPage = 20
	}
	if perPage > 100 {
		perPage = 100
	}

	offset := (page - 1) * perPage
	entries, total := s.store.Query(filter, offset, perPage)

	totalPages := (total + perPage - 1) / perPage
	if totalPages < 1 {
		totalPages = 1
	}

	return &ListResult{
		Entries:    entries,
		Total:      total,
		Page:       page,
		PerPage:    perPage,
		TotalPages: totalPages,
	}, nil
}

// Stats retrieves aggregated statistics for the given duration.
func (s *Service) Stats(duration time.Duration) (*store.AccessLogStats, error) {
	since := time.Now().Add(-duration)
	return s.store.Stats(since)
}

// Aggregate groups access log entries by the specified dimension.
func (s *Service) Aggregate(duration time.Duration, groupBy, sortBy, sortOrder string, page, perPage int) store.AggregateResult {
	if page < 1 {
		page = 1
	}
	if perPage < 1 {
		perPage = 20
	}
	if perPage > 100 {
		perPage = 100
	}

	// Validate groupBy
	switch groupBy {
	case "route_id", "client_ip", "username", "status_code", "auth_result":
		// valid
	default:
		groupBy = "client_ip"
	}

	// Validate sortBy
	switch sortBy {
	case "count", "errors", "avg_latency", "p95_latency":
		// valid
	default:
		sortBy = "count"
	}

	if sortOrder != "asc" {
		sortOrder = "desc"
	}

	since := time.Now().Add(-duration)
	offset := (page - 1) * perPage

	return s.store.Aggregate(store.AggregateFilter{
		Since:     &since,
		GroupBy:   groupBy,
		SortBy:    sortBy,
		SortOrder: sortOrder,
		Offset:    offset,
		Limit:     perPage,
	})
}
