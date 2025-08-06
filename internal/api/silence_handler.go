package api

import (
	"fmt"
	"net/http"
	"regexp"
	"time"

	"alertbot/internal/models"
	"alertbot/internal/service"

	"github.com/gin-gonic/gin"
)

type SilenceHandler struct {
	services *service.Services
	response *ResponseHelper
}

func NewSilenceHandler(services *service.Services) *SilenceHandler {
	return &SilenceHandler{
		services: services,
		response: NewResponseHelper(),
	}
}

// SilenceRequest represents the request body for creating silences
type SilenceRequest struct {
	Matchers []SilenceMatcher `json:"matchers" binding:"required,min=1"`
	StartsAt time.Time        `json:"starts_at" binding:"required"`
	EndsAt   time.Time        `json:"ends_at" binding:"required"`
	Creator  string           `json:"creator" binding:"required"`
	Comment  string           `json:"comment"`
}

// SilenceMatcher represents a matcher for silence rules
type SilenceMatcher struct {
	Name    string `json:"name" binding:"required"`
	Value   string `json:"value" binding:"required"`
	IsRegex bool   `json:"is_regex"`
}

// ListSilences retrieves all silences
func (h *SilenceHandler) ListSilences(c *gin.Context) {
	silences, err := h.services.Silence.ListSilences(c.Request.Context())
	if err != nil {
		h.response.InternalServerError(c, "Failed to retrieve silences", err.Error())
		return
	}

	// Enrich silences with status information
	enrichedSilences := make([]gin.H, len(silences))
	now := time.Now()

	for i, silence := range silences {
		status := h.getSilenceStatus(silence, now)
		enrichedSilences[i] = gin.H{
			"id":         silence.ID,
			"matchers":   silence.Matchers,
			"starts_at":  silence.StartsAt,
			"ends_at":    silence.EndsAt,
			"creator":    silence.Creator,
			"comment":    silence.Comment,
			"created_at": silence.CreatedAt,
			"status":     status,
		}
	}

	h.response.Success(c, gin.H{
		"items": enrichedSilences,
		"total": len(enrichedSilences),
	}, "Silences retrieved successfully")
}

// CreateSilence creates a new silence
func (h *SilenceHandler) CreateSilence(c *gin.Context) {
	var req SilenceRequest
	if !h.response.BindAndValidate(c, &req) {
		return
	}

	// Validate time range
	if !req.EndsAt.After(req.StartsAt) {
		h.response.ValidationError(c, "End time must be after start time", nil)
		return
	}

	// Don't allow silences that end in the past
	if req.EndsAt.Before(time.Now()) {
		h.response.ValidationError(c, "End time must be in the future", nil)
		return
	}

	// Convert request to model
	silence := &models.Silence{
		Matchers: h.convertMatchersToJSONB(req.Matchers),
		StartsAt: req.StartsAt,
		EndsAt:   req.EndsAt,
		Creator:  req.Creator,
		Comment:  req.Comment,
	}

	// Validate matchers
	if err := h.validateMatchers(req.Matchers); err != nil {
		h.response.ValidationError(c, "Invalid matchers", err.Error())
		return
	}

	if err := h.services.Silence.CreateSilence(c.Request.Context(), silence); err != nil {
		h.response.InternalServerError(c, "Failed to create silence", err.Error())
		return
	}

	// Return enriched response
	status := h.getSilenceStatus(*silence, time.Now())
	responseData := gin.H{
		"id":         silence.ID,
		"matchers":   silence.Matchers,
		"starts_at":  silence.StartsAt,
		"ends_at":    silence.EndsAt,
		"creator":    silence.Creator,
		"comment":    silence.Comment,
		"created_at": silence.CreatedAt,
		"status":     status,
	}

	h.response.SuccessWithStatus(c, http.StatusCreated, responseData, "Silence created successfully")
}

// GetSilence retrieves a silence by ID
func (h *SilenceHandler) GetSilence(c *gin.Context) {
	id, ok := h.response.ParseUintParam(c, "id")
	if !ok {
		return
	}

	silence, err := h.services.Silence.GetSilence(c.Request.Context(), id)
	if err != nil {
		h.response.NotFound(c, "Silence")
		return
	}

	// Enrich with status information
	status := h.getSilenceStatus(*silence, time.Now())
	responseData := gin.H{
		"id":         silence.ID,
		"matchers":   silence.Matchers,
		"starts_at":  silence.StartsAt,
		"ends_at":    silence.EndsAt,
		"creator":    silence.Creator,
		"comment":    silence.Comment,
		"created_at": silence.CreatedAt,
		"status":     status,
	}

	h.response.Success(c, responseData, "Silence retrieved successfully")
}

