package httpserver

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"mygoproject/pkg/util"
)

func AuthMiddleware(jwtSecret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		token := util.ExtractToken(c.Request)
		if token == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "missing token"})
			c.Abort()
			return
		}

		userID, err := util.ParseJWT(token, jwtSecret)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			c.Abort()
			return
		}

		// store user_id in context so handlers can use it
		c.Set("user_id", userID)

		c.Next()
	}
}

