package handler

import (
	"bytes"
	"fmt"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
)

type MailProxyHandler struct {
	ingestionServiceURL string
	httpClient          *http.Client
}

func NewMailProxyHandler(ingestionServiceURL string) *MailProxyHandler {
	return &MailProxyHandler{
		ingestionServiceURL: ingestionServiceURL,
		httpClient:          &http.Client{},
	}
}

// SimulateNewEmail proxies POST /email/simulate to mail-ingestion-service
func (h *MailProxyHandler) SimulateNewEmail(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not authenticated"})
		return
	}

	// Read request body
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	// Forward to mail-ingestion-service
	url := h.ingestionServiceURL + "/email/simulate"
	req, err := http.NewRequestWithContext(c.Request.Context(), "POST", url, bytes.NewReader(body))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create request"})
		return
	}

	// Copy headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-ID", fmt.Sprintf("%d", userID.(int)))

	// Forward request
	resp, err := h.httpClient.Do(req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to forward request"})
		return
	}
	defer resp.Body.Close()

	// Read response
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to read response"})
		return
	}

	// Forward response
	c.Data(resp.StatusCode, "application/json", respBody)
}

