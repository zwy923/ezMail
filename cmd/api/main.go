package main

import (
	"log"

	"mygoproject/internal/config"
	"mygoproject/internal/db"
	"mygoproject/internal/handler"
	"mygoproject/internal/httpserver"
	"mygoproject/internal/mq"
	redisclient "mygoproject/internal/redis"
	"mygoproject/internal/repository"
	"mygoproject/internal/service/email"
	"mygoproject/internal/service/user"

	"go.uber.org/zap"
)

func main() {
	// Load config
	cfg := config.Load()

	logger, _ := zap.NewProduction()
	defer logger.Sync()

	// Init DB
	dbConn, err := db.NewConnection(cfg.DB, logger)
	if err != nil {
		logger.Fatal("DB initialization failed", zap.Error(err))
	}
	defer dbConn.Close()

	// Init Redis
	rdb := redisclient.NewRedisClient(cfg.Redis)
	defer rdb.Close()

	// Init RabbitMQ Publisher
	publisher, err := mq.NewPublisher(cfg.MQ.URL)
	if err != nil {
		log.Fatalf("failed to init publisher: %v", err)
	}
	defer publisher.Close()

	// Init Repositories
	userRepo := repository.NewUserRepository(dbConn)
	emailRepo := repository.NewEmailRepository(dbConn)

	// Init Services
	authService := user.NewService(userRepo, cfg.JWT.Secret)
	mailService := email.NewService(emailRepo, publisher)

	// Init Handlers
	authHandler := handler.NewAuthHandler(authService)
	mailHandler := handler.NewMailHandler(mailService)
	emailQueryHandler := handler.NewEmailQueryHandler(emailRepo)

	// Router
	router := httpserver.NewRouter(authHandler, mailHandler, emailQueryHandler, cfg.JWT.Secret)

	// Start API server
	if err := router.Run(cfg.Server.Port); err != nil {
		logger.Fatal("server start failed", zap.Error(err))
	}
}
