package middleware

import (
	"net/http"
	"runtime/debug"

	"alertbot/internal/errors"
	"alertbot/internal/metrics"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// ErrorHandler middleware for centralized error handling
func ErrorHandler(logger *logrus.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				// Log panic with stack trace
				logger.WithFields(logrus.Fields{
					"panic":      err,
					"stack":      string(debug.Stack()),
					"path":       c.Request.URL.Path,
					"method":     c.Request.Method,
					"client_ip":  c.ClientIP(),
					"user_agent": c.Request.UserAgent(),
				}).Error("Panic recovered")

				// Record panic metric
				metrics.RecordDatabaseError("panic", "server_panic")

				// Return internal server error
				c.JSON(http.StatusInternalServerError, errors.ToResponse(
					errors.NewInternalError("Internal server error occurred", nil),
				))
				c.Abort()
			}
		}()

		c.Next()

		// Handle errors that were set during request processing
		if len(c.Errors) > 0 {
			err := c.Errors.Last()
			
			// Log the error
			logger.WithFields(logrus.Fields{
				"error":      err.Error(),
				"path":       c.Request.URL.Path,
				"method":     c.Request.Method,
				"client_ip":  c.ClientIP(),
				"user_agent": c.Request.UserAgent(),
			}).Error("Request error")

			// Get HTTP status and response
			httpStatus := errors.GetHTTPStatus(err.Err)
			response := errors.ToResponse(err.Err)

			// Record error metric
			if httpStatus >= 500 {
				metrics.RecordDatabaseError("http", "server_error")
			} else if httpStatus >= 400 {
				metrics.RecordDatabaseError("http", "client_error")
			}

			c.JSON(httpStatus, response)
			c.Abort()
		}
	}
}

// HandleError is a helper function to handle errors in handlers
func HandleError(c *gin.Context, err error) {
	if err == nil {
		return
	}

	// Add error to context for the error handler middleware
	c.Error(err)
}

// AbortWithError aborts the request with an error
func AbortWithError(c *gin.Context, err error) {
	HandleError(c, err)
	c.Abort()
}

// RespondWithError sends an error response and aborts the request
func RespondWithError(c *gin.Context, err error) {
	httpStatus := errors.GetHTTPStatus(err)
	response := errors.ToResponse(err)
	c.JSON(httpStatus, response)
	c.Abort()
}