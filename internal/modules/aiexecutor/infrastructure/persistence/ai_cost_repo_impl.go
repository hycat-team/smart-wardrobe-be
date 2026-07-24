package persistence

import (
	"context"
	"errors"
	"time"

	"smart-wardrobe-be/internal/modules/aiexecutor/application/dto"
	"smart-wardrobe-be/internal/modules/aiexecutor/domain/constants/aienforcementmode"
	"smart-wardrobe-be/internal/modules/aiexecutor/domain/constants/aioperation"
	"smart-wardrobe-be/internal/modules/aiexecutor/domain/constants/airoute"
	"smart-wardrobe-be/internal/modules/aiexecutor/domain/constants/aiusageeventstatus"
	aiexecutor_entities "smart-wardrobe-be/internal/modules/aiexecutor/domain/entities"
	"smart-wardrobe-be/internal/modules/aiexecutor/domain/repositories"
	shared_repos "smart-wardrobe-be/internal/shared/infrastructure/repositories"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type AICostRepository struct {
	*shared_repos.GenericRepository[aiexecutor_entities.AICostPolicy, uuid.UUID]
}

func NewAICostRepository(db *gorm.DB) repositories.IAICostRepository {
	relations := []string{"Operations"}
	return &AICostRepository{GenericRepository: shared_repos.NewGenericRepository[aiexecutor_entities.AICostPolicy, uuid.UUID](
		db, relations)}
}

func (r *AICostRepository) ResolvePolicy(ctx context.Context, subscriptionPlanID uuid.UUID, userID uuid.UUID, operation aioperation.AIOperation, now time.Time) (*aiexecutor_entities.AIPolicyGrant, *aiexecutor_entities.AICostPolicy, *aiexecutor_entities.AICostPolicyOperation, error) {
	db := r.GetDB(ctx)
	var grant aiexecutor_entities.AIPolicyGrant
	err := db.Where("user_id = ? AND effective_from <= ? AND (effective_to IS NULL OR effective_to > ?)", userID, now, now).Order("effective_from DESC").First(&grant).Error
	if err == nil && subscriptionPlanID != uuid.Nil && grant.PlanID != subscriptionPlanID {
		_ = db.Model(&grant).Updates(map[string]any{"effective_to": now, "status": "closed", "updated_at": now}).Error
		err = gorm.ErrRecordNotFound
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil, nil, errors.New("subscription plan AI cost policy not yet migrated - need grant lookup")
	} else if err != nil {
		return nil, nil, nil, err
	}

	var policy aiexecutor_entities.AICostPolicy
	if err := r.GetQueryWithPreload(ctx).First(&policy, "id = ?", grant.PolicyID).Error; err != nil {
		return nil, nil, nil, errors.New("AI policy lookup not yet migrated")
	}
	for _, op := range policy.Operations {
		if op.Operation == operation && op.IsEnabled {
			return &grant, &policy, op, nil
		}
	}
	return nil, nil, nil, errors.New("AI operation is not configured for policy")
}

func periodFor(grant *aiexecutor_entities.AIPolicyGrant, days int, now time.Time) (int, time.Time, time.Time) {
	if days <= 0 {
		days = 30
	}
	duration := time.Duration(days) * 24 * time.Hour
	index := int(now.Sub(grant.EffectiveFrom) / duration)
	if index < 0 {
		index = 0
	}
	start := grant.EffectiveFrom.Add(time.Duration(index) * duration)
	end := start.Add(duration)
	if grant.EffectiveTo != nil && grant.EffectiveTo.Before(end) {
		end = *grant.EffectiveTo
	}
	return index, start, end
}

func (r *AICostRepository) Reserve(ctx context.Context, grant *aiexecutor_entities.AIPolicyGrant, policy *aiexecutor_entities.AICostPolicy, operation *aiexecutor_entities.AICostPolicyOperation, event *aiexecutor_entities.AIUsageEvent, now time.Time) (*aiexecutor_entities.AIUsagePeriodLedger, bool, error) {
	var ledger aiexecutor_entities.AIUsagePeriodLedger
	admitted := false
	err := r.GetDB(ctx).Transaction(func(tx *gorm.DB) error {
		idx, start, end := periodFor(grant, policy.PeriodDays, now)
		err := tx.Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "grant_id"}, {Name: "period_index"}}, DoNothing: true}).Create(&aiexecutor_entities.AIUsagePeriodLedger{GrantID: grant.ID, UserID: grant.UserID, PeriodIndex: idx, PeriodStart: start, PeriodEnd: end}).Error
		if err != nil {
			return err
		}
		if err = tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("grant_id=? AND period_index=?", grant.ID, idx).First(&ledger).Error; err != nil {
			return err
		}
		if event.LogicalRoute != airoute.Paid {
			admitted = true
		} else if policy.EnforcementMode == aienforcementmode.ObserveOnly {
			var attempts int64
			day := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
			if err = tx.Model(&aiexecutor_entities.AIUsageEvent{}).Where("user_id=? AND operation=? AND created_at>=? AND logical_route=?", grant.UserID, operation.Operation, day, airoute.Paid).Count(&attempts).Error; err != nil {
				return err
			}
			admitted = attempts < int64(operation.MaxPaidAttemptsPerDay)
		} else if policy.EnforcementMode == aienforcementmode.Strict && policy.HardCostMicroVND != nil {
			threshold := *policy.HardCostMicroVND * int64(policy.FreeRouteThresholdBPS) / 10000
			var unknown int64
			day := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
			if err = tx.Model(&aiexecutor_entities.AIUsageEvent{}).Where("user_id=? AND status IN ? AND created_at>=?", grant.UserID, []string{string(aiusageeventstatus.UnknownUsage), string(aiusageeventstatus.ExpiredUnverified)}, day).Count(&unknown).Error; err != nil {
				return err
			}
			admitted = unknown < int64(policy.MaxUnknownPaidRequestsPerDay) && ledger.ActualCostMicroVND+ledger.ReservedCostMicroVND+event.ReservedCostMicroVND <= threshold
		}
		if !admitted {
			return nil
		}
		event.LedgerID = &ledger.ID
		if err = tx.Create(event).Error; err != nil {
			return err
		}
		if event.ReservedCostMicroVND > 0 {
			if err = tx.Model(&ledger).UpdateColumn("reserved_cost_micro_vnd", gorm.Expr("reserved_cost_micro_vnd + ?", event.ReservedCostMicroVND)).Error; err != nil {
				return err
			}
		}
		return nil
	})
	return &ledger, admitted, err
}

