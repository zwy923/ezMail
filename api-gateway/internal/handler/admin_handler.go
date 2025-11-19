package handler

import (
	"net/http"
	"strconv"

	"mygoproject/pkg/outbox"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type AdminHandler struct {
	replayService *outbox.ReplayService
	logger        *zap.Logger
}

func NewAdminHandler(replayService *outbox.ReplayService, logger *zap.Logger) *AdminHandler {
	return &AdminHandler{
		replayService: replayService,
		logger:        logger,
	}
}

// ReplayOutboxEvent 重放指定的 Outbox 事件
// POST /admin/outbox/replay?id=xxx
func (h *AdminHandler) ReplayOutboxEvent(c *gin.Context) {
	idStr := c.Query("id")
	if idStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing id parameter"})
		return
	}

	eventID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id parameter"})
		return
	}

	if err := h.replayService.ReplayEvent(c.Request.Context(), eventID); err != nil {
		h.logger.Error("Failed to replay event",
			zap.Int64("event_id", eventID),
			zap.Error(err),
		)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "failed to replay event",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "replayed",
		"event_id": eventID,
	})
}

// ReplayFailedEvents 重放所有失败的事件
// POST /admin/outbox/replay-failed?limit=100
func (h *AdminHandler) ReplayFailedEvents(c *gin.Context) {
	limitStr := c.DefaultQuery("limit", "100")
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 {
		limit = 100
	}

	successCount, err := h.replayService.ReplayFailedEvents(c.Request.Context(), limit)
	if err != nil {
		h.logger.Error("Failed to replay failed events", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "failed to replay failed events",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":        "completed",
		"success_count": successCount,
		"limit":         limit,
	})
}

