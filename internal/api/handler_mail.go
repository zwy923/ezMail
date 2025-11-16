package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"mygoproject/internal/service"
)

type MailHandler struct {
	mailService *service.MailService
}

func NewMailHandler(mailService *service.MailService) *MailHandler {
	return &MailHandler{
		mailService: mailService,
	}
}

// SimulateNewEmail handles POST /simulate/new_email
func (h *MailHandler) SimulateNewEmail(c *gin.Context) {
	var req struct {
		Subject string `json:"subject"`
		Body    string `json:"body"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not authenticated"})
		return
	}

	emailID, err := h.mailService.CreateRawAndPublish(c.Request.Context(), userID.(int), req.Subject, req.Body)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create email"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"email_id": emailID,
		"status":   "queued",
	})
}
