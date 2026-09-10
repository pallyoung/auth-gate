package admin

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/pallyoung/auth-gate/packages/server/internal/auth"
	"github.com/pallyoung/auth-gate/packages/server/internal/router"
	"github.com/pallyoung/auth-gate/packages/server/internal/store"
)

func accessLogTestAPI(t *testing.T) (*gin.Engine, store.Store, *store.AccessLogStore, map[string]string, string) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	auth.ConfigureJWTSecret("access-log-test-secret")
	db := newTestDB(t)
	dir := t.TempDir()
	logs, err := store.NewAccessLogStore(dir, 100)
	if err != nil {
		t.Fatal(err)
	}
	tokens := map[string]string{}
	for _, role := range []string{store.RoleAdmin, store.RoleEditor, store.RoleViewer} {
		user := seedUser(t, db, role, "password123", role)
		token, err := auth.GenerateToken(user.ID, user.Username, role)
		if err != nil {
			t.Fatal(err)
		}
		tokens[role] = token
	}
	engine := gin.New()
	group := engine.Group("/api")
	group.Use(auth.AuthMiddleware(db))
	RegisterRoutes(group, router.NewManager(db), db, nil, nil, logs, nil, nil)
	return engine, db, logs, tokens, dir
}

func decodeLogJSON(t *testing.T, resp *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var value map[string]any
	if err := json.Unmarshal(resp.Body.Bytes(), &value); err != nil {
		t.Fatal(err)
	}
	return value
}

func TestAccessLogReadEndpointsShareFiltersAndValidation(t *testing.T) {
	engine, _, logs, tokens, _ := accessLogTestAPI(t)
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	end := start.Add(7 * 24 * time.Hour)
	for _, stamp := range []time.Time{start, end} {
		logs.Record(store.AccessLogEntry{Host: "example.com", Path: "/api", AuthResult: "fail", StatusCode: 302, Timestamp: stamp})
		logs.Record(store.AccessLogEntry{Host: "elsewhere.example", Path: "/api", AuthResult: "fail", StatusCode: 302, Timestamp: stamp})
	}
	params := url.Values{"duration": {"all"}, "host": {"EXAMPLE.COM.:443"}, "path": {"/api"}, "auth_result": {"fail"}, "status_code": {"302"}, "start_time": {start.Format(time.RFC3339)}, "end_time": {end.Format(time.RFC3339)}}
	for _, endpoint := range []string{"/access-logs", "/access-logs/stats", "/access-logs/aggregate"} {
		query := params.Encode()
		if strings.HasSuffix(endpoint, "aggregate") {
			query += "&group_by=host"
		}
		resp := performRequest(t, engine, http.MethodGet, "/api"+endpoint+"?"+query, tokens[store.RoleViewer], nil)
		if resp.Code != http.StatusOK {
			t.Fatalf("%s: %d %s", endpoint, resp.Code, resp.Body)
		}
		body := decodeLogJSON(t, resp)
		switch endpoint {
		case "/access-logs":
			if body["total"] != float64(2) {
				t.Fatalf("list = %v", body)
			}
			entry := body["entries"].([]any)[0].(map[string]any)
			if entry["host"] != "example.com" || entry["timestamp"] != end.Format(time.RFC3339) {
				t.Fatalf("latest entry = %v", entry)
			}
		case "/access-logs/stats":
			if body["total_requests"] != float64(2) || body["auth_failures"] != float64(2) || body["error_count"] != float64(0) {
				t.Fatalf("stats = %v", body)
			}
		case "/access-logs/aggregate":
			groups := body["groups"].([]any)
			if len(groups) != 1 || groups[0].(map[string]any)["key"] != "example.com" || groups[0].(map[string]any)["count"] != float64(2) {
				t.Fatalf("aggregate = %v", body)
			}
		}
		for _, invalid := range []string{
			"start_time=bad", "end_time=bad", "start_time=2026-02-01T00%3A00%3A00Z&end_time=2026-01-01T00%3A00%3A00Z",
			"status_code=abc", "status_code=99", "auth_result=anything", "host=https%3A%2F%2Fexample.com", "host=%20", "duration=-1h", "client_ip=bad",
		} {
			resp := performRequest(t, engine, http.MethodGet, "/api"+endpoint+"?group_by=host&"+invalid, tokens[store.RoleViewer], nil)
			if resp.Code != http.StatusBadRequest {
				t.Errorf("%s accepted %s: %d", endpoint, invalid, resp.Code)
			}
		}
	}
}

