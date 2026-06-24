package proxy

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/pallyoung/auth-gate/packages/server/internal/router"
	"github.com/pallyoung/auth-gate/packages/server/internal/store"
)

func newTestDB(t *testing.T) store.Store {
	t.Helper()

	db, err := store.NewJSONStore(t.TempDir())
	if err != nil {
		t.Fatalf("NewJSONStore() error = %v", err)
	}
	t.Cleanup(func() {
		_ = db.Close()
	})
	return db
}

func TestRegisterRoutes_ProxySuccessDoesNotKeepNoRouteStatus(t *testing.T) {
	gin.SetMode(gin.TestMode)

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("upstream-ok"))
	}))
	defer upstream.Close()

	db := newTestDB(t)
	if err := db.CreateRoute(&store.Route{
		ID:         "route-1",
		Name:       "proxy-success",
		PathPrefix: "/api",
		Backend:    upstream.URL,
		Enabled:    true,
	}); err != nil {
		t.Fatalf("CreateRoute() error = %v", err)
	}

	engine := gin.New()
	RegisterRoutes(engine, router.NewManager(db), nil)

	req := httptest.NewRequest(http.MethodGet, "/api/users", nil)
	resp := httptest.NewRecorder()
	engine.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body=%s", resp.Code, http.StatusOK, resp.Body.String())
	}
	if resp.Body.String() != "upstream-ok" {
		t.Fatalf("body = %q, want %q", resp.Body.String(), "upstream-ok")
	}
}
