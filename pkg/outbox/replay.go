package outbox

import (
	"context"
	"encoding/json"
	"fmt"

	"mygoproject/pkg/mq"
	"mygoproject/pkg/trace"
)

// ReplayService 提供重放 Outbox 事件的服务
type ReplayService struct {
	repo      *Repository
	publisher *mq.Publisher
}

// NewReplayService 创建新的 ReplayService
func NewReplayService(repo *Repository, publisher *mq.Publisher) *ReplayService {
	return &ReplayService{
		repo:      repo,
		publisher: publisher,
	}
}

// ReplayEvent 重放指定的事件
func (s *ReplayService) ReplayEvent(ctx context.Context, eventID int64) error {
	// 获取事件
	event, err := s.repo.GetEventByID(ctx, eventID)
	if err != nil {
		return fmt.Errorf("failed to get event: %w", err)
	}

	// 解析 payload
	var payload interface{}
	if err := json.Unmarshal(event.Payload, &payload); err != nil {
		return fmt.Errorf("failed to unmarshal payload: %w", err)
	}

	// 发布到 MQ
	ctx = s.extractTraceIDFromPayload(ctx, event.Payload)
	if err := s.publisher.PublishWithContext(ctx, event.RoutingKey, payload); err != nil {
		// 发布失败，标记为失败
		if markErr := s.repo.MarkAsFailed(ctx, eventID, 5); markErr != nil {
			return fmt.Errorf("failed to publish and mark as failed: %w (mark error: %v)", err, markErr)
		}
		return fmt.Errorf("failed to publish: %w", err)
	}

	// 发布成功，标记为已发送
	if err := s.repo.MarkAsSent(ctx, eventID); err != nil {
		return fmt.Errorf("failed to mark as sent: %w", err)
	}

	return nil
}

// ReplayFailedEvents 重放所有失败的事件
func (s *ReplayService) ReplayFailedEvents(ctx context.Context, limit int) (int, error) {
	events, err := s.repo.GetFailedEvents(ctx, limit)
	if err != nil {
		return 0, fmt.Errorf("failed to get failed events: %w", err)
	}

	successCount := 0
	for _, event := range events {
		if err := s.ReplayEvent(ctx, event.ID); err != nil {
			// 记录错误但继续处理其他事件
			continue
		}
		successCount++
	}

	return successCount, nil
}

// extractTraceIDFromPayload 从 payload 中提取 trace_id
func (s *ReplayService) extractTraceIDFromPayload(ctx context.Context, payload json.RawMessage) context.Context {
	var payloadMap map[string]interface{}
	if err := json.Unmarshal(payload, &payloadMap); err != nil {
		return ctx
	}

	if traceID, ok := payloadMap["trace_id"].(string); ok && traceID != "" {
		ctx = trace.WithContext(ctx, traceID)
	}

	return ctx
}

