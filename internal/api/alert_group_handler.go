package api

import (
	"net/http"

	"alertbot/internal/models"
	"alertbot/internal/service"

	"github.com/gin-gonic/gin"
)

type AlertGroupHandler struct {
	services *service.Services
	response *ResponseHelper
}

func NewAlertGroupHandler(services *service.Services) *AlertGroupHandler {
	return &AlertGroupHandler{
		services: services,
		response: NewResponseHelper(),
	}
}

// Alert Groups Handlers

// ListAlertGroups lists all alert groups with optional filters
func (h *AlertGroupHandler) ListAlertGroups(c *gin.Context) {
	// Parse filters from query parameters
	filters := &models.AlertFilters{}
	if err := c.ShouldBindQuery(filters); err != nil {
		h.response.ValidationError(c, "Invalid query parameters", err.Error())
		return
	}
	
	// Set default pagination if not provided
	if filters.Page == 0 {
		filters.Page = 1
	}
	if filters.Size == 0 {
		filters.Size = 20
	}
	
	groups, err := h.services.AlertGroup.ListAlertGroups(c.Request.Context(), filters)
	if err != nil {
		h.response.InternalServerError(c, "Failed to retrieve alert groups", err.Error())
		return
	}
	
	// Calculate total for pagination (simplified - in production, would use separate count query)
	total := len(groups)
	pages := (total + filters.Size - 1) / filters.Size
	
	responseData := gin.H{
		"items": groups,
		"total": total,
		"page":  filters.Page,
		"size":  filters.Size,
		"pages": pages,
	}
	
	h.response.Success(c, responseData, "Alert groups retrieved successfully")
}

// GetAlertGroup retrieves a specific alert group by ID
func (h *AlertGroupHandler) GetAlertGroup(c *gin.Context) {
	id, ok := h.response.ParseUintParam(c, "id")
	if !ok {
		return
	}
	
	group, err := h.services.AlertGroup.GetAlertGroup(c.Request.Context(), id)
	if err != nil {
		h.response.NotFound(c, "Alert group")
		return
	}
	
	h.response.Success(c, group, "Alert group retrieved successfully")
}

// Alert Group Rules Handlers

// ListAlertGroupRules lists all alert group rules
func (h *AlertGroupHandler) ListAlertGroupRules(c *gin.Context) {
	rules, err := h.services.AlertGroup.ListAlertGroupRules(c.Request.Context())
	if err != nil {
		h.response.InternalServerError(c, "Failed to retrieve alert group rules", err.Error())
		return
	}
	
	responseData := gin.H{
		"items": rules,
		"total": len(rules),
	}
	
	h.response.Success(c, responseData, "Alert group rules retrieved successfully")
}

// GetAlertGroupRule retrieves a specific alert group rule by ID
func (h *AlertGroupHandler) GetAlertGroupRule(c *gin.Context) {
	id, ok := h.response.ParseUintParam(c, "id")
	if !ok {
		return
	}
	
	rule, err := h.services.AlertGroup.GetAlertGroupRule(c.Request.Context(), id)
	if err != nil {
		h.response.NotFound(c, "Alert group rule")
		return
	}
	
	h.response.Success(c, rule, "Alert group rule retrieved successfully")
}

// CreateAlertGroupRule creates a new alert group rule
func (h *AlertGroupHandler) CreateAlertGroupRule(c *gin.Context) {
	var req AlertGroupRuleRequest
	if !h.response.BindAndValidate(c, &req) {
		return
	}
	
	rule := &models.AlertGroupRule{
		Name:           req.Name,
		Description:    req.Description,
		GroupBy:        req.GroupBy,
		GroupWait:      req.GroupWait,
		GroupInterval:  req.GroupInterval,
		RepeatInterval: req.RepeatInterval,
		Matchers:       req.Matchers,
		Priority:       req.Priority,
		Enabled:        req.Enabled,
	}
	
	err := h.services.AlertGroup.CreateAlertGroupRule(c.Request.Context(), rule)
	if err != nil {
		h.response.InternalServerError(c, "Failed to create alert group rule", err.Error())
		return
	}
	
	h.response.SuccessWithStatus(c, http.StatusCreated, rule, "Alert group rule created successfully")
}

