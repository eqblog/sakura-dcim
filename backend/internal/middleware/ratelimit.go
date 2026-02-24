package middleware

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

// RateLimit implements a Redis sliding window rate limiter.
// maxRequests is the maximum number of requests allowed within the given window.
func RateLimit(rdb *redis.Client, maxRequests int, window time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Build key: per-user if authenticated, else per-IP
		userID := GetUserID(c)
		var key string
		if userID.String() != "00000000-0000-0000-0000-000000000000" {
			key = fmt.Sprintf("rl:%s:%s", c.FullPath(), userID.String())
		} else {
			key = fmt.Sprintf("rl:%s:%s", c.FullPath(), c.ClientIP())
		}

		ctx := context.Background()
		now := time.Now().UnixMilli()
		windowMs := window.Milliseconds()

		pipe := rdb.Pipeline()
		// Remove entries outside the window
		pipe.ZRemRangeByScore(ctx, key, "0", fmt.Sprintf("%d", now-windowMs))
		// Add current request
		pipe.ZAdd(ctx, key, redis.Z{Score: float64(now), Member: now})
		// Count requests in window
		countCmd := pipe.ZCard(ctx, key)
		// Set key expiry to window duration
		pipe.Expire(ctx, key, window)

		if _, err := pipe.Exec(ctx); err != nil {
			// On Redis error, allow the request (fail-open)
			c.Next()
			return
		}

		count := countCmd.Val()
		if count > int64(maxRequests) {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error":       "rate limit exceeded",
				"retry_after": window.Seconds(),
			})
			c.Abort()
			return
		}

		c.Next()
	}
}
