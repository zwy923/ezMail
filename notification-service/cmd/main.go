package main

import (
	"os"
	"os/signal"
	"syscall"

	"context"
	"net/http"
	"time"

	"mygoproject/pkg/db"
	"mygoproject/pkg/logger"
	"mygoproject/pkg/mq"
	"notification-service/internal/config"
	"notification-service/internal/httpserver"
	"notification-service/internal/mqhandler"
	"notification-service/internal/repository"
	"notification-service/internal/service"

	"go.uber.org/zap"
)

func main() {
	cfg := config.Load()

	log := logger.NewLogger()
	defer log.Sync()

	log.Info("Starting notification-service...",
		zap.String("db_host", cfg.DB.Host),
		zap.Int("db_port", cfg.DB.Port),
		zap.String("mq_url", cfg.MQ.URL),
	)

	// DB
	log.Info("Initializing database connection...")
	dbConn, err := db.NewConnection(cfg.DB, log)
	if err != nil {
		log.Fatal("Failed to init DB", zap.Error(err))
	}
	defer dbConn.Close()
	log.Info("Database connection established successfully")

	// MQ Publisher
	publisher, err := mq.NewPublisher(cfg.MQ.URL)
	if err != nil {
		log.Fatal("Failed to init MQ publisher", zap.Error(err))
	}
	defer publisher.Close()

	// Repositories
	notificationRepo := repository.NewNotificationRepository(dbConn, log)

	// Services
	notificationSender := service.NewNotificationSender(notificationRepo, publisher, log)

	// MQ Handlers
	notificationCreatedHandler := mqhandler.NewNotificationCreatedHandler(notificationRepo, notificationSender, log)

	// MQ Consumer for notification.created
	log.Info("Initializing MQ consumer for notification.created...",
		zap.String("queue", "notification.created.q"),
		zap.String("routing_key", "notification.created"),
	)
	consumer, err := mq.NewConsumer(cfg.MQ.URL, "notification.created.q", "notification.created", log)
	if err != nil {
		log.Fatal("Failed to init consumer", zap.Error(err))
	}
	defer consumer.Close()

	consumer.SetHandler(notificationCreatedHandler.Handle)

	go func() {
		log.Info("Starting notification.created consumer...")
		if err := consumer.StartConsuming(); err != nil {
			log.Fatal("Notification consumer failed", zap.Error(err))
		}
	}()
	log.Info("notification.created consumer started successfully")

	// HTTP Server (for health checks)
	log.Info("Initializing HTTP server...", zap.String("port", "8085"))
	router := httpserver.NewRouter(log, dbConn)
	srv := &http.Server{
		Addr:    ":8085",
		Handler: router.Engine,
	}

	go func() {
		log.Info("HTTP server starting on :8085")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal("HTTP server failed", zap.Error(err))
		}
	}()

	log.Info("notification-service is fully initialized and running")

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info("Shutting down notification-service gracefully...")

	// Stop MQ consumer
	consumer.Stop()

	// Close HTTP server
	log.Info("Shutting down HTTP server...")
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Error("HTTP server shutdown error", zap.Error(err))
	} else {
		log.Info("HTTP server stopped")
	}

	// Close connections
	publisher.Close()
	dbConn.Close()

	log.Info("notification-service shutdown complete")
}

