package dto

// AccessLogEntry represents an access log entry in API responses.
type AccessLogEntry struct {
	RequestID        string `json:"request_id"`
	RouteID          string `json:"route_id"`
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
	TotalRequests      int               `json:"total_requests"`
	SuccessCount       int               `json:"success_count"`
	ErrorCount         int               `json:"error_count"`
	AvgLatencyMs       float64           `json:"avg_latency_ms"`
	P95LatencyMs       int64             `json:"p95_latency_ms"`
	RequestsPerMinute  []TimeBucket      `json:"requests_per_minute"`
	ErrorRatePerHour   []TimeBucket      `json:"error_rate_per_hour"`
	LatencyPerHour     []LatencyBucket   `json:"latency_per_hour"`
	TopPaths           []PathCount       `json:"top_paths"`
	TopIPs             []IPCount         `json:"top_ips"`
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

// AccessLogQueryParams contains query parameters for filtering access logs.
type AccessLogQueryParams struct {
	ClientIP   string `form:"client_ip"`
	Path       string `form:"path"`
	Username   string `form:"username"`
	AuthResult string `form:"auth_result"`
	RouteID    string `form:"route_id"`
	StatusCode *int   `form:"status_code"`
	StartTime  string `form:"start_time"`
	EndTime    string `form:"end_time"`
	Page       int    `form:"page"`
	PerPage    int    `form:"per_page"`
}
