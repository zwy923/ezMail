package main

import (
	"log"
	"os"

	"api-gateway/internal/config"
	"api-gateway/internal/handler"
	"api-gateway/internal/httpserver"
	"api-gateway/internal/repository"
	"api-gateway/internal/service/auth"
	"context"
	"mygoproject/pkg/db"
	"mygoproject/pkg/logger"
	"mygoproject/pkg/mq"
	"mygoproject/pkg/otel"
	"mygoproject/pkg/outbox"

	"go.uber.org/zap"
)

func main() {
	// Load config
	cfg := config.Load()

	logger := logger.NewLogger()
	defer logger.Sync()

	// Init OpenTelemetry
	otelEndpoint := os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")
	if otelEndpoint == "" {
		otelEndpoint = "otel-collector:4317"
	}
	shutdown, err := otel.Init(otel.Config{
		ServiceName:    "api-gateway",
		ServiceVersion: "1.0.0",
		Endpoint:       otelEndpoint,
		Enabled:        true,
	}, logger)
	if err != nil {
		logger.Fatal("Failed to init OpenTelemetry", zap.Error(err))
	}
	defer shutdown()

	// Init DB (for user auth and email query)
	dbConn, err := db.NewConnection(cfg.DB, logger)
	if err != nil {
		logger.Fatal("DB initialization failed", zap.Error(err))
	}
	defer dbConn.Close()

	// Init Repositories
	userRepo := repository.NewUserRepository(dbConn)
	emailRepo := repository.NewEmailRepository(dbConn)

	// Init MQ Publisher
	taskPublisher, err := mq.NewPublisher(cfg.MQ.URL)
	if err != nil {
		logger.Fatal("Failed to init MQ publisher", zap.Error(err))
	}
	defer taskPublisher.Close()

	// Init Services
	authService := auth.NewService(userRepo, cfg.JWT.Secret)

	// Init Outbox
	outboxRepo := outbox.NewRepository(dbConn)
	replayService := outbox.NewReplayService(outboxRepo, taskPublisher)

	// Init Handlers
	authHandler := handler.NewAuthHandler(authService)
	mailProxyHandler := handler.NewMailProxyHandler(cfg.MailIngestionServiceURL)
	emailQueryHandler := handler.NewEmailQueryHandler(emailRepo)
	taskController := handler.NewTaskController(dbConn, cfg.AgentServiceURL, cfg.TaskServiceURL, taskPublisher, logger)
	adminHandler := handler.NewAdminHandler(replayService, logger)

	// Init Outbox Dispatcher
	dispatcher := outbox.NewDispatcher(outboxRepo, taskPublisher, logger)
	go dispatcher.Start(context.Background())

	// Router
	router := httpserver.NewRouter(
		authHandler,
		mailProxyHandler,
		emailQueryHandler,
		taskController,
		adminHandler,
		cfg.JWT.Secret,
		dbConn,
	)

	// Start API server
	logger.Info("Starting API Gateway", zap.String("port", cfg.Server.Port))
	if err := router.Run(cfg.Server.Port); err != nil {
		log.Fatalf("server start failed: %v", err)
	}
}
