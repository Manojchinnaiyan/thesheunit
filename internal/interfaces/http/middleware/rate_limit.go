package middleware

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/your-org/ecommerce-backend/internal/config"
)

// RateLimit implements rate limiting using Redis
func RateLimit(cfg *config.Config, redisClient *redis.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get client IP
		clientIP := c.ClientIP()

		// Create rate limit key
		key := fmt.Sprintf("rate_limit:%s", clientIP)

		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()

		// Get current count
		current, err := redisClient.Get(ctx, key).Int()
		if err != nil && err.Error() != "redis: nil" {
			// If Redis is down, allow the request
			c.Next()
			return
		}

		// Check if limit exceeded
		if current >= cfg.Security.RateLimitPerMinute {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error":       "Rate limit exceeded",
				"retry_after": 60,
			})
			c.Abort()
			return
		}

		// Increment counter
		pipe := redisClient.Pipeline()
		pipe.Incr(ctx, key)
		pipe.Expire(ctx, key, time.Minute)
		_, err = pipe.Exec(ctx)
		if err != nil {
			// If Redis operation fails, log but allow request
			// In production, you might want to handle this differently
		}

		// Add rate limit headers
		c.Header("X-RateLimit-Limit", strconv.Itoa(cfg.Security.RateLimitPerMinute))
		c.Header("X-RateLimit-Remaining", strconv.Itoa(cfg.Security.RateLimitPerMinute-current-1))
		c.Header("X-RateLimit-Reset", strconv.FormatInt(time.Now().Add(time.Minute).Unix(), 10))

		c.Next()
	}
}
