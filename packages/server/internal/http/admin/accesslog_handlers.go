package admin

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/pallyoung/auth-gate/packages/server/internal/api/dto"
	"github.com/pallyoung/auth-gate/packages/server/internal/routehost"
	"github.com/pallyoung/auth-gate/packages/server/internal/store"
)

// parseAccessLogFilter is shared by list, statistics, aggregation and cleanup.
func parseAccessLogFilter(params dto.AccessLogFilterParams, duration, defaultDuration string) (store.AccessLogFilter, error) {
	filter := store.AccessLogFilter{
		Host: store.NormalizeLogHost(params.Host), ClientIP: params.ClientIP,
		Path: params.Path, Username: params.Username, AuthResult: params.AuthResult,
		RouteID: params.RouteID, StatusCode: params.StatusCode,
	}
	if params.Host != "" && (filter.Host == "" || !routehost.IsValid(filter.Host)) {
		return filter, fmt.Errorf("invalid host")
	}
	if filter.ClientIP != "" && net.ParseIP(filter.ClientIP) == nil {
		return filter, fmt.Errorf("invalid client_ip")
	}
	for name, value := range map[string]string{"path": filter.Path, "username": filter.Username, "route_id": filter.RouteID} {
		if value != "" && strings.TrimSpace(value) == "" {
			return filter, fmt.Errorf("invalid %s", name)
		}
	}
	switch filter.AuthResult {
	case "", "pass", "fail", "none":
	default:
		return filter, fmt.Errorf("invalid auth_result")
	}
	if filter.StatusCode != nil && (*filter.StatusCode < 100 || *filter.StatusCode > 599) {
		return filter, fmt.Errorf("status_code must be between 100 and 599")
	}
	if params.StartTime != "" {
		at, err := time.Parse(time.RFC3339Nano, params.StartTime)
		if err != nil {
			return filter, fmt.Errorf("invalid start_time: use RFC3339")
		}
		filter.StartTime = &at
	}
	if params.EndTime != "" {
		at, err := time.Parse(time.RFC3339Nano, params.EndTime)
		if err != nil {
			return filter, fmt.Errorf("invalid end_time: use RFC3339")
		}
		filter.EndTime = &at
	}
	if duration == "" && filter.StartTime == nil && filter.EndTime == nil {
		duration = defaultDuration
	}
	if duration != "" && duration != "all" {
		period, err := time.ParseDuration(duration)
		if err != nil || period <= 0 {
			return filter, fmt.Errorf("duration must be a positive duration or all")
		}
		if filter.StartTime == nil && filter.EndTime == nil {
			end := time.Now()
			start := end.Add(-period)
			filter.StartTime, filter.EndTime = &start, &end
		}
	}
	if filter.StartTime != nil && filter.EndTime != nil && filter.StartTime.After(*filter.EndTime) {
		return filter, fmt.Errorf("start_time must not be after end_time")
	}
	return filter, nil
}

func hasAccessLogFilter(filter store.AccessLogFilter) bool {
	return filter.Host != "" || filter.ClientIP != "" || filter.Path != "" || filter.Username != "" ||
		filter.AuthResult != "" || filter.RouteID != "" || filter.StatusCode != nil || filter.StartTime != nil || filter.EndTime != nil
}

func purgeAccessLogs(db store.Store, logs *store.AccessLogStore) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req dto.AccessLogPurgeRequest
		decoder := json.NewDecoder(http.MaxBytesReader(c.Writer, c.Request.Body, 64<<10))
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&req); err != nil {
			writeError(c, http.StatusBadRequest, "invalid_request", "invalid cleanup request")
			return
		}
		if err := decoder.Decode(&struct{}{}); err != io.EOF {
			writeError(c, http.StatusBadRequest, "invalid_request", "expected one cleanup request")
			return
		}
		params := dto.AccessLogFilterParams{}
		if req.Filters != nil {
			params = *req.Filters
		}
		filter, err := parseAccessLogFilter(params, "", "")
		if err != nil {
			writeError(c, http.StatusBadRequest, "invalid_params", err.Error())
			return
		}
		switch req.Mode {
		case "filtered":
			if !hasAccessLogFilter(filter) {
				writeError(c, http.StatusBadRequest, "missing_filter", "filtered cleanup requires at least one filter")
				return
			}
		case "expired", "all":
			if hasAccessLogFilter(filter) {
				writeError(c, http.StatusBadRequest, "invalid_scope", "filters require filtered cleanup mode")
				return
			}
		default:
			writeError(c, http.StatusBadRequest, "invalid_mode", "mode must be expired, filtered or all")
			return
		}
		now := time.Now().UTC()
		cutoff := now
		if !req.Preview || req.CutoffTime != "" {
			cutoff, err = time.Parse(time.RFC3339Nano, req.CutoffTime)
			if err != nil || cutoff.After(now) {
				writeError(c, http.StatusBadRequest, "invalid_cutoff_time", "execution requires a preview cutoff_time that is not in the future")
				return
			}
		}
		var olderThan *time.Time
		retentionDays := 0
		if req.Mode == "expired" {
			value, err := db.GetSetting(settingLogRetentionDays)
			if err != nil {
				writeError(c, http.StatusInternalServerError, "read_failed", "failed to read retention setting")
				return
			}
			retentionDays, _ = strconv.Atoi(value)
			if retentionDays > 0 {
				// Expired previews return the expiration threshold, not the
				// current clock time. Execution clamps to that threshold so a
				// retention policy change cannot expand the previewed scope.
				currentThreshold := now.AddDate(0, 0, -retentionDays)
				if req.Preview || currentThreshold.Before(cutoff) {
					cutoff = currentThreshold
				}
				olderThan = &cutoff
			} else if req.Preview {
				// A disabled preview must remain empty if retention is enabled
				// before execution. Its zero cutoff cannot include stored logs.
				cutoff = time.Time{}
			}
		}
		cutoffText := cutoff.UTC().Format(time.RFC3339Nano)
		if req.Preview {
			matched := 0
			if req.Mode != "expired" || retentionDays > 0 {
				matched = logs.CountMatching(filter, cutoff, olderThan)
			}
			result := gin.H{"matched": matched, "cutoff_time": cutoffText}
			if req.Mode == "expired" {
				result["retention_days"] = retentionDays
			}
			c.JSON(http.StatusOK, result)
			return
		}
		removed := 0
		if req.Mode != "expired" || retentionDays > 0 {
			removed, err = logs.PurgeMatching(filter, cutoff, olderThan)
			if err != nil {
				writeError(c, http.StatusInternalServerError, "purge_failed", "failed to persist access log cleanup")
				return
			}
		}
		c.JSON(http.StatusOK, gin.H{"removed": removed, "cutoff_time": cutoffText})
	}
}
