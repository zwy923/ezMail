package main

import (
	"time"

	"worker-service/internal/config"
	"worker-service/internal/mqhandler"
	"worker-service/internal/repository"
	"worker-service/internal/service"
	"mygoproject/pkg/db"
	"mygoproject/pkg/logger"
	"mygoproject/pkg/mq"
	"mygoproject/pkg/redis"
	"mygoproject/pkg/util"

	"go.uber.org/zap"
)

func main() {
	// Load config
	cfg := config.Load()

	logger := logger.NewLogger()
	defer logger.Sync()

	logger.Info("Starting worker service...")

	// Init Redis
	rdb := redis.NewRedisClient(cfg.Redis)
	defer rdb.Close()

	deduper := util.NewDeduperWithLogger(rdb, time.Hour, logger)
	retryCounter := util.NewRetryCounter(rdb, time.Hour)

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

	// Init Agent Client
	agentClient := service.NewAgentClient(cfg.AgentServiceURL)

	// Init Handlers
	classifyHandler := mqhandler.NewEmailReceivedClassifyHandler(emailRepo, metadataRepo, agentClient, retryCounter, deduper, logger)
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

