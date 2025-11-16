package service

import (
	"context"
	"encoding/json"
	"mygoproject/internal/mq"
	"mygoproject/internal/repository"
	"strings"
)

type ClassifyService struct {
	emailRepo    *repository.EmailRepository
	metadataRepo *repository.MetadataRepository
}

func NewClassifyService(emailRepo *repository.EmailRepository, metadataRepo *repository.MetadataRepository) *ClassifyService {
	return &ClassifyService{
		emailRepo:    emailRepo,
		metadataRepo: metadataRepo,
	}
}

// HandleEmailReceived processes an EmailReceivedEvent and stores classification.
func (s *ClassifyService) HandleEmailReceived(ctx context.Context, raw json.RawMessage) error {
	var p mq.EmailReceivedPayload
	if err := json.Unmarshal(raw, &p); err != nil {
		return err
	}

	var category string
	subj := strings.ToLower(p.Subject)
	switch {
	case strings.Contains(subj, "payment"):
		category = "finance"
	case strings.Contains(subj, "meeting"):
		category = "schedule"
	default:
		category = "other"
	}

	if err := s.metadataRepo.Insert(ctx, p.EmailID, category, 1.0); err != nil {
		return err
	}

	if err := s.emailRepo.UpdateStatus(ctx, p.EmailID, "classified"); err != nil {
		return err
	}

	return nil
}
