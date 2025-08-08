package api

import (
	"alertbot/internal/config"
	"alertbot/internal/middleware"
	"alertbot/internal/monitoring"
	"alertbot/internal/service"
	"alertbot/internal/websocket"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"
)

func NewRouter(services *service.Services, logger *logrus.Logger, hub *websocket.Hub, cfg *config.Config, monitoringService *monitoring.MonitoringService, backgroundMonitor *monitoring.BackgroundMonitor) *gin.Engine {
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
	
	// Alertmanager compatibility endpoints (for direct Prometheus integration)
	router.GET("/api/v1/status", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"cluster": gin.H{
				"status": "ready",
				"name":   "alertbot",
			},
			"versionInfo": gin.H{
				"version":   "1.0.0",
				"revision":  "alertbot",
				"branch":    "main",
				"goVersion": "go1.21",
			},
			"uptime": "0h",
		})
	})

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
			alerts.GET("/:fingerprint/history", alertHandler.GetAlertHistory)
			alerts.GET("/:fingerprint/relations", alertHandler.GetAlertRelations)
			// 批量操作路由
			alerts.PUT("/batch/silence", alertHandler.BatchSilenceAlerts)
			alerts.PUT("/batch/ack", alertHandler.BatchAcknowledgeAlerts)
			alerts.DELETE("/batch/resolve", alertHandler.BatchResolveAlerts)
		}
		
		// 告警去重配置路由
		deduplication := v1.Group("/deduplication")
		{
			deduplication.GET("/config", alertHandler.GetDeduplicationConfig)
			deduplication.PUT("/config", alertHandler.UpdateDeduplicationConfig)
		}
		
		// 告警历史路由
		alertHistory := v1.Group("/alert-history")
		{
			alertHistory.GET("", alertHandler.ListAlertHistory)
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

		// 抑制规则相关路由
		inhibitionHandler := NewInhibitionHandler(services)
		inhibitions := v1.Group("/inhibitions")
		{
			inhibitions.GET("", inhibitionHandler.ListInhibitionRules)
			inhibitions.POST("", inhibitionHandler.CreateInhibitionRule)
			inhibitions.GET("/:id", inhibitionHandler.GetInhibitionRule)
			inhibitions.PUT("/:id", inhibitionHandler.UpdateInhibitionRule)
			inhibitions.DELETE("/:id", inhibitionHandler.DeleteInhibitionRule)
			inhibitions.POST("/test", inhibitionHandler.TestInhibitionRule)
		}

		// 告警分组相关路由
		groupHandler := NewAlertGroupHandler(services)
		groups := v1.Group("/alert-groups")
		{
			groups.GET("", groupHandler.ListAlertGroups)
			groups.GET("/:id", groupHandler.GetAlertGroup)
		}
		
		// 告警分组规则相关路由
		groupRules := v1.Group("/alert-group-rules")
		{
			groupRules.GET("", groupHandler.ListAlertGroupRules)
			groupRules.POST("", groupHandler.CreateAlertGroupRule)
			groupRules.GET("/:id", groupHandler.GetAlertGroupRule)
			groupRules.PUT("/:id", groupHandler.UpdateAlertGroupRule)
			groupRules.DELETE("/:id", groupHandler.DeleteAlertGroupRule)
			groupRules.POST("/test", groupHandler.TestAlertGroupRule)
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

		// 监控相关路由
		monitoringHandler := NewMonitoringHandler(monitoringService, backgroundMonitor)
		monitoring := v1.Group("/monitoring")
		{
			monitoring.GET("/health", monitoringHandler.GetSystemHealth)
			monitoring.GET("/health/simple", monitoringHandler.GetHealthCheck)
			monitoring.GET("/health/ready", monitoringHandler.GetReadinessCheck)
			monitoring.GET("/health/live", monitoringHandler.GetLivenessCheck)
			monitoring.GET("/health/component/:component", monitoringHandler.GetComponentHealth)
			monitoring.GET("/health/history", monitoringHandler.GetHealthHistory)
			monitoring.POST("/health/trigger", monitoringHandler.TriggerHealthCheck)
			
			monitoring.GET("/metrics/summary", monitoringHandler.GetMetricsSummary)
			monitoring.GET("/metrics/performance", monitoringHandler.GetPerformanceStats)
			monitoring.GET("/metrics/system", monitoringHandler.GetSystemMetrics)
			monitoring.GET("/metrics/export", monitoringHandler.ExportMetrics)
			
			monitoring.GET("/system/info", monitoringHandler.GetSystemInfo)
			monitoring.GET("/alerts", monitoringHandler.GetAlerts)
			
			monitoring.GET("/config", monitoringHandler.GetMonitoringConfig)
			monitoring.PUT("/config", monitoringHandler.UpdateMonitoringConfig)
		}

		// 设置相关路由
		settingsHandler := NewSettingsHandler(services.Settings)
		settings := v1.Group("/settings")
		{
			settings.GET("/system", settingsHandler.GetSystemSettings)
			settings.PUT("/system", settingsHandler.UpdateSystemSettings)
			settings.GET("/prometheus", settingsHandler.GetPrometheusSettings)
			settings.PUT("/prometheus", settingsHandler.UpdatePrometheusSettings)
			settings.POST("/prometheus/test", settingsHandler.TestPrometheusConnection)
			settings.GET("/notification", settingsHandler.GetNotificationSettings)
			settings.PUT("/notification", settingsHandler.UpdateNotificationSettings)
		}

		// WebSocket路由
		wsHandler := NewWebSocketHandler(services, logger, hub)
		v1.GET("/ws/alerts", middleware.OptionalJWTAuth(cfg), wsHandler.HandleWebSocket)
	}

	// Prometheus v2 API兼容路由
	// Prometheus 2.x默认使用/api/v2/alerts路径发送告警
	v2 := router.Group("/api/v2")
	{
		alertHandler := NewAlertHandler(services)
		// 将Prometheus v2的告警转发到v1处理器
		v2.POST("/alerts", alertHandler.ReceiveAlerts)
	}

	// 同时支持Prometheus配置了path_prefix的情况
	// 处理错误的双重路径问题 /api/v1/api/v2/alerts
	v1compat := router.Group("/api/v1/api/v2")
	{
		alertHandler := NewAlertHandler(services)
		v1compat.POST("/alerts", alertHandler.ReceiveAlerts)
	}

	return router
}