// UpdateAlertGroupRule updates an existing alert group rule
func (h *AlertGroupHandler) UpdateAlertGroupRule(c *gin.Context) {
	id, ok := h.response.ParseUintParam(c, "id")
	if !ok {
		return
	}
	
	var req AlertGroupRuleRequest
	if !h.response.BindAndValidate(c, &req) {
		return
	}
	
	updateData := &models.AlertGroupRule{
		Name:           req.Name,
		Description:    req.Description,
		GroupBy:        req.GroupBy,
		GroupWait:      req.GroupWait,
		GroupInterval:  req.GroupInterval,
		RepeatInterval: req.RepeatInterval,
		Matchers:       req.Matchers,
		Priority:       req.Priority,
		Enabled:        req.Enabled,
	}
	
	err := h.services.AlertGroup.UpdateAlertGroupRule(c.Request.Context(), id, updateData)
	if err != nil {
		h.response.InternalServerError(c, "Failed to update alert group rule", err.Error())
		return
	}
	
	// Get updated rule
	updatedRule, err := h.services.AlertGroup.GetAlertGroupRule(c.Request.Context(), id)
	if err != nil {
		h.response.InternalServerError(c, "Failed to retrieve updated rule", err.Error())
		return
	}
	
	h.response.Success(c, updatedRule, "Alert group rule updated successfully")
}

// DeleteAlertGroupRule deletes an alert group rule
func (h *AlertGroupHandler) DeleteAlertGroupRule(c *gin.Context) {
	id, ok := h.response.ParseUintParam(c, "id")
	if !ok {
		return
	}
	
	// Check if rule exists
	_, err := h.services.AlertGroup.GetAlertGroupRule(c.Request.Context(), id)
	if err != nil {
		h.response.NotFound(c, "Alert group rule")
		return
	}
	
	err = h.services.AlertGroup.DeleteAlertGroupRule(c.Request.Context(), id)
	if err != nil {
		h.response.InternalServerError(c, "Failed to delete alert group rule", err.Error())
		return
	}
	
	h.response.Success(c, nil, "Alert group rule deleted successfully")
}

// TestAlertGroupRule tests if an alert would match a group rule
func (h *AlertGroupHandler) TestAlertGroupRule(c *gin.Context) {
	var req struct {
		GroupBy models.JSONB          `json:"group_by" binding:"required"`
		Matchers models.JSONB         `json:"matchers"`
		TestAlert map[string]string   `json:"test_alert" binding:"required"`
	}
	
	if !h.response.BindAndValidate(c, &req) {
		return
	}
	
	// For now, return a simple match result
	// In a full implementation, this would use the actual grouping logic
	result := gin.H{
		"matched":     true,
		"group_key":   "test-group-key",
		"group_by":    req.GroupBy,
		"test_alert":  req.TestAlert,
		"explanation": "This is a simplified test result. The alert would be grouped based on the specified labels.",
	}
	
	h.response.Success(c, result, "Group rule test completed successfully")
}

// Request/Response types

type AlertGroupRuleRequest struct {
	Name           string       `json:"name" binding:"required,max=255"`
	Description    string       `json:"description"`
	GroupBy        models.JSONB `json:"group_by" binding:"required"`
	GroupWait      int          `json:"group_wait" binding:"min=0,max=3600"`
	GroupInterval  int          `json:"group_interval" binding:"min=60,max=86400"`
	RepeatInterval int          `json:"repeat_interval" binding:"min=300,max=604800"`
	Matchers       models.JSONB `json:"matchers"`
	Priority       int          `json:"priority"`
	Enabled        bool         `json:"enabled"`
}