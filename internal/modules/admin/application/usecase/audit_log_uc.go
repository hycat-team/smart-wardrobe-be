package usecase

import (
	"context"
	"time"

	"smart-wardrobe-be/internal/modules/admin/application/dto"
	uc_interfaces "smart-wardrobe-be/internal/modules/admin/application/interface/usecase"
	"smart-wardrobe-be/internal/modules/admin/application/mapper"
	"smart-wardrobe-be/internal/modules/admin/domain/entities"
	"smart-wardrobe-be/internal/modules/admin/domain/repositories"
)

type AuditLogUseCase struct {
	auditRepo repositories.IAdminAuditLogRepository
}

func NewAuditLogUseCase(auditRepo repositories.IAdminAuditLogRepository) uc_interfaces.IAuditLogUseCase {
	return &AuditLogUseCase{
		auditRepo: auditRepo,
	}
}

func (uc *AuditLogUseCase) Record(ctx context.Context, input dto.RecordAuditLogInput) error {
	payload := input.Payload
	if payload == nil {
		payload = make(map[string]any)
	}

	log := &entities.AdminAuditLog{
		ActorUserID: input.ActorUserID,
		Action:      input.Action,
		TargetType:  input.TargetType,
		TargetID:    input.TargetID,
		Payload:     entities.JSONDocument(payload),
		RequestIP:   input.RequestIP,
		CreatedAt:   time.Now(),
	}

	return uc.auditRepo.Create(ctx, log)
}

func (uc *AuditLogUseCase) List(ctx context.Context, filter dto.AuditLogFilter, page, limit int) ([]*dto.AdminAuditLogDTO, int64, error) {
	repoFilter := repositories.AuditLogFilter{
		ActorUserID: filter.ActorUserID,
		Action:      filter.Action,
		TargetType:  filter.TargetType,
		TargetID:    filter.TargetID,
		FromDate:    filter.FromDate,
		ToDate:      filter.ToDate,
	}
	logs, total, err := uc.auditRepo.List(ctx, repoFilter, page, limit)
	if err != nil {
		return nil, 0, err
	}

	return mapper.MapAdminAuditLogsToDTO(logs), total, nil
}

func (uc *AuditLogUseCase) DeleteOlderThan(ctx context.Context, cutoff time.Time) (int64, error) {
	return uc.auditRepo.DeleteOlderThan(ctx, cutoff)
}
