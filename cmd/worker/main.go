package main

import (
	"mygoproject/internal/config"
	"mygoproject/internal/db"
	"mygoproject/internal/mq"
	"mygoproject/internal/mqhandler"
	"mygoproject/internal/repository"
	 redisclient "mygoproject/internal/redis"
	"mygoproject/internal/util"
	"go.uber.org/zap"
	"time"
)

func main() {
	// Load config
	cfg := config.Load()

	// Init Redis
	rdb := redisclient.NewRedisClient(cfg.Redis)
	defer rdb.Close()

	deduper := util.NewDeduper(rdb, time.Hour)

	logger, _ := zap.NewProduction()
	defer logger.Sync()

	logger.Info("Starting worker service...")

	// Init DB
	dbConn, err := db.NewConnection(cfg.DB, logger)
	if err != nil {
		logger.Fatal("DB initialization failed", zap.Error(err))
	}
	defer dbConn.Close()

	logger.Info("Database connection established")

	// Init Repositories
	emailRepo := repository.NewEmailRepository(dbConn)
	metadataRepo := repository.NewMetadataRepository(dbConn)
	notiLogRepo := repository.NewNotificationLogRepository(dbConn)
	notiRepo := repository.NewNotificationRepository(dbConn)

	// Init Handlers
	classifyHandler := mqhandler.NewEmailReceivedClassifyHandler(emailRepo, metadataRepo, logger)
	notiLogHandler := mqhandler.NewEmailReceivedNotificationLogHandler(notiLogRepo, logger)
	notiHandler := mqhandler.NewEmailReceivedNotificationHandler(notiRepo, logger, deduper)
	

	// (1) Consumer for classification
	logger.Info("Initializing classify consumer", zap.String("queue", "email.received.classify.q"))
	consumerClassify, err := mq.NewConsumer(cfg.MQ.URL, "email.received.classify.q", "email.received", logger)
	if err != nil {
		logger.Fatal("failed to init classify consumer", zap.Error(err))
	}
	consumerClassify.SetHandler(classifyHandler.HandleEmailReceived)
	go func() {
		logger.Info("Starting classify consumer")
		if err := consumerClassify.StartConsuming(); err != nil {
			logger.Fatal("classify consumer failed", zap.Error(err))
		}
	}()
	defer consumerClassify.Close()

	// (2) Consumer for notification-log
	logger.Info("Initializing notification-log consumer", zap.String("queue", "email.received.log.q"))
	consumerNotiLog, err := mq.NewConsumer(cfg.MQ.URL, "email.received.log.q", "email.received", logger)
	if err != nil {
		logger.Fatal("failed to init noti-log consumer", zap.Error(err))
	}
	consumerNotiLog.SetHandler(notiLogHandler.HandleEmailReceived)
	go func() {
		logger.Info("Starting notification-log consumer")
		if err := consumerNotiLog.StartConsuming(); err != nil {
			logger.Fatal("notification-log consumer failed", zap.Error(err))
		}
	}()
	defer consumerNotiLog.Close()

	// (3) Consumer for notification-service
	logger.Info("Initializing notification consumer", zap.String("queue", "email.received.notify.q"))
	consumerNoti, err := mq.NewConsumer(cfg.MQ.URL, "email.received.notify.q", "email.received", logger)
	if err != nil {
		logger.Fatal("failed to init noti consumer", zap.Error(err))
	}
	consumerNoti.SetHandler(notiHandler.HandleEmailReceived)
	go func() {
		logger.Info("Starting notification consumer")
		if err := consumerNoti.StartConsuming(); err != nil {
			logger.Fatal("notification consumer failed", zap.Error(err))
		}
	}()
	defer consumerNoti.Close()

	logger.Info("All consumers started, worker is ready to process messages")

	// Keep worker running
	select {}
}
