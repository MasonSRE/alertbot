package api

import (
	"alertbot/internal/models"
	"alertbot/internal/service"

	"github.com/gin-gonic/gin"
)

type AlertHandler struct {
	services *service.Services
	response *ResponseHelper
}

func NewAlertHandler(services *service.Services) *AlertHandler {
	return &AlertHandler{
		services: services,
		response: NewResponseHelper(),
	}
}

func (h *AlertHandler) ReceiveAlerts(c *gin.Context) {
	var alerts []models.PrometheusAlert
	if !h.response.BindAndValidate(c, &alerts) {
		return
	}

	if len(alerts) == 0 {
		h.response.BadRequest(c, "No alerts provided", nil)
		return
	}

	err := h.services.Alert.ReceiveAlerts(c.Request.Context(), alerts)
	if err != nil {
		h.response.InternalServerError(c, "Failed to process alerts", err.Error())
		return
	}

	h.response.Success(c, gin.H{
		"received":   len(alerts),
		"processed":  len(alerts),
		"duplicates": 0,
	}, "Alerts processed successfully")
}

func (h *AlertHandler) ListAlerts(c *gin.Context) {
	var filters models.AlertFilters
	if !h.response.BindQueryAndValidate(c, &filters) {
		return
	}

	alerts, total, err := h.services.Alert.ListAlerts(c.Request.Context(), filters)
	if err != nil {
		h.response.InternalServerError(c, "Failed to retrieve alerts", err.Error())
		return
	}

	page := filters.Page
	if page == 0 {
		page = 1
	}
	size := filters.Size
	if size == 0 {
		size = 20
	}

	h.response.Paginated(c, alerts, total, page, size, "Alerts retrieved successfully")
}

func (h *AlertHandler) GetAlert(c *gin.Context) {
	fingerprint := c.Param("fingerprint")
	if fingerprint == "" {
		h.response.BadRequest(c, "Alert fingerprint is required", nil)
		return
	}

	alert, err := h.services.Alert.GetAlert(c.Request.Context(), fingerprint)
	if err != nil {
		h.response.NotFound(c, "Alert")
		return
	}

	h.response.Success(c, alert, "Alert retrieved successfully")
}

func (h *AlertHandler) SilenceAlert(c *gin.Context) {
	fingerprint := c.Param("fingerprint")
	if fingerprint == "" {
		h.response.BadRequest(c, "Alert fingerprint is required", nil)
		return
	}

	var req struct {
		Duration string `json:"duration" binding:"required"`
		Comment  string `json:"comment"`
	}

	if !h.response.BindAndValidate(c, &req) {
		return
	}

	err := h.services.Alert.SilenceAlert(c.Request.Context(), fingerprint, req.Duration, req.Comment)
	if err != nil {
		h.response.InternalServerError(c, "Failed to silence alert", err.Error())
		return
	}

	h.response.Success(c, nil, "Alert silenced successfully")
}

func (h *AlertHandler) AcknowledgeAlert(c *gin.Context) {
	fingerprint := c.Param("fingerprint")
	if fingerprint == "" {
		h.response.BadRequest(c, "Alert fingerprint is required", nil)
		return
	}

	var req struct {
		Comment string `json:"comment"`
	}

	if !h.response.BindAndValidate(c, &req) {
		return
	}

	err := h.services.Alert.AcknowledgeAlert(c.Request.Context(), fingerprint, req.Comment)
	if err != nil {
		h.response.InternalServerError(c, "Failed to acknowledge alert", err.Error())
		return
	}

	h.response.Success(c, nil, "Alert acknowledged successfully")
}

func (h *AlertHandler) ResolveAlert(c *gin.Context) {
	fingerprint := c.Param("fingerprint")
	if fingerprint == "" {
		h.response.BadRequest(c, "Alert fingerprint is required", nil)
		return
	}

	var req struct {
		Comment string `json:"comment"`
	}

	// For resolve, comment is optional, so don't fail if bind fails
	if err := c.ShouldBindJSON(&req); err != nil {
		req.Comment = "Manually resolved"
	}

	err := h.services.Alert.ResolveAlert(c.Request.Context(), fingerprint, req.Comment)
	if err != nil {
		h.response.InternalServerError(c, "Failed to resolve alert", err.Error())
		return
	}

	h.response.Success(c, nil, "Alert resolved successfully")
}

