package store

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net"
	"net/netip"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

// AccessLogEntry represents a single access log record.
type AccessLogEntry struct {
	Host             string    `json:"host"`
	RequestID        string    `json:"request_id"`
	RouteID          string    `json:"route_id"`
	RouteName        string    `json:"route_name,omitempty"`
	Method           string    `json:"method"`
	Path             string    `json:"path"`
	BackendURL       string    `json:"backend_url"`
	BackendLatencyMs int64     `json:"backend_latency_ms"`
	StatusCode       int       `json:"status_code"`
	ClientIP         string    `json:"client_ip"`
	UserAgent        string    `json:"user_agent"`
	Username         string    `json:"username,omitempty"`
	AuthResult       string    `json:"auth_result"` // "pass", "fail", "none"
	Timestamp        time.Time `json:"timestamp"`
}

// AccessLogFilter defines query filters for access logs.
type AccessLogFilter struct {
	Host       string
	ClientIP   string
	Path       string
	Username   string
	AuthResult string
	RouteID    string
	StatusCode *int
	StartTime  *time.Time
	EndTime    *time.Time
}

// AccessLogStats contains aggregated statistics.
type AccessLogStats struct {
	AuthFailures          int             `json:"auth_failures"`
	BucketIntervalSeconds int64           `json:"bucket_interval_seconds"`
	PeriodStart           time.Time       `json:"period_start"`
	PeriodEnd             time.Time       `json:"period_end"`
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
	Time  time.Time `json:"time"`
	Count int       `json:"count"`
}

// LatencyBucket represents latency metrics at a specific time.
type LatencyBucket struct {
	Time  time.Time `json:"time"`
	AvgMs float64   `json:"avg_ms"`
	P95Ms int64     `json:"p95_ms"`
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

// AggregateGroup represents aggregated metrics for a single group.
type AggregateGroup struct {
	Key          string    `json:"key"`
	RouteName    string    `json:"route_name,omitempty"` // populated when group_by=route_id
	Count        int       `json:"count"`
	Errors       int       `json:"errors"`
	ErrorRate    float64   `json:"error_rate"`
	AuthFailures int       `json:"auth_failures"`
	AvgLatencyMs float64   `json:"avg_latency_ms"`
	P95LatencyMs int64     `json:"p95_latency_ms"`
	FirstSeen    time.Time `json:"first_seen"`
	LastSeen     time.Time `json:"last_seen"`
}

// AggregateResult holds the paginated result of an aggregation query.
type AggregateResult struct {
	GroupBy     string           `json:"group_by"`
	TotalGroups int              `json:"total_groups"`
	Page        int              `json:"page"`
	PerPage     int              `json:"per_page"`
	TotalPages  int              `json:"total_pages"`
	Groups      []AggregateGroup `json:"groups"`
}

// AggregateFilter defines parameters for aggregation queries.
type AggregateFilter struct {
	Filter    AccessLogFilter
	Since     *time.Time
	GroupBy   string // host, route_id, client_ip, username, status_code, auth_result
	SortBy    string // count, errors, avg_latency, p95_latency
	SortOrder string // asc, desc
	Offset    int
	Limit     int
}

// AccessLogStore is an in-memory ring buffer with periodic file persistence.
type AccessLogStore struct {
	mu         sync.RWMutex
	diskMu     sync.Mutex // acquired before mu; serializes snapshots and purges
	stopOnce   sync.Once
	workers    sync.WaitGroup
	entries    []AccessLogEntry
	writeIndex int
	count      int
	capacity   int
	dir        string
	doneChan   chan struct{}
}

// NewAccessLogStore creates a new access log store with the given capacity.
func NewAccessLogStore(dir string, capacity int) (*AccessLogStore, error) {
	if capacity <= 0 {
		capacity = 10000
	}

	store := &AccessLogStore{
		entries:  make([]AccessLogEntry, capacity),
		capacity: capacity,
		dir:      dir,
		doneChan: make(chan struct{}),
	}

	// Load existing logs from file
	if err := store.load(); err != nil {
		return nil, fmt.Errorf("failed to load access logs: %w", err)
	}

	return store, nil
}

// Record adds a new entry to the ring buffer.
func (s *AccessLogStore) Record(entry AccessLogEntry) {
	entry.Host = NormalizeLogHost(entry.Host)
	s.mu.Lock()
	defer s.mu.Unlock()

	s.entries[s.writeIndex] = entry
	s.writeIndex = (s.writeIndex + 1) % s.capacity
	if s.count < s.capacity {
		s.count++
	}
}

// Query retrieves entries matching the filter with pagination.
func (s *AccessLogStore) Query(filter AccessLogFilter, offset, limit int) ([]AccessLogEntry, int) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Collect newest records first, preserving insertion order for equal timestamps.
	all := s.collectEntries()

	// Apply filters
	filtered := make([]AccessLogEntry, 0, len(all))
	for i := len(all) - 1; i >= 0; i-- {
		entry := all[i]
		if s.matchesFilter(entry, filter) {
			filtered = append(filtered, entry)
		}
	}

	sort.SliceStable(filtered, func(i, j int) bool { return filtered[i].Timestamp.After(filtered[j].Timestamp) })
	total := len(filtered)
	if offset < 0 {
		offset = 0
	}
	if limit <= 0 {
		limit = total
	}

	// Apply pagination
	if offset >= total {
		return []AccessLogEntry{}, total
	}

	end := offset + limit
	if end > total {
		end = total
	}

	return filtered[offset:end], total
}

