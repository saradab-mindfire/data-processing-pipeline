package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func newInternalTokenTestRouter(token string) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(RequireInternalToken(token))
	r.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "pong"})
	})
	return r
}

func TestRequireInternalTokenMissingHeader(t *testing.T) {
	r := newInternalTokenTestRouter("secret")
	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestRequireInternalTokenWrongToken(t *testing.T) {
	r := newInternalTokenTestRouter("secret")
	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	req.Header.Set("X-Internal-Token", "wrong")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestRequireInternalTokenCorrectToken(t *testing.T) {
	r := newInternalTokenTestRouter("secret")
	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	req.Header.Set("X-Internal-Token", "secret")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestRequireInternalTokenEmptyConfiguredTokenAlwaysRejects(t *testing.T) {
	r := newInternalTokenTestRouter("")
	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	req.Header.Set("X-Internal-Token", "")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}
