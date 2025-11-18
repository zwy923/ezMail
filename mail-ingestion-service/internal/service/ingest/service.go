package ingest

import (
	"context"
	"fmt"
	"time"

	"mail-ingestion-service/internal/repository"
	dbcontracts "mygoproject/contracts/db"
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

// 事务边界处理：如果 MQ 发布失败，记录到 failed_events 表并返回错误（让 api-gateway 重试）
func (s *Service) CreateRawAndPublish(ctx context.Context, userID int, subject, body string) (int, error) {

	// 1. Insert raw email
	raw := &dbcontracts.Email{
		UserID:    userID,
		Subject:   subject,
		Body:      body,
		RawJSON:   "{}",
		Status:    "received",
		CreatedAt: time.Now(),
	}

	emailID, err := s.emailRepo.CreateRawEmail(ctx, raw)
	if err != nil {
		s.logger.Error("Failed to create raw email", zap.Error(err))
		return 0, fmt.Errorf("failed to create email: %w", err)
	}

	// 2. Construct event payload
	payload := mqcontracts.EmailReceivedPayload{
		EmailID:    emailID,
		UserID:     userID,
		Subject:    subject,
		Body:       body,
		ReceivedAt: time.Now(),
	}

	// 3 routing keys
	routingKeys := []string{
		"email.received.agent",
		"email.received.log",
		"email.received.notify",
	}

	// 3. Publish to all routing keys
	for _, rk := range routingKeys {
		if err := s.publisher.Publish(rk, payload); err != nil {
			// log + write failed_events
			s.logger.Error("Failed to publish MQ event",
				zap.String("routing_key", rk),
				zap.Int("email_id", emailID),
				zap.Error(err),
			)

			// record failed event for retry
			if recordErr := s.failedEventRepo.InsertFailedEvent(
				ctx,
				emailID,
				userID,
				"email.received", // event type
				rk,
				payload,
				err.Error(),
			); recordErr != nil {
				s.logger.Error("Failed to record failed event", zap.Error(recordErr))
			}

			// return error → make gateway retry (5xx)
			return emailID, fmt.Errorf("failed to publish event to %s: %w", rk, err)
		}
	}

	s.logger.Info("Email created and all events published successfully",
		zap.Int("email_id", emailID),
		zap.Int("user_id", userID),
		zap.Any("routing_keys", routingKeys),
	)

	return emailID, nil
}
