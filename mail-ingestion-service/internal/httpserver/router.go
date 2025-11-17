package httpserver

import (
	"mail-ingestion-service/internal/handler"
	"github.com/gin-gonic/gin"
)

type Router struct {
	Engine *gin.Engine
}

func NewRouter(ingestHandler *handler.IngestHandler) *Router {
	r := gin.Default()

	// Email ingestion endpoint
	r.POST("/email/simulate", ingestHandler.SimulateNewEmail)

	return &Router{Engine: r}
}

func (r *Router) Run(port string) error {
	return r.Engine.Run(port)
}

