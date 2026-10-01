package persistence

import (
	"context"
	"errors"
	"time"

	aiexecutor_entities "smart-wardrobe-be/services/ai/internal/modules/aiexecutor/domain/entities"
	"smart-wardrobe-be/shared/domain/constants/aipolicygrantstatus"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Grant lifecycle operations on aiexecutor.ai_policy_grants.
//
// They live here, next to the rest of the cost-policy access, because they are
// the writes that the plan-changed handler performs. Every one of them resolves
// the handle through GetDB, so a transaction injected into ctx by
// IUnitOfWork.Execute is reused: closing the previous grant and creating the new
// one have to be atomic, and two independent handles would make that impossible.

// LockGrantsForAccount takes a transaction-scoped advisory lock keyed on the
// account, so grant writes for one account are serialised (024 T163, T164).
//
// An advisory lock rather than `SELECT ... FOR UPDATE` because the case that
// matters most is the first grant: a brand-new account has no rows, so a row lock
// locks nothing and two concurrent events for the same account both proceed. That is
// how two `active` grants appeared, and how a redelivered event created a second
// grant for a `source_event_id` the first one had already written.
//
// `hashtextextended` maps the account UUID onto a 64-bit key. The hash is not a
// security boundary; collisions only mean two unrelated accounts briefly serialise
// against each other, which costs a few milliseconds and is not worth a lookup
// table.
//
// The lock is released at COMMIT or ROLLBACK, so it cannot leak beyond the
// transaction the caller opened - which is why the handler must take it INSIDE
// `IUnitOfWork.Execute` and re-run its duplicate check afterwards.
func (r *AICostRepository) LockGrantsForAccount(ctx context.Context, userID uuid.UUID) error {
	return r.GetDB(ctx).Exec(
		"SELECT pg_advisory_xact_lock(hashtextextended(?, 0))",
		"ai_policy_grants:"+userID.String(),
	).Error
}

// CloseActiveGrants closes every grant currently in force for an account.
func (r *AICostRepository) CloseActiveGrants(ctx context.Context, userID uuid.UUID, cutoff, at time.Time) error {
	return r.GetDB(ctx).
		Model(&aiexecutor_entities.AIPolicyGrant{}).
		Where("user_id = ? AND status = ? AND effective_from <= ?", userID, aipolicygrantstatus.Active, cutoff).
		Updates(map[string]any{
			"effective_to": cutoff,
			"status":       string(aipolicygrantstatus.Closed),
			"updated_at":   at,
		}).Error
}

// DeleteFutureGrants removes pending rows for an account.
func (r *AICostRepository) DeleteFutureGrants(ctx context.Context, userID uuid.UUID) error {
	return r.GetDB(ctx).
		Where("user_id = ? AND status = ?", userID, aipolicygrantstatus.Future).
		Delete(&aiexecutor_entities.AIPolicyGrant{}).Error
}

// TruncateGrantsBefore moves the end of every grant that spans cutoff.
//
// A renewal does not close the previous grant at "now": the account paid for one
// continuous stretch of time, so the previous row is truncated at the exact
// point the new one starts. Closing it at "now" instead would leave a gap, and an
// account in that gap is refused by every AI call.
//
// The row is still marked closed, because its window has ended. Leaving the
// status as active would make "a live grant whose period has already passed"
// look like a real state, and that is one of the six things the verification
// script exists to catch.
func (r *AICostRepository) TruncateGrantsBefore(ctx context.Context, userID uuid.UUID, cutoff, at time.Time) error {
	return r.GetDB(ctx).
		Model(&aiexecutor_entities.AIPolicyGrant{}).
		Where("user_id = ? AND status IN ? AND effective_from < ? AND (effective_to IS NULL OR effective_to > ?)",
			userID, []aipolicygrantstatus.AIPolicyGrantStatus{aipolicygrantstatus.Active, aipolicygrantstatus.Future}, cutoff, cutoff).
		Updates(map[string]any{
			"effective_to": cutoff,
			"status":       string(aipolicygrantstatus.Closed),
			"updated_at":   at,
		}).Error
}

// GetCurrentGrant returns the grant currently in force, if any.
func (r *AICostRepository) GetCurrentGrant(ctx context.Context, userID uuid.UUID) (*aiexecutor_entities.AIPolicyGrant, error) {
	var grant aiexecutor_entities.AIPolicyGrant
	now := time.Now()
	err := r.GetDB(ctx).
		Where("user_id = ? AND effective_from <= ? AND (effective_to IS NULL OR effective_to > ?)", userID, now, now).
		Order("effective_from DESC").
		First(&grant).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &grant, nil
}

// GetNewestGrantForOrdering returns the most recently scheduled grant for an account,
// regardless of status or window.
//
// Contrast GetCurrentGrant above, which returns what applies right now. Ordering has
// to be decided against the newest thing ever applied, or a `future` grant is
// invisible to it and an older event slips in behind it. See the interface for the
// full failure this produced.
func (r *AICostRepository) GetNewestGrantForOrdering(ctx context.Context, userID uuid.UUID) (*aiexecutor_entities.AIPolicyGrant, error) {
	var grant aiexecutor_entities.AIPolicyGrant
	// No status filter and no window predicate, deliberately. `closed` rows are the
	// newest record of what the account was told, and ignoring them would let a
	// delayed event re-apply a revoked entitlement.
	err := r.GetDB(ctx).
		Where("user_id = ?", userID).
		Order("effective_from DESC").
		First(&grant).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &grant, nil
}

// GetGrantBySourceEventID is the duplicate-key lookup (FR-033).
func (r *AICostRepository) GetGrantBySourceEventID(ctx context.Context, sourceEventID uuid.UUID) (*aiexecutor_entities.AIPolicyGrant, error) {
	var grant aiexecutor_entities.AIPolicyGrant
	err := r.GetDB(ctx).Where("source_event_id = ?", sourceEventID).First(&grant).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &grant, nil
}
