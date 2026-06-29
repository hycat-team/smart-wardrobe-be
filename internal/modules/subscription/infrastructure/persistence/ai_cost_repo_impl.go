package persistence

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"smart-wardrobe-be/internal/modules/subscription/domain/repositories"
	"smart-wardrobe-be/internal/shared/domain/entities"
	"smart-wardrobe-be/internal/shared/domain/constants/subscription/aienforcementmode"
	"smart-wardrobe-be/internal/shared/domain/constants/subscription/aipolicygrantstatus"
	"smart-wardrobe-be/internal/shared/domain/constants/subscription/aiusageeventstatus"
	shared_repos "smart-wardrobe-be/internal/shared/infrastructure/repositories"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type AICostRepository struct {
	*shared_repos.GenericRepository[entities.AICostPolicy, uuid.UUID]
}

func NewAICostRepository(db *gorm.DB) repositories.IAICostRepository {
	return &AICostRepository{GenericRepository: shared_repos.NewGenericRepository[entities.AICostPolicy, uuid.UUID](db, nil)}
}

func (r *AICostRepository) ResolvePolicy(ctx context.Context, userID uuid.UUID, operation string, now time.Time) (*entities.UserAIPolicyGrant, *entities.AICostPolicy, *entities.AICostPolicyOperation, error) {
	db := r.GetDB(ctx)
	var sub entities.UserSubscription
	if err := db.Preload("SubscriptionPlan.AICostPolicy.Operations").First(&sub, "user_id = ?", userID).Error; err != nil {
		return nil, nil, nil, err
	}
	var grant entities.UserAIPolicyGrant
	err := db.Where("user_id = ? AND effective_from <= ? AND (effective_to IS NULL OR effective_to > ?)", userID, now, now).Order("effective_from DESC").First(&grant).Error
	if err == nil && grant.PlanID != sub.SubscriptionPlanID {
		_ = db.Model(&grant).Updates(map[string]any{"effective_to": now, "status": "closed", "updated_at": now}).Error
		err = gorm.ErrRecordNotFound
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		policy := sub.SubscriptionPlan.AICostPolicy
		if policy == nil {
			return nil, nil, nil, errors.New("subscription plan has no AI cost policy")
		}
		snapshot, _ := json.Marshal(policy)
		grant = entities.UserAIPolicyGrant{UserID: userID, PolicyID: policy.ID, PlanID: sub.SubscriptionPlanID, PlanCode: sub.CurrentPlanCode, TierRank: sub.CurrentTierRank, PolicySnapshot: entities.JSONDocument(snapshot), EffectiveFrom: sub.StartedAt, EffectiveTo: sub.ExpiresAt, Status: aipolicygrantstatus.Active}
		if err = db.Create(&grant).Error; err != nil {
			return nil, nil, nil, err
		}
	} else if err != nil {
		return nil, nil, nil, err
	}
	var policy entities.AICostPolicy
	if err = db.Preload("Operations").First(&policy, "id = ?", grant.PolicyID).Error; err != nil {
		return nil, nil, nil, err
	}
	for _, op := range policy.Operations {
		if op.Operation == operation && op.IsEnabled {
			return &grant, &policy, op, nil
		}
	}
	return nil, nil, nil, errors.New("AI operation is not configured for policy")
}

