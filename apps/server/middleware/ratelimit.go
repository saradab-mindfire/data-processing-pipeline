package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

const (
	rateLimitRequestsPerSecond = 5
	rateLimitBurst             = 20
)

func RateLimit() gin.HandlerFunc {
	limiter := rate.NewLimiter(rateLimitRequestsPerSecond, rateLimitBurst)

	return func(c *gin.Context) {
		if !limiter.Allow() {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"error": "rate limit exceeded"})
			return
		}
		c.Next()
	}
}
