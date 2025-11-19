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
	habitRepo := repository.NewHabitRepository(dbConn, log)
	projectRepo := repository.NewProjectRepository(dbConn, log)
	milestoneRepo := repository.NewMilestoneRepository(dbConn, log)

	taskCreatedHandler := mqhandler.NewTaskCreatedHandler(taskRepo, log)
	taskBulkCreatedHandler := mqhandler.NewTaskBulkCreatedHandler(taskRepo, log)
	habitCreatedHandler := mqhandler.NewHabitCreatedHandler(habitRepo, log)
	projectCreatedHandler := mqhandler.NewProjectCreatedHandler(projectRepo, milestoneRepo, taskRepo, log)
	taskOverdueHandler := mqhandler.NewTaskOverdueHandler(taskRepo, log)
	taskUnlockedHandler := mqhandler.NewTaskUnlockedHandler(taskRepo, log)
	habitTaskGeneratedHandler := mqhandler.NewHabitTaskGeneratedHandler(taskRepo, log)

	// MQ Consumer for task.created
	log.Info("Initializing MQ consumer for task.created...",
		zap.String("queue", "task.created.q"),
		zap.String("routing_key", "task.created"),
	)
	consumer, err := mq.NewConsumer(cfg.MQ.URL, "task.created.q", "task.created", log)
	if err != nil {
		log.Fatal("Failed to init consumer", zap.Error(err))
	}
	defer consumer.Close()

	consumer.SetHandler(taskCreatedHandler.Handle)

	go func() {
		log.Info("Starting task.created consumer...")
		if err := consumer.StartConsuming(); err != nil {
			log.Fatal("Task consumer failed", zap.Error(err))
		}
	}()
	log.Info("task.created consumer started successfully")

	// MQ Consumer for task.bulk_created
	log.Info("Initializing MQ consumer for task.bulk_created...",
		zap.String("queue", "task.bulk_created.q"),
		zap.String("routing_key", "task.bulk_created"),
	)
	bulkConsumer, err := mq.NewConsumer(cfg.MQ.URL, "task.bulk_created.q", "task.bulk_created", log)
	if err != nil {
		log.Fatal("Failed to init bulk consumer", zap.Error(err))
	}
	defer bulkConsumer.Close()

	bulkConsumer.SetHandler(taskBulkCreatedHandler.Handle)

	go func() {
		log.Info("Starting task.bulk_created consumer...")
		if err := bulkConsumer.StartConsuming(); err != nil {
			log.Fatal("Bulk task consumer failed", zap.Error(err))
		}
	}()
	log.Info("task.bulk_created consumer started successfully")

	// MQ Consumer for habit.created
	log.Info("Initializing MQ consumer for habit.created...",
		zap.String("queue", "habit.created.q"),
		zap.String("routing_key", "habit.created"),
	)
	habitConsumer, err := mq.NewConsumer(cfg.MQ.URL, "habit.created.q", "habit.created", log)
	if err != nil {
		log.Fatal("Failed to init habit consumer", zap.Error(err))
	}
	defer habitConsumer.Close()

	habitConsumer.SetHandler(habitCreatedHandler.Handle)

	go func() {
		log.Info("Starting habit.created consumer...")
		if err := habitConsumer.StartConsuming(); err != nil {
			log.Fatal("Habit consumer failed", zap.Error(err))
		}
	}()
	log.Info("habit.created consumer started successfully")

	// MQ Consumer for project.created
	log.Info("Initializing MQ consumer for project.created...",
		zap.String("queue", "project.created.q"),
		zap.String("routing_key", "project.created"),
	)
	projectConsumer, err := mq.NewConsumer(cfg.MQ.URL, "project.created.q", "project.created", log)
	if err != nil {
		log.Fatal("Failed to init project consumer", zap.Error(err))
	}
	defer projectConsumer.Close()

	projectConsumer.SetHandler(projectCreatedHandler.Handle)

	go func() {
		log.Info("Starting project.created consumer...")
		if err := projectConsumer.StartConsuming(); err != nil {
			log.Fatal("Project consumer failed", zap.Error(err))
		}
	}()
	log.Info("project.created consumer started successfully")

	// MQ Consumer for task.overdue
	log.Info("Initializing MQ consumer for task.overdue...",
		zap.String("queue", "task.overdue.q"),
		zap.String("routing_key", "task.overdue"),
	)
	overdueConsumer, err := mq.NewConsumer(cfg.MQ.URL, "task.overdue.q", "task.overdue", log)
	if err != nil {
		log.Fatal("Failed to init overdue consumer", zap.Error(err))
	}
	defer overdueConsumer.Close()
	overdueConsumer.SetHandler(taskOverdueHandler.Handle)
	go func() {
		log.Info("Starting task.overdue consumer...")
		if err := overdueConsumer.StartConsuming(); err != nil {
			log.Fatal("Overdue consumer failed", zap.Error(err))
		}
	}()
	log.Info("task.overdue consumer started successfully")

	// MQ Consumer for task.unlocked
	log.Info("Initializing MQ consumer for task.unlocked...",
		zap.String("queue", "task.unlocked.q"),
		zap.String("routing_key", "task.unlocked"),
	)
	unlockedConsumer, err := mq.NewConsumer(cfg.MQ.URL, "task.unlocked.q", "task.unlocked", log)
	if err != nil {
		log.Fatal("Failed to init unlocked consumer", zap.Error(err))
	}
	defer unlockedConsumer.Close()
	unlockedConsumer.SetHandler(taskUnlockedHandler.Handle)
	go func() {
		log.Info("Starting task.unlocked consumer...")
		if err := unlockedConsumer.StartConsuming(); err != nil {
			log.Fatal("Unlocked consumer failed", zap.Error(err))
		}
	}()
	log.Info("task.unlocked consumer started successfully")

	// MQ Consumer for habit.task.generated
	log.Info("Initializing MQ consumer for habit.task.generated...",
		zap.String("queue", "habit.task.generated.q"),
		zap.String("routing_key", "habit.task.generated"),
	)
	habitTaskGenConsumer, err := mq.NewConsumer(cfg.MQ.URL, "habit.task.generated.q", "habit.task.generated", log)
	if err != nil {
		log.Fatal("Failed to init habit task gen consumer", zap.Error(err))
	}
	defer habitTaskGenConsumer.Close()
	habitTaskGenConsumer.SetHandler(habitTaskGeneratedHandler.Handle)
	go func() {
		log.Info("Starting habit.task.generated consumer...")
		if err := habitTaskGenConsumer.StartConsuming(); err != nil {
			log.Fatal("Habit task gen consumer failed", zap.Error(err))
		}
	}()
	log.Info("habit.task.generated consumer started successfully")

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

	// Note: Task orchestration (expiration check, habit generation, dependency unlock)
	// has been moved to task-runner-service. This service now only handles
	// task CRUD operations and event consumption.

	log.Info("task-service is fully initialized and running",
		zap.String("http_port", "8082"),
		zap.String("mq_queue_created", "task.created.q"),
		zap.String("mq_queue_bulk", "task.bulk_created.q"),
	)

	// 优雅退出处理
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info("Shutting down task-service gracefully...")

	// 停止 MQ 消费者
	log.Info("Stopping MQ consumers...")
	consumer.Stop()
	bulkConsumer.Stop()
	habitConsumer.Stop()
	projectConsumer.Stop()
	overdueConsumer.Stop()
	unlockedConsumer.Stop()
	habitTaskGenConsumer.Stop()

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
