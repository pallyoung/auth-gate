package dto

// AccessLogEntry represents an access log entry in API responses.
type AccessLogEntry struct {
	Host             string `json:"host"`
	RequestID        string `json:"request_id"`
	RouteID          string `json:"route_id"`
	RouteName        string `json:"route_name,omitempty"`
	Method           string `json:"method"`
	Path             string `json:"path"`
	BackendURL       string `json:"backend_url"`
	BackendLatencyMs int64  `json:"backend_latency_ms"`
	StatusCode       int    `json:"status_code"`
	ClientIP         string `json:"client_ip"`
	UserAgent        string `json:"user_agent"`
	Username         string `json:"username,omitempty"`
	AuthResult       string `json:"auth_result"`
	Timestamp        string `json:"timestamp"`
}

// AccessLogListResponse is the response for listing access logs.
type AccessLogListResponse struct {
	Entries    []AccessLogEntry `json:"entries"`
	Total      int              `json:"total"`
	Page       int              `json:"page"`
	PerPage    int              `json:"per_page"`
	TotalPages int              `json:"total_pages"`
}

// AccessLogStatsResponse contains aggregated statistics.
type AccessLogStatsResponse struct {
	AuthFailures          int             `json:"auth_failures"`
	BucketIntervalSeconds int64           `json:"bucket_interval_seconds"`
	PeriodStart           string          `json:"period_start"`
	PeriodEnd             string          `json:"period_end"`
	TotalRequests         int             `json:"total_requests"`
	SuccessCount          int             `json:"success_count"`
	ErrorCount            int             `json:"error_count"`
	AvgLatencyMs          float64         `json:"avg_latency_ms"`
	P95LatencyMs          int64           `json:"p95_latency_ms"`
	RequestsPerMinute     []TimeBucket    `json:"requests_per_minute"`
	ErrorRatePerHour      []TimeBucket    `json:"error_rate_per_hour"`
	LatencyPerHour        []LatencyBucket `json:"latency_per_hour"`
	TopPaths              []PathCount     `json:"top_paths"`
	TopIPs                []IPCount       `json:"top_ips"`
}

// TimeBucket represents a count at a specific time.
type TimeBucket struct {
	Time  string `json:"time"`
	Count int    `json:"count"`
}

// LatencyBucket represents latency metrics at a specific time.
type LatencyBucket struct {
	Time  string  `json:"time"`
	AvgMs float64 `json:"avg_ms"`
	P95Ms int64   `json:"p95_ms"`
}

// PathCount represents request count for a path.
type PathCount struct {
	Path  string `json:"path"`
	Count int    `json:"count"`
}

// IPCount represents request count for an IP.
type IPCount struct {
	IP    string `json:"ip"`
	Count int    `json:"count"`
}

// AccessLogFilterParams is shared by log queries and cleanup requests.
type AccessLogFilterParams struct {
	Host       string `form:"host" json:"host"`
	ClientIP   string `form:"client_ip" json:"client_ip"`
	Path       string `form:"path" json:"path"`
	Username   string `form:"username" json:"username"`
	AuthResult string `form:"auth_result" json:"auth_result"`
	RouteID    string `form:"route_id" json:"route_id"`
	StatusCode *int   `form:"status_code" json:"status_code"`
	StartTime  string `form:"start_time" json:"start_time"`
	EndTime    string `form:"end_time" json:"end_time"`
}

type AccessLogQueryParams struct {
	AccessLogFilterParams
	Duration string `form:"duration"`
	Page     int    `form:"page"`
	PerPage  int    `form:"per_page"`
}

// AccessLogAggregateQueryParams contains query parameters for log aggregation.
type AccessLogAggregateQueryParams struct {
	AccessLogFilterParams
	GroupBy   string `form:"group_by" binding:"required"`
	Duration  string `form:"duration"`
	SortBy    string `form:"sort_by"`
	SortOrder string `form:"sort_order"`
	Page      int    `form:"page"`
	PerPage   int    `form:"per_page"`
}

type AccessLogPurgeRequest struct {
	Mode       string                 `json:"mode"`
	Filters    *AccessLogFilterParams `json:"filters"`
	Preview    bool                   `json:"preview"`
	CutoffTime string                 `json:"cutoff_time"`
}

// AccessLogAggregateGroup represents aggregated metrics for a single group.
type AccessLogAggregateGroup struct {
	Key          string  `json:"key"`
	RouteName    string  `json:"route_name,omitempty"`
	Count        int     `json:"count"`
	Errors       int     `json:"errors"`
	ErrorRate    float64 `json:"error_rate"`
	AuthFailures int     `json:"auth_failures"`
	AvgLatencyMs float64 `json:"avg_latency_ms"`
	P95LatencyMs int64   `json:"p95_latency_ms"`
	FirstSeen    string  `json:"first_seen"`
	LastSeen     string  `json:"last_seen"`
}

// AccessLogAggregateResponse is the response for log aggregation.
type AccessLogAggregateResponse struct {
	GroupBy     string                    `json:"group_by"`
	Duration    string                    `json:"duration"`
	TotalGroups int                       `json:"total_groups"`
	Page        int                       `json:"page"`
	PerPage     int                       `json:"per_page"`
	TotalPages  int                       `json:"total_pages"`
	Groups      []AccessLogAggregateGroup `json:"groups"`
}