// Stats retains the legacy duration-based store interface.
func (s *AccessLogStore) Stats(since time.Time) (*AccessLogStats, error) {
	return s.StatsFiltered(AccessLogFilter{StartTime: &since})
}

// StatsFiltered uses the same inclusive filters as list and aggregate.
func (s *AccessLogStore) StatsFiltered(filter AccessLogFilter) (*AccessLogStats, error) {
	if filter.StartTime != nil && filter.EndTime != nil && filter.StartTime.After(*filter.EndTime) {
		return nil, fmt.Errorf("start_time must not be after end_time")
	}
	s.mu.RLock()
	all := s.collectEntries()
	s.mu.RUnlock()
	entries := make([]AccessLogEntry, 0, len(all))
	for _, entry := range all {
		if s.matchesFilter(entry, filter) {
			entries = append(entries, entry)
		}
	}
	end := time.Now()
	if filter.EndTime != nil {
		end = *filter.EndTime
	}
	start := end
	if filter.StartTime != nil {
		start = *filter.StartTime
	} else {
		for _, entry := range entries {
			if entry.Timestamp.Before(start) {
				start = entry.Timestamp
			}
		}
	}
	// For unbounded queries include future-dated retained entries as well.
	if filter.EndTime == nil {
		if start.After(end) {
			end = start
		}
		for _, entry := range entries {
			if entry.Timestamp.After(end) {
				end = entry.Timestamp
			}
		}
	}
	// At most 168 buckets, with a one-minute minimum resolution. The final
	// bucket includes the inclusive end boundary (no extra terminal bucket).
	spanSeconds := logElapsedSeconds(start, end)
	intervalSeconds := int64(math.Ceil(spanSeconds/168/60)) * 60
	if intervalSeconds < 60 {
		intervalSeconds = 60
	}
	count := int(math.Ceil(spanSeconds / float64(intervalSeconds)))
	if count < 1 {
		count = 1
	}
	stats := &AccessLogStats{
		TotalRequests: len(entries), BucketIntervalSeconds: intervalSeconds,
		PeriodStart: start, PeriodEnd: end,
		RequestsPerMinute: make([]TimeBucket, count),
		ErrorRatePerHour:  make([]TimeBucket, count),
		LatencyPerHour:    make([]LatencyBucket, count),
	}
	bucketLatencies := make([][]int64, count)
	for i := 0; i < count; i++ {
		at := time.Unix(start.Unix()+int64(i)*intervalSeconds, int64(start.Nanosecond())).In(start.Location())
		stats.RequestsPerMinute[i].Time = at
		stats.ErrorRatePerHour[i].Time = at
		stats.LatencyPerHour[i].Time = at
	}
	latencies := make([]int64, 0, len(entries))
	for _, entry := range entries {
		index := int(logElapsedSeconds(start, entry.Timestamp) / float64(intervalSeconds))
		if index >= count {
			index = count - 1
		}
		if index < 0 {
			index = 0
		}
		stats.RequestsPerMinute[index].Count++
		if entry.StatusCode >= 400 {
			stats.ErrorCount++
			stats.ErrorRatePerHour[index].Count++
		} else if entry.StatusCode >= 200 {
			stats.SuccessCount++
		}
		if entry.AuthResult == "fail" {
			stats.AuthFailures++
		}
		latencies = append(latencies, entry.BackendLatencyMs)
		bucketLatencies[index] = append(bucketLatencies[index], entry.BackendLatencyMs)
	}
	stats.AvgLatencyMs, stats.P95LatencyMs = latencyMetrics(latencies)
	for i := range bucketLatencies {
		stats.LatencyPerHour[i].AvgMs, stats.LatencyPerHour[i].P95Ms = latencyMetrics(bucketLatencies[i])
	}
	stats.TopPaths = s.computeTopPaths(entries, 10)
	stats.TopIPs = s.computeTopIPs(entries, 10)
	return stats, nil
}

