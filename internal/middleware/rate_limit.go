package middleware

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"alertbot/internal/config"
	"alertbot/internal/metrics"

	"github.com/gin-gonic/gin"
	"github.com/patrickmn/go-cache"
	"github.com/sirupsen/logrus"
	"golang.org/x/time/rate"
)

// RateLimitRule defines rate limiting rules for different scenarios
type RateLimitRule struct {
	RPS   int `json:"rps"`
	Burst int `json:"burst"`
}

// RateLimitConfig holds all rate limiting configurations
type RateLimitConfig struct {
	Global        RateLimitRule            `json:"global"`
	PerUser       RateLimitRule            `json:"per_user"`
	PerEndpoint   map[string]RateLimitRule `json:"per_endpoint"`
	Notification  RateLimitRule            `json:"notification"`
	BurstProtection RateLimitRule          `json:"burst_protection"`
}

// MemoryRateLimiter implements in-memory rate limiting with multiple strategies
type MemoryRateLimiter struct {
	limiters *cache.Cache
	mu       sync.RWMutex
	config   RateLimitConfig
	logger   *logrus.Logger
}

func NewMemoryRateLimiter(cfg *config.Config, logger *logrus.Logger) *MemoryRateLimiter {
	// Default rate limit configuration
	rateLimitConfig := RateLimitConfig{
		Global: RateLimitRule{
			RPS:   cfg.RateLimit.RPS,
			Burst: cfg.RateLimit.Burst,
		},
		PerUser: RateLimitRule{
			RPS:   cfg.RateLimit.RPS / 2, // Half of global limit per user
			Burst: cfg.RateLimit.Burst / 2,
		},
		PerEndpoint: map[string]RateLimitRule{
			"/api/v1/alerts": {
				RPS:   cfg.RateLimit.RPS * 2, // Alerts endpoint gets higher limit
				Burst: cfg.RateLimit.Burst * 2,
			},
			"/api/v1/channels/*/test": {
				RPS:   5,  // Test endpoints are limited to prevent abuse
				Burst: 10,
			},
			"/api/v1/auth/login": {
				RPS:   10, // Login attempts are strictly limited
				Burst: 20,
			},
		},
		Notification: RateLimitRule{
			RPS:   30,  // 30 notifications per second
			Burst: 100, // Allow bursts up to 100
		},
		BurstProtection: RateLimitRule{
			RPS:   cfg.RateLimit.RPS * 5, // 5x normal rate for burst detection
			Burst: 1,                     // Only allow 1 burst request
		},
	}

	return &MemoryRateLimiter{
		limiters: cache.New(time.Hour, 10*time.Minute),
		config:   rateLimitConfig,
		logger:   logger,
	}
}

func (rl *MemoryRateLimiter) GetLimiter(key string, rule RateLimitRule) *rate.Limiter {
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

	limiter := rate.NewLimiter(rate.Limit(rule.RPS), rule.Burst)
	rl.limiters.Set(key, limiter, cache.DefaultExpiration)
	return limiter
}

func (rl *MemoryRateLimiter) Allow(key string, rule RateLimitRule) bool {
	return rl.GetLimiter(key, rule).Allow()
}

// AllowN checks if n requests can be made
func (rl *MemoryRateLimiter) AllowN(key string, rule RateLimitRule, n int) bool {
	return rl.GetLimiter(key, rule).AllowN(time.Now(), n)
}

// GetRuleForEndpoint returns the appropriate rate limit rule for an endpoint
func (rl *MemoryRateLimiter) GetRuleForEndpoint(path string) RateLimitRule {
	// Check for exact match first
	if rule, exists := rl.config.PerEndpoint[path]; exists {
		return rule
	}
	
	// Check for pattern matches
	for pattern, rule := range rl.config.PerEndpoint {
		if matchEndpointPattern(pattern, path) {
			return rule
		}
	}
	
	// Return global rule as default
	return rl.config.Global
}

// GetNotificationRule returns the notification rate limit rule
func (rl *MemoryRateLimiter) GetNotificationRule() RateLimitRule {
	return rl.config.Notification
}

// GetUserRule returns the per-user rate limit rule
func (rl *MemoryRateLimiter) GetUserRule() RateLimitRule {
	return rl.config.PerUser
}

