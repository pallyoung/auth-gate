package store

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"
)

// AccessLogEntry represents a single access log record.
type AccessLogEntry struct {
	RequestID        string    `json:"request_id"`
	RouteID          string    `json:"route_id"`
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
	TotalRequests      int             `json:"total_requests"`
	SuccessCount       int             `json:"success_count"`
	ErrorCount         int             `json:"error_count"`
	AvgLatencyMs       float64         `json:"avg_latency_ms"`
	P95LatencyMs       int64           `json:"p95_latency_ms"`
	RequestsPerMinute  []TimeBucket    `json:"requests_per_minute"`
	ErrorRatePerHour   []TimeBucket    `json:"error_rate_per_hour"`
	LatencyPerHour     []LatencyBucket `json:"latency_per_hour"`
	TopPaths           []PathCount     `json:"top_paths"`
	TopIPs             []IPCount       `json:"top_ips"`
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

// AccessLogStore is an in-memory ring buffer with periodic file persistence.
type AccessLogStore struct {
	mu         sync.RWMutex
	entries    []AccessLogEntry
	writeIndex int
	count      int
	capacity   int
	dir        string
	flushChan  chan struct{}
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
		flushChan: make(chan struct{}, 1),
		doneChan:  make(chan struct{}),
	}

	// Load existing logs from file
	if err := store.load(); err != nil {
		return nil, fmt.Errorf("failed to load access logs: %w", err)
	}

	return store, nil
}

// Record adds a new entry to the ring buffer.
func (s *AccessLogStore) Record(entry AccessLogEntry) {
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

	// Collect all entries in chronological order
	all := s.collectEntries()

	// Apply filters
	filtered := make([]AccessLogEntry, 0, len(all))
	for _, entry := range all {
		if s.matchesFilter(entry, filter) {
			filtered = append(filtered, entry)
		}
	}

	total := len(filtered)

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

// Stats computes aggregated statistics for entries since the given time.
func (s *AccessLogStore) Stats(since time.Time) (*AccessLogStats, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	all := s.collectEntries()

	// Filter by time
	entries := make([]AccessLogEntry, 0, len(all))
	for _, entry := range all {
		if entry.Timestamp.After(since) || entry.Timestamp.Equal(since) {
			entries = append(entries, entry)
		}
	}

	if len(entries) == 0 {
		return &AccessLogStats{}, nil
	}

	stats := &AccessLogStats{
		TotalRequests: len(entries),
	}

	// Compute success/error counts and latency
	latencies := make([]int64, 0, len(entries))
	for _, entry := range entries {
		if entry.StatusCode >= 200 && entry.StatusCode < 400 {
			stats.SuccessCount++
		} else {
			stats.ErrorCount++
		}
		latencies = append(latencies, entry.BackendLatencyMs)
	}

	// Average latency
	var totalLatency int64
	for _, l := range latencies {
		totalLatency += l
	}
	stats.AvgLatencyMs = float64(totalLatency) / float64(len(latencies))

	// P95 latency
	sort.Slice(latencies, func(i, j int) bool { return latencies[i] < latencies[j] })
	p95Index := int(float64(len(latencies)) * 0.95)
	if p95Index >= len(latencies) {
		p95Index = len(latencies) - 1
	}
	stats.P95LatencyMs = latencies[p95Index]

	// Requests per minute (last 60 minutes)
	stats.RequestsPerMinute = s.computeRequestsPerMinute(entries)

	// Error rate per hour (last 24 hours)
	stats.ErrorRatePerHour = s.computeErrorRatePerHour(entries)

	// Latency per hour (last 24 hours)
	stats.LatencyPerHour = s.computeLatencyPerHour(entries)

	// Top paths
	stats.TopPaths = s.computeTopPaths(entries, 10)

	// Top IPs
	stats.TopIPs = s.computeTopIPs(entries, 10)

	return stats, nil
}

// StartFlusher starts a goroutine that periodically flushes entries to disk.
func (s *AccessLogStore) StartFlusher(interval time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				s.flush()
			case <-s.doneChan:
				return
			}
		}
	}()
}

// StopFlusher stops the flusher goroutine and performs a final flush.
func (s *AccessLogStore) StopFlusher() {
	close(s.doneChan)
	s.flush()
}

// PurgeOlderThan removes all entries with a Timestamp before cutoff.
// It returns the number of entries removed and flushes the result to disk.
func (s *AccessLogStore) PurgeOlderThan(cutoff time.Time) int {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.count == 0 {
		return 0
	}

	// Collect surviving entries
	all := s.collectEntries()
	keep := make([]AccessLogEntry, 0, len(all))
	for _, e := range all {
		if !e.Timestamp.Before(cutoff) {
			keep = append(keep, e)
		}
	}
	removed := len(all) - len(keep)
	if removed == 0 {
		return 0
	}

	// Rebuild the ring buffer with surviving entries
	s.entries = make([]AccessLogEntry, s.capacity)
	s.writeIndex = 0
	s.count = 0
	for _, e := range keep {
		s.entries[s.writeIndex] = e
		s.writeIndex = (s.writeIndex + 1) % s.capacity
		if s.count < s.capacity {
			s.count++
		}
	}

	// Persist immediately (collectEntries works on the rebuilt buffer)
	s.writeToDisk(s.collectEntries())

	return removed
}