func latencyMetrics(values []int64) (float64, int64) {
	if len(values) == 0 {
		return 0, 0
	}
	var total int64
	for _, value := range values {
		total += value
	}
	sort.Slice(values, func(i, j int) bool { return values[i] < values[j] })
	// Nearest-rank percentile.
	index := int(math.Ceil(float64(len(values))*.95)) - 1
	return float64(total) / float64(len(values)), values[index]
}

// time.Duration saturates for ranges longer than about 292 years. RFC3339
// accepts a wider range, so bucket positioning uses seconds instead.
func logElapsedSeconds(start, end time.Time) float64 {
	return float64(end.Unix()-start.Unix()) + float64(end.Nanosecond()-start.Nanosecond())/1e9
}

// Aggregate groups entries by the specified dimension and computes per-group metrics.
func (s *AccessLogStore) Aggregate(filter AggregateFilter) AggregateResult {
	s.mu.RLock()
	defer s.mu.RUnlock()

	all := s.collectEntries()

	query := filter.Filter
	if query.StartTime == nil {
		query.StartTime = filter.Since
	}
	entries := make([]AccessLogEntry, 0, len(all))
	for _, entry := range all {
		if s.matchesFilter(entry, query) {
			entries = append(entries, entry)
		}
	}

	type bucket struct {
		key       string
		count     int
		errors    int
		authFails int
		latencies []int64
		firstSeen time.Time
		lastSeen  time.Time
	}

	buckets := make(map[string]*bucket)

	for _, entry := range entries {
		var key string
		switch filter.GroupBy {
		case "host":
			key = NormalizeLogHost(entry.Host)
		case "route_id":
			key = entry.RouteID
		case "client_ip":
			key = entry.ClientIP
		case "username":
			key = entry.Username
		case "status_code":
			key = fmt.Sprintf("%d", entry.StatusCode)
		case "auth_result":
			key = entry.AuthResult
		default:
			key = entry.ClientIP
		}

		b, ok := buckets[key]
		if !ok {
			b = &bucket{key: key, firstSeen: entry.Timestamp, lastSeen: entry.Timestamp}
			buckets[key] = b
		}
		b.count++
		if entry.StatusCode >= 400 {
			b.errors++
		}
		if entry.AuthResult == "fail" {
			b.authFails++
		}
		b.latencies = append(b.latencies, entry.BackendLatencyMs)
		if entry.Timestamp.Before(b.firstSeen) {
			b.firstSeen = entry.Timestamp
		}
		if entry.Timestamp.After(b.lastSeen) {
			b.lastSeen = entry.Timestamp
		}
	}

	// Build groups with computed metrics
	groups := make([]AggregateGroup, 0, len(buckets))
	for _, b := range buckets {
		g := AggregateGroup{
			Key:          b.key,
			Count:        b.count,
			Errors:       b.errors,
			AuthFailures: b.authFails,
			FirstSeen:    b.firstSeen,
			LastSeen:     b.lastSeen,
		}
		if b.count > 0 {
			g.ErrorRate = float64(b.errors) * 100 / float64(b.count)
		}
		g.AvgLatencyMs, g.P95LatencyMs = latencyMetrics(b.latencies)
		groups = append(groups, g)
	}

	// Use the key to break ties so pagination remains deterministic.
	sort.Slice(groups, func(i, j int) bool {
		var left, right float64
		switch filter.SortBy {
		case "errors":
			left, right = float64(groups[i].Errors), float64(groups[j].Errors)
		case "avg_latency":
			left, right = groups[i].AvgLatencyMs, groups[j].AvgLatencyMs
		case "p95_latency":
			left, right = float64(groups[i].P95LatencyMs), float64(groups[j].P95LatencyMs)
		default:
			left, right = float64(groups[i].Count), float64(groups[j].Count)
		}
		if left == right {
			return groups[i].Key < groups[j].Key
		}
		if filter.SortOrder == "asc" {
			return left < right
		}
		return left > right
	})

	total := len(groups)
	if filter.Offset < 0 {
		filter.Offset = 0
	}
	page := 1
	if filter.Limit > 0 {
		page = filter.Offset/filter.Limit + 1
	}
	totalPages := 1
	if filter.Limit > 0 {
		totalPages = (total + filter.Limit - 1) / filter.Limit
		if totalPages == 0 {
			totalPages = 1
		}
	}

	// Paginate
	start := filter.Offset
	if start > total {
		start = total
	}
	end := start + filter.Limit
	if end > total || filter.Limit <= 0 {
		end = total
	}

	return AggregateResult{
		GroupBy:     filter.GroupBy,
		TotalGroups: total,
		Page:        page,
		PerPage:     filter.Limit,
		TotalPages:  totalPages,
		Groups:      groups[start:end],
	}
}

