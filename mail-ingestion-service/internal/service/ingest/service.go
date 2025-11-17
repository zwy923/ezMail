package ingest

import (
	"context"
	"fmt"
	"time"

	"mail-ingestion-service/internal/model"
	"mail-ingestion-service/internal/repository"
	mqcontracts "mygoproject/contracts/mq"
	"mygoproject/pkg/mq"

	"go.uber.org/zap"
)

type Service struct {
	emailRepo       *repository.EmailRepository
	failedEventRepo *repository.FailedEventRepository
	publisher       *mq.Publisher
	logger          *zap.Logger
}

func NewService(
	emailRepo *repository.EmailRepository,
	failedEventRepo *repository.FailedEventRepository,
	publisher *mq.Publisher,
	logger *zap.Logger,
) *Service {
	return &Service{
		emailRepo:       emailRepo,
		failedEventRepo: failedEventRepo,
		publisher:       publisher,
		logger:          logger,
	}
}

// CreateRawAndPublish creates a raw email record and publishes `email.received` event.
// 事务边界处理：如果 MQ 发布失败，记录到 failed_events 表并返回错误（让 api-gateway 重试）
func (s *Service) CreateRawAndPublish(ctx context.Context, userID int, subject, body string) (int, error) {
	raw := &model.EmailRaw{
		UserID:    userID,
		Subject:   subject,
		Body:      body,
		RawJSON:   "{}",
		Status:    "received",
		CreatedAt: time.Now(),
	}

	// 1. 先插入数据库
	emailID, err := s.emailRepo.CreateRawEmail(ctx, raw)
	if err != nil {
		s.logger.Error("Failed to create raw email",
			zap.Int("user_id", userID),
			zap.String("subject", subject),
			zap.Error(err),
		)
		return 0, fmt.Errorf("failed to create email: %w", err)
	}

	// 2. 构造事件 payload
	payload := mqcontracts.EmailReceivedPayload{
		EmailID:    emailID,
		UserID:     userID,
		Subject:    subject,
		Body:       body,
		ReceivedAt: time.Now(),
	}

	// 3. 发布事件到 MQ
	routingKey := "email.received"
	if err := s.publisher.Publish(routingKey, payload); err != nil {
		// MQ 发布失败：记录到 failed_events 表
		s.logger.Error("Failed to publish MQ event, recording to failed_events",
			zap.Int("email_id", emailID),
			zap.Int("user_id", userID),
			zap.String("routing_key", routingKey),
			zap.Error(err),
		)

		// 记录失败事件（用于后续重试）
		if recordErr := s.failedEventRepo.InsertFailedEvent(
			ctx,
			emailID,
			userID,
			"email.received",
			routingKey,
			payload,
			err.Error(),
		); recordErr != nil {
			// 如果记录失败事件也失败，记录日志但继续返回原始错误
			s.logger.Error("Failed to record failed event",
				zap.Int("email_id", emailID),
				zap.Error(recordErr),
			)
		}

		// 返回错误，让 api-gateway 重试（5xx）
		return emailID, fmt.Errorf("failed to publish event: %w", err)
	}

	s.logger.Info("Email created and event published successfully",
		zap.Int("email_id", emailID),
		zap.Int("user_id", userID),
		zap.String("routing_key", routingKey),
	)

	return emailID, nil
}