func TestAccessLogPurgeAdminPreviewExecutionAndCutoff(t *testing.T) {
	engine, _, logs, tokens, dir := accessLogTestAPI(t)
	logs.Record(store.AccessLogEntry{RequestID: "selected", Host: "example.com", Timestamp: time.Now().Add(-time.Hour)})
	logs.Record(store.AccessLogEntry{RequestID: "other", Host: "other.example", Timestamp: time.Now().Add(-time.Hour)})
	request := map[string]any{"mode": "filtered", "filters": map[string]any{"host": "example.com"}, "preview": true}
	for _, role := range []string{store.RoleEditor, store.RoleViewer, ""} {
		resp := performRequest(t, engine, http.MethodPost, "/api/access-logs/purge", tokens[role], request)
		want := http.StatusForbidden
		if role == "" {
			want = http.StatusUnauthorized
		}
		if resp.Code != want {
			t.Fatalf("role %q: %d %s", role, resp.Code, resp.Body)
		}
	}
	resp := performRequest(t, engine, http.MethodPost, "/api/access-logs/purge", tokens[store.RoleAdmin], request)
	if resp.Code != http.StatusOK {
		t.Fatalf("preview: %d %s", resp.Code, resp.Body)
	}
	preview := decodeLogJSON(t, resp)
	if preview["matched"] != float64(1) {
		t.Fatalf("preview = %v", preview)
	}
	if _, total := logs.Query(store.AccessLogFilter{}, 0, 100); total != 2 {
		t.Fatal("preview changed logs")
	}
	cutoff, err := time.Parse(time.RFC3339Nano, preview["cutoff_time"].(string))
	if err != nil {
		t.Fatal(err)
	}
	logs.Record(store.AccessLogEntry{RequestID: "new", Host: "example.com", Timestamp: cutoff.Add(time.Nanosecond)})
	request["preview"], request["cutoff_time"] = false, preview["cutoff_time"]
	resp = performRequest(t, engine, http.MethodPost, "/api/access-logs/purge", tokens[store.RoleAdmin], request)
	if resp.Code != http.StatusOK || decodeLogJSON(t, resp)["removed"] != float64(1) {
		t.Fatalf("execute: %d %s", resp.Code, resp.Body)
	}
	reopened, err := store.NewAccessLogStore(dir, 100)
	if err != nil {
		t.Fatal(err)
	}
	entries, total := reopened.Query(store.AccessLogFilter{}, 0, 100)
	if total != 2 || entries[0].RequestID != "new" {
		t.Fatalf("remaining logs = %+v", entries)
	}
}

