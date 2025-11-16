package api

import (
	"github.com/gin-gonic/gin"
)

type Router struct {
	Engine *gin.Engine
}

func NewRouter(
	authHandler *AuthHandler,
	mailHandler *MailHandler,
	emailQueryHandler *EmailQueryHandler,
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
		auth.POST("/simulate/new_email", mailHandler.SimulateNewEmail)
		auth.GET("/emails", emailQueryHandler.GetEmails)
	}

	return &Router{Engine: r}
}

func (r *Router) Run(port string) error {
	return r.Engine.Run(port)
}
