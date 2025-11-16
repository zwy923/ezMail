package main

import (
	"log"

	"go.uber.org/zap"
	"mygoproject/config"
	"mygoproject/internal/api"
	"mygoproject/internal/db"
	"mygoproject/internal/mq"
	"mygoproject/internal/repository"
	"mygoproject/internal/service"
)

func main() {
	// 1. Load config
	cfg := config.Load()

	logger, _ := zap.NewProduction()
	defer logger.Sync()

	// 2. Init DB
	dbConn, err := db.NewConnection(cfg.DB, logger)
	if err != nil {
		logger.Fatal("DB initialization failed", zap.Error(err))
	}

	// 3. Init RabbitMQ Producer
	producer, err := mq.NewProducer(cfg.MQ.URL)
	if err != nil {
		log.Fatalf("failed to init producer: %v", err)
	}
	defer producer.Close()

	// 4. Init RabbitMQ Consumer
	consumer, err := mq.NewConsumer(cfg.MQ.URL)
	if err != nil {
		log.Fatalf("failed to init consumer: %v", err)
	}
	defer consumer.Close()

	// 5. Init repositories
	userRepo := repository.NewUserRepository(dbConn)
	emailRepo := repository.NewEmailRepository(dbConn)
	metadataRepo := repository.NewMetadataRepository(dbConn)

	// 6. Init services
	authService := service.NewAuthService(userRepo, cfg.JWT.Secret)
	mailService := service.NewMailService(emailRepo, producer)
	classifyService := service.NewClassifyService(emailRepo, metadataRepo)

	// 7. Inject classify handler into consumer
	mqrouter := mq.NewRouter()
	mqrouter.Register("email.received", classifyService.HandleEmailReceived)
	consumer.SetHandler(mqrouter.Handle)

	// Start consumer goroutine
	go func() {
		if err := consumer.StartConsuming(); err != nil {
			logger.Fatal("consumer start failed", zap.Error(err))
		}
	}()

	// 8. Init handlers
	authHandler := api.NewAuthHandler(authService)
	mailHandler := api.NewMailHandler(mailService)
	emailQueryHandler := api.NewEmailQueryHandler(emailRepo)

	// 9. Init router
	router := api.NewRouter(authHandler, mailHandler, emailQueryHandler, cfg.JWT.Secret)

	// 10. Run server
	if err := router.Run(cfg.Server.Port); err != nil {
		logger.Fatal("server start failed", zap.Error(err))
	}
}
