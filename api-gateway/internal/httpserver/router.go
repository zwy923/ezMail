package httpserver

import (
	"api-gateway/internal/handler"
	"github.com/gin-gonic/gin"
)

type Router struct {
	Engine *gin.Engine
}

func NewRouter(
	authHandler *handler.AuthHandler,
	mailProxyHandler *handler.MailProxyHandler,
	emailQueryHandler *handler.EmailQueryHandler,
	jwtSecret string,
) *Router {
	r := gin.Default()

	// Public
	r.POST("/register", authHandler.Register)
	r.POST("/login", authHandler.Login)

	// Protected
	auth := r.Group("/")
	auth.Use(AuthMiddleware(jwtSecret))
	{
		auth.POST("/email/simulate", mailProxyHandler.SimulateNewEmail)
		auth.GET("/emails", emailQueryHandler.GetEmails)
	}

	return &Router{Engine: r}
}

func (r *Router) Run(port string) error {
	return r.Engine.Run(port)
}

