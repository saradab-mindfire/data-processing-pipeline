package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func newAPIKeyTestRouter(apiKey string) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(RequireAPIKey(apiKey))
	r.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "pong"})
	})
	return r
}

func TestRequireAPIKeyMissingHeader(t *testing.T) {
	r := newAPIKeyTestRouter("secret")
	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestRequireAPIKeyWrongKey(t *testing.T) {
	r := newAPIKeyTestRouter("secret")
	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	req.Header.Set("X-API-Key", "wrong")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestRequireAPIKeyCorrectKey(t *testing.T) {
	r := newAPIKeyTestRouter("secret")
	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	req.Header.Set("X-API-Key", "secret")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestRequireAPIKeyEmptyConfiguredKeyAlwaysRejects(t *testing.T) {
	r := newAPIKeyTestRouter("")
	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	req.Header.Set("X-API-Key", "")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}
