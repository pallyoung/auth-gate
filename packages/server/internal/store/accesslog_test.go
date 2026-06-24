package store

import (
	"fmt"
	"testing"
	"time"
)

func TestAccessLogStore_Record(t *testing.T) {
	store, err := NewAccessLogStore(t.TempDir(), 3)
	if err != nil {
		t.Fatal(err)
	}

	// Record 3 entries
	for i := 0; i < 3; i++ {
		store.Record(AccessLogEntry{
			RequestID:  fmt.Sprintf("req-%d", i),
			StatusCode: 200,
			Timestamp:  time.Now(),
		})
	}

	entries, total := store.Query(AccessLogFilter{}, 0, 10)
	if total != 3 {
		t.Errorf("expected 3 entries, got %d", total)
	}
	if len(entries) != 3 {
		t.Errorf("expected 3 entries, got %d", len(entries))
	}
}

func TestAccessLogStore_RingBufferOverflow(t *testing.T) {
	store, err := NewAccessLogStore(t.TempDir(), 3)
	if err != nil {
		t.Fatal(err)
	}

	// Record 5 entries (more than capacity)
	for i := 0; i < 5; i++ {
		store.Record(AccessLogEntry{
			RequestID:  fmt.Sprintf("req-%d", i),
			StatusCode: 200,
			Timestamp:  time.Now().Add(time.Duration(i) * time.Minute),
		})
	}

	entries, total := store.Query(AccessLogFilter{}, 0, 10)
	if total != 3 {
		t.Errorf("expected 3 entries (capacity), got %d", total)
	}

	// Should have entries 2, 3, 4 (oldest 2 were overwritten)
	if entries[0].RequestID != "req-2" {
		t.Errorf("expected req-2, got %s", entries[0].RequestID)
	}
	if entries[1].RequestID != "req-3" {
		t.Errorf("expected req-3, got %s", entries[1].RequestID)
	}
	if entries[2].RequestID != "req-4" {
		t.Errorf("expected req-4, got %s", entries[2].RequestID)
	}
}

func TestAccessLogStore_QueryFilter(t *testing.T) {
	store, err := NewAccessLogStore(t.TempDir(), 100)
	if err != nil {
		t.Fatal(err)
	}

	now := time.Now()

	// Record entries with different attributes
	store.Record(AccessLogEntry{
		RequestID:  "req-1",
		ClientIP:   "192.168.1.1",
		Path:       "/api/users",
		StatusCode: 200,
		Username:   "alice",
		Timestamp:  now,
	})
	store.Record(AccessLogEntry{
		RequestID:  "req-2",
		ClientIP:   "192.168.1.2",
		Path:       "/api/users",
		StatusCode: 401,
		Username:   "bob",
		Timestamp:  now.Add(1 * time.Minute),
	})
	store.Record(AccessLogEntry{
		RequestID:  "req-3",
		ClientIP:   "192.168.1.1",
		Path:       "/api/orders",
		StatusCode: 200,
		Username:   "alice",
		Timestamp:  now.Add(2 * time.Minute),
	})

	// Test filter by IP
	_, total := store.Query(AccessLogFilter{ClientIP: "192.168.1.1"}, 0, 10)
	if total != 2 {
		t.Errorf("expected 2 entries for IP 192.168.1.1, got %d", total)
	}

	// Test filter by path
	_, total = store.Query(AccessLogFilter{Path: "/api/users"}, 0, 10)
	if total != 2 {
		t.Errorf("expected 2 entries for path /api/users, got %d", total)
	}

	// Test filter by username
	_, total = store.Query(AccessLogFilter{Username: "bob"}, 0, 10)
	if total != 1 {
		t.Errorf("expected 1 entry for user bob, got %d", total)
	}

	// Test filter by status code
	statusCode := 200
	_, total = store.Query(AccessLogFilter{StatusCode: &statusCode}, 0, 10)
	if total != 2 {
		t.Errorf("expected 2 entries for status 200, got %d", total)
	}

	// Test filter by time range
	startTime := now.Add(1 * time.Minute)
	endTime := now.Add(2 * time.Minute)
	_, total = store.Query(AccessLogFilter{
		StartTime: &startTime,
		EndTime:   &endTime,
	}, 0, 10)
	if total != 2 {
		t.Errorf("expected 2 entries in time range, got %d", total)
	}
}

func TestAccessLogStore_QueryPagination(t *testing.T) {
	store, err := NewAccessLogStore(t.TempDir(), 100)
	if err != nil {
		t.Fatal(err)
	}

	// Record 10 entries
	for i := 0; i < 10; i++ {
		store.Record(AccessLogEntry{
			RequestID:  fmt.Sprintf("req-%d", i),
			StatusCode: 200,
			Timestamp:  time.Now().Add(time.Duration(i) * time.Minute),
		})
	}

	// Get first page
	entries, total := store.Query(AccessLogFilter{}, 0, 3)
	if total != 10 {
		t.Errorf("expected total 10, got %d", total)
	}
	if len(entries) != 3 {
		t.Errorf("expected 3 entries, got %d", len(entries))
	}
	if entries[0].RequestID != "req-0" {
		t.Errorf("expected req-0, got %s", entries[0].RequestID)
	}

	// Get second page
	entries, total = store.Query(AccessLogFilter{}, 3, 3)
	if len(entries) != 3 {
		t.Errorf("expected 3 entries, got %d", len(entries))
	}
	if entries[0].RequestID != "req-3" {
		t.Errorf("expected req-3, got %s", entries[0].RequestID)
	}

	// Get last page
	entries, total = store.Query(AccessLogFilter{}, 9, 3)
	if len(entries) != 1 {
		t.Errorf("expected 1 entry, got %d", len(entries))
	}
	if entries[0].RequestID != "req-9" {
		t.Errorf("expected req-9, got %s", entries[0].RequestID)
	}

	// Get page beyond total
	entries, total = store.Query(AccessLogFilter{}, 20, 3)
	if len(entries) != 0 {
		t.Errorf("expected 0 entries, got %d", len(entries))
	}
}

