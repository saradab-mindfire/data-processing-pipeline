package middleware

import (
	"crypto/subtle"
	"net/http"

	"github.com/gin-gonic/gin"
)

// RequireAPIKey checks the X-API-Key header against apiKey. apiKey must be
// non-empty (config.Load enforces this); an empty apiKey fails every
// request closed rather than silently allowing them through.
func RequireAPIKey(apiKey string) gin.HandlerFunc {
	return func(c *gin.Context) {
		got := c.GetHeader("X-API-Key")
		if apiKey == "" || subtle.ConstantTimeCompare([]byte(got), []byte(apiKey)) != 1 {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid or missing API key"})
			return
		}

		c.Next()
	}
}