// StartCleanup launches a goroutine that purges expired access logs daily
// at midnight. getRetentionDays is called each cycle to read the current
// setting; a value <= 0 disables cleanup.
func (s *AccessLogStore) StartCleanup(getRetentionDays func() int) {
	go func() {
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
				removed := s.PurgeOlderThan(cutoff)
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
	s.mu.RLock()
	entries := s.collectEntries()
	s.mu.RUnlock()

	return s.writeToDisk(entries)
}

// writeToDisk atomically writes the given entries to the JSONL file.
// Does not acquire any lock; the caller is responsible for consistency.
func (s *AccessLogStore) writeToDisk(entries []AccessLogEntry) error {
	if len(entries) == 0 {
		return nil
	}

	filePath := filepath.Join(s.dir, "access-logs.jsonl")
	tmpPath := filePath + ".tmp"

	f, err := os.Create(tmpPath)
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	defer f.Close()

	writer := bufio.NewWriter(f)
	for _, entry := range entries {
		data, err := json.Marshal(entry)
		if err != nil {
			continue
		}
		writer.Write(data)
		writer.WriteByte('\n')
	}

	if err := writer.Flush(); err != nil {
		return fmt.Errorf("failed to flush writer: %w", err)
	}

	if err := f.Close(); err != nil {
		return fmt.Errorf("failed to close file: %w", err)
	}

	return os.Rename(tmpPath, filePath)
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

// computeRequestsPerMinute computes request counts per minute for the last 60 minutes.
func (s *AccessLogStore) computeRequestsPerMinute(entries []AccessLogEntry) []TimeBucket {
	now := time.Now()
	buckets := make([]TimeBucket, 60)

	// Initialize buckets
	for i := 0; i < 60; i++ {
		buckets[i] = TimeBucket{
			Time:  now.Add(time.Duration(-59+i) * time.Minute).Truncate(time.Minute),
			Count: 0,
		}
	}

	// Count entries
	for _, entry := range entries {
		entryTime := entry.Timestamp.Truncate(time.Minute)
		for i := 0; i < 60; i++ {
			if entryTime.Equal(buckets[i].Time) {
				buckets[i].Count++
				break
			}
		}
	}

	return buckets
}

// computeErrorRatePerHour computes error counts per hour for the last 24 hours.
func (s *AccessLogStore) computeErrorRatePerHour(entries []AccessLogEntry) []TimeBucket {
	now := time.Now()
	buckets := make([]TimeBucket, 24)

	// Initialize buckets
	for i := 0; i < 24; i++ {
		buckets[i] = TimeBucket{
			Time:  now.Add(time.Duration(-23+i) * time.Hour).Truncate(time.Hour),
			Count: 0,
		}
	}

	// Count error entries
	for _, entry := range entries {
		if entry.StatusCode >= 400 {
			entryTime := entry.Timestamp.Truncate(time.Hour)
			for i := 0; i < 24; i++ {
				if entryTime.Equal(buckets[i].Time) {
					buckets[i].Count++
					break
				}
			}
		}
	}

	return buckets
}

// computeLatencyPerHour computes average and P95 latency per hour for the last 24 hours.
func (s *AccessLogStore) computeLatencyPerHour(entries []AccessLogEntry) []LatencyBucket {
	now := time.Now()
	type bucketData struct {
		latencies []int64
	}
	bucketMap := make(map[time.Time]*bucketData)

	// Initialize buckets
	for i := 0; i < 24; i++ {
		t := now.Add(time.Duration(-23+i) * time.Hour).Truncate(time.Hour)
		bucketMap[t] = &bucketData{}
	}

	// Collect latencies
	for _, entry := range entries {
		entryTime := entry.Timestamp.Truncate(time.Hour)
		if bd, ok := bucketMap[entryTime]; ok {
			bd.latencies = append(bd.latencies, entry.BackendLatencyMs)
		}
	}

	// Compute stats
	buckets := make([]LatencyBucket, 24)
	for i := 0; i < 24; i++ {
		t := now.Add(time.Duration(-23+i) * time.Hour).Truncate(time.Hour)
		bd := bucketMap[t]

		bucket := LatencyBucket{Time: t}
		if len(bd.latencies) > 0 {
			var total int64
			for _, l := range bd.latencies {
				total += l
			}
			bucket.AvgMs = float64(total) / float64(len(bd.latencies))

			sort.Slice(bd.latencies, func(i, j int) bool { return bd.latencies[i] < bd.latencies[j] })
			p95Index := int(float64(len(bd.latencies)) * 0.95)
			if p95Index >= len(bd.latencies) {
				p95Index = len(bd.latencies) - 1
			}
			bucket.P95Ms = bd.latencies[p95Index]
		}
		buckets[i] = bucket
	}

	return buckets
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