func TestAccessLogStore_Stats(t *testing.T) {
	store, err := NewAccessLogStore(t.TempDir(), 100)
	if err != nil {
		t.Fatal(err)
	}

	now := time.Now()

	// Record entries with different latencies and status codes
	store.Record(AccessLogEntry{
		RequestID:        "req-1",
		Path:             "/api/users",
		ClientIP:         "192.168.1.1",
		StatusCode:       200,
		BackendLatencyMs: 10,
		Timestamp:        now,
	})
	store.Record(AccessLogEntry{
		RequestID:        "req-2",
		Path:             "/api/users",
		ClientIP:         "192.168.1.2",
		StatusCode:       500,
		BackendLatencyMs: 50,
		Timestamp:        now,
	})
	store.Record(AccessLogEntry{
		RequestID:        "req-3",
		Path:             "/api/orders",
		ClientIP:         "192.168.1.1",
		StatusCode:       200,
		BackendLatencyMs: 30,
		Timestamp:        now,
	})

	since := now.Add(-1 * time.Hour)
	stats, err := store.Stats(since)
	if err != nil {
		t.Fatal(err)
	}

	if stats.TotalRequests != 3 {
		t.Errorf("expected 3 total requests, got %d", stats.TotalRequests)
	}
	if stats.SuccessCount != 2 {
		t.Errorf("expected 2 success count, got %d", stats.SuccessCount)
	}
	if stats.ErrorCount != 1 {
		t.Errorf("expected 1 error count, got %d", stats.ErrorCount)
	}

	// Average latency: (10 + 50 + 30) / 3 = 30
	if stats.AvgLatencyMs != 30 {
		t.Errorf("expected avg latency 30, got %f", stats.AvgLatencyMs)
	}

	// P95 latency: sorted [10, 30, 50], P95 index = 2, value = 50
	if stats.P95LatencyMs != 50 {
		t.Errorf("expected P95 latency 50, got %d", stats.P95LatencyMs)
	}

	// Top paths
	if len(stats.TopPaths) != 2 {
		t.Errorf("expected 2 top paths, got %d", len(stats.TopPaths))
	}
	if stats.TopPaths[0].Path != "/api/users" {
		t.Errorf("expected top path /api/users, got %s", stats.TopPaths[0].Path)
	}
	if stats.TopPaths[0].Count != 2 {
		t.Errorf("expected top path count 2, got %d", stats.TopPaths[0].Count)
	}

	// Top IPs
	if len(stats.TopIPs) != 2 {
		t.Errorf("expected 2 top IPs, got %d", len(stats.TopIPs))
	}
}

func TestAccessLogStore_Persistence(t *testing.T) {
	dir := t.TempDir()

	// Create store and record entries
	store1, err := NewAccessLogStore(dir, 100)
	if err != nil {
		t.Fatal(err)
	}

	store1.Record(AccessLogEntry{
		RequestID:  "req-1",
		StatusCode: 200,
		Timestamp:  time.Now(),
	})
	store1.Record(AccessLogEntry{
		RequestID:  "req-2",
		StatusCode: 401,
		Timestamp:  time.Now(),
	})

	// Flush to disk
	if err := store1.flush(); err != nil {
		t.Fatal(err)
	}

	// Create new store and verify it loads the entries
	store2, err := NewAccessLogStore(dir, 100)
	if err != nil {
		t.Fatal(err)
	}

	entries, total := store2.Query(AccessLogFilter{}, 0, 10)
	if total != 2 {
		t.Errorf("expected 2 entries loaded, got %d", total)
	}
	if entries[0].RequestID != "req-1" {
		t.Errorf("expected req-1, got %s", entries[0].RequestID)
	}
	if entries[1].RequestID != "req-2" {
		t.Errorf("expected req-2, got %s", entries[1].RequestID)
	}
}

func TestAccessLogStore_PersistenceRingBuffer(t *testing.T) {
	dir := t.TempDir()

	// Create store with capacity 3
	store1, err := NewAccessLogStore(dir, 3)
	if err != nil {
		t.Fatal(err)
	}

	// Record 5 entries (exceeds capacity)
	for i := 0; i < 5; i++ {
		store1.Record(AccessLogEntry{
			RequestID:  fmt.Sprintf("req-%d", i),
			StatusCode: 200,
			Timestamp:  time.Now(),
		})
	}

	// Flush to disk
	if err := store1.flush(); err != nil {
		t.Fatal(err)
	}

	// Create new store
	store2, err := NewAccessLogStore(dir, 3)
	if err != nil {
		t.Fatal(err)
	}

	entries, total := store2.Query(AccessLogFilter{}, 0, 10)
	if total != 3 {
		t.Errorf("expected 3 entries loaded (capacity), got %d", total)
	}

	// Should have entries 2, 3, 4
	if entries[0].RequestID != "req-2" {
		t.Errorf("expected req-2, got %s", entries[0].RequestID)
	}
	if entries[1].RequestID != "req-3" {
		t.Errorf("expected req-3, got %s", entries[1].RequestID)
	}
	if entries[2].RequestID != "req-4" {
		t.Errorf("expected req-4, got %s", entries[2].RequestID)
	}
}
