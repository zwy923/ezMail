package httpserver

import (
	"api-gateway/internal/handler"
	"context"
	"time"

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
	jwtSecret string,
	db *pgxpool.Pool,
) *Router {
	r := gin.Default()

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
		auth.POST("/tasks/from-text", taskController.CreateTasksFromText)
		auth.POST("/tasks/plan-project", taskController.PlanProject)
		
	}

	return &Router{Engine: r}
}

func (r *Router) Run(port string) error {
	return r.Engine.Run(port)
}
