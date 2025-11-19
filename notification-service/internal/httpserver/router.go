package httpserver

import (
	"context"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Router struct {
	Engine *gin.Engine
}

func NewRouter(logger interface{}, db *pgxpool.Pool) *Router {
	r := gin.Default()

	// Health endpoints
	r.GET("/healthz", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})
	r.HEAD("/healthz", func(c *gin.Context) {
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

	return &Router{Engine: r}
}

func (r *Router) Run(port string) error {
	return r.Engine.Run(port)
}