// DeleteSilence deletes a silence
func (h *SilenceHandler) DeleteSilence(c *gin.Context) {
	id, ok := h.response.ParseUintParam(c, "id")
	if !ok {
		return
	}

	// Check if silence exists
	silence, err := h.services.Silence.GetSilence(c.Request.Context(), id)
	if err != nil {
		h.response.NotFound(c, "Silence")
		return
	}

	// Don't allow deletion of expired silences (they're automatically inactive)
	if silence.EndsAt.Before(time.Now()) {
		h.response.BadRequest(c, "Cannot delete expired silence", nil)
		return
	}

	if err := h.services.Silence.DeleteSilence(c.Request.Context(), id); err != nil {
		h.response.InternalServerError(c, "Failed to delete silence", err.Error())
		return
	}

	h.response.Success(c, nil, "Silence deleted successfully")
}

// TestSilence tests if a silence would match a given set of labels
func (h *SilenceHandler) TestSilence(c *gin.Context) {
	var req struct {
		Matchers []SilenceMatcher   `json:"matchers" binding:"required,min=1"`
		Labels   map[string]string  `json:"labels" binding:"required"`
	}

	if !h.response.BindAndValidate(c, &req) {
		return
	}

	// Validate matchers
	if err := h.validateMatchers(req.Matchers); err != nil {
		h.response.ValidationError(c, "Invalid matchers", err.Error())
		return
	}

	// Test if matchers would match the provided labels
	matched, err := h.testMatchers(req.Matchers, req.Labels)
	if err != nil {
		h.response.InternalServerError(c, "Failed to test matchers", err.Error())
		return
	}

	h.response.Success(c, gin.H{
		"matched":      matched,
		"matchers":     req.Matchers,
		"test_labels":  req.Labels,
	}, "Silence test completed successfully")
}

// getSilenceStatus determines the current status of a silence
func (h *SilenceHandler) getSilenceStatus(silence models.Silence, now time.Time) string {
	if now.Before(silence.StartsAt) {
		return "pending"
	}
	if now.After(silence.EndsAt) {
		return "expired"
	}
	return "active"
}

// convertMatchersToJSONB converts SilenceMatcher slice to JSONB format
func (h *SilenceHandler) convertMatchersToJSONB(matchers []SilenceMatcher) models.JSONB {
	jsonbMatchers := make([]interface{}, len(matchers))
	for i, matcher := range matchers {
		jsonbMatchers[i] = gin.H{
			"name":     matcher.Name,
			"value":    matcher.Value,
			"is_regex": matcher.IsRegex,
		}
	}
	return models.JSONB{"matchers": jsonbMatchers}
}

// validateMatchers validates silence matchers
func (h *SilenceHandler) validateMatchers(matchers []SilenceMatcher) error {
	for _, matcher := range matchers {
		// Validate matcher name (should be a valid label name)
		if matcher.Name == "" {
			return fmt.Errorf("matcher name cannot be empty")
		}

		// Validate matcher value
		if matcher.Value == "" {
			return fmt.Errorf("matcher value cannot be empty")
		}

		// If it's a regex matcher, validate the regex
		if matcher.IsRegex {
			if _, err := regexp.Compile(matcher.Value); err != nil {
				return fmt.Errorf("invalid regex pattern '%s': %v", matcher.Value, err)
			}
		}

		// Validate matcher name format (basic label name validation)
		if !isValidLabelName(matcher.Name) {
			return fmt.Errorf("invalid matcher name '%s': must be a valid label name", matcher.Name)
		}
	}

	return nil
}

// testMatchers tests if the given matchers would match the provided labels
func (h *SilenceHandler) testMatchers(matchers []SilenceMatcher, labels map[string]string) (bool, error) {
	for _, matcher := range matchers {
		labelValue, exists := labels[matcher.Name]
		if !exists {
			return false, nil // If label doesn't exist, no match
		}

		var matched bool
		if matcher.IsRegex {
			regex, err := regexp.Compile(matcher.Value)
			if err != nil {
				return false, fmt.Errorf("invalid regex pattern: %v", err)
			}
			matched = regex.MatchString(labelValue)
		} else {
			matched = labelValue == matcher.Value
		}

		if !matched {
			return false, nil
		}
	}

	return true, nil
}

// isValidLabelName checks if a string is a valid Prometheus label name
func isValidLabelName(name string) bool {
	// Basic validation for label names (simplified)
	if name == "" {
		return false
	}
	
	// Label names must start with letter or underscore, followed by letters, digits, or underscores
	matched, _ := regexp.MatchString(`^[a-zA-Z_][a-zA-Z0-9_]*$`, name)
	return matched
}