package api

import (
	"net/http"

	"alertbot/internal/models"
	"alertbot/internal/service"

	"github.com/gin-gonic/gin"
)

type AlertHandler struct {
	services *service.Services
}

func NewAlertHandler(services *service.Services) *AlertHandler {
	return &AlertHandler{services: services}
}

func (h *AlertHandler) ReceiveAlerts(c *gin.Context) {
	var alerts []models.PrometheusAlert
	if err := c.ShouldBindJSON(&alerts); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "INVALID_REQUEST",
				"message": "Invalid alert format",
				"details": err.Error(),
			},
		})
		return
	}

	if len(alerts) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "EMPTY_ALERTS",
				"message": "No alerts provided",
			},
		})
		return
	}

	err := h.services.Alert.ReceiveAlerts(c.Request.Context(), alerts)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "INTERNAL_ERROR",
				"message": "Failed to process alerts",
				"details": err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"received":   len(alerts),
			"processed":  len(alerts),
			"duplicates": 0,
		},
		"message": "Alerts processed successfully",
	})
}

func (h *AlertHandler) ListAlerts(c *gin.Context) {
	var filters models.AlertFilters
	if err := c.ShouldBindQuery(&filters); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "INVALID_QUERY",
				"message": "Invalid query parameters",
				"details": err.Error(),
			},
		})
		return
	}

	alerts, total, err := h.services.Alert.ListAlerts(c.Request.Context(), filters)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "INTERNAL_ERROR",
				"message": "Failed to retrieve alerts",
				"details": err.Error(),
			},
		})
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

	pages := int((total + int64(size) - 1) / int64(size))

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"items": alerts,
			"total": total,
			"page":  page,
			"size":  size,
			"pages": pages,
		},
	})
}

func (h *AlertHandler) GetAlert(c *gin.Context) {
	fingerprint := c.Param("fingerprint")
	if fingerprint == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "MISSING_FINGERPRINT",
				"message": "Alert fingerprint is required",
			},
		})
		return
	}

	alert, err := h.services.Alert.GetAlert(c.Request.Context(), fingerprint)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "NOT_FOUND",
				"message": "Alert not found",
			},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    alert,
	})
}

func (h *AlertHandler) SilenceAlert(c *gin.Context) {
	fingerprint := c.Param("fingerprint")
	if fingerprint == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "MISSING_FINGERPRINT",
				"message": "Alert fingerprint is required",
			},
		})
		return
	}

	var req struct {
		Duration string `json:"duration" binding:"required"`
		Comment  string `json:"comment"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "INVALID_REQUEST",
				"message": "Invalid request body",
				"details": err.Error(),
			},
		})
		return
	}

	err := h.services.Alert.SilenceAlert(c.Request.Context(), fingerprint, req.Duration, req.Comment)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "INTERNAL_ERROR",
				"message": "Failed to silence alert",
				"details": err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Alert silenced successfully",
	})
}

func (h *AlertHandler) AcknowledgeAlert(c *gin.Context) {
	fingerprint := c.Param("fingerprint")
	if fingerprint == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "MISSING_FINGERPRINT",
				"message": "Alert fingerprint is required",
			},
		})
		return
	}

	var req struct {
		Comment string `json:"comment"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "INVALID_REQUEST",
				"message": "Invalid request body",
				"details": err.Error(),
			},
		})
		return
	}

	err := h.services.Alert.AcknowledgeAlert(c.Request.Context(), fingerprint, req.Comment)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "INTERNAL_ERROR",
				"message": "Failed to acknowledge alert",
				"details": err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Alert acknowledged successfully",
	})
}

func (h *AlertHandler) ResolveAlert(c *gin.Context) {
	fingerprint := c.Param("fingerprint")
	if fingerprint == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "MISSING_FINGERPRINT",
				"message": "Alert fingerprint is required",
			},
		})
		return
	}

	var req struct {
		Comment string `json:"comment"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		req.Comment = "Manually resolved"
	}

	err := h.services.Alert.ResolveAlert(c.Request.Context(), fingerprint, req.Comment)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "INTERNAL_ERROR",
				"message": "Failed to resolve alert",
				"details": err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Alert resolved successfully",
	})
}