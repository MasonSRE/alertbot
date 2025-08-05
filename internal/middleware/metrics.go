package middleware

import (
	"strconv"
	"time"

	"alertbot/internal/metrics"

	"github.com/gin-gonic/gin"
)

// MetricsMiddleware records HTTP request metrics
func MetricsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		// Process request
		c.Next()

		// Record metrics
		duration := time.Since(start).Seconds()
		method := c.Request.Method
		endpoint := c.FullPath()
		statusCode := strconv.Itoa(c.Writer.Status())

		// Sanitize endpoint for metrics (remove dynamic parts)
		endpoint = sanitizeEndpoint(endpoint)

		metrics.RecordHTTPRequest(method, endpoint, statusCode, duration)
	}
}

// sanitizeEndpoint removes dynamic parts from endpoint for cleaner metrics
func sanitizeEndpoint(endpoint string) string {
	if endpoint == "" {
		return "unknown"
	}

	// Replace common dynamic parts with placeholders
	// This helps reduce cardinality in metrics
	replacements := map[string]string{
		"/api/v1/alerts/:fingerprint":  "/api/v1/alerts/{fingerprint}",
		"/api/v1/rules/:id":           "/api/v1/rules/{id}",
		"/api/v1/channels/:id":        "/api/v1/channels/{id}",
		"/api/v1/silences/:id":        "/api/v1/silences/{id}",
	}

	for pattern, replacement := range replacements {
		if endpoint == pattern {
			return replacement
		}
	}

	return endpoint
}