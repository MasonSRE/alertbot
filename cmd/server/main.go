package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"alertbot/internal/api"
	"alertbot/internal/config"
	"alertbot/internal/monitor"
	"alertbot/internal/repository"
	"alertbot/internal/service"
	"alertbot/internal/websocket"
	"alertbot/pkg/logger"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		logrus.Fatalf("Failed to load config: %v", err)
	}

	log := logger.New(cfg.Logger)

	db, err := repository.NewDatabase(cfg.Database)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	repos := repository.NewRepositories(db)
	
	// Initialize WebSocket hub
	hub := websocket.NewHub(log)
	go hub.Run()

	// Initialize system monitor
	systemMonitor := monitor.NewSystemMonitor(log, 30*time.Second)
	go systemMonitor.Start()
	
	services := service.NewServices(service.ServiceDependencies{
		Repositories: repos,
		Logger:       log,
		Config:       cfg,
		WebSocketHub: hub,
	})

	if cfg.Env == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := api.NewRouter(services, log, hub, cfg)

	srv := &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.Server.Port),
		Handler: router,
	}

	go func() {
		log.Infof("Starting AlertBot server on port %d", cfg.Server.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Info("Server exited gracefully")
}