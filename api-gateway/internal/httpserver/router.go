package httpserver

import (
	"api-gateway/internal/handler"
	"context"
	"fmt"
	"time"

	"mygoproject/pkg/metrics"
	"mygoproject/pkg/otel"
	"mygoproject/pkg/trace"

	"mygoproject/pkg/rbac"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Router struct {
	Engine *gin.Engine
}

func NewRouter(
	authHandler *handler.AuthHandler,
	mailProxyHandler *handler.MailProxyHandler,
	emailQueryHandler *handler.EmailQueryHandler,
	taskController *handler.TaskController,
	adminHandler *handler.AdminHandler,
	jwtSecret string,
	db *pgxpool.Pool,
) *Router {
	r := gin.Default()

	// OpenTelemetry 追踪中间件（必须在最前面）
	r.Use(otel.GinMiddleware())

	// Trace ID 中间件：生成或提取 trace_id（向后兼容）
	r.Use(func(c *gin.Context) {
		// 从请求头中提取 trace_id，如果没有则生成新的
		traceID := trace.FromHeader(c.GetHeader(trace.HeaderName()))
		if traceID == "" {
			traceID = trace.GenerateTraceID()
		}

		// 将 trace_id 添加到 context
		ctx := trace.WithContext(c.Request.Context(), traceID)
		c.Request = c.Request.WithContext(ctx)

		// 在响应头中返回 trace_id
		c.Header(trace.HeaderName(), traceID)

		c.Next()
	})

	// HTTP 请求指标和日志中间件
	r.Use(func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		method := c.Request.Method

		c.Next()

		latency := time.Since(start)
		status := c.Writer.Status()

		// 记录 Prometheus 指标
		metrics.RecordHTTPRequestDuration(method, path, fmt.Sprintf("%d", status), latency)

		// 记录日志（包含 trace_id）
		traceID := trace.FromContext(c.Request.Context())
		// 这里可以添加日志记录，但需要 logger，暂时跳过
		_ = traceID
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

		c.JSON(200, gin.H{"status": "ready"})
	})

	// Public
	r.POST("/register", authHandler.Register)
	r.POST("/login", authHandler.Login)

	// Protected
	auth := r.Group("/")
	auth.Use(AuthMiddleware(jwtSecret))
	{
		auth.POST("/email/simulate", mailProxyHandler.SimulateNewEmail)
		auth.GET("/emails", emailQueryHandler.GetEmails)
		// Task endpoints (统一由 TaskController 处理)
		auth.GET("/tasks", taskController.GetTasks)
		auth.POST("/tasks/:id/complete", taskController.CompleteTask)

		// 敏感操作：需要 RBAC 验证
		auth.POST("/tasks/from-text",
			RequirePermission(rbac.PermissionBulkCreateTask),
			taskController.CreateTasksFromText)
		auth.POST("/tasks/plan-project",
			RequirePermission(rbac.PermissionCreateProject),
			taskController.PlanProject)

		// Admin endpoints (需要 admin 权限，暂时使用 user 权限)
		auth.POST("/admin/outbox/replay", adminHandler.ReplayOutboxEvent)
		auth.POST("/admin/outbox/replay-failed", adminHandler.ReplayFailedEvents)
	}

	return &Router{Engine: r}
}

func (r *Router) Run(port string) error {
	return r.Engine.Run(port)
}
