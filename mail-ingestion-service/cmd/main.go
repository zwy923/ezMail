package main

import (
	"log"
	"os"

	"mail-ingestion-service/internal/config"
	"mail-ingestion-service/internal/handler"
	"mail-ingestion-service/internal/httpserver"
	"mail-ingestion-service/internal/repository"
	"mail-ingestion-service/internal/service/ingest"
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
		ServiceName:    "mail-ingestion-service",
		ServiceVersion: "1.0.0",
		Endpoint:       otelEndpoint,
		Enabled:        true,
	}, logger)
	if err != nil {
		logger.Fatal("Failed to init OpenTelemetry", zap.Error(err))
	}
	defer shutdown()

	// Init DB
	dbConn, err := db.NewConnection(cfg.DB, logger)
	if err != nil {
		logger.Fatal("DB initialization failed", zap.Error(err))
	}
	defer dbConn.Close()

	// Init RabbitMQ Publisher
	publisher, err := mq.NewPublisher(cfg.MQ.URL)
	if err != nil {
		log.Fatalf("failed to init publisher: %v", err)
	}
	defer publisher.Close()

	// Init Repositories
	emailRepo := repository.NewEmailRepository(dbConn)

	// Init Services
	ingestService := ingest.NewService(dbConn, emailRepo, logger)

	// Init Outbox Dispatcher
	outboxRepo := outbox.NewRepository(dbConn)
	dispatcher := outbox.NewDispatcher(outboxRepo, publisher, logger)
	go dispatcher.Start(context.Background())

	// Init Handlers
	ingestHandler := handler.NewIngestHandler(ingestService)

	// Router
	router := httpserver.NewRouter(ingestHandler, dbConn, publisher)

	// Start server
	logger.Info("Starting mail ingestion service", zap.String("port", cfg.Server.Port))
	if err := router.Run(cfg.Server.Port); err != nil {
		log.Fatalf("server start failed: %v", err)
	}
}

