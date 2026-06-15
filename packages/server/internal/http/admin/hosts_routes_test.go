package admin

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/pallyoung/auth-gate/packages/server/internal/auth"
	hostservice "github.com/pallyoung/auth-gate/packages/server/internal/service/hosts"
	"github.com/pallyoung/auth-gate/packages/server/internal/store"
)

func newTestHostRouter(t *testing.T) (*gin.Engine, store.Store) {
	t.Helper()

	db, err := store.NewJSONStore(t.TempDir())
	if err != nil {
		t.Fatalf("NewJSONStore() error = %v", err)
	}
	t.Cleanup(func() {
		_ = db.Close()
	})

	auth.ConfigureJWTSecret("test-secret")
	svc := hostservice.NewService(db, nil)
	gin.SetMode(gin.TestMode)
	r := gin.New()
	group := r.Group("/_authgate/api")
	group.Use(auth.AuthMiddleware(db))
	RegisterRoutes(group, nil, db, nil, svc)
	return r, db
}

func hostToken(t *testing.T, db store.Store, role string) string {
	t.Helper()

	username := "host-test-user"
	if _, err := db.EnsureAdmin(username, "password123"); err != nil {
		t.Fatalf("EnsureAdmin() error = %v", err)
	}
	user, err := db.GetUserByUsername(username)
	if err != nil {
		t.Fatalf("GetUserByUsername() error = %v", err)
	}
	if user.Role != role {
		user.Role = role
		if err := db.UpdateUser(user); err != nil {
			t.Fatalf("UpdateUser() error = %v", err)
		}
	}
	tok, err := auth.GenerateToken(user.ID, user.Username, user.Role)
	if err != nil {
		t.Fatalf("GenerateToken() error = %v", err)
	}
	return tok
}

func hostDo(r *gin.Engine, method, path, token, body string) *httptest.ResponseRecorder {
	var reader *bytes.Reader
	if body == "" {
		reader = bytes.NewReader(nil)
	} else {
		reader = bytes.NewReader([]byte(body))
	}
	req := httptest.NewRequest(method, path, reader)
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestHTTP_ListHostProfiles_Empty(t *testing.T) {
	r, db := newTestHostRouter(t)
	tok := hostToken(t, db, store.RoleAdmin)

	w := hostDo(r, http.MethodGet, "/_authgate/api/host-profiles", tok, "")
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", w.Code, w.Body.String())
	}
	var resp struct {
		Profiles []map[string]any `json:"profiles"`
		ActiveID string           `json:"active_id"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if len(resp.Profiles) != 0 {
		t.Fatalf("len(Profiles) = %d, want 0", len(resp.Profiles))
	}
	if resp.ActiveID != "" {
		t.Fatalf("ActiveID = %q, want empty", resp.ActiveID)
	}
}

func TestHTTP_CreateHostProfile_RejectsInvalidName(t *testing.T) {
	r, db := newTestHostRouter(t)
	tok := hostToken(t, db, store.RoleAdmin)

	w := hostDo(r, http.MethodPost, "/_authgate/api/host-profiles", tok, `{"name":"bad/name"}`)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body=%s", w.Code, w.Body.String())
	}
	if !bytes.Contains(w.Body.Bytes(), []byte("invalid_host_profile_name")) {
		t.Fatalf("body missing invalid_host_profile_name code: %s", w.Body.String())
	}
}

func TestHTTP_ActivateProfile_RendererUnconfigured(t *testing.T) {
	r, db := newTestHostRouter(t)
	tok := hostToken(t, db, store.RoleAdmin)

	// Create a profile so the activate endpoint has a target.
	createW := hostDo(r, http.MethodPost, "/_authgate/api/host-profiles", tok, `{"name":"dev"}`)
	if createW.Code != http.StatusCreated {
		t.Fatalf("create profile status = %d, body=%s", createW.Code, createW.Body.String())
	}

	// Fetch the profile ID via list.
	listW := hostDo(r, http.MethodGet, "/_authgate/api/host-profiles", tok, "")
	if listW.Code != http.StatusOK {
		t.Fatalf("list status = %d, body=%s", listW.Code, listW.Body.String())
	}
	var listResp struct {
		Profiles []struct {
			ID string `json:"id"`
		} `json:"profiles"`
	}
	if err := json.Unmarshal(listW.Body.Bytes(), &listResp); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if len(listResp.Profiles) == 0 {
		t.Fatal("expected one profile after create")
	}
	profileID := listResp.Profiles[0].ID

	// Without a renderer wired, ActivateProfile returns host_render_failure.
	w := hostDo(r, http.MethodPost, "/_authgate/api/host-profiles/"+profileID+"/activate", tok, "")
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500 (no renderer wired); body=%s", w.Code, w.Body.String())
	}
	if !bytes.Contains(w.Body.Bytes(), []byte("host_render_failure")) {
		t.Fatalf("body missing host_render_failure code: %s", w.Body.String())
	}
}

func TestHTTP_CreateHostProfile_ViewerIsForbidden(t *testing.T) {
	r, db := newTestHostRouter(t)
	tok := hostToken(t, db, store.RoleViewer)

	w := hostDo(r, http.MethodPost, "/_authgate/api/host-profiles", tok, `{"name":"dev"}`)
	if w.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403; body=%s", w.Code, w.Body.String())
	}
}