func periodFor(grant *entities.UserAIPolicyGrant, days int, now time.Time) (int, time.Time, time.Time) {
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

func (r *AICostRepository) Reserve(ctx context.Context, grant *entities.UserAIPolicyGrant, policy *entities.AICostPolicy, operation *entities.AICostPolicyOperation, event *entities.AIUsageEvent, now time.Time) (*entities.AIUsagePeriodLedger, bool, error) {
	var ledger entities.AIUsagePeriodLedger
	admitted := false
	err := r.GetDB(ctx).Transaction(func(tx *gorm.DB) error {
		idx, start, end := periodFor(grant, policy.PeriodDays, now)
		err := tx.Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "grant_id"}, {Name: "period_index"}}, DoNothing: true}).Create(&entities.AIUsagePeriodLedger{GrantID: grant.ID, UserID: grant.UserID, PeriodIndex: idx, PeriodStart: start, PeriodEnd: end}).Error
		if err != nil {
			return err
		}
		if err = tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("grant_id=? AND period_index=?", grant.ID, idx).First(&ledger).Error; err != nil {
			return err
		}
		if event.LogicalRoute != "paid" {
			admitted = true
		} else if policy.EnforcementMode == aienforcementmode.ObserveOnly {
			var attempts int64
			day := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
			if err = tx.Model(&entities.AIUsageEvent{}).Where("user_id=? AND operation=? AND created_at>=? AND logical_route=?", grant.UserID, operation.Operation, day, "paid").Count(&attempts).Error; err != nil {
				return err
			}
			admitted = attempts < int64(operation.MaxPaidAttemptsPerDay)
		} else if policy.EnforcementMode == aienforcementmode.Strict && policy.HardCostMicroVND != nil {
			threshold := *policy.HardCostMicroVND * int64(policy.FreeRouteThresholdBPS) / 10000
			var unknown int64
			day := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
			if err = tx.Model(&entities.AIUsageEvent{}).Where("user_id=? AND status IN ? AND created_at>=?", grant.UserID, []string{string(aiusageeventstatus.UnknownUsage), string(aiusageeventstatus.ExpiredUnverified)}, day).Count(&unknown).Error; err != nil {
				return err
			}
			admitted = unknown < int64(policy.MaxUnknownPaidRequestsPerDay) && ledger.ActualCostMicroVND+ledger.ReservedCostMicroVND+event.ReservedCostMicroVND <= threshold
		}
		if !admitted {
			return nil
		}
		event.LedgerID = ledger.ID
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
	return r.GetDB(ctx).Model(&entities.AIUsageEvent{}).Where("request_id=?", id).Updates(map[string]any{"status": aiusageeventstatus.InFlight, "sent_at": at, "updated_at": at}).Error
}

func (r *AICostRepository) Confirm(ctx context.Context, id uuid.UUID, prompt, output, thinking, cost int64, provider, model, finish string, at time.Time) error {
	return r.GetDB(ctx).Transaction(func(tx *gorm.DB) error {
		var e entities.AIUsageEvent
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&e, "request_id=?", id).Error; err != nil {
			return err
		}
		if e.Status == aiusageeventstatus.Confirmed || e.Status == aiusageeventstatus.Released || e.Status == aiusageeventstatus.ExpiredUnverified {
			return nil
		}
		paid := e.LogicalRoute == "paid"
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
		return tx.Model(&entities.AIUsagePeriodLedger{}).Where("id=?", e.LedgerID).Updates(ledgerUpdates).Error
	})
}

func (r *AICostRepository) Release(ctx context.Context, id uuid.UUID, reason string, at time.Time) error {
	return r.finishWithoutCost(ctx, id, string(aiusageeventstatus.Released), reason, nil, at)
}
func (r *AICostRepository) MarkUnknown(ctx context.Context, id uuid.UUID, reason string, expires time.Time) error {
	updates := map[string]any{"status": aiusageeventstatus.UnknownUsage, "error_code": reason, "updated_at": time.Now()}
	if !expires.IsZero() {
		updates["unknown_expires_at"] = expires
	}
	return r.GetDB(ctx).Model(&entities.AIUsageEvent{}).Where("request_id=? AND status IN ?", id, []aiusageeventstatus.AIUsageEventStatus{aiusageeventstatus.Reserved, aiusageeventstatus.InFlight}).Updates(updates).Error
}
func (r *AICostRepository) finishWithoutCost(ctx context.Context, id uuid.UUID, status, reason string, expires *time.Time, at time.Time) error {
	return r.GetDB(ctx).Transaction(func(tx *gorm.DB) error {
		var e entities.AIUsageEvent
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&e, "request_id=?", id).Error; err != nil {
			return err
		}
		if e.Status == aiusageeventstatus.Confirmed || e.Status == aiusageeventstatus.Released || e.Status == aiusageeventstatus.ExpiredUnverified {
			return nil
		}
		if err := tx.Model(&e).Updates(map[string]any{"status": status, "error_code": reason, "reserved_cost_micro_vnd": 0, "unknown_expires_at": expires, "completed_at": at, "updated_at": at}).Error; err != nil {
			return err
		}
		return tx.Model(&entities.AIUsagePeriodLedger{}).Where("id=?", e.LedgerID).UpdateColumn("reserved_cost_micro_vnd", gorm.Expr("GREATEST(reserved_cost_micro_vnd - ?,0)", e.ReservedCostMicroVND)).Error
	})
}
func (r *AICostRepository) ExpireUnknown(ctx context.Context, now time.Time, limit int) (int64, error) {
	var ids []uuid.UUID
	err := r.GetDB(ctx).Model(&entities.AIUsageEvent{}).Where("status IN ? AND unknown_expires_at <= ?", []aiusageeventstatus.AIUsageEventStatus{aiusageeventstatus.Reserved, aiusageeventstatus.UnknownUsage}, now).Limit(limit).Pluck("request_id", &ids).Error
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
