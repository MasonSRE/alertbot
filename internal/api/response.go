package api

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// APIResponse represents the standard API response format
type APIResponse struct {
	Success   bool        `json:"success"`
	Data      interface{} `json:"data,omitempty"`
	Error     *APIError   `json:"error,omitempty"`
	Message   string      `json:"message,omitempty"`
	Timestamp string      `json:"timestamp"`
	RequestID string      `json:"request_id,omitempty"`
}

// APIError represents the standard error response format
type APIError struct {
	Code    string      `json:"code"`
	Message string      `json:"message"`
	Details interface{} `json:"details,omitempty"`
}

// PaginatedData represents paginated response data
type PaginatedData struct {
	Items interface{} `json:"items"`
	Total int64       `json:"total"`
	Page  int         `json:"page"`
	Size  int         `json:"size"`
	Pages int         `json:"pages"`
}

// ResponseHelper provides helper methods for standardized API responses
type ResponseHelper struct{}

// NewResponseHelper creates a new ResponseHelper instance
func NewResponseHelper() *ResponseHelper {
	return &ResponseHelper{}
}

// Success sends a successful response
func (r *ResponseHelper) Success(c *gin.Context, data interface{}, message string) {
	response := APIResponse{
		Success:   true,
		Data:      data,
		Message:   message,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		RequestID: r.getRequestID(c),
	}
	c.JSON(http.StatusOK, response)
}

// SuccessWithStatus sends a successful response with custom status code
func (r *ResponseHelper) SuccessWithStatus(c *gin.Context, statusCode int, data interface{}, message string) {
	response := APIResponse{
		Success:   true,
		Data:      data,
		Message:   message,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		RequestID: r.getRequestID(c),
	}
	c.JSON(statusCode, response)
}

// Paginated sends a paginated response
func (r *ResponseHelper) Paginated(c *gin.Context, items interface{}, total int64, page, size int, message string) {
	pages := int((total + int64(size) - 1) / int64(size))
	if pages < 1 {
		pages = 1
	}

	data := PaginatedData{
		Items: items,
		Total: total,
		Page:  page,
		Size:  size,
		Pages: pages,
	}

	r.Success(c, data, message)
}

// Error sends an error response
func (r *ResponseHelper) Error(c *gin.Context, statusCode int, code, message string, details interface{}) {
	response := APIResponse{
		Success: false,
		Error: &APIError{
			Code:    code,
			Message: message,
			Details: details,
		},
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		RequestID: r.getRequestID(c),
	}
	c.JSON(statusCode, response)
}

// BadRequest sends a 400 Bad Request response
func (r *ResponseHelper) BadRequest(c *gin.Context, message string, details interface{}) {
	r.Error(c, http.StatusBadRequest, "INVALID_REQUEST", message, details)
}

// Unauthorized sends a 401 Unauthorized response
func (r *ResponseHelper) Unauthorized(c *gin.Context, message string) {
	r.Error(c, http.StatusUnauthorized, "UNAUTHORIZED", message, nil)
}

// Forbidden sends a 403 Forbidden response
func (r *ResponseHelper) Forbidden(c *gin.Context, message string) {
	r.Error(c, http.StatusForbidden, "FORBIDDEN", message, nil)
}

// NotFound sends a 404 Not Found response
func (r *ResponseHelper) NotFound(c *gin.Context, resource string) {
	message := "Resource not found"
	if resource != "" {
		message = resource + " not found"
	}
	r.Error(c, http.StatusNotFound, "NOT_FOUND", message, nil)
}

// Conflict sends a 409 Conflict response
func (r *ResponseHelper) Conflict(c *gin.Context, message string) {
	r.Error(c, http.StatusConflict, "CONFLICT", message, nil)
}

// ValidationError sends a 422 Unprocessable Entity response
func (r *ResponseHelper) ValidationError(c *gin.Context, message string, details interface{}) {
	r.Error(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", message, details)
}

// InternalServerError sends a 500 Internal Server Error response
func (r *ResponseHelper) InternalServerError(c *gin.Context, message string, details interface{}) {
	r.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", message, details)
}

// ServiceUnavailable sends a 503 Service Unavailable response
func (r *ResponseHelper) ServiceUnavailable(c *gin.Context, message string) {
	r.Error(c, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", message, nil)
}

// RateLimited sends a 429 Too Many Requests response
func (r *ResponseHelper) RateLimited(c *gin.Context, message string) {
	r.Error(c, http.StatusTooManyRequests, "RATE_LIMITED", message, nil)
}

// getRequestID extracts request ID from context or generates one
func (r *ResponseHelper) getRequestID(c *gin.Context) string {
	if requestID, exists := c.Get("request_id"); exists {
		if id, ok := requestID.(string); ok {
			return id
		}
	}
	
	// Try to get from headers
	if requestID := c.GetHeader("X-Request-ID"); requestID != "" {
		return requestID
	}
	
	return ""
}

// BindAndValidate binds request data and validates it
func (r *ResponseHelper) BindAndValidate(c *gin.Context, obj interface{}) bool {
	if err := c.ShouldBindJSON(obj); err != nil {
		r.BadRequest(c, "Invalid request format", err.Error())
		return false
	}
	return true
}

// BindQueryAndValidate binds query parameters and validates them
func (r *ResponseHelper) BindQueryAndValidate(c *gin.Context, obj interface{}) bool {
	if err := c.ShouldBindQuery(obj); err != nil {
		r.BadRequest(c, "Invalid query parameters", err.Error())
		return false
	}
	return true
}

// ParseUintParam parses a URL parameter as uint
func (r *ResponseHelper) ParseUintParam(c *gin.Context, paramName string) (uint, bool) {
	paramStr := c.Param(paramName)
	if paramStr == "" {
		r.BadRequest(c, paramName+" is required", nil)
		return 0, false
	}

	// Simple uint parsing for IDs
	var id uint
	if _, err := fmt.Sscanf(paramStr, "%d", &id); err != nil || id == 0 {
		r.BadRequest(c, "Invalid "+paramName, nil)
		return 0, false
	}

	return id, true
}