func TestAccessLogPurgeRejectsInvalidOrMissingScope(t *testing.T) {
	engine, _, logs, tokens, _ := accessLogTestAPI(t)
	logs.Record(store.AccessLogEntry{Timestamp: time.Now().Add(-time.Hour)})
	for _, body := range []map[string]any{
		{}, {"mode": "unknown", "preview": true}, {"mode": "filtered", "preview": true},
		{"mode": "filtered", "filters": map[string]any{"host": " "}, "preview": true},
		{"mode": "filtered", "filters": map[string]any{"status_code": 0}, "preview": true},
		{"mode": "all", "filters": map[string]any{"host": "example.com"}, "preview": true},
		{"mode": "all", "preview": false},
		{"mode": "all", "preview": false, "cutoff_time": "bad"},
		{"mode": "all", "preview": false, "cutoff_time": time.Now().Add(time.Hour).Format(time.RFC3339Nano)},
		{"mode": "all", "preview": true, "typo": true},
		{"mode": "filtered", "preview": true, "filters": map[string]any{"typo": "example.com"}},
	} {
		resp := performRequest(t, engine, http.MethodPost, "/api/access-logs/purge", tokens[store.RoleAdmin], body)
		if resp.Code != http.StatusBadRequest {
			t.Errorf("accepted %v: %d %s", body, resp.Code, resp.Body)
		}
	}
	if _, total := logs.Query(store.AccessLogFilter{}, 0, 100); total != 1 {
		t.Fatal("invalid request deleted logs")
	}
}

func TestAccessLogPurgeRetentionDisabledAndPersistenceFailure(t *testing.T) {
	engine, db, logs, tokens, dir := accessLogTestAPI(t)
	logs.Record(store.AccessLogEntry{Timestamp: time.Now().Add(-48 * time.Hour)})
	resp := performRequest(t, engine, http.MethodPost, "/api/access-logs/purge", tokens[store.RoleAdmin], map[string]any{"mode": "expired", "preview": true})
	preview := decodeLogJSON(t, resp)
	if resp.Code != http.StatusOK || preview["matched"] != float64(0) || preview["retention_days"] != float64(0) {
		t.Fatalf("disabled: %d %v", resp.Code, preview)
	}
	if err := db.SetSetting(settingLogRetentionDays, "1"); err != nil {
		t.Fatal(err)
	}
	resp = performRequest(t, engine, http.MethodPost, "/api/access-logs/purge", tokens[store.RoleAdmin], map[string]any{"mode": "expired", "preview": true})
	preview = decodeLogJSON(t, resp)
	if resp.Code != http.StatusOK || preview["matched"] != float64(1) {
		t.Fatalf("expired preview: %d %v", resp.Code, preview)
	}
	if err := os.Mkdir(filepath.Join(dir, "access-logs.jsonl"), 0700); err != nil {
		t.Fatal(err)
	}
	resp = performRequest(t, engine, http.MethodPost, "/api/access-logs/purge", tokens[store.RoleAdmin], map[string]any{"mode": "expired", "preview": false, "cutoff_time": preview["cutoff_time"]})
	if resp.Code != http.StatusInternalServerError {
		t.Fatalf("write failure: %d %s", resp.Code, resp.Body)
	}
	resp = performRequest(t, engine, http.MethodPost, "/api/settings/log-retention/purge", tokens[store.RoleAdmin], nil)
	if resp.Code != http.StatusInternalServerError {
		t.Fatalf("legacy write failure: %d %s", resp.Code, resp.Body)
	}
	if _, total := logs.Query(store.AccessLogFilter{}, 0, 100); total != 1 {
		t.Fatal("failed persistence changed memory")
	}
}

