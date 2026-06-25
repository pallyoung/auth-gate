package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func makeTestContext(method, path string, fns ...func(*http.Request)) (*gin.Context, *httptest.ResponseRecorder) {
	req := httptest.NewRequest(method, path, nil)
	for _, fn := range fns {
		fn(req)
	}
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req
	return c, w
}

func TestRequireAuth(t *testing.T) {
	handler := RequireAuth(`Basic realm="test"`)
	c, w := makeTestContext("GET", "/")
	handler(c)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
	if h := w.Header().Get("WWW-Authenticate"); h != `Basic realm="test"` {
		t.Errorf("WWW-Authenticate = %q, want %q", h, `Basic realm="test"`)
	}
}
