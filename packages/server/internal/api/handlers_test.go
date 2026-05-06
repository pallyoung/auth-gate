package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/pallyoung/auth-gate/packages/server/internal/router"
	"github.com/pallyoung/auth-gate/packages/server/internal/store"

	"github.com/gin-gonic/gin"
)

func newHandlerTestDB(t *testing.T) *store.SQLite {
	t.Helper()

	db, err := store.NewSQLite(filepath.Join(t.TempDir(), "auth-gate.db"))
	if err != nil {
		t.Fatalf("NewSQLite() error = %v", err)
	}
	t.Cleanup(func() {
		_ = db.Close()
	})
	return db
}

func createHandlerTestRoute(t *testing.T, db *store.SQLite, routeID string) {
	t.Helper()

	err := db.CreateRoute(&store.Route{
		ID:         routeID,
		Name:       "test",
		PathPrefix: "/svc",
		Backend:    "http://example.com",
		Enabled:    true,
	})
	if err != nil {
		t.Fatalf("CreateRoute() error = %v", err)
	}
}

func performJSONRequest(t *testing.T, handler gin.HandlerFunc, method, target string, body any) *httptest.ResponseRecorder {
	t.Helper()

	var payload []byte
	if body != nil {
		var err error
		payload, err = json.Marshal(body)
		if err != nil {
			t.Fatalf("json.Marshal() error = %v", err)
		}
	}

	req := httptest.NewRequest(method, target, bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	handler(c)
	return w
}

func TestCreateAuthRule_RejectsDuplicateRoute(t *testing.T) {
	gin.SetMode(gin.TestMode)

	db := newHandlerTestDB(t)
	createHandlerTestRoute(t, db, "route-1")
	routerMgr := router.NewManager(db)

	first := performJSONRequest(t, createAuthRule(db, routerMgr), http.MethodPost, "/api/auth-rules", map[string]any{
		"route_id": "route-1",
		"type":     "apikey",
		"config": map[string]any{
			"secret": "secret-1",
		},
	})
	if first.Code != http.StatusCreated {
		t.Fatalf("first create status = %d, want %d, body=%s", first.Code, http.StatusCreated, first.Body.String())
	}

	second := performJSONRequest(t, createAuthRule(db, routerMgr), http.MethodPost, "/api/auth-rules", map[string]any{
		"route_id": "route-1",
		"type":     "bearer",
		"config": map[string]any{
			"secret": "secret-2",
		},
	})
	if second.Code != http.StatusBadRequest {
		t.Fatalf("second create status = %d, want %d, body=%s", second.Code, http.StatusBadRequest, second.Body.String())
	}
	if second.Body.String() != "{\"error\":\"route already has an auth rule\"}" {
		t.Fatalf("second create body = %s", second.Body.String())
	}
}

func TestListRoutes_ReturnsEmptyArrayWhenNoRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)

	db := newHandlerTestDB(t)

	w := performJSONRequest(t, listRoutes(db), http.MethodGet, "/api/routes", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body=%s", w.Code, http.StatusOK, w.Body.String())
	}
	if w.Body.String() != "[]" {
		t.Fatalf("body = %s, want []", w.Body.String())
	}
}

func TestListAuthRules_ReturnsEmptyArrayWhenNoRules(t *testing.T) {
	gin.SetMode(gin.TestMode)

	db := newHandlerTestDB(t)

	w := performJSONRequest(t, listAuthRules(db), http.MethodGet, "/api/auth-rules", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body=%s", w.Code, http.StatusOK, w.Body.String())
	}
	if w.Body.String() != "[]" {
		t.Fatalf("body = %s, want []", w.Body.String())
	}
}

func TestListUsers_ReturnsEmptyArrayWhenNoUsers(t *testing.T) {
	gin.SetMode(gin.TestMode)

	db := newHandlerTestDB(t)

	w := performJSONRequest(t, listUsers(db), http.MethodGet, "/api/users", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body=%s", w.Code, http.StatusOK, w.Body.String())
	}
	if w.Body.String() != "[]" {
		t.Fatalf("body = %s, want []", w.Body.String())
	}
}

func TestCreateAuthRule_RejectsMissingSecret(t *testing.T) {
	gin.SetMode(gin.TestMode)

	db := newHandlerTestDB(t)
	createHandlerTestRoute(t, db, "route-1")
	routerMgr := router.NewManager(db)

	tests := []struct {
		name string
		body map[string]any
		want string
	}{
		{
			name: "apikey",
			body: map[string]any{
				"route_id": "route-1",
				"type":     "apikey",
				"config":   map[string]any{},
			},
			want: "{\"error\":\"apikey secret required\"}",
		},
		{
			name: "bearer",
			body: map[string]any{
				"route_id": "route-1",
				"type":     "bearer",
				"config":   map[string]any{},
			},
			want: "{\"error\":\"bearer secret required\"}",
		},
		{
			name: "basic",
			body: map[string]any{
				"route_id": "route-1",
				"type":     "basic",
				"config": map[string]any{
					"username": "admin",
				},
			},
			want: "{\"error\":\"basic username and password required\"}",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := performJSONRequest(t, createAuthRule(db, routerMgr), http.MethodPost, "/api/auth-rules", tt.body)
			if w.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want %d, body=%s", w.Code, http.StatusBadRequest, w.Body.String())
			}
			if w.Body.String() != tt.want {
				t.Fatalf("body = %s, want %s", w.Body.String(), tt.want)
			}
		})
	}
}

func TestUpdateRoute_RejectsInvalidBackend(t *testing.T) {
	gin.SetMode(gin.TestMode)

	db := newHandlerTestDB(t)
	createHandlerTestRoute(t, db, "route-1")
	routerMgr := router.NewManager(db)

	w := performJSONRequest(t, updateRoute(db, routerMgr), http.MethodPut, "/api/routes/route-1", map[string]any{
		"name":         "broken",
		"path_prefix":  "/svc",
		"backend":      "ftp://example.com",
		"strip_prefix": true,
		"enabled":      true,
	})
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d, body=%s", w.Code, http.StatusBadRequest, w.Body.String())
	}
	if w.Body.String() != "{\"error\":\"backend must be a valid http or https URL\"}" {
		t.Fatalf("body = %s", w.Body.String())
	}
}

func TestCreateUser_RejectsInvalidRole(t *testing.T) {
	gin.SetMode(gin.TestMode)

	db := newHandlerTestDB(t)

	w := performJSONRequest(t, createUser(db), http.MethodPost, "/api/users", map[string]any{
		"username": "alice",
		"password": "password123",
		"role":     "owner",
	})
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d, body=%s", w.Code, http.StatusBadRequest, w.Body.String())
	}
	if w.Body.String() != "{\"error\":\"invalid role\"}" {
		t.Fatalf("body = %s", w.Body.String())
	}
}
