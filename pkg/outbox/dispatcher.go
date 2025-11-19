package outbox

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"mygoproject/pkg/mq"
	"mygoproject/pkg/trace"

	"go.uber.org/zap"
)

// Dispatcher 负责从 outbox 中读取事件并发布到 MQ
type Dispatcher struct {
	repo      *Repository
	publisher *mq.Publisher
	logger    *zap.Logger
	maxRetries int
	interval   time.Duration
	batchSize  int
}

// NewDispatcher 创建新的 Dispatcher
func NewDispatcher(
	repo *Repository,
	publisher *mq.Publisher,
	logger *zap.Logger,
) *Dispatcher {
	return &Dispatcher{
		repo:       repo,
		publisher:  publisher,
		logger:     logger,
		maxRetries: 5,              // 默认最大重试5次
		interval:   1 * time.Second, // 默认每秒扫描一次
		batchSize:  100,             // 默认每次处理100个事件
	}
}

// WithMaxRetries 设置最大重试次数
func (d *Dispatcher) WithMaxRetries(maxRetries int) *Dispatcher {
	d.maxRetries = maxRetries
	return d
}

// WithInterval 设置扫描间隔
func (d *Dispatcher) WithInterval(interval time.Duration) *Dispatcher {
	d.interval = interval
	return d
}

// WithBatchSize 设置批次大小
func (d *Dispatcher) WithBatchSize(batchSize int) *Dispatcher {
	d.batchSize = batchSize
	return d
}

// Start 启动 Dispatcher（在 goroutine 中运行）
func (d *Dispatcher) Start(ctx context.Context) {
	d.logger.Info("Starting Outbox Dispatcher",
		zap.Int("max_retries", d.maxRetries),
		zap.Duration("interval", d.interval),
		zap.Int("batch_size", d.batchSize),
	)

	ticker := time.NewTicker(d.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			d.logger.Info("Outbox Dispatcher stopped")
			return
		case <-ticker.C:
			d.processPendingEvents(ctx)
		}
	}
}

// processPendingEvents 处理待发送的事件
func (d *Dispatcher) processPendingEvents(ctx context.Context) {
	events, err := d.repo.GetPendingEvents(ctx, d.batchSize)
	if err != nil {
		d.logger.Error("Failed to get pending events", zap.Error(err))
		return
	}

	if len(events) == 0 {
		return // 没有待处理的事件
	}

	d.logger.Debug("Processing pending events",
		zap.Int("count", len(events)),
	)

	for _, event := range events {
		if err := d.publishEvent(ctx, event); err != nil {
			d.logger.Error("Failed to publish event",
				zap.Int64("event_id", event.ID),
				zap.String("routing_key", event.RoutingKey),
				zap.Error(err),
			)
			
			// 标记为失败或增加重试次数
			if err := d.repo.MarkAsFailed(ctx, event.ID, d.maxRetries); err != nil {
				d.logger.Error("Failed to mark event as failed",
					zap.Int64("event_id", event.ID),
					zap.Error(err),
				)
			}
			continue
		}

		// 发布成功，标记为已发送
		if err := d.repo.MarkAsSent(ctx, event.ID); err != nil {
			d.logger.Error("Failed to mark event as sent",
				zap.Int64("event_id", event.ID),
				zap.Error(err),
			)
		} else {
			d.logger.Debug("Event published successfully",
				zap.Int64("event_id", event.ID),
				zap.String("routing_key", event.RoutingKey),
			)
		}
	}
}

// publishEvent 发布单个事件到 MQ
func (d *Dispatcher) publishEvent(ctx context.Context, event *Event) error {
	// 解析 payload
	var payload interface{}
	if err := json.Unmarshal(event.Payload, &payload); err != nil {
		return fmt.Errorf("failed to unmarshal payload: %w", err)
	}

	// 发布到 MQ
	// 注意：这里使用 PublishWithContext 以支持 trace_id 传播
	// 如果 payload 中有 trace_id，需要提取并传递
	ctx = d.extractTraceIDFromPayload(ctx, event.Payload)
	
	if err := d.publisher.PublishWithContext(ctx, event.RoutingKey, payload); err != nil {
		return fmt.Errorf("failed to publish to MQ: %w", err)
	}

	return nil
}

// extractTraceIDFromPayload 从 payload 中提取 trace_id（如果存在）
func (d *Dispatcher) extractTraceIDFromPayload(ctx context.Context, payload json.RawMessage) context.Context {
	var payloadMap map[string]interface{}
	if err := json.Unmarshal(payload, &payloadMap); err != nil {
		return ctx
	}
	
	if traceID, ok := payloadMap["trace_id"].(string); ok && traceID != "" {
		ctx = trace.WithContext(ctx, traceID)
	}
	
	return ctx
}

