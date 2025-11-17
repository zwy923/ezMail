package main

import (
	"log"

	"api-gateway/internal/config"
	"api-gateway/internal/handler"
	"api-gateway/internal/httpserver"
	"api-gateway/internal/repository"
	"api-gateway/internal/service/auth"
	"mygoproject/pkg/db"
	"mygoproject/pkg/logger"

	"go.uber.org/zap"
)

func main() {
	// Load config
	cfg := config.Load()

	logger := logger.NewLogger()
	defer logger.Sync()

	// Init DB (for user auth and email query)
	dbConn, err := db.NewConnection(cfg.DB, logger)
	if err != nil {
		logger.Fatal("DB initialization failed", zap.Error(err))
	}
	defer dbConn.Close()

	// Init Repositories
	userRepo := repository.NewUserRepository(dbConn)
	emailRepo := repository.NewEmailRepository(dbConn)

	// Init Services
	authService := auth.NewService(userRepo, cfg.JWT.Secret)

	// Init Handlers
	authHandler := handler.NewAuthHandler(authService)
	mailProxyHandler := handler.NewMailProxyHandler(cfg.MailIngestionServiceURL)
	emailQueryHandler := handler.NewEmailQueryHandler(emailRepo)

	// Router
	router := httpserver.NewRouter(authHandler, mailProxyHandler, emailQueryHandler, cfg.JWT.Secret)

	// Start API server
	logger.Info("Starting API Gateway", zap.String("port", cfg.Server.Port))
	if err := router.Run(cfg.Server.Port); err != nil {
		log.Fatalf("server start failed: %v", err)
	}
}

