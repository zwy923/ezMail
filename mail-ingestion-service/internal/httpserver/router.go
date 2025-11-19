package httpserver

import (
	"context"
	"fmt"
	"mail-ingestion-service/internal/handler"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"mygoproject/pkg/mq"
	"mygoproject/pkg/metrics"
	"mygoproject/pkg/otel"
	"mygoproject/pkg/trace"
)

type Router struct {
	Engine *gin.Engine
}

func NewRouter(ingestHandler *handler.IngestHandler, db *pgxpool.Pool, publisher *mq.Publisher) *Router {
	r := gin.Default()

	// OpenTelemetry 追踪中间件（必须在最前面）
	r.Use(otel.GinMiddleware())

	// Trace ID 中间件：提取或生成 trace_id（向后兼容）
	r.Use(func(c *gin.Context) {
		traceID := trace.FromHeader(c.GetHeader(trace.HeaderName()))
		if traceID == "" {
			traceID = trace.GenerateTraceID()
		}
		ctx := trace.WithContext(c.Request.Context(), traceID)
		c.Request = c.Request.WithContext(ctx)
		c.Header(trace.HeaderName(), traceID)
		c.Next()
	})

	// HTTP 请求指标中间件
	r.Use(func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		method := c.Request.Method
		c.Next()
		latency := time.Since(start)
		status := c.Writer.Status()
		metrics.RecordHTTPRequestDuration(method, path, fmt.Sprintf("%d", status), latency)
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

		if publisher != nil && !publisher.IsConnected() {
			c.JSON(500, gin.H{"status": "mq_not_ready"})
			return
		}

		c.JSON(200, gin.H{"status": "ready"})
	})

	// Email ingestion endpoint
	r.POST("/email/simulate", ingestHandler.SimulateNewEmail)

	return &Router{Engine: r}
}

func (r *Router) Run(port string) error {
	return r.Engine.Run(port)
}