// GetBurstProtectionRule returns the burst protection rule
func (rl *MemoryRateLimiter) GetBurstProtectionRule() RateLimitRule {
	return rl.config.BurstProtection
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
		globalRateLimiter = NewMemoryRateLimiter(cfg, logger)
	}

	return func(c *gin.Context) {
		clientIP := c.ClientIP()
		path := c.Request.URL.Path
		method := c.Request.Method
		
		// Get appropriate rate limit rule for this endpoint
		rule := globalRateLimiter.GetRuleForEndpoint(path)
		
		// Create composite key for rate limiting
		keys := []string{
			fmt.Sprintf("global:%s", clientIP),                    // Global per-IP limit
			fmt.Sprintf("endpoint:%s:%s:%s", method, path, clientIP), // Per-endpoint per-IP limit
		}
		
		// Add user-based key if user is authenticated
		if userID, exists := c.Get("user_id"); exists {
			keys = append(keys, fmt.Sprintf("user:%v", userID))
		}
		
		// Check each rate limit
		for i, key := range keys {
			var currentRule RateLimitRule
			var limitType string
			
			switch i {
			case 0: // Global limit
				currentRule = globalRateLimiter.config.Global
				limitType = "global"
			case 1: // Endpoint limit
				currentRule = rule
				limitType = "endpoint"
			case 2: // User limit
				currentRule = globalRateLimiter.GetUserRule()
				limitType = "user"
			}
			
			if !globalRateLimiter.Allow(key, currentRule) {
				c.Header("X-RateLimit-Limit", strconv.Itoa(currentRule.RPS))
				c.Header("X-RateLimit-Remaining", "0")
				c.Header("X-RateLimit-Type", limitType)
				c.Header("Retry-After", "60")

				logger.WithFields(logrus.Fields{
					"client_ip":   clientIP,
					"path":        path,
					"method":      method,
					"limit_type":  limitType,
					"rate_limit":  currentRule.RPS,
				}).Warn("Rate limit exceeded")

				// Record rate limit metric
				metrics.RecordRateLimitedRequest(clientIP)

				c.JSON(http.StatusTooManyRequests, gin.H{
					"success": false,
					"error": gin.H{
						"code":        "RATE_LIMITED",
						"message":     fmt.Sprintf("%s rate limit exceeded, please try again later", limitType),
						"limit_type":  limitType,
						"retry_after": 60,
					},
				})
				c.Abort()
				return
			}
		}

		c.Next()
	}
}

// NotificationRateLimit provides rate limiting specifically for notifications
func NotificationRateLimit(logger *logrus.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		if globalRateLimiter == nil {
			c.Next()
			return
		}
		
		channelID := c.Param("id")
		if channelID == "" {
			c.Next()
			return
		}
		
		key := fmt.Sprintf("notification:%s", channelID)
		rule := globalRateLimiter.GetNotificationRule()
		
		if !globalRateLimiter.Allow(key, rule) {
			logger.WithFields(logrus.Fields{
				"channel_id": channelID,
				"rate_limit": rule.RPS,
			}).Warn("Notification rate limit exceeded")
			
			c.JSON(http.StatusTooManyRequests, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "NOTIFICATION_RATE_LIMITED",
					"message": "Notification rate limit exceeded, please reduce notification frequency",
				},
			})
			c.Abort()
			return
		}
		
		c.Next()
	}
}

// BurstProtection detects and prevents burst attacks
func BurstProtection(logger *logrus.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		if globalRateLimiter == nil {
			c.Next()
			return
		}
		
		clientIP := c.ClientIP()
		key := fmt.Sprintf("burst:%s", clientIP)
		rule := globalRateLimiter.GetBurstProtectionRule()
		
		// Check if this looks like a burst attack (many requests in short time)
		if !globalRateLimiter.AllowN(key, rule, 10) { // 10 requests threshold
			logger.WithFields(logrus.Fields{
				"client_ip": clientIP,
				"path":      c.Request.URL.Path,
			}).Warn("Potential burst attack detected")
			
			// Temporarily block this IP
			c.JSON(http.StatusTooManyRequests, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "BURST_DETECTED",
					"message": "Unusual burst activity detected, please wait before retrying",
				},
			})
			c.Abort()
			return
		}
		
		c.Next()
	}
}

// matchEndpointPattern matches endpoint patterns like "/api/v1/channels/*/test"
func matchEndpointPattern(pattern, path string) bool {
	patternParts := strings.Split(pattern, "/")
	pathParts := strings.Split(path, "/")
	
	if len(patternParts) != len(pathParts) {
		return false
	}
	
	for i, part := range patternParts {
		if part == "*" {
			continue // Wildcard matches anything
		}
		if part != pathParts[i] {
			return false
		}
	}
	
	return true
}