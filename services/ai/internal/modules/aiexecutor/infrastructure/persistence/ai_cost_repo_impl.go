package persistence

import (
	"context"
	"errors"
	"time"

	"smart-wardrobe-be/services/ai/dto/aiexecutor"
	aierrors "smart-wardrobe-be/services/ai/internal/modules/aiexecutor/application/errors"
	"smart-wardrobe-be/services/ai/internal/modules/aiexecutor/domain/constants/aienforcementmode"
	"smart-wardrobe-be/services/ai/internal/modules/aiexecutor/domain/constants/airoute"
	"smart-wardrobe-be/services/ai/internal/modules/aiexecutor/domain/constants/aiusageeventstatus"
	aiexecutor_entities "smart-wardrobe-be/services/ai/internal/modules/aiexecutor/domain/entities"
	"smart-wardrobe-be/services/ai/internal/modules/aiexecutor/domain/repositories"
	shared_repos "smart-wardrobe-be/shared/infrastructure/repositories"
	"smart-wardrobe-be/shared/pkg/utils/timeutils"

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

// fallbackEpoch anchors the ledger period of a request with no grant.
//
// It has to be a fixed instant, not the request time: the fallback ledger row is
// deduped on (user_id, period_index), so anchoring on "now" would pin every
// request to index 0 and the running total would never reset - a period ledger
// that never rolls over is not a period. The Unix epoch is used so the index is
// deterministic and identical for every process and every restart.
var fallbackEpoch = time.Unix(0, 0).UTC()

// periodFor returns the ledger period index and bounds for a request.
//
// With a grant the window is anchored on the grant's effective_from, so renewing
// a plan continues the existing period instead of handing the account a fresh
// budget. With no grant (fallback tier 1) there is no such anchor, so the
// window is anchored on a fixed epoch and advances on its own.
func periodFor(grant *aiexecutor_entities.AIPolicyGrant, days int, now time.Time) (int, time.Time, time.Time) {
	if days <= 0 {
		days = 30
	}
	anchor := fallbackEpoch
	if grant != nil {
		anchor = grant.EffectiveFrom
	}
	duration := time.Duration(days) * 24 * time.Hour
	index := int(now.Sub(anchor) / duration)
	if index < 0 {
		index = 0
	}
	start := anchor.Add(time.Duration(index) * duration)
	end := start.Add(duration)
	if grant != nil && grant.EffectiveTo != nil && grant.EffectiveTo.Before(end) {
		end = *grant.EffectiveTo
	}
	return index, start, end
}

// userIDFor picks the account whose ledger is being reserved. A nil grant means
// the account has no policy grant row, but the request still belongs to a user
// and the ledger row is keyed on that user.
func userIDFor(grant *aiexecutor_entities.AIPolicyGrant, event *aiexecutor_entities.AIUsageEvent) uuid.UUID {
	if grant != nil {
		return grant.UserID
	}
	if event.UserID != nil {
		return *event.UserID
	}
	return uuid.Nil
}

// nullableGrantID maps a missing grant to SQL NULL.
//
// A fallback ledger row is keyed on the account instead, which is why the
// column has to be nullable: a nil grant means the account has no policy grant
// row at all, so there is no id to point at.
func nullableGrantID(grant *aiexecutor_entities.AIPolicyGrant) *uuid.UUID {
	if grant == nil {
		return nil
	}
	return &grant.ID
}

func (r *AICostRepository) Reserve(ctx context.Context, grant *aiexecutor_entities.AIPolicyGrant, policy *aiexecutor_entities.AICostPolicy, operation *aiexecutor_entities.AICostPolicyOperation, maxPaidAttemptsPerDay *int, event *aiexecutor_entities.AIUsageEvent, now time.Time) (*aiexecutor_entities.AIUsagePeriodLedger, bool, error) {
	var ledger aiexecutor_entities.AIUsagePeriodLedger
	admitted := false
	var observeOnlyCapMissing bool
	userID := userIDFor(grant, event)

	err := r.GetDB(ctx).Transaction(func(tx *gorm.DB) error {
		idx, start, end := periodFor(grant, policy.PeriodDays, now)

		// Two insert strategies, one per case, and the reason is the shape of
		// the dedup key. The existing UNIQUE (grant_id, period_index) cannot
		// dedupe anything when grant_id is NULL, because SQL treats NULL as
		// distinct from NULL - every call would open a fresh ledger and the
		// anti-overspend control of constitution VI would evaporate. The
		// partial unique index ux_ledgers_user_period_fallback exists for
		// exactly those rows, so ON CONFLICT has to name the same predicate.
		//
		// Either way the row is read back under FOR UPDATE afterwards: the
		// lock is the anti-overspend guarantee, the insert is only how the row
		// comes to exist. Verified on PostgreSQL 17.10 (research.md R8).
		ledgerRow := &aiexecutor_entities.AIUsagePeriodLedger{
			GrantID:     nullableGrantID(grant),
			UserID:      userID,
			PeriodIndex: idx,
			PeriodStart: start,
			PeriodEnd:   end,
		}
		if grant == nil {
			if err := tx.Clauses(clause.OnConflict{
				Columns: []clause.Column{{Name: "user_id"}, {Name: "period_index"}},
				// TargetWhere, not Where: GORM's OnConflict.Where is the WHERE
				// of a DO UPDATE SET clause, and using it here emits
				// "ON CONFLICT (...) DO NOTHING" with no predicate â€” a syntax
				// error. TargetWhere is the one that renders the partial-index
				// inference clause, so it has to match
				// ux_ledgers_user_period_fallback exactly.
				TargetWhere: clause.Where{Exprs: []clause.Expression{clause.Eq{Column: clause.Column{Name: "grant_id"}, Value: nil}}},
				DoNothing:   true,
			}).Create(ledgerRow).Error; err != nil {
				return err
			}
			if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
				Where("user_id = ? AND period_index = ? AND grant_id IS NULL", userID, idx).
				First(&ledger).Error; err != nil {
				return err
			}
		} else {
			if err := tx.Clauses(clause.OnConflict{
				Columns:   []clause.Column{{Name: "grant_id"}, {Name: "period_index"}},
				DoNothing: true,
			}).Create(ledgerRow).Error; err != nil {
				return err
			}
			if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
				Where("grant_id = ? AND period_index = ?", grant.ID, idx).
				First(&ledger).Error; err != nil {
				return err
			}
		}

		if event.LogicalRoute != airoute.Paid {
			admitted = true
		} else if policy.EnforcementMode == aienforcementmode.ObserveOnly {
			// In observe_only there is no hard cap, so a per-day attempt limit is
			// the ONLY thing standing between the account and unbounded spend.
			//
			// The cap arrives from the caller, which takes it from the grant
			// snapshot (FR-029b). A nil here is therefore a misconfiguration, not
			// a legitimate "no limit": silently treating it as unlimited would let
			// the free policy run uncapped. The paid route is refused and the
			// caller escalates to fallback tier 2, which warns (FR-005e).
			if maxPaidAttemptsPerDay == nil {
				observeOnlyCapMissing = true
				admitted = false
			} else {
				var attempts int64
				day := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
				if err := tx.Model(&aiexecutor_entities.AIUsageEvent{}).Where("user_id=? AND operation=? AND created_at>=? AND logical_route=?", userID, operation.Operation, day, airoute.Paid).Count(&attempts).Error; err != nil {
					return err
				}
				admitted = attempts < int64(*maxPaidAttemptsPerDay)
			}
		} else if policy.EnforcementMode == aienforcementmode.Strict && policy.HardCostMicroVND != nil {
			threshold := *policy.HardCostMicroVND * int64(policy.FreeRouteThresholdBPS) / 10000
			var unknown int64
			day := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
			if err := tx.Model(&aiexecutor_entities.AIUsageEvent{}).Where("user_id=? AND status IN ? AND created_at>=?", userID, []string{string(aiusageeventstatus.UnknownUsage), string(aiusageeventstatus.ExpiredUnverified)}, day).Count(&unknown).Error; err != nil {
				return err
			}
			admitted = unknown < int64(policy.MaxUnknownPaidRequestsPerDay) && ledger.ActualCostMicroVND+ledger.ReservedCostMicroVND+event.ReservedCostMicroVND <= threshold
		}
		if !admitted {
			return nil
		}
		event.LedgerID = &ledger.ID
		if err := tx.Create(event).Error; err != nil {
			return err
		}
		if event.ReservedCostMicroVND > 0 {
			if err := tx.Model(&ledger).UpdateColumn("reserved_cost_micro_vnd", gorm.Expr("reserved_cost_micro_vnd + ?", event.ReservedCostMicroVND)).Error; err != nil {
				return err
			}
		}
		return nil
	})
	if err == nil && observeOnlyCapMissing {
		// Reported separately from err so the caller can distinguish "the
		// configuration is broken" from "the transaction failed" without having
		// to string-match a message.
		err = aierrors.ErrObserveOnlyMissingAttemptCap
	}
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
	db := r.GetDB(ctx)
	err := db.Model(&aiexecutor_entities.AIUsageEvent{}).
		Where("status IN ? AND unknown_expires_at <= ? AND status NOT IN ?",
			[]aiusageeventstatus.AIUsageEventStatus{aiusageeventstatus.Reserved, aiusageeventstatus.UnknownUsage},
			now,
			[]aiusageeventstatus.AIUsageEventStatus{aiusageeventstatus.Confirmed, aiusageeventstatus.Released, aiusageeventstatus.ExpiredUnverified}).
		Limit(limit).Pluck("request_id", &ids).Error
	if err != nil {
		return 0, err
	}
	if len(ids) == 0 {
		return 0, nil
	}
	result := db.Model(&aiexecutor_entities.AIUsageEvent{}).
		Where("request_id IN ?", ids).
		Where("status NOT IN ?",
			[]aiusageeventstatus.AIUsageEventStatus{aiusageeventstatus.Confirmed, aiusageeventstatus.Released, aiusageeventstatus.ExpiredUnverified}).
		Updates(map[string]any{
			"status":                  aiusageeventstatus.ExpiredUnverified,
			"error_code":              "usage_unverified_expired",
			"reserved_cost_micro_vnd": 0,
			"completed_at":            now,
			"updated_at":              now,
		})
	if err = result.Error; err != nil {
		return 0, err
	}
	n := result.RowsAffected
	if n > 0 {
		err = db.Exec(`UPDATE ai_usage_period_ledgers SET reserved_cost_micro_vnd = GREATEST(reserved_cost_micro_vnd - sub.reserved, 0), updated_at = ?
FROM (SELECT ledger_id, SUM(reserved_cost_micro_vnd) AS reserved FROM ai_usage_events WHERE request_id IN ? AND ledger_id IS NOT NULL GROUP BY ledger_id) AS sub
WHERE ai_usage_period_ledgers.id = sub.ledger_id`, now, ids).Error
	}
	return n, err
}

func (r *AICostRepository) GetAdminAIUsageStats(ctx context.Context, snapshotDate time.Time) (*dto.AIUsageAdminStatsDTO, error) {
	db := r.GetDB(ctx)

	// Daily stats cut on the Vietnam calendar day.
	startOfDay, endOfDay := timeutils.VNDayRange(snapshotDate)

	var reqCount int64
	_ = db.Model(&aiexecutor_entities.AIUsageEvent{}).
		Where("created_at >= ? AND created_at < ?", startOfDay, endOfDay).
		Count(&reqCount).Error

	// A failed request ends in one of the terminal error states: Released
	// (provider call failed), UnknownUsage (usage could not be attributed),
	// or ExpiredUnverified (reserved call never confirmed before expiry).
	aiErrorStatuses := []aiusageeventstatus.AIUsageEventStatus{
		aiusageeventstatus.Released,
		aiusageeventstatus.UnknownUsage,
		aiusageeventstatus.ExpiredUnverified,
	}

	var errCount int64
	_ = db.Model(&aiexecutor_entities.AIUsageEvent{}).
		Where("created_at >= ? AND created_at < ? AND status IN ?", startOfDay, endOfDay, aiErrorStatuses).
		Count(&errCount).Error

	var paidTokens int64
	_ = db.Model(&aiexecutor_entities.AIUsageEvent{}).
		Where("created_at >= ? AND created_at < ?", startOfDay, endOfDay).
		Select("COALESCE(SUM(prompt_tokens + output_tokens + thinking_tokens), 0)").Scan(&paidTokens).Error

	var actualCost int64
	_ = db.Model(&aiexecutor_entities.AIUsageEvent{}).
		Where("created_at >= ? AND created_at < ?", startOfDay, endOfDay).
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

func (r *AICostRepository) GetAIUsageByOperation(ctx context.Context, fromDate, toDate time.Time) ([]dto.AIUsageByOperationDTO, error) {
	query := r.GetDB(ctx).Model(&aiexecutor_entities.AIUsageEvent{}).
		Select("operation, logical_route, COUNT(*) AS request_count, " +
			"COALESCE(SUM(actual_cost_micro_vnd), 0) AS total_cost_micro_vnd, " +
			"COALESCE(SUM(prompt_tokens), 0) AS total_prompt_tokens, " +
			"COALESCE(SUM(output_tokens), 0) AS total_output_tokens")
	if !fromDate.IsZero() {
		query = query.Where("created_at >= ?", fromDate)
	}
	if !toDate.IsZero() {
		query = query.Where("created_at <= ?", toDate)
	}
	var rows []dto.AIUsageByOperationDTO
	err := query.Group("operation, logical_route").
		Order("request_count DESC").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	return rows, nil
}

func (r *AICostRepository) GetTotalAIRequestCount(ctx context.Context, fromDate, toDate time.Time) (int64, error) {
	query := r.GetDB(ctx).Model(&aiexecutor_entities.AIUsageEvent{})
	if !fromDate.IsZero() {
		query = query.Where("created_at >= ?", fromDate)
	}
	if !toDate.IsZero() {
		query = query.Where("created_at <= ?", toDate)
	}
	var count int64
	if err := query.Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

func (r *AICostRepository) GetAIErrorBreakdown(ctx context.Context) ([]dto.AIErrorBreakdownDTO, error) {
	var rows []dto.AIErrorBreakdownDTO
	err := r.GetDB(ctx).Model(&aiexecutor_entities.AIUsageEvent{}).
		Select("error_code, provider, model, COUNT(*) AS error_count").
		Where("status = ?", string(aiusageeventstatus.Released)).
		Group("error_code, provider, model").
		Order("error_count DESC").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	return rows, nil
}

func (r *AICostRepository) GetUsageByUser(ctx context.Context, userID uuid.UUID, fromDate, toDate *time.Time, page, limit int) ([]*aiexecutor_entities.AIUsageEvent, int64, error) {
	db := r.GetDB(ctx).Model(&aiexecutor_entities.AIUsageEvent{}).Where("user_id = ?", userID)
	if fromDate != nil && !fromDate.IsZero() {
		db = db.Where("created_at >= ?", *fromDate)
	}
	if toDate != nil && !toDate.IsZero() {
		db = db.Where("created_at <= ?", *toDate)
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

	var events []*aiexecutor_entities.AIUsageEvent
	err := db.Order("created_at DESC").Offset(offset).Limit(limit).Find(&events).Error
	if err != nil {
		return nil, 0, err
	}

	return events, total, nil
}

func (r *AICostRepository) ListGrantsByUser(ctx context.Context, userID uuid.UUID) ([]*aiexecutor_entities.AIPolicyGrant, error) {
	var grants []*aiexecutor_entities.AIPolicyGrant
	err := r.GetDB(ctx).
		Where("user_id = ?", userID).
		Order("created_at DESC").
		Find(&grants).Error
	if err != nil {
		return nil, err
	}
	return grants, nil
}

func (r *AICostRepository) GetPolicyByID(ctx context.Context, policyID uuid.UUID) (*aiexecutor_entities.AICostPolicy, error) {
	return r.getPolicyBy(ctx, "id = ?", policyID)
}

func (r *AICostRepository) getPolicyBy(ctx context.Context, condition string, args ...any) (*aiexecutor_entities.AICostPolicy, error) {
	var policy aiexecutor_entities.AICostPolicy
	// GetQueryWithPreload is mandatory here, not a convenience. This policy is
	// the source of the entitlement snapshot, so a missing Operations preload
	// would silently produce a snapshot with zero operations and an
	// entitlement set that fails the "all four operations present" check
	// (FR-029b) on every single grant.
	queryArgs := append([]any{condition}, args...)
	if err := r.GetQueryWithPreload(ctx).First(&policy, queryArgs...).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &policy, nil
}
