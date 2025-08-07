package api

import (
	"alertbot/internal/models"
	"alertbot/internal/service"
	"strconv"

	"github.com/gin-gonic/gin"
)

type InhibitionHandler struct {
	services *service.Services
	response *ResponseHelper
}

func NewInhibitionHandler(services *service.Services) *InhibitionHandler {
	return &InhibitionHandler{
		services: services,
		response: NewResponseHelper(),
	}
}

func (h *InhibitionHandler) ListInhibitionRules(c *gin.Context) {
	rules, err := h.services.Inhibition.ListInhibitionRules(c.Request.Context())
	if err != nil {
		h.response.InternalServerError(c, "Failed to retrieve inhibition rules", err.Error())
		return
	}

	h.response.Success(c, rules, "Inhibition rules retrieved successfully")
}

func (h *InhibitionHandler) GetInhibitionRule(c *gin.Context) {
	idStr := c.Param("id")
	if idStr == "" {
		h.response.BadRequest(c, "Inhibition rule ID is required", nil)
		return
	}

	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		h.response.BadRequest(c, "Invalid inhibition rule ID", nil)
		return
	}

	rule, err := h.services.Inhibition.GetInhibitionRule(c.Request.Context(), uint(id))
	if err != nil {
		h.response.NotFound(c, "Inhibition rule")
		return
	}

	h.response.Success(c, rule, "Inhibition rule retrieved successfully")
}

func (h *InhibitionHandler) CreateInhibitionRule(c *gin.Context) {
	var rule models.InhibitionRule
	if !h.response.BindAndValidate(c, &rule) {
		return
	}

	if err := h.services.Inhibition.CreateInhibitionRule(c.Request.Context(), &rule); err != nil {
		h.response.InternalServerError(c, "Failed to create inhibition rule", err.Error())
		return
	}

	h.response.SuccessWithStatus(c, 201, rule, "Inhibition rule created successfully")
}

func (h *InhibitionHandler) UpdateInhibitionRule(c *gin.Context) {
	idStr := c.Param("id")
	if idStr == "" {
		h.response.BadRequest(c, "Inhibition rule ID is required", nil)
		return
	}

	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		h.response.BadRequest(c, "Invalid inhibition rule ID", nil)
		return
	}

	var rule models.InhibitionRule
	if !h.response.BindAndValidate(c, &rule) {
		return
	}

	if err := h.services.Inhibition.UpdateInhibitionRule(c.Request.Context(), uint(id), &rule); err != nil {
		h.response.InternalServerError(c, "Failed to update inhibition rule", err.Error())
		return
	}

	rule.ID = uint(id)
	h.response.Success(c, rule, "Inhibition rule updated successfully")
}

func (h *InhibitionHandler) DeleteInhibitionRule(c *gin.Context) {
	idStr := c.Param("id")
	if idStr == "" {
		h.response.BadRequest(c, "Inhibition rule ID is required", nil)
		return
	}

	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		h.response.BadRequest(c, "Invalid inhibition rule ID", nil)
		return
	}

	if err := h.services.Inhibition.DeleteInhibitionRule(c.Request.Context(), uint(id)); err != nil {
		h.response.InternalServerError(c, "Failed to delete inhibition rule", err.Error())
		return
	}

	h.response.Success(c, nil, "Inhibition rule deleted successfully")
}

func (h *InhibitionHandler) TestInhibitionRule(c *gin.Context) {
	var req struct {
		Rule        models.InhibitionRule `json:"rule" binding:"required"`
		SourceAlert map[string]string     `json:"source_alert" binding:"required"`
		TargetAlert map[string]string     `json:"target_alert" binding:"required"`
	}

	if !h.response.BindAndValidate(c, &req) {
		return
	}

	result, err := h.services.Inhibition.TestInhibitionRule(c.Request.Context(), &req.Rule, req.SourceAlert, req.TargetAlert)
	if err != nil {
		h.response.InternalServerError(c, "Failed to test inhibition rule", err.Error())
		return
	}

	h.response.Success(c, gin.H{
		"inhibited": result,
		"test_rule": req.Rule.Name,
	}, "Inhibition rule test completed successfully")
}