func (r *AICostRepository) MarkInFlight(ctx context.Context, id uuid.UUID, at time.Time) error {
	return r.GetDB(ctx).Model(&aiexecutor_entities.AIUsageEvent{}).Where("request_id=?", id).Updates(map[string]any{"status": aiusageeventstatus.InFlight, "sent_at": at, "updated_at": at}).Error
}

func (r *AICostRepository) Confirm(ctx context.Context, id uuid.UUID, prompt, output, thinking, cost int64, provider, model, finish string, at time.Time) error {
	return r.GetDB(ctx).Transaction(func(tx *gorm.DB) error {
		var e aiexecutor_entities.AIUsageEvent
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&e, "request_id=?", id).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil
			}
			return err
		}
		if e.Status == aiusageeventstatus.Confirmed || e.Status == aiusageeventstatus.Released || e.Status == aiusageeventstatus.ExpiredUnverified {
			return nil
		}
		paid := e.LogicalRoute == airoute.Paid
		if !paid {
			cost = 0
		}
		updates := map[string]any{"status": aiusageeventstatus.Confirmed, "prompt_tokens": prompt, "output_tokens": output, "thinking_tokens": thinking, "actual_cost_micro_vnd": cost, "reserved_cost_micro_vnd": 0, "provider": provider, "model": model, "finish_reason": finish, "completed_at": at, "updated_at": at}
		if err := tx.Model(&e).Updates(updates).Error; err != nil {
			return err
		}
		ledgerUpdates := map[string]any{"reserved_cost_micro_vnd": gorm.Expr("GREATEST(reserved_cost_micro_vnd - ?,0)", e.ReservedCostMicroVND), "updated_at": at}
		if paid {
			ledgerUpdates["paid_input_tokens"] = gorm.Expr("paid_input_tokens + ?", prompt)
			ledgerUpdates["paid_output_tokens"] = gorm.Expr("paid_output_tokens + ?", output+thinking)
			ledgerUpdates["actual_cost_micro_vnd"] = gorm.Expr("actual_cost_micro_vnd + ?", cost)
		} else {
			ledgerUpdates["free_input_tokens"] = gorm.Expr("free_input_tokens + ?", prompt)
			ledgerUpdates["free_output_tokens"] = gorm.Expr("free_output_tokens + ?", output+thinking)
		}
		return tx.Model(&aiexecutor_entities.AIUsagePeriodLedger{}).Where("id=?", e.LedgerID).Updates(ledgerUpdates).Error
	})
}

