package middleware

import (
	"net/http"
	"strconv"
	"sync"
	"time"

	"alertbot/internal/config"
	"alertbot/internal/metrics"

	"github.com/gin-gonic/gin"
	"github.com/patrickmn/go-cache"
	"github.com/sirupsen/logrus"
	"golang.org/x/time/rate"
)

// MemoryRateLimiter implements in-memory rate limiting
type MemoryRateLimiter struct {
	limiters *cache.Cache
	mu       sync.RWMutex
	rps      int
	burst    int
	logger   *logrus.Logger
}

func NewMemoryRateLimiter(rps, burst int, logger *logrus.Logger) *MemoryRateLimiter {
	return &MemoryRateLimiter{
		limiters: cache.New(time.Hour, 10*time.Minute),
		rps:      rps,
		burst:    burst,
		logger:   logger,
	}
}

func (rl *MemoryRateLimiter) GetLimiter(key string) *rate.Limiter {
	rl.mu.RLock()
	if limiter, found := rl.limiters.Get(key); found {
		rl.mu.RUnlock()
		return limiter.(*rate.Limiter)
	}
	rl.mu.RUnlock()

	rl.mu.Lock()
	defer rl.mu.Unlock()
	
	// Double-check after acquiring write lock
	if limiter, found := rl.limiters.Get(key); found {
		return limiter.(*rate.Limiter)
	}

	limiter := rate.NewLimiter(rate.Limit(rl.rps), rl.burst)
	rl.limiters.Set(key, limiter, cache.DefaultExpiration)
	return limiter
}

func (rl *MemoryRateLimiter) Allow(key string) bool {
	return rl.GetLimiter(key).Allow()
}

var globalRateLimiter *MemoryRateLimiter

func RateLimit(cfg *config.Config, logger *logrus.Logger) gin.HandlerFunc {
	if !cfg.RateLimit.Enabled {
		return func(c *gin.Context) {
			c.Next()
		}
	}

	// Initialize rate limiter
	if globalRateLimiter == nil {
		globalRateLimiter = NewMemoryRateLimiter(cfg.RateLimit.RPS, cfg.RateLimit.Burst, logger)
	}

	return func(c *gin.Context) {
		clientIP := c.ClientIP()
		key := clientIP

		if !globalRateLimiter.Allow(key) {
			c.Header("X-RateLimit-Limit", strconv.Itoa(cfg.RateLimit.RPS))
			c.Header("X-RateLimit-Remaining", "0")
			c.Header("Retry-After", "60")

			logger.WithField("client_ip", clientIP).Warn("Rate limit exceeded")

			// Record rate limit metric
			metrics.RecordRateLimitedRequest(clientIP)

			c.JSON(http.StatusTooManyRequests, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "RATE_LIMITED",
					"message": "Too many requests, please try again later",
				},
			})
			c.Abort()
			return
		}

		c.Next()
	}
}