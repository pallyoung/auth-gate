package store

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

func TestAccessLogPurgeEmptySurvivesRestart(t *testing.T) {
	dir := t.TempDir()
	s, err := NewAccessLogStore(dir, 10)
	if err != nil {
		t.Fatal(err)
	}
	s.Record(AccessLogEntry{RequestID: "expired", Timestamp: time.Now().Add(-48 * time.Hour)})
	if err := s.flush(); err != nil {
		t.Fatal(err)
	}
	if removed, err := s.PurgeOlderThan(time.Now()); err != nil || removed != 1 {
		t.Fatalf("purge = %d, %v", removed, err)
	}
	reopened, err := NewAccessLogStore(dir, 10)
	if err != nil {
		t.Fatal(err)
	}
	_, total := reopened.Query(AccessLogFilter{}, 0, 10)
	if total != 0 {
		t.Fatalf("purged logs reappeared after restart: %d", total)
	}
}

func TestAccessLogPurgePersistenceFailuresPreserveMemory(t *testing.T) {
	for _, failure := range []string{"create", "encode", "replace"} {
		t.Run(failure, func(t *testing.T) {
			dir := t.TempDir()
			s, err := NewAccessLogStore(dir, 10)
			if err != nil {
				t.Fatal(err)
			}
			now := time.Now()
			s.Record(AccessLogEntry{RequestID: "remove", Timestamp: now.Add(-time.Hour)})
			s.Record(AccessLogEntry{RequestID: "keep", Timestamp: now.Add(time.Hour)})
			switch failure {
			case "create":
				s.dir = filepath.Join(dir, "missing")
			case "encode":
				s.Record(AccessLogEntry{RequestID: "unencodable", Timestamp: time.Date(10000, 1, 1, 0, 0, 0, 0, time.UTC)})
			case "replace":
				if err := os.Mkdir(filepath.Join(dir, "access-logs.jsonl"), 0700); err != nil {
					t.Fatal(err)
				}
			}
			_, before := s.Query(AccessLogFilter{}, 0, 10)
			removed, err := s.PurgeOlderThan(now)
			if err == nil || removed != 0 {
				t.Fatalf("purge = %d, %v; want error and no removal", removed, err)
			}
			_, after := s.Query(AccessLogFilter{}, 0, 10)
			if after != before {
				t.Fatalf("memory changed on persistence error: %d -> %d", before, after)
			}
			temps, err := filepath.Glob(filepath.Join(dir, "access-logs-*.tmp"))
			if err != nil || len(temps) != 0 {
				t.Fatalf("temporary files leaked: %v (%v)", temps, err)
			}
		})
	}
}

func TestAccessLogConcurrentFlushPurgeNeverRestoresSnapshot(t *testing.T) {
	dir := t.TempDir()
	s, err := NewAccessLogStore(dir, 100)
	if err != nil {
		t.Fatal(err)
	}
	cutoff := time.Now()
	for i := 0; i < 50; i++ {
		s.Record(AccessLogEntry{Timestamp: cutoff.Add(-time.Hour)})
	}
	s.Record(AccessLogEntry{RequestID: "new", Timestamp: cutoff.Add(time.Hour)})
	if err := s.flush(); err != nil {
		t.Fatal(err)
	}
	var workers sync.WaitGroup
	for i := 0; i < 20; i++ {
		workers.Add(2)
		go func() {
			defer workers.Done()
			if err := s.flush(); err != nil {
				t.Error(err)
			}
		}()
		go func() {
			defer workers.Done()
			if _, err := s.PurgeMatching(AccessLogFilter{}, cutoff, nil); err != nil {
				t.Error(err)
			}
		}()
	}
	workers.Wait()
	reopened, err := NewAccessLogStore(dir, 100)
	if err != nil {
		t.Fatal(err)
	}
	entries, total := reopened.Query(AccessLogFilter{}, 0, 100)
	if total != 1 || entries[0].RequestID != "new" {
		t.Fatalf("stale flush restored purged logs: %+v", entries)
	}
}

