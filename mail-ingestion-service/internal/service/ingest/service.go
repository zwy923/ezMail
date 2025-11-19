package ingest

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"mail-ingestion-service/internal/repository"
	dbcontracts "mygoproject/contracts/db"
	mqcontracts "mygoproject/contracts/mq"
	"mygoproject/pkg/outbox"
	"mygoproject/pkg/trace"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

type Service struct {
	db         *pgxpool.Pool
	emailRepo  *repository.EmailRepository
	outboxRepo *outbox.Repository
	logger     *zap.Logger
}

func NewService(
	db *pgxpool.Pool,
	emailRepo *repository.EmailRepository,
	logger *zap.Logger,
) *Service {
	return &Service{
		db:         db,
		emailRepo:  emailRepo,
		outboxRepo: outbox.NewRepository(db),
		logger:     logger,
	}
}

// CreateRawAndPublish 使用 Outbox 模式：在事务中写入 email 和 outbox 事件
func (s *Service) CreateRawAndPublish(ctx context.Context, userID int, subject, body string) (int, error) {
	// 开始事务
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// 1. Insert raw email（在事务中）
	raw := &dbcontracts.Email{
		UserID:    userID,
		Subject:   subject,
		Body:      body,
		RawJSON:   "{}",
		Status:    "received",
		CreatedAt: time.Now(),
	}

	emailID, err := s.emailRepo.CreateRawEmailTx(ctx, tx, raw)
	if err != nil {
		s.logger.Error("Failed to create raw email", zap.Error(err))
		return 0, fmt.Errorf("failed to create email: %w", err)
	}

	// 2. Construct event payload
	traceID := trace.FromContext(ctx)
	payload := mqcontracts.EmailReceivedPayload{
		EmailID:    emailID,
		UserID:     userID,
		Subject:    subject,
		Body:       body,
		ReceivedAt: time.Now(),
		TraceID:    traceID,
	}

	// 3. 将事件写入 outbox（在同一个事务中）
	routingKeys := []string{
		"email.received.agent",
		"email.received.log",
		"email.received.notify",
	}

	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return 0, fmt.Errorf("failed to marshal payload: %w", err)
	}

	emailID64 := int64(emailID)
	for _, rk := range routingKeys {
		event := &outbox.Event{
			AggregateType: "email",
			AggregateID:   &emailID64,
			RoutingKey:    rk,
			Payload:       payloadJSON,
			Status:        "pending",
		}

		if err := s.outboxRepo.InsertEvent(ctx, tx, event); err != nil {
			s.logger.Error("Failed to insert outbox event",
				zap.String("routing_key", rk),
				zap.Error(err),
			)
			return 0, fmt.Errorf("failed to insert outbox event: %w", err)
		}
	}

	// 4. 提交事务（email 和 outbox 事件一起提交）
	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("failed to commit transaction: %w", err)
	}

	s.logger.Info("Email created and outbox events inserted successfully",
		zap.Int("email_id", emailID),
		zap.Int("user_id", userID),
		zap.Any("routing_keys", routingKeys),
	)

	return emailID, nil
}
