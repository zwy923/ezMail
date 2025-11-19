package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"mygoproject/pkg/db"
	"mygoproject/pkg/logger"
	"mygoproject/pkg/mq"
	"task-service/internal/config"
	"task-service/internal/handler"
	"task-service/internal/httpserver"
	"task-service/internal/mqhandler"
	"task-service/internal/repository"

	"go.uber.org/zap"
)

func main() {
	cfg := config.Load()

	log := logger.NewLogger()
	defer log.Sync()

	log.Info("Starting task-service...",
		zap.String("db_host", cfg.DB.Host),
		zap.Int("db_port", cfg.DB.Port),
		zap.String("mq_url", cfg.MQ.URL),
	)

	// DB
	log.Info("Initializing database connection...")
	dbConn, err := db.NewConnection(cfg.DB, log)
	if err != nil {
		log.Fatal("Failed to init DB", zap.Error(err))
	}
	defer dbConn.Close()
	log.Info("Database connection established successfully")

	taskRepo := repository.NewTaskRepository(dbConn, log)
	mqHandler := mqhandler.NewTaskCreatedHandler(taskRepo, log)

	// MQ Consumer
	log.Info("Initializing MQ consumer...",
		zap.String("queue", "task.created.q"),
		zap.String("routing_key", "task.created"),
	)
	consumer, err := mq.NewConsumer(cfg.MQ.URL, "task.created.q", "task.created", log)
	if err != nil {
		log.Fatal("Failed to init consumer", zap.Error(err))
	}
	defer consumer.Close()

	consumer.SetHandler(mqHandler.Handle)

	go func() {
		log.Info("Starting MQ consumer...")
		if err := consumer.StartConsuming(); err != nil {
			log.Fatal("Task consumer failed", zap.Error(err))
		}
	}()
	log.Info("MQ consumer started successfully")

	// HTTP Server
	log.Info("Initializing HTTP server...", zap.String("port", "8082"))
	taskHandler := handler.NewTaskHandler(taskRepo, log)
	router := httpserver.NewRouter(taskHandler, log, dbConn, consumer)

	srv := &http.Server{
		Addr:    ":8082",
		Handler: router,
	}

	go func() {
		log.Info("HTTP server starting on :8082")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal("HTTP server failed", zap.Error(err))
		}
	}()

	// Task Expiration Checker
	log.Info("Starting task expiration checker (runs every 1 minute)...")
	expirationCtx, expirationCancel := context.WithCancel(context.Background())
	defer expirationCancel()

	go func() {
		ticker := time.NewTicker(1 * time.Minute)
		defer ticker.Stop()
		for {
			select {
			case <-expirationCtx.Done():
				log.Info("Task expiration checker stopped")
				return
			case <-ticker.C:
				log.Info("Running task expiration check...")
				if err := taskRepo.MarkExpired(context.Background()); err != nil {
					log.Error("Task expiration check failed", zap.Error(err))
				} else {
					log.Debug("Task expiration check completed successfully")
				}
			}
		}
	}()

	log.Info("task-service is fully initialized and running",
		zap.String("http_port", "8082"),
		zap.String("mq_queue", "task.created.q"),
	)

	// 优雅退出处理
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info("Shutting down task-service gracefully...")

	// 停止任务过期检查器
	expirationCancel()

	// 停止 MQ 消费者
	log.Info("Stopping MQ consumer...")
	consumer.Stop()

	// 关闭 HTTP 服务器
	log.Info("Shutting down HTTP server...")
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Error("HTTP server shutdown error", zap.Error(err))
	} else {
		log.Info("HTTP server stopped")
	}

	// 关闭数据库连接
	log.Info("Closing database connection...")
	dbConn.Close()

	log.Info("task-service shutdown complete")
}
