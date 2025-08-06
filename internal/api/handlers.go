package api

import (
	"net/http"

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
	response *ResponseHelper
}

func NewRoutingRuleHandler(services *service.Services) *RoutingRuleHandler {
	return &RoutingRuleHandler{
		services: services,
		response: NewResponseHelper(),
	}
}

func (h *RoutingRuleHandler) ListRules(c *gin.Context) {
	rules, err := h.services.RoutingRule.ListRules(c.Request.Context())
	if err != nil {
		h.response.InternalServerError(c, "Failed to retrieve routing rules", err.Error())
		return
	}

	h.response.Success(c, gin.H{
		"items": rules,
		"total": len(rules),
	}, "Routing rules retrieved successfully")
}

func (h *RoutingRuleHandler) CreateRule(c *gin.Context) {
	var rule models.RoutingRule
	if !h.response.BindAndValidate(c, &rule) {
		return
	}

	if err := h.services.RoutingRule.CreateRule(c.Request.Context(), &rule); err != nil {
		h.response.InternalServerError(c, "Failed to create rule", err.Error())
		return
	}

	h.response.SuccessWithStatus(c, http.StatusCreated, rule, "Rule created successfully")
}

func (h *RoutingRuleHandler) GetRule(c *gin.Context) {
	id, ok := h.response.ParseUintParam(c, "id")
	if !ok {
		return
	}

	rule, err := h.services.RoutingRule.GetRule(c.Request.Context(), id)
	if err != nil {
		h.response.NotFound(c, "Rule")
		return
	}

	h.response.Success(c, rule, "Rule retrieved successfully")
}

func (h *RoutingRuleHandler) UpdateRule(c *gin.Context) {
	id, ok := h.response.ParseUintParam(c, "id")
	if !ok {
		return
	}

	var rule models.RoutingRule
	if !h.response.BindAndValidate(c, &rule) {
		return
	}

	rule.ID = id
	if err := h.services.RoutingRule.UpdateRule(c.Request.Context(), &rule); err != nil {
		h.response.InternalServerError(c, "Failed to update rule", err.Error())
		return
	}

	h.response.Success(c, rule, "Rule updated successfully")
}

func (h *RoutingRuleHandler) DeleteRule(c *gin.Context) {
	id, ok := h.response.ParseUintParam(c, "id")
	if !ok {
		return
	}

	if err := h.services.RoutingRule.DeleteRule(c.Request.Context(), id); err != nil {
		h.response.InternalServerError(c, "Failed to delete rule", err.Error())
		return
	}

	h.response.Success(c, nil, "Rule deleted successfully")
}

func (h *RoutingRuleHandler) TestRule(c *gin.Context) {
	var req struct {
		Conditions  map[string]interface{} `json:"conditions" binding:"required"`
		SampleAlert models.Alert           `json:"sample_alert" binding:"required"`
	}

	if !h.response.BindAndValidate(c, &req) {
		return
	}

	matched, matchedRules, err := h.services.RoutingRule.TestRule(c.Request.Context(), req.Conditions, req.SampleAlert)
	if err != nil {
		h.response.InternalServerError(c, "Failed to test rule", err.Error())
		return
	}

	h.response.Success(c, gin.H{
		"matched":       matched,
		"matched_rules": matchedRules,
	}, "Rule test completed successfully")
}

// NotificationChannelHandler is now in notification_channel_handler.go
// SilenceHandler is now in silence_handler.go  
// StatsHandler is now in stats_handler.go

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