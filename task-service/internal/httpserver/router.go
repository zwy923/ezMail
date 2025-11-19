package httpserver

import (
	"context"
	"task-service/internal/handler"
	"time"

	"mygoproject/pkg/mq"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

func NewRouter(taskHandler *handler.TaskHandler, logger *zap.Logger, db *pgxpool.Pool, consumer *mq.Consumer) *gin.Engine {
	r := gin.Default()

	// 添加请求日志中间件
	r.Use(func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery

		c.Next()

		latency := time.Since(start)
		logger.Info("HTTP Request",
			zap.String("method", c.Request.Method),
			zap.String("path", path),
			zap.String("query", query),
			zap.Int("status", c.Writer.Status()),
			zap.Duration("latency", latency),
			zap.String("client_ip", c.ClientIP()),
			zap.String("user_agent", c.Request.UserAgent()),
		)
	})

	// Health endpoints (放在最前面)
	r.GET("/healthz", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})
	r.HEAD("/healthz", func(c *gin.Context) {
		c.Status(200)
	})
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})
	r.HEAD("/health", func(c *gin.Context) {
		c.Status(200)
	})

	r.GET("/readyz", func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c, 1*time.Second)
		defer cancel()

		if err := db.Ping(ctx); err != nil {
			c.JSON(500, gin.H{"status": "db_not_ready", "error": err.Error()})
			return
		}

		if consumer != nil && !consumer.IsConnected() {
			c.JSON(500, gin.H{"status": "mq_not_ready"})
			return
		}

		c.JSON(200, gin.H{"status": "ready"})
	})

	r.GET("/tasks", taskHandler.ListTasks)
	r.POST("/tasks/:id/complete", taskHandler.CompleteTask)
	return r
}
