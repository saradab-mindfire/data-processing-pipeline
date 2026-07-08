package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func RequireAPIKey(apiKey string) gin.HandlerFunc {
	return func(c *gin.Context) {
		got := c.GetHeader("X-API-Key")

		if apiKey == "" || got != apiKey {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid or missing API key"})
			return
		}

		c.Next()
	}
}
