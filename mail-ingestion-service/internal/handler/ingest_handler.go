package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"mail-ingestion-service/internal/service/ingest"
)

type IngestHandler struct {
	ingestService *ingest.Service
}

func NewIngestHandler(ingestService *ingest.Service) *IngestHandler {
	return &IngestHandler{
		ingestService: ingestService,
	}
}

// SimulateNewEmail handles POST /email/simulate
func (h *IngestHandler) SimulateNewEmail(c *gin.Context) {
	var req struct {
		Subject string `json:"subject"`
		Body    string `json:"body"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	// Get user ID from header (set by api-gateway)
	userIDStr := c.GetHeader("X-User-ID")
	if userIDStr == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not authenticated"})
		return
	}

	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user id"})
		return
	}

	emailID, err := h.ingestService.CreateRawAndPublish(c.Request.Context(), userID, req.Subject, req.Body)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create email"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"email_id": emailID,
		"status":   "queued",
	})
}

