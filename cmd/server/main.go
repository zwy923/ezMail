package main

import (
	"log"

	"mygoproject/config"
	"mygoproject/internal/api"
	"mygoproject/internal/db"
	"mygoproject/internal/mq"
	"mygoproject/internal/repository"
	serviceapi "mygoproject/internal/service/api"
    serviceworker "mygoproject/internal/service/worker"
	redisclient "mygoproject/internal/redis"
	"go.uber.org/zap"	
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
	defer dbConn.Close()

	// 3. Init Redis
	rdb := redisclient.NewRedisClient(cfg.Redis)
	defer rdb.Close()

	// 4. Init RabbitMQ Producer
	producer, err := mq.NewProducer(cfg.MQ.URL)
	if err != nil {
		log.Fatalf("failed to init producer: %v", err)
	}
	defer producer.Close()

	// 5. Init repositories
	userRepo := repository.NewUserRepository(dbConn)
	emailRepo := repository.NewEmailRepository(dbConn)
	metadataRepo := repository.NewMetadataRepository(dbConn)
	notiLogRepo := repository.NewNotificationLogRepository(dbConn)
	notiRepo := repository.NewNotificationRepository(dbConn)

	// 6. Init services for API
	authService := serviceapi.NewAuthService(userRepo, cfg.JWT.Secret)
	mailService := serviceapi.NewMailService(emailRepo, producer)
	classifyService := serviceworker.NewClassifyService(emailRepo, metadataRepo)
	notiLogService := serviceworker.NewNotificationLogService(notiLogRepo)
	notiService := serviceworker.NewNotificationService(notiRepo)


	// (1) Consumer for classification
	consumerClassify, err := mq.NewConsumer(cfg.MQ.URL, "email.received.classify.q", "email.received")
	if err != nil {
		log.Fatalf("failed to init classify consumer: %v", err)
	}
	consumerClassify.SetHandler(classifyService.HandleEmailReceived)
	go consumerClassify.StartConsuming()
	defer consumerClassify.Close()

	// (2) Consumer for notification-log
	consumerNotiLog, err := mq.NewConsumer(cfg.MQ.URL, "email.received.log.q", "email.received")
	if err != nil {
		log.Fatalf("failed to init noti-log consumer: %v", err)
	}
	consumerNotiLog.SetHandler(notiLogService.HandleEmailReceived)
	go consumerNotiLog.StartConsuming()
	defer consumerNotiLog.Close()

	// (3) Consumer for notification-service
	consumerNoti, err := mq.NewConsumer(cfg.MQ.URL, "email.received.notify.q", "email.received")
	if err != nil {
		log.Fatalf("failed to init noti consumer: %v", err)
	}
	consumerNoti.SetHandler(notiService.HandleEmailReceived)
	go consumerNoti.StartConsuming()
	defer consumerNoti.Close()

	// --------------------------------------------------------------------

	// 7. Init API handlers
	authHandler := api.NewAuthHandler(authService)
	mailHandler := api.NewMailHandler(mailService)
	emailQueryHandler := api.NewEmailQueryHandler(emailRepo)

	// 8. Init router for API
	router := api.NewRouter(authHandler, mailHandler, emailQueryHandler, cfg.JWT.Secret)

	// 9. Run server
	if err := router.Run(cfg.Server.Port); err != nil {
		logger.Fatal("server start failed", zap.Error(err))
	}
}
