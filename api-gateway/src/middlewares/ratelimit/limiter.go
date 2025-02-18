package ratelimit

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

// RateLimiter implements rate limiting using Redis
type RateLimiter struct {
	redisClient *redis.Client
	logger      *zap.Logger
	// Configurable limits
	requestsPerSecond int
	burstSize         int
}

// Config holds rate limiter configuration
type Config struct {
	RequestsPerSecond int
	BurstSize         int
	RedisClient       *redis.Client
	Logger            *zap.Logger
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(config Config) *RateLimiter {
	return &RateLimiter{
		redisClient:       config.RedisClient,
		logger:            config.Logger,
		requestsPerSecond: config.RequestsPerSecond,
		burstSize:         config.BurstSize,
	}
}

// Limit is the middleware function to limit requests
func (rl *RateLimiter) Limit() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get client identifier (IP or user ID)
		identifier := rl.getClientIdentifier(c)

		allowed, remaining, err := rl.isAllowed(c.Request.Context(), identifier)
		if err != nil {
			rl.logger.Error("rate limiter error", zap.Error(err))
			c.Next() // Allow request on error
			return
		}

		if !allowed {
			c.Header("X-RateLimit-Remaining", "0")
			c.Header("X-RateLimit-Reset", fmt.Sprintf("%d", time.Now().Unix()+1))
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error": "rate limit exceeded",
			})
			c.Abort()
			return
		}

		c.Header("X-RateLimit-Remaining", fmt.Sprintf("%d", remaining))
		c.Next()
	}
}

// getClientIdentifier returns a unique identifier for the client
func (rl *RateLimiter) getClientIdentifier(c *gin.Context) string {
	// Try to get user ID from context first
	if userID, exists := c.Get("userID"); exists {
		return fmt.Sprintf("user:%s", userID)
	}

	// Fall back to IP address
	return fmt.Sprintf("ip:%s", c.ClientIP())
}

// isAllowed checks if the request is allowed based on rate limits
func (rl *RateLimiter) isAllowed(ctx context.Context, identifier string) (bool, int, error) {
	key := fmt.Sprintf("ratelimit:%s", identifier)

	pipe := rl.redisClient.Pipeline()

	// Get current count
	//currentCount := pipe.Get(ctx, key)
	// Increment count
	increment := pipe.Incr(ctx, key)
	// Set expiry if key is new
	pipe.Expire(ctx, key, time.Second)

	_, err := pipe.Exec(ctx)
	if err != nil && err != redis.Nil {
		return false, 0, fmt.Errorf("redis pipeline error: %w", err)
	}

	count, err := increment.Result()
	if err != nil {
		return false, 0, fmt.Errorf("increment error: %w", err)
	}

	remaining := rl.requestsPerSecond - int(count)
	if remaining < 0 {
		remaining = 0
	}

	return count <= int64(rl.requestsPerSecond), remaining, nil
}