func TestAccessLogSharedFiltersInclusiveAndHostNormalization(t *testing.T) {
	s, err := NewAccessLogStore(t.TempDir(), 20)
	if err != nil {
		t.Fatal(err)
	}
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	end := start.Add(7 * 24 * time.Hour)
	entry := AccessLogEntry{Host: "EXAMPLE.COM.:8443", ClientIP: "192.0.2.1", Path: "/api", Username: "alice", AuthResult: "fail", RouteID: "route-1", StatusCode: 403, BackendLatencyMs: 20, Timestamp: start}
	s.Record(entry)
	entry.Timestamp = end
	s.Record(entry)
	for _, change := range []func(*AccessLogEntry){
		func(e *AccessLogEntry) { e.Host = "other.example" },
		func(e *AccessLogEntry) { e.ClientIP = "192.0.2.2" },
		func(e *AccessLogEntry) { e.Path = "/other" },
		func(e *AccessLogEntry) { e.Username = "bob" },
		func(e *AccessLogEntry) { e.AuthResult = "pass" },
		func(e *AccessLogEntry) { e.RouteID = "route-2" },
		func(e *AccessLogEntry) { e.StatusCode = 200 },
		func(e *AccessLogEntry) { e.Timestamp = start.Add(-time.Nanosecond) },
		func(e *AccessLogEntry) { e.Timestamp = end.Add(time.Nanosecond) },
	} {
		other := entry
		change(&other)
		s.Record(other)
	}
	status := 403
	filter := AccessLogFilter{Host: "Example.Com:443", ClientIP: "192.0.2.1", Path: "/api", Username: "alice", AuthResult: "fail", RouteID: "route-1", StatusCode: &status, StartTime: &start, EndTime: &end}
	entries, total := s.Query(filter, 0, 20)
	if total != 2 || !entries[0].Timestamp.Equal(end) {
		t.Fatalf("list filter/order = %+v", entries)
	}
	stats, err := s.StatsFiltered(filter)
	if err != nil {
		t.Fatal(err)
	}
	if stats.TotalRequests != total || stats.AuthFailures != 2 || stats.ErrorCount != 2 {
		t.Fatalf("stats differ from list: %+v", stats)
	}
	groups := s.Aggregate(AggregateFilter{Filter: filter, GroupBy: "host", Limit: 10})
	if len(groups.Groups) != 1 || groups.Groups[0].Count != total || groups.Groups[0].Key != "example.com" {
		t.Fatalf("aggregate differs from list: %+v", groups)
	}
	if len(stats.RequestsPerMinute) > 168 || len(stats.RequestsPerMinute) == 0 {
		t.Fatalf("bucket count = %d", len(stats.RequestsPerMinute))
	}
	if stats.RequestsPerMinute[0].Count != 1 || stats.RequestsPerMinute[len(stats.RequestsPerMinute)-1].Count != 1 {
		t.Fatal("inclusive boundary entries missing from graph")
	}
	if !stats.PeriodStart.Equal(start) || !stats.PeriodEnd.Equal(end) {
		t.Fatal("graph period differs from selected range")
	}
}

func TestAccessLogEmptyStatsAndAggregateHaveUsableShape(t *testing.T) {
	s, err := NewAccessLogStore(t.TempDir(), 10)
	if err != nil {
		t.Fatal(err)
	}
	start, end := time.Now().Add(-24*time.Hour), time.Now()
	stats, err := s.StatsFiltered(AccessLogFilter{StartTime: &start, EndTime: &end})
	if err != nil {
		t.Fatal(err)
	}
	if len(stats.RequestsPerMinute) == 0 || stats.TopIPs == nil || stats.TopPaths == nil {
		t.Fatalf("empty chart shape = %+v", stats)
	}
	result := s.Aggregate(AggregateFilter{GroupBy: "host"})
	if result.Groups == nil || result.Page != 1 {
		t.Fatalf("empty aggregate = %+v", result)
	}
}

func TestAccessLogStatsExtremeRFC3339RangeRemainsBoundedAndComplete(t *testing.T) {
	s, err := NewAccessLogStore(t.TempDir(), 10)
	if err != nil {
		t.Fatal(err)
	}
	start := time.Date(1, 1, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(9999, 12, 31, 23, 59, 59, 0, time.UTC)
	s.Record(AccessLogEntry{Timestamp: end.Add(-time.Hour), StatusCode: 200})
	stats, err := s.StatsFiltered(AccessLogFilter{StartTime: &start, EndTime: &end})
	if err != nil {
		t.Fatal(err)
	}
	count := len(stats.RequestsPerMinute)
	if count == 0 || count > 168 {
		t.Fatalf("bucket count = %d", count)
	}
	last := stats.RequestsPerMinute[count-1].Time
	if last.Unix()+stats.BucketIntervalSeconds < end.Unix() {
		t.Fatalf("buckets end before selected range: last=%s interval=%d end=%s", last, stats.BucketIntervalSeconds, end)
	}
	if stats.RequestsPerMinute[count-1].Count != 1 {
		t.Fatal("last entry missing")
	}
}

func TestAccessLogStatsBucketsIncludeWholeSelectedPeriod(t *testing.T) {
	s, err := NewAccessLogStore(t.TempDir(), 10)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	s.Record(AccessLogEntry{Timestamp: now.Add(-6 * 24 * time.Hour), StatusCode: 500})
	s.Record(AccessLogEntry{Timestamp: now.Add(-30 * time.Minute), StatusCode: 200})
	stats, err := s.Stats(now.Add(-7 * 24 * time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	total, errors := 0, 0
	for _, bucket := range stats.RequestsPerMinute {
		total += bucket.Count
	}
	for _, bucket := range stats.ErrorRatePerHour {
		errors += bucket.Count
	}
	if total != 2 || errors != 1 {
		t.Fatalf("buckets dropped selected logs: requests=%d errors=%d", total, errors)
	}
}
