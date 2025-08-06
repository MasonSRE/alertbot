package api

import (
	"alertbot/internal/config"
	"alertbot/internal/middleware"
	"alertbot/internal/service"
	"alertbot/internal/websocket"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"
)

func NewRouter(services *service.Services, logger *logrus.Logger, hub *websocket.Hub, cfg *config.Config) *gin.Engine {
	router := gin.New()

	// 全局中间件 - 顺序很重要
	router.Use(middleware.CORS())
	router.Use(middleware.ErrorHandler(logger))  // 错误处理中间件必须在最前面
	router.Use(middleware.Logging(logger))
	router.Use(middleware.MetricsMiddleware())
	router.Use(middleware.RateLimit(cfg, logger))
	// 移除 gin.Recovery() 因为 ErrorHandler 已经包含了 panic 恢复

	// 健康检查
	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status": "ok",
			"service": "alertbot",
			"version": "1.0.0",
		})
	})

	// Prometheus metrics endpoint
	router.GET("/metrics", gin.WrapH(promhttp.Handler()))

	// API v1路由组
	v1 := router.Group("/api/v1")
	{
		// 认证相关路由（无需认证）
		authHandler := NewAuthHandler(services)
		auth := v1.Group("/auth")
		{
			auth.POST("/login", authHandler.Login)
			auth.POST("/logout", authHandler.Logout)
			auth.POST("/refresh", authHandler.RefreshToken)
			auth.GET("/profile", authHandler.GetProfile) // In real implementation, this would require JWT auth
		}
		// 告警相关路由
		alertHandler := NewAlertHandler(services)
		alerts := v1.Group("/alerts")
		{
			alerts.POST("", alertHandler.ReceiveAlerts)
			alerts.GET("", alertHandler.ListAlerts)
			alerts.GET("/:fingerprint", alertHandler.GetAlert)
			alerts.PUT("/:fingerprint/silence", alertHandler.SilenceAlert)
			alerts.PUT("/:fingerprint/ack", alertHandler.AcknowledgeAlert)
			alerts.DELETE("/:fingerprint", alertHandler.ResolveAlert)
		}

		// 规则相关路由
		ruleHandler := NewRoutingRuleHandler(services)
		rules := v1.Group("/rules")
		{
			rules.GET("", ruleHandler.ListRules)
			rules.POST("", ruleHandler.CreateRule)
			rules.GET("/:id", ruleHandler.GetRule)
			rules.PUT("/:id", ruleHandler.UpdateRule)
			rules.DELETE("/:id", ruleHandler.DeleteRule)
			rules.POST("/test", ruleHandler.TestRule)
		}

		// 通知渠道相关路由
		channelHandler := NewNotificationChannelHandler(services)
		channels := v1.Group("/channels")
		{
			channels.GET("", channelHandler.ListChannels)
			channels.POST("", channelHandler.CreateChannel)
			channels.GET("/:id", channelHandler.GetChannel)
			channels.PUT("/:id", channelHandler.UpdateChannel)
			channels.DELETE("/:id", channelHandler.DeleteChannel)
			channels.POST("/:id/test", channelHandler.TestChannel)
		}

		// 静默相关路由
		silenceHandler := NewSilenceHandler(services)
		silences := v1.Group("/silences")
		{
			silences.GET("", silenceHandler.ListSilences)
			silences.POST("", silenceHandler.CreateSilence)
			silences.GET("/:id", silenceHandler.GetSilence)
			silences.DELETE("/:id", silenceHandler.DeleteSilence)
			silences.POST("/test", silenceHandler.TestSilence) // Added test endpoint
		}

		// 统计相关路由
		statsHandler := NewStatsHandler(services)
		stats := v1.Group("/stats")
		{
			stats.GET("/alerts", statsHandler.GetAlertStats)
			stats.GET("/notifications", statsHandler.GetNotificationStats)
			stats.GET("/system", statsHandler.GetSystemStats)
		}

		// 系统健康检查路由
		v1.GET("/health", statsHandler.GetHealthStatus)

		// WebSocket路由
		wsHandler := NewWebSocketHandler(services, logger, hub)
		v1.GET("/ws/alerts", middleware.OptionalJWTAuth(cfg), wsHandler.HandleWebSocket)
	}

	return router
}