func (r *AICostRepository) Release(ctx context.Context, id uuid.UUID, reason string, at time.Time) error {
	return r.finishWithoutCost(ctx, id, string(aiusageeventstatus.Released), reason, nil, at)
}

func (r *AICostRepository) ExpireGrantsByUserID(ctx context.Context, userID uuid.UUID, now time.Time) error {
	return r.GetDB(ctx).
		Model(&aiexecutor_entities.AIPolicyGrant{}).
		Where("user_id = ? AND effective_from <= ? AND (effective_to IS NULL OR effective_to > ?)", userID, now, now).
		Updates(map[string]any{"effective_to": now, "updated_at": now, "status": "closed"}).
		Error
}

func (r *AICostRepository) CreateGrant(ctx context.Context, grant *aiexecutor_entities.AIPolicyGrant) error {
	return r.GetDB(ctx).Create(grant).Error
}

func (r *AICostRepository) CreateEvent(ctx context.Context, event *aiexecutor_entities.AIUsageEvent) error {
	return r.GetDB(ctx).Create(event).Error
}

func (r *AICostRepository) MarkUnknown(ctx context.Context, id uuid.UUID, reason string, expires time.Time) error {
	updates := map[string]any{"status": aiusageeventstatus.UnknownUsage, "error_code": reason, "updated_at": time.Now()}
	if !expires.IsZero() {
		updates["unknown_expires_at"] = expires
	}
	return r.GetDB(ctx).Model(&aiexecutor_entities.AIUsageEvent{}).Where("request_id=? AND status IN ?", id, []aiusageeventstatus.AIUsageEventStatus{aiusageeventstatus.Reserved, aiusageeventstatus.InFlight}).Updates(updates).Error
}
func (r *AICostRepository) finishWithoutCost(ctx context.Context, id uuid.UUID, status, reason string, expires *time.Time, at time.Time) error {
	return r.GetDB(ctx).Transaction(func(tx *gorm.DB) error {
		var e aiexecutor_entities.AIUsageEvent
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&e, "request_id=?", id).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil
			}
			return err
		}
		if e.Status == aiusageeventstatus.Confirmed || e.Status == aiusageeventstatus.Released || e.Status == aiusageeventstatus.ExpiredUnverified {
			return nil
		}
		if err := tx.Model(&e).Updates(map[string]any{"status": status, "error_code": reason, "reserved_cost_micro_vnd": 0, "unknown_expires_at": expires, "completed_at": at, "updated_at": at}).Error; err != nil {
			return err
		}
		return tx.Model(&aiexecutor_entities.AIUsagePeriodLedger{}).Where("id=?", e.LedgerID).UpdateColumn("reserved_cost_micro_vnd", gorm.Expr("GREATEST(reserved_cost_micro_vnd - ?,0)", e.ReservedCostMicroVND)).Error
	})
}
func (r *AICostRepository) ExpireUnknown(ctx context.Context, now time.Time, limit int) (int64, error) {
	var ids []uuid.UUID
	err := r.GetDB(ctx).Model(&aiexecutor_entities.AIUsageEvent{}).Where("status IN ? AND unknown_expires_at <= ?", []aiusageeventstatus.AIUsageEventStatus{aiusageeventstatus.Reserved, aiusageeventstatus.UnknownUsage}, now).Limit(limit).Pluck("request_id", &ids).Error
	if err != nil {
		return 0, err
	}
	var n int64
	for _, id := range ids {
		if err = r.finishWithoutCost(ctx, id, string(aiusageeventstatus.ExpiredUnverified), "usage_unverified_expired", nil, now); err != nil {
			return n, err
		}
		n++
	}
	return n, nil
}

