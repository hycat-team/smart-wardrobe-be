package usecase

import (
	"context"
	"time"

	"smart-wardrobe-be/services/account/dto/admin"
	uc_interfaces "smart-wardrobe-be/services/account/internal/modules/admin/application/interface/usecase"
	"smart-wardrobe-be/services/account/internal/modules/admin/application/mapper"
	"smart-wardrobe-be/services/account/internal/modules/admin/domain/entities"
	"smart-wardrobe-be/services/account/internal/modules/admin/domain/repositories"
	"smart-wardrobe-be/shared/domain/constants/adminaudit"
)

type AuditLogUseCase struct {
	auditRepo repositories.IAdminAuditLogRepository
}

func NewAuditLogUseCase(auditRepo repositories.IAdminAuditLogRepository) uc_interfaces.IAuditLogUseCase {
	return &AuditLogUseCase{
		auditRepo: auditRepo,
	}
}

// gate2DedupeWindow bounds how far back the duplicate check looks.
//
// The outbox retries a failed publish every outbox.retry_interval_minutes (5)
// and the consumer queue is quorum with x-delivery-limit 5, so a redelivered
// event can surface minutes after the first attempt. The window has to cover
// that whole span or a retry would still double-log.
//
// Known limitation, accepted on purpose: a genuine second edit of the same
// field to the same value by the same administrator inside this window is
// treated as a duplicate and dropped. The alternative — a unique index — would
// drop that edit permanently instead of just for this window, and would also
// wrongly drop a legitimate repeat. See contracts/admin-cost-policy-api.md §7.3.
const gate2DedupeWindow = 10 * time.Minute

func (uc *AuditLogUseCase) Record(ctx context.Context, input dto.RecordAuditLogInput) error {
	payload := input.Payload
	if payload == nil {
		payload = make(map[string]any)
	}

	// Only the gate-2 action is de-duplicated. Other audit sources legitimately
	// emit identical payloads — two pricing updates that leave the price alone,
	// for instance — and must not be silently discarded.
	if input.Action == string(adminaudit.AICostPolicyUpdate) {
		duplicate, err := uc.isDuplicateGate2Change(ctx, input)
		if err != nil {
			return err
		}
		if duplicate {
			return nil
		}
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

// isDuplicateGate2Change answers "has this exact change already been recorded?".
// Any missing identity field makes the answer "no": an event we cannot identify
// must be logged, not dropped, because losing an audit row is worse than
// keeping a redundant one.
func (uc *AuditLogUseCase) isDuplicateGate2Change(ctx context.Context, input dto.RecordAuditLogInput) (bool, error) {
	policyCode, _ := input.Payload["policyCode"].(string)
	operation, _ := input.Payload["operation"].(string)
	fieldName, _ := input.Payload["fieldName"].(string)
	if policyCode == "" || operation == "" || fieldName == "" {
		return false, nil
	}

	var newValue *string
	if raw, ok := input.Payload["newValue"].(string); ok {
		newValue = &raw
	}

	return uc.auditRepo.ExistsRecentChange(ctx, repositories.RecentChangeQuery{
		ActorUserID: input.ActorUserID,
		Action:      input.Action,
		TargetType:  input.TargetType,
		PolicyCode:  policyCode,
		Operation:   operation,
		FieldName:   fieldName,
		NewValue:    newValue,
		Since:       time.Now().Add(-gate2DedupeWindow),
	})
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
