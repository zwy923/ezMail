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
// This method is idempotent: calling it multiple times with the same event
// will have the same effect as calling it once.
func (s *ClassifyService) HandleEmailReceived(ctx context.Context, raw json.RawMessage) error {
	var p mq.EmailReceivedPayload
	if err := json.Unmarshal(raw, &p); err != nil {
		return err
	}

	// 幂等性检查：如果邮件已经分类，直接返回
	email, err := s.emailRepo.FindRawByID(ctx, p.EmailID)
	if err != nil {
		return err
	}
	if email.Status == "classified" {
		// 已经分类过，幂等返回
		return nil
	}

	// 检查metadata是否已存在（双重检查）
	exists, err := s.metadataRepo.Exists(ctx, p.EmailID)
	if err != nil {
		return err
	}
	if exists {
		// metadata已存在，只需更新状态
		if email.Status != "classified" {
			if err := s.emailRepo.UpdateStatus(ctx, p.EmailID, "classified"); err != nil {
				return err
			}
		}
		return nil
	}

	// 执行分类逻辑
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

	// 插入metadata（内部已做幂等检查）
	if err := s.metadataRepo.Insert(ctx, p.EmailID, category, 1.0); err != nil {
		return err
	}

	// 更新状态
	if err := s.emailRepo.UpdateStatus(ctx, p.EmailID, "classified"); err != nil {
		return err
	}

	return nil
}