func TestAccessLogExpiredPreviewCannotBroadenAfterPolicyChange(t *testing.T) {
	engine, db, logs, tokens, _ := accessLogTestAPI(t)
	now := time.Now()
	logs.Record(store.AccessLogEntry{RequestID: "expired", Timestamp: now.Add(-8 * 24 * time.Hour)})
	logs.Record(store.AccessLogEntry{RequestID: "retained", Timestamp: now.Add(-2 * 24 * time.Hour)})
	if err := db.SetSetting(settingLogRetentionDays, "7"); err != nil {
		t.Fatal(err)
	}
	resp := performRequest(t, engine, http.MethodPost, "/api/access-logs/purge", tokens[store.RoleAdmin], map[string]any{"mode": "expired", "preview": true})
	preview := decodeLogJSON(t, resp)
	if preview["matched"] != float64(1) {
		t.Fatalf("preview = %v", preview)
	}
	cutoff, err := time.Parse(time.RFC3339Nano, preview["cutoff_time"].(string))
	if err != nil {
		t.Fatal(err)
	}
	if cutoff.After(now.Add(-6 * 24 * time.Hour)) {
		t.Fatal("expired cutoff must be the expiration threshold")
	}
	if err := db.SetSetting(settingLogRetentionDays, "1"); err != nil {
		t.Fatal(err)
	}
	resp = performRequest(t, engine, http.MethodPost, "/api/access-logs/purge", tokens[store.RoleAdmin], map[string]any{"mode": "expired", "preview": false, "cutoff_time": preview["cutoff_time"]})
	if resp.Code != http.StatusOK || decodeLogJSON(t, resp)["removed"] != float64(1) {
		t.Fatalf("execute: %d %s", resp.Code, resp.Body)
	}
	entries, total := logs.Query(store.AccessLogFilter{}, 0, 10)
	if total != 1 || entries[0].RequestID != "retained" {
		t.Fatalf("policy change expanded scope: %+v", entries)
	}
	if err := db.SetSetting(settingLogRetentionDays, "0"); err != nil {
		t.Fatal(err)
	}
	resp = performRequest(t, engine, http.MethodPost, "/api/access-logs/purge", tokens[store.RoleAdmin], map[string]any{"mode": "expired", "preview": true})
	disabled := decodeLogJSON(t, resp)
	if err := db.SetSetting(settingLogRetentionDays, "1"); err != nil {
		t.Fatal(err)
	}
	resp = performRequest(t, engine, http.MethodPost, "/api/access-logs/purge", tokens[store.RoleAdmin], map[string]any{"mode": "expired", "preview": false, "cutoff_time": disabled["cutoff_time"]})
	if resp.Code != http.StatusOK || decodeLogJSON(t, resp)["removed"] != float64(0) {
		t.Fatalf("disabled preview expanded: %d %s", resp.Code, resp.Body)
	}
}

func TestAccessLogPurgeAllAndLegacyRetentionPersist(t *testing.T) {
	engine, db, logs, tokens, dir := accessLogTestAPI(t)
	logs.Record(store.AccessLogEntry{Timestamp: time.Now().Add(-48 * time.Hour)})
	logs.Record(store.AccessLogEntry{Timestamp: time.Now().Add(-time.Hour)})
	if err := db.SetSetting(settingLogRetentionDays, "1"); err != nil {
		t.Fatal(err)
	}
	resp := performRequest(t, engine, http.MethodPost, "/api/settings/log-retention/purge", tokens[store.RoleAdmin], nil)
	legacy := decodeLogJSON(t, resp)
	if resp.Code != http.StatusOK || legacy["removed"] != float64(1) || legacy["cutoff_days"] != float64(1) {
		t.Fatalf("legacy: %d %v", resp.Code, legacy)
	}
	resp = performRequest(t, engine, http.MethodPost, "/api/access-logs/purge", tokens[store.RoleAdmin], map[string]any{"mode": "all", "preview": true})
	preview := decodeLogJSON(t, resp)
	if preview["matched"] != float64(1) {
		t.Fatalf("all preview = %v", preview)
	}
	resp = performRequest(t, engine, http.MethodPost, "/api/access-logs/purge", tokens[store.RoleAdmin], map[string]any{"mode": "all", "preview": false, "cutoff_time": preview["cutoff_time"]})
	if resp.Code != http.StatusOK || decodeLogJSON(t, resp)["removed"] != float64(1) {
		t.Fatalf("all execute: %d %s", resp.Code, resp.Body)
	}
	reopened, err := store.NewAccessLogStore(dir, 100)
	if err != nil {
		t.Fatal(err)
	}
	if _, total := reopened.Query(store.AccessLogFilter{}, 0, 100); total != 0 {
		t.Fatal("all cleanup reappeared after restart")
	}
}
