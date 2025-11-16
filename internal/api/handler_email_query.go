package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"mygoproject/internal/repository"
)

type EmailQueryHandler struct {
	emailRepo *repository.EmailRepository
}

func NewEmailQueryHandler(emailRepo *repository.EmailRepository) *EmailQueryHandler {
	return &EmailQueryHandler{
		emailRepo: emailRepo,
	}
}

// GetEmails handles GET /emails
func (h *EmailQueryHandler) GetEmails(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not authenticated"})
		return
	}

	emails, err := h.emailRepo.ListEmailsWithMetadata(c.Request.Context(), userID.(int))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch emails"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"emails": emails,
	})
}