func (r *AICostRepository) GetAdminAIUsageStats(ctx context.Context, snapshotDate time.Time) (*dto.AIUsageAdminStatsDTO, error) {
	db := r.GetDB(ctx)

	startOfDay := time.Date(snapshotDate.Year(), snapshotDate.Month(), snapshotDate.Day(), 0, 0, 0, 0, snapshotDate.Location())
	endOfDay := startOfDay.Add(24 * time.Hour)

	var reqCount int64
	_ = db.Model(&aiexecutor_entities.AIUsageEvent{}).
		Where("created_at >= ? AND created_at < ?", startOfDay, endOfDay).
		Count(&reqCount).Error

	var errCount int64
	_ = db.Model(&aiexecutor_entities.AIUsageEvent{}).
		Where("created_at >= ? AND created_at < ? AND status IN ?", startOfDay, endOfDay, []string{"FAILED", "RELEASED", "EXPIRED"}).
		Count(&errCount).Error

	var paidTokens int64
	_ = db.Model(&aiexecutor_entities.AIUsageEvent{}).
		Where("created_at >= ? AND created_at < ?", startOfDay, endOfDay).
		Select("COALESCE(SUM(total_tokens), 0)").Scan(&paidTokens).Error

	var actualCost int64
	_ = db.Model(&aiexecutor_entities.AIUsagePeriodLedger{}).
		Where("period_start >= ? AND period_start < ?", startOfDay, endOfDay).
		Select("COALESCE(SUM(actual_cost_micro_vnd), 0)").Scan(&actualCost).Error

	return &dto.AIUsageAdminStatsDTO{
		SnapshotDate:         snapshotDate,
		AIRequestsCount:      int(reqCount),
		AIErrorCount:         int(errCount),
		AIPaidTokens:         paidTokens,
		AIActualCostMicroVND: actualCost,
	}, nil
}

func (r *AICostRepository) GetAIEventsDrillDown(ctx context.Context, status string, page, limit int) ([]dto.AIEventDrillDownDTO, int64, error) {
	db := r.GetDB(ctx).Model(&aiexecutor_entities.AIUsageEvent{})

	if status != "" {
		db = db.Where("status = ?", status)
	}

	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}
	offset := (page - 1) * limit

	var events []aiexecutor_entities.AIUsageEvent
	err := db.Order("created_at DESC").Offset(offset).Limit(limit).Find(&events).Error
	if err != nil {
		return nil, 0, err
	}

	items := make([]dto.AIEventDrillDownDTO, 0, len(events))
	for _, e := range events {
		providerStr := ""
		if e.Provider != nil {
			providerStr = *e.Provider
		}
		modelStr := ""
		if e.Model != nil {
			modelStr = *e.Model
		}

		items = append(items, dto.AIEventDrillDownDTO{
			RequestID:    e.RequestID.String(),
			Provider:     providerStr,
			Model:        modelStr,
			StatusCode:   200,
			ErrorMessage: "",
			CreatedAt:    e.CreatedAt,
		})
	}

	return items, total, nil
}
