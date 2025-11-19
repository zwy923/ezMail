package main

import (
	"os"
	"os/signal"
	"syscall"
	"time"

	"context"
	"mygoproject/pkg/db"
	"mygoproject/pkg/logger"
	"mygoproject/pkg/mq"
	"mygoproject/pkg/outbox"
	"mygoproject/pkg/redis"
	"mygoproject/pkg/util"
	"email-processor-service/internal/config"
	"email-processor-service/internal/mqhandler"
	"email-processor-service/internal/repository"
	"email-processor-service/internal/service"

	"go.uber.org/zap"
)

func main() {
	cfg := config.Load()
	logger := logger.NewLogger()
	defer logger.Sync()

	logger.Info("Starting email-processor-service...")

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
	notiLogRepo := repository.NewNotificationLogRepository(dbConn)

	// agent client
	agentClient := service.NewAgentClient(cfg.AgentServiceURL)

	// task publisher (also used for notification events)
	taskPublisher, err := mq.NewPublisher(cfg.MQ.URL)
	if err != nil {
		logger.Fatal("failed to init task publisher", zap.Error(err))
	}
	defer taskPublisher.Close()

	// handlers
	agentHandler := mqhandler.NewAgentDecisionHandler(
		dbConn,
		emailRepo,
		metadataRepo,
		agentClient,
		retryCounter,
		deduper,
		taskPublisher,
		logger,
	)

	// Init Outbox Dispatcher
	outboxRepo := outbox.NewRepository(dbConn)
	dispatcher := outbox.NewDispatcher(outboxRepo, taskPublisher, logger)
	go dispatcher.Start(context.Background())

	notiLogHandler := mqhandler.NewEmailReceivedNotificationLogHandler(notiLogRepo, logger)
	// NotificationHandler now publishes notification.created events (handled by notification-service)
	notiHandler := mqhandler.NewEmailReceivedNotificationHandler(taskPublisher, logger, deduper)

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

	// 优雅退出处理
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down email-processor-service gracefully...")

	// 停止所有消费者
	logger.Info("Stopping MQ consumers...")
	consumerAgent.Stop()
	consumerNotiLog.Stop()
	consumerNoti.Stop()

	// 关闭数据库连接
	logger.Info("Closing database connection...")
	dbConn.Close()

	// 关闭 Redis 连接
	logger.Info("Closing Redis connection...")
	rdb.Close()

	// 关闭任务发布者
	logger.Info("Closing task publisher...")
	taskPublisher.Close()

	logger.Info("email-processor-service shutdown complete")
}