func (h *AlertHandler) BatchSilenceAlerts(c *gin.Context) {
	var req struct {
		Fingerprints []string `json:"fingerprints" binding:"required"`
		Duration     string   `json:"duration" binding:"required"`
		Comment      string   `json:"comment"`
	}

	if !h.response.BindAndValidate(c, &req) {
		return
	}

	if len(req.Fingerprints) == 0 {
		h.response.BadRequest(c, "No alert fingerprints provided", nil)
		return
	}

	err := h.services.Alert.BatchSilenceAlerts(c.Request.Context(), req.Fingerprints, req.Duration, req.Comment)
	if err != nil {
		h.response.InternalServerError(c, "Failed to batch silence alerts", err.Error())
		return
	}

	h.response.Success(c, gin.H{
		"processed": len(req.Fingerprints),
		"action":    "silenced",
	}, "Alerts silenced successfully")
}

func (h *AlertHandler) BatchAcknowledgeAlerts(c *gin.Context) {
	var req struct {
		Fingerprints []string `json:"fingerprints" binding:"required"`
		Comment      string   `json:"comment"`
	}

	if !h.response.BindAndValidate(c, &req) {
		return
	}

	if len(req.Fingerprints) == 0 {
		h.response.BadRequest(c, "No alert fingerprints provided", nil)
		return
	}

	err := h.services.Alert.BatchAcknowledgeAlerts(c.Request.Context(), req.Fingerprints, req.Comment)
	if err != nil {
		h.response.InternalServerError(c, "Failed to batch acknowledge alerts", err.Error())
		return
	}

	h.response.Success(c, gin.H{
		"processed": len(req.Fingerprints),
		"action":    "acknowledged",
	}, "Alerts acknowledged successfully")
}

func (h *AlertHandler) BatchResolveAlerts(c *gin.Context) {
	var req struct {
		Fingerprints []string `json:"fingerprints" binding:"required"`
		Comment      string   `json:"comment"`
	}

	if !h.response.BindAndValidate(c, &req) {
		return
	}

	if len(req.Fingerprints) == 0 {
		h.response.BadRequest(c, "No alert fingerprints provided", nil)
		return
	}

	err := h.services.Alert.BatchResolveAlerts(c.Request.Context(), req.Fingerprints, req.Comment)
	if err != nil {
		h.response.InternalServerError(c, "Failed to batch resolve alerts", err.Error())
		return
	}

	h.response.Success(c, gin.H{
		"processed": len(req.Fingerprints),
		"action":    "resolved",
	}, "Alerts resolved successfully")
}

func (h *AlertHandler) GetAlertHistory(c *gin.Context) {
	fingerprint := c.Param("fingerprint")
	if fingerprint == "" {
		h.response.BadRequest(c, "Alert fingerprint is required", nil)
		return
	}

	history, err := h.services.Alert.GetAlertHistory(c.Request.Context(), fingerprint)
	if err != nil {
		h.response.InternalServerError(c, "Failed to retrieve alert history", err.Error())
		return
	}

	h.response.Success(c, history, "Alert history retrieved successfully")
}

func (h *AlertHandler) ListAlertHistory(c *gin.Context) {
	var filters models.AlertHistoryFilters
	if !h.response.BindQueryAndValidate(c, &filters) {
		return
	}

	history, total, err := h.services.Alert.ListAlertHistory(c.Request.Context(), filters)
	if err != nil {
		h.response.InternalServerError(c, "Failed to retrieve alert history", err.Error())
		return
	}

	page := filters.Page
	if page == 0 {
		page = 1
	}
	size := filters.Size
	if size == 0 {
		size = 50
	}

	h.response.Paginated(c, history, total, page, size, "Alert history retrieved successfully")
}