package httpserver

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"mygoproject/pkg/rbac"
)

// RequirePermission 中间件：要求用户具有指定权限
func RequirePermission(permission string) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, exists := c.Get("user_id")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "user not authenticated"})
			c.Abort()
			return
		}

		uid, ok := userID.(int)
		if !ok {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "invalid user_id"})
			c.Abort()
			return
		}

		if err := rbac.CheckPermission(uid, permission); err != nil {
			c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
			c.Abort()
			return
		}

		c.Next()
	}
}

