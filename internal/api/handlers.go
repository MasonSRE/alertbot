package api

import (
	"net/http"
	"strconv"

	"alertbot/internal/models"
	"alertbot/internal/service"
	websocketPkg "alertbot/internal/websocket"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
)

// 占位符处理器，后续会实现具体功能

type RoutingRuleHandler struct {
	services *service.Services
}

func NewRoutingRuleHandler(services *service.Services) *RoutingRuleHandler {
	return &RoutingRuleHandler{services: services}
}

func (h *RoutingRuleHandler) ListRules(c *gin.Context) {
	rules, err := h.services.RoutingRule.ListRules(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "INTERNAL_ERROR",
				"message": "Failed to retrieve routing rules",
				"details": err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"items": rules,
			"total": len(rules),
		},
	})
}

func (h *RoutingRuleHandler) CreateRule(c *gin.Context) {
	var rule models.RoutingRule
	if err := c.ShouldBindJSON(&rule); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "INVALID_REQUEST",
				"message": "Invalid rule format",
				"details": err.Error(),
			},
		})
		return
	}

	if err := h.services.RoutingRule.CreateRule(c.Request.Context(), &rule); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "INTERNAL_ERROR",
				"message": "Failed to create rule",
				"details": err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"data":    rule,
		"message": "Rule created successfully",
	})
}

func (h *RoutingRuleHandler) GetRule(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "INVALID_ID",
				"message": "Invalid rule ID",
			},
		})
		return
	}

	rule, err := h.services.RoutingRule.GetRule(c.Request.Context(), uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "NOT_FOUND",
				"message": "Rule not found",
			},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    rule,
	})
}

func (h *RoutingRuleHandler) UpdateRule(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "INVALID_ID",
				"message": "Invalid rule ID",
			},
		})
		return
	}

	var rule models.RoutingRule
	if err := c.ShouldBindJSON(&rule); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "INVALID_REQUEST",
				"message": "Invalid rule format",
				"details": err.Error(),
			},
		})
		return
	}

	rule.ID = uint(id)
	if err := h.services.RoutingRule.UpdateRule(c.Request.Context(), &rule); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "INTERNAL_ERROR",
				"message": "Failed to update rule",
				"details": err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    rule,
		"message": "Rule updated successfully",
	})
}

func (h *RoutingRuleHandler) DeleteRule(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "INVALID_ID",
				"message": "Invalid rule ID",
			},
		})
		return
	}

	if err := h.services.RoutingRule.DeleteRule(c.Request.Context(), uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "INTERNAL_ERROR",
				"message": "Failed to delete rule",
				"details": err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Rule deleted successfully",
	})
}

func (h *RoutingRuleHandler) TestRule(c *gin.Context) {
	var req struct {
		Conditions  map[string]interface{} `json:"conditions" binding:"required"`
		SampleAlert models.Alert           `json:"sample_alert" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "INVALID_REQUEST",
				"message": "Invalid test request format",
				"details": err.Error(),
			},
		})
		return
	}

	matched, matchedRules, err := h.services.RoutingRule.TestRule(c.Request.Context(), req.Conditions, req.SampleAlert)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "INTERNAL_ERROR",
				"message": "Failed to test rule",
				"details": err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"matched":       matched,
			"matched_rules": matchedRules,
		},
	})
}

type NotificationChannelHandler struct {
	services *service.Services
}

func NewNotificationChannelHandler(services *service.Services) *NotificationChannelHandler {
	return &NotificationChannelHandler{services: services}
}

func (h *NotificationChannelHandler) ListChannels(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"success": true, "data": []interface{}{}})
}

func (h *NotificationChannelHandler) CreateChannel(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "Channel created"})
}

func (h *NotificationChannelHandler) GetChannel(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{}})
}

func (h *NotificationChannelHandler) UpdateChannel(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "Channel updated"})
}

func (h *NotificationChannelHandler) DeleteChannel(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "Channel deleted"})
}

func (h *NotificationChannelHandler) TestChannel(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "Test sent"})
}

type SilenceHandler struct {
	services *service.Services
}

func NewSilenceHandler(services *service.Services) *SilenceHandler {
	return &SilenceHandler{services: services}
}

func (h *SilenceHandler) ListSilences(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"success": true, "data": []interface{}{}})
}

func (h *SilenceHandler) CreateSilence(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "Silence created"})
}

func (h *SilenceHandler) GetSilence(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{}})
}

func (h *SilenceHandler) DeleteSilence(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "Silence deleted"})
}

type StatsHandler struct {
	services *service.Services
}

func NewStatsHandler(services *service.Services) *StatsHandler {
	return &StatsHandler{services: services}
}

func (h *StatsHandler) GetAlertStats(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{"total_alerts": 0}})
}

func (h *StatsHandler) GetNotificationStats(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{"total_sent": 0}})
}

type WebSocketHandler struct {
	services *service.Services
	logger   *logrus.Logger
	hub      *websocketPkg.Hub
}

func NewWebSocketHandler(services *service.Services, logger *logrus.Logger, hub *websocketPkg.Hub) *WebSocketHandler {
	return &WebSocketHandler{
		services: services,
		logger:   logger,
		hub:      websocketPkg.NewHub(logger),
	}
}

func (h *WebSocketHandler) HandleWebSocket(c *gin.Context) {
	// Upgrade HTTP connection to WebSocket
	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			// TODO: Implement proper origin checking in production
			return true
		},
	}

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		h.logger.WithError(err).Error("Failed to upgrade WebSocket connection")
		return
	}

	// Get user information from context (set by auth middleware)
	userID, _ := c.Get("user_id")
	username, _ := c.Get("username")
	role, _ := c.Get("role")

	// Create and start new client
	client := h.hub.NewClient(conn, getUserID(userID), getUserString(username), getUserString(role))
	client.Start()

	h.logger.WithFields(logrus.Fields{
		"client_id": client.GetInfo()["id"],
		"user_id":   getUserID(userID),
		"username":  getUserString(username),
	}).Info("WebSocket client connected")
}

func getUserID(userID interface{}) uint {
	if id, ok := userID.(uint); ok {
		return id
	}
	return 0
}

func getUserString(value interface{}) string {
	if str, ok := value.(string); ok {
		return str
	}
	return ""
}