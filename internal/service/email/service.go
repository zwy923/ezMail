package email

import (
	"context"
	"time"

	"mygoproject/internal/model"
	"mygoproject/internal/mq"
	"mygoproject/internal/repository"
)

type Service struct {
	emailRepo *repository.EmailRepository
	publisher *mq.Publisher
}

func NewService(emailRepo *repository.EmailRepository, publisher *mq.Publisher) *Service {
	return &Service{
		emailRepo: emailRepo,
		publisher: publisher,
	}
}

// CreateRawAndPublish creates a raw email record and publishes `email.received` event.
func (s *Service) CreateRawAndPublish(ctx context.Context, userID int, subject, body string) (int, error) {
	raw := &model.EmailRaw{
		UserID:    userID,
		Subject:   subject,
		Body:      body,
		RawJSON:   "{}",
		Status:    "received",
		CreatedAt: time.Now(),
	}

	emailID, err := s.emailRepo.CreateRawEmail(ctx, raw)
	if err != nil {
		return 0, err
	}

	// 构造事件 payload
	payload := mq.EmailReceivedPayload{
		EmailID:    emailID,
		UserID:     userID,
		Subject:    subject,
		Body:       body,
		ReceivedAt: time.Now(),
	}

	// 发布事件，使用 routing key "email.received"
	if err := s.publisher.Publish("email.received", payload); err != nil {
		return 0, err
	}

	return emailID, nil
}