// StartFlusher starts a goroutine that periodically flushes entries to disk.
func (s *AccessLogStore) StartFlusher(interval time.Duration) {
	s.workers.Add(1)
	go func() {
		defer s.workers.Done()
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				if err := s.flush(); err != nil {
					log.Printf("access log flush failed: %v", err)
				}
			case <-s.doneChan:
				return
			}
		}
	}()
}

// StopFlusher stops the flusher goroutine and performs a final flush.
func (s *AccessLogStore) StopFlusher() error {
	s.stopOnce.Do(func() { close(s.doneChan) })
	s.workers.Wait()
	return s.flush()
}

// PurgeOlderThan retains the legacy retention behavior and reports IO failures.
func (s *AccessLogStore) PurgeOlderThan(cutoff time.Time) (int, error) {
	return s.PurgeMatching(AccessLogFilter{}, cutoff, &cutoff)
}

func (s *AccessLogStore) CountMatching(filter AccessLogFilter, cutoff time.Time, olderThan *time.Time) int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	count := 0
	for _, entry := range s.collectEntries() {
		if s.matchesPurge(entry, filter, cutoff, olderThan) {
			count++
		}
	}
	return count
}

func (s *AccessLogStore) matchesPurge(entry AccessLogEntry, filter AccessLogFilter, cutoff time.Time, olderThan *time.Time) bool {
	return !entry.Timestamp.After(cutoff) && (olderThan == nil || entry.Timestamp.Before(*olderThan)) && s.matchesFilter(entry, filter)
}

