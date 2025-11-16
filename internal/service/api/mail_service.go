package service

import (
	"context"
	"time"

	"mygoproject/internal/model"
	"mygoproject/internal/mq"
	"mygoproject/internal/repository"
)

type MailService struct {
	emailRepo *repository.EmailRepository
	producer  *mq.Producer
}

func NewMailService(emailRepo *repository.EmailRepository, producer *mq.Producer) *MailService {
	return &MailService{
		emailRepo: emailRepo,
		producer:  producer,
	}
}

// CreateRawAndPublish creates a raw email record and publishes `email.received` event.
func (s *MailService) CreateRawAndPublish(ctx context.Context, userID int, subject, body string) (int, error) {
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
	if err := s.producer.Publish("email.received", payload); err != nil {
		return 0, err
	}

	return emailID, nil
}
