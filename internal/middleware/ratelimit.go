package middleware

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

type RateLimitConfig struct {
	WindowSeconds int
	MaxRequests   int
	KeyPrefix     string
}

func RateLimit(rdb *redis.Client, cfg RateLimitConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		ip := c.ClientIP()
		if ip == "" {
			ip = "127.0.0.1"
		}
		key := cfg.KeyPrefix + ip
		ctx := c.Request.Context()

		count, err := rdb.Incr(ctx, key).Result()
		if err != nil {
			c.Next()
			return
		}

		if count == 1 {
			rdb.Expire(ctx, key, time.Duration(cfg.WindowSeconds)*time.Second)
		}

		if count > int64(cfg.MaxRequests) {
			ttl, _ := rdb.TTL(ctx, key).Result()
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error":      "Too many requests",
				"retryAfter": int(ttl.Seconds()),
			})
			return
		}

		c.Next()
	}
}