// PurgeMatching persists the survivors before changing memory. Lock ordering
// prevents a concurrent flush from restoring an older snapshot after deletion.
func (s *AccessLogStore) PurgeMatching(filter AccessLogFilter, cutoff time.Time, olderThan *time.Time) (int, error) {
	s.diskMu.Lock()
	defer s.diskMu.Unlock()
	s.mu.Lock()
	defer s.mu.Unlock()
	all := s.collectEntries()
	keep := make([]AccessLogEntry, 0, len(all))
	for _, entry := range all {
		if !s.matchesPurge(entry, filter, cutoff, olderThan) {
			keep = append(keep, entry)
		}
	}
	removed := len(all) - len(keep)
	if err := s.writeToDisk(keep); err != nil {
		return 0, err
	}
	s.entries = make([]AccessLogEntry, s.capacity)
	copy(s.entries, keep)
	s.count = len(keep)
	s.writeIndex = s.count % s.capacity
	return removed, nil
}

// StartCleanup launches a goroutine that purges expired access logs daily
// at midnight. getRetentionDays is called each cycle to read the current
// setting; a value <= 0 disables cleanup.
func (s *AccessLogStore) StartCleanup(getRetentionDays func() int) {
	s.workers.Add(1)
	go func() {
		defer s.workers.Done()
		// Use a 1-hour ticker and only act in the 00:00–00:30 window.
		ticker := time.NewTicker(1 * time.Hour)
		defer ticker.Stop()
		lastRun := ""
		for {
			select {
			case <-ticker.C:
				now := time.Now()
				today := now.Format("2006-01-02")
				// Only run once per day, in the first 30 minutes after midnight
				if now.Hour() != 0 || lastRun == today {
					continue
				}
				lastRun = today
				days := getRetentionDays()
				if days <= 0 {
					continue
				}
				cutoff := now.AddDate(0, 0, -days)
				removed, err := s.PurgeOlderThan(cutoff)
				if err != nil {
					log.Printf("access log cleanup failed: %v", err)
					continue
				}
				if removed > 0 {
					log.Printf("access log cleanup: purged %d entries older than %d days", removed, days)
				}
			case <-s.doneChan:
				return
			}
		}
	}()
}

// flush writes all entries to disk.
func (s *AccessLogStore) flush() error {
	s.diskMu.Lock()
	defer s.diskMu.Unlock()
	s.mu.RLock()
	entries := s.collectEntries()
	s.mu.RUnlock()

	return s.writeToDisk(entries)
}

// writeToDisk atomically writes the given entries to the JSONL file.
// Does not acquire any lock; the caller is responsible for consistency.
func (s *AccessLogStore) writeToDisk(entries []AccessLogEntry) error {
	filePath := filepath.Join(s.dir, "access-logs.jsonl")
	f, err := os.CreateTemp(s.dir, "access-logs-*.tmp")
	if err != nil {
		return fmt.Errorf("create access log temp file: %w", err)
	}
	tmpPath := f.Name()
	defer os.Remove(tmpPath)
	defer f.Close()
	writer := bufio.NewWriter(f)
	encoder := json.NewEncoder(writer)
	for _, entry := range entries {
		if err := encoder.Encode(entry); err != nil {
			return fmt.Errorf("encode access log: %w", err)
		}
	}
	if err := writer.Flush(); err != nil {
		return fmt.Errorf("flush access log: %w", err)
	}
	if err := f.Sync(); err != nil {
		return fmt.Errorf("sync access log: %w", err)
	}
	if err := f.Close(); err != nil {
		return fmt.Errorf("close access log: %w", err)
	}
	if err := os.Rename(tmpPath, filePath); err != nil {
		return fmt.Errorf("replace access log: %w", err)
	}
	return nil
}

// load reads entries from disk.
func (s *AccessLogStore) load() error {
	filePath := filepath.Join(s.dir, "access-logs.jsonl")

	f, err := os.Open(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("failed to open log file: %w", err)
	}
	defer f.Close()

	s.mu.Lock()
	defer s.mu.Unlock()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		var entry AccessLogEntry
		if err := json.Unmarshal(scanner.Bytes(), &entry); err != nil {
			continue
		}

		s.entries[s.writeIndex] = entry
		s.writeIndex = (s.writeIndex + 1) % s.capacity
		if s.count < s.capacity {
			s.count++
		}
	}

	return scanner.Err()
}

