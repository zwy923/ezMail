package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"net/http"

	"mygoproject/pkg/db"
	"mygoproject/pkg/logger"
	"mygoproject/pkg/mq"
	"mygoproject/pkg/outbox"
	"task-runner-service/internal/config"
	"task-runner-service/internal/httpserver"
	"task-runner-service/internal/repository"
	"task-runner-service/internal/service"

	"go.uber.org/zap"
)

func main() {
	cfg := config.Load()

	log := logger.NewLogger()
	defer log.Sync()

	log.Info("Starting task-runner-service...",
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

	// MQ Publisher
	publisher, err := mq.NewPublisher(cfg.MQ.URL)
	if err != nil {
		log.Fatal("Failed to init MQ publisher", zap.Error(err))
	}
	defer publisher.Close()

	// Repositories
	taskRepo := repository.NewTaskRepository(dbConn, log)
	habitRepo := repository.NewHabitRepository(dbConn, log)

	// Orchestrator
	orchestrator := service.NewOrchestrator(dbConn, taskRepo, habitRepo, publisher, log)

	// Init Outbox Dispatcher
	outboxRepo := outbox.NewRepository(dbConn)
	dispatcher := outbox.NewDispatcher(outboxRepo, publisher, log)
	go dispatcher.Start(context.Background())

	// Task Orchestrator - runs every 1 minute
	log.Info("Starting task orchestrator (runs every 1 minute)...")
	orchestratorCtx, orchestratorCancel := context.WithCancel(context.Background())
	defer orchestratorCancel()

	go func() {
		ticker := time.NewTicker(1 * time.Minute)
		defer ticker.Stop()
		
		// Run immediately on startup
		orchestrator.CheckAndMarkOverdue(context.Background())
		orchestrator.CheckAndUnlockTasks(context.Background())

		for {
			select {
			case <-orchestratorCtx.Done():
				log.Info("Task orchestrator stopped")
				return
			case <-ticker.C:
				log.Info("Running task orchestrator...")
				ctx := context.Background()
				
				// Check overdue tasks
				if err := orchestrator.CheckAndMarkOverdue(ctx); err != nil {
					log.Error("Overdue check failed", zap.Error(err))
				}
				
				// Check unlockable tasks
				if err := orchestrator.CheckAndUnlockTasks(ctx); err != nil {
					log.Error("Unlock check failed", zap.Error(err))
				}
			}
		}
	}()

	// Habit Task Generator - runs daily at 00:00
	log.Info("Starting habit task generator (runs daily at 00:00)...")
	habitGenCtx, habitGenCancel := context.WithCancel(context.Background())
	defer habitGenCancel()

	go func() {
		// Calculate time until next midnight
		now := time.Now()
		nextMidnight := time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, now.Location())
		initialDelay := nextMidnight.Sub(now)

		log.Info("Habit generator will start at midnight",
			zap.Duration("delay", initialDelay),
		)

		// Wait until midnight
		time.Sleep(initialDelay)

		// Run immediately if it's already past midnight (for testing)
		if now.Hour() == 0 && now.Minute() < 5 {
			log.Info("Running initial habit task generation...")
			if err := orchestrator.GenerateHabitTasks(context.Background()); err != nil {
				log.Error("Initial habit task generation failed", zap.Error(err))
			}
		}

		// Then run every 24 hours
		ticker := time.NewTicker(24 * time.Hour)
		defer ticker.Stop()
		for {
			select {
			case <-habitGenCtx.Done():
				log.Info("Habit task generator stopped")
				return
			case <-ticker.C:
				log.Info("Running daily habit task generation...")
				if err := orchestrator.GenerateHabitTasks(context.Background()); err != nil {
					log.Error("Habit task generation failed", zap.Error(err))
				} else {
					log.Info("Habit task generation completed successfully")
				}
			}
		}
	}()

	// HTTP Server (for health checks)
	log.Info("Initializing HTTP server...", zap.String("port", "8084"))
	router := httpserver.NewRouter(log, dbConn)
	srv := &http.Server{
		Addr:    ":8084",
		Handler: router.Engine,
	}

	go func() {
		log.Info("HTTP server starting on :8084")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal("HTTP server failed", zap.Error(err))
		}
	}()

	log.Info("task-runner-service is fully initialized and running")

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info("Shutting down task-runner-service gracefully...")

	// Stop orchestrators
	orchestratorCancel()
	habitGenCancel()

	// Close HTTP server
	log.Info("Shutting down HTTP server...")
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Error("HTTP server shutdown error", zap.Error(err))
	} else {
		log.Info("HTTP server stopped")
	}

	// Close connections
	publisher.Close()
	dbConn.Close()

	log.Info("task-runner-service shutdown complete")
}

