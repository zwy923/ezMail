package main

import (
	"time"

	"mygoproject/pkg/db"
	"mygoproject/pkg/logger"
	"mygoproject/pkg/mq"
	"mygoproject/pkg/redis"
	"mygoproject/pkg/util"
	"worker-service/internal/config"
	"worker-service/internal/mqhandler"
	"worker-service/internal/repository"
	"worker-service/internal/service"

	"go.uber.org/zap"
)

func main() {
	cfg := config.Load()
	logger := logger.NewLogger()
	defer logger.Sync()

	logger.Info("Starting worker service...")

	// Redis
	rdb := redis.NewRedisClient(cfg.Redis)
	defer rdb.Close()

	deduper := util.NewDeduperWithLogger(rdb, time.Hour, logger)
	retryCounter := util.NewRetryCounter(rdb, time.Hour)

	// DB
	dbConn, err := db.NewConnection(cfg.DB, logger)
	if err != nil {
		logger.Fatal("DB connection failed", zap.Error(err))
	}
	defer dbConn.Close()

	logger.Info("DB ready")

	// repositories
	emailRepo := repository.NewEmailRepository(dbConn)
	metadataRepo := repository.NewMetadataRepository(dbConn)
	taskRepo := repository.NewTaskRepository(dbConn)
	notiRepo := repository.NewNotificationRepository(dbConn)
	notiLogRepo := repository.NewNotificationLogRepository(dbConn)

	// agent client
	agentClient := service.NewAgentClient(cfg.AgentServiceURL)

	// handlers
	agentHandler := mqhandler.NewAgentDecisionHandler(
		emailRepo, metadataRepo, taskRepo, notiRepo,
		agentClient, retryCounter, deduper, logger,
	)

	notiLogHandler := mqhandler.NewEmailReceivedNotificationLogHandler(notiLogRepo, logger)
	notiHandler := mqhandler.NewEmailReceivedNotificationHandler(notiRepo, logger, deduper)

	// -------------------------
	// Agent Decision Consumer
	// -------------------------
	logger.Info("Init consumer: email.received.agent.q")
	consumerAgent, err := mq.NewConsumer(
		cfg.MQ.URL,
		"email.received.agent.q",
		"email.received.agent",
		logger,
	)
	if err != nil {
		logger.Fatal("Agent consumer init failed", zap.Error(err))
	}
	consumerAgent.SetHandler(agentHandler.Handle)

	go func() {
		if err := consumerAgent.StartConsuming(); err != nil {
			logger.Fatal("Agent consumer crashed", zap.Error(err))
		}
	}()
	defer consumerAgent.Close()

	// -------------------------
	// Notification Log Consumer
	// -------------------------
	logger.Info("Init consumer: email.received.log.q")
	consumerNotiLog, err := mq.NewConsumer(
		cfg.MQ.URL,
		"email.received.log.q",
		"email.received.log",
		logger,
	)
	if err != nil {
		logger.Fatal("Noti-log consumer init failed", zap.Error(err))
	}
	consumerNotiLog.SetHandler(notiLogHandler.HandleEmailReceived)
	go func() {
		if err := consumerNotiLog.StartConsuming(); err != nil {
			logger.Fatal("Noti-log consumer crashed", zap.Error(err))
		}
	}()
	defer consumerNotiLog.Close()

	// -------------------------
	// Notification Consumer
	// -------------------------
	logger.Info("Init consumer: email.received.notify.q")
	consumerNoti, err := mq.NewConsumer(
		cfg.MQ.URL,
		"email.received.notify.q",
		"email.received.notify",
		logger,
	)
	if err != nil {
		logger.Fatal("Noti consumer init failed", zap.Error(err))
	}
	consumerNoti.SetHandler(notiHandler.HandleEmailReceived)
	go func() {
		if err := consumerNoti.StartConsuming(); err != nil {
			logger.Fatal("Noti consumer crashed", zap.Error(err))
		}
	}()
	defer consumerNoti.Close()

	logger.Info("Worker running")
	select {}
}