// collectEntries returns all entries in chronological order.
func (s *AccessLogStore) collectEntries() []AccessLogEntry {
	if s.count == 0 {
		return nil
	}

	result := make([]AccessLogEntry, 0, s.count)

	if s.count < s.capacity {
		// Buffer not full, entries are at [0, count)
		result = append(result, s.entries[:s.count]...)
	} else {
		// Buffer full, oldest entry is at writeIndex
		result = append(result, s.entries[s.writeIndex:]...)
		result = append(result, s.entries[:s.writeIndex]...)
	}

	return result
}

// matchesFilter checks if an entry matches the given filter.
func (s *AccessLogStore) matchesFilter(entry AccessLogEntry, filter AccessLogFilter) bool {
	if filter.Host != "" && NormalizeLogHost(entry.Host) != NormalizeLogHost(filter.Host) {
		return false
	}
	if filter.ClientIP != "" && entry.ClientIP != filter.ClientIP {
		return false
	}
	if filter.Path != "" && entry.Path != filter.Path {
		return false
	}
	if filter.Username != "" && entry.Username != filter.Username {
		return false
	}
	if filter.AuthResult != "" && entry.AuthResult != filter.AuthResult {
		return false
	}
	if filter.RouteID != "" && entry.RouteID != filter.RouteID {
		return false
	}
	if filter.StatusCode != nil && entry.StatusCode != *filter.StatusCode {
		return false
	}
	if filter.StartTime != nil && entry.Timestamp.Before(*filter.StartTime) {
		return false
	}
	if filter.EndTime != nil && entry.Timestamp.After(*filter.EndTime) {
		return false
	}
	return true
}

// NormalizeLogHost records the incoming hostname, without its transport port.
// Historical entries with no host remain unknown rather than using a route host.
func NormalizeLogHost(host string) string {
	host = strings.ToLower(strings.TrimSpace(host))
	if value, port, err := net.SplitHostPort(host); err == nil {
		if number, err := strconv.Atoi(port); err == nil && number >= 0 && number <= 65535 {
			host = value
		}
	}
	host = strings.TrimSuffix(host, ".")
	if strings.HasPrefix(host, "[") && strings.HasSuffix(host, "]") {
		if address, err := netip.ParseAddr(host[1 : len(host)-1]); err == nil {
			return address.String()
		}
	}
	if address, err := netip.ParseAddr(host); err == nil {
		return address.String()
	}
	return host
}

// computeTopPaths returns the top N paths by request count.
func (s *AccessLogStore) computeTopPaths(entries []AccessLogEntry, n int) []PathCount {
	pathCounts := make(map[string]int)
	for _, entry := range entries {
		pathCounts[entry.Path]++
	}

	counts := make([]PathCount, 0, len(pathCounts))
	for path, count := range pathCounts {
		counts = append(counts, PathCount{Path: path, Count: count})
	}

	sort.Slice(counts, func(i, j int) bool {
		return counts[i].Count > counts[j].Count
	})

	if len(counts) > n {
		counts = counts[:n]
	}

	return counts
}

// computeTopIPs returns the top N IPs by request count.
func (s *AccessLogStore) computeTopIPs(entries []AccessLogEntry, n int) []IPCount {
	ipCounts := make(map[string]int)
	for _, entry := range entries {
		ipCounts[entry.ClientIP]++
	}

	counts := make([]IPCount, 0, len(ipCounts))
	for ip, count := range ipCounts {
		counts = append(counts, IPCount{IP: ip, Count: count})
	}

	sort.Slice(counts, func(i, j int) bool {
		return counts[i].Count > counts[j].Count
	})

	if len(counts) > n {
		counts = counts[:n]
	}

	return counts
}
