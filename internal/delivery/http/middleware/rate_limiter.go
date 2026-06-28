package middleware

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis_rate/v10"
	"github.com/redis/go-redis/v9"
)

// RateLimiterMiddleware creates a rate limiting middleware using Redis
func RateLimiterMiddleware(redisClient *redis.Client, limit redis_rate.Limit) gin.HandlerFunc {
	limiter := redis_rate.NewLimiter(redisClient)

	return func(c *gin.Context) {
		// Identify user by JWT token if exists, otherwise by IP
		identifier := c.ClientIP()
		if userID, exists := c.Get("user_id"); exists {
			identifier = userID.(string)
		}

		key := fmt.Sprintf("rate_limit:%s:%s", c.Request.URL.Path, identifier)

		res, err := limiter.Allow(c.Request.Context(), key, limit)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "rate limiter error"})
			c.Abort()
			return
		}

		if res.Allowed == 0 {
			c.Header("X-RateLimit-Limit", fmt.Sprintf("%d", res.Limit.Rate))
			c.Header("X-RateLimit-Remaining", fmt.Sprintf("%d", res.Remaining))
			c.Header("X-RateLimit-Reset", fmt.Sprintf("%d", res.ResetAfter.Milliseconds()))
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error": "too many requests, please try again later",
			})
			c.Abort()
			return
		}

		// Set headers for allowed requests too
		c.Header("X-RateLimit-Limit", fmt.Sprintf("%d", res.Limit.Rate))
		c.Header("X-RateLimit-Remaining", fmt.Sprintf("%d", res.Remaining))

		c.Next()
	}
}
