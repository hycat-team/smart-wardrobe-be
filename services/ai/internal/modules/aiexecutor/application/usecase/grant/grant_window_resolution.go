package grant

import (
	"context"
	"time"

	aiexecutor_entities "smart-wardrobe-be/services/ai/internal/modules/aiexecutor/domain/entities"

	"go.uber.org/zap"
)

// staleFloor derives the ordering floor from one already-read grant.
//
// It is PURE on purpose: it takes a grant and a clock, reads no database and takes
// no context. That is what lets both ordering checks share it - the create path's
// resolveGrantWindow and the unpriceable-downgrade path's rejectStaleEvent - so
// neither can keep the old "refuse when the snapshot envelope is unreadable"
// behaviour (FR-019).
//
// It reads created_at, not occurred_at from the policy snapshot:
//   - created_at is the time the record was written, and it is what FR-001 names
//     as the fallback clock. The snapshot envelope is unreadable on rows written
//     before the envelope existed, and reading it was the production lock-up.
//   - When created_at is zero OR in the future relative to the application clock,
//     the column is not trustworthy - a clock skew or a hand-edited value could
//     otherwise block every later grant for the account - so the floor falls back
//     to effective_from, which carries the CHECK constraint (ADR 014 §2, R-003).
//
// The bool reports whether the fallback happened. The caller MUST record FR-018
// (user_id + grant_id) whenever it is true, because that branch is the weak clock
// an operator needs to be able to count.
//
// This function answers ONLY "is this event newer than the last write". The second
// floor - max(staleFloor, prior.EffectiveFrom) - lives in resolveGrantWindow,
// because a `future` grant has created_at in the past and effective_from in the
// future, and a floor taken from created_at alone cannot see it (FR-004a).
func staleFloor(grant *aiexecutor_entities.AIPolicyGrant, now time.Time) (time.Time, bool) {
	if grant == nil {
		return time.Time{}, false
	}
	floor := grant.CreatedAt
	if floor.IsZero() || floor.After(now) {
		return grant.EffectiveFrom, true
	}
	return floor, false
}

// warnStaleFloorFallback records FR-018 once per time the stale floor had to fall
// back to effective_from.
//
// It carries user_id AND grant_id deliberately: an operator reading the line can
// tell which account is running on the weak clock, and can count how many grants
// are in that state. A line carrying only the reason would force a second lookup
// at exactly the moment nobody is watching.
func (uc *GrantUseCase) warnStaleFloorFallback(grant *aiexecutor_entities.AIPolicyGrant) {
	uc.logger.Warn("grant created_at is unusable for ordering; falling back to effective_from",
		zap.String("user_id", grant.UserID.String()),
		zap.String("grant_id", grant.ID.String()),
	)
}

// resolveGrantWindow settles the window the new grant may actually occupy against
// the newest stored grant for the account.
//
// It is NOT pure: it reads GetNewestGrantForOrdering, so it must run INSIDE the
// transaction, AFTER LockGrantsForAccount. Running it before the lock - or keeping
// its result outside the transaction - would let two events decide from two
// different states and write two overlapping active windows (FR-005, contract §2).
//
// It has TWO floors, and the distinction is the whole point (FR-004a):
//
//   - the STALE floor (staleFloor) answers "is this event newer than the last
//     write". FR-002 refuses on it. It is the only floor the unpriceable-downgrade
//     path needs.
//   - the ORDER floor is max(staleFloor, prior.EffectiveFrom). FR-004/FR-006 shift
//     against it, so the new window can never start before a stored `future`
//     window. A floor taken from created_at alone cannot see that row, because its
//     created_at is in the past while its effective_from is not.
//
// The returned EffectiveFrom must be used for BOTH the new row and the truncation
// point; recomputing either is the documented way to leave a 1µs overlap or gap
// (contract §3, ADR 014 §2.2). The window is otherwise the event's own window:
// when it is already past the order floor, the output is byte-for-byte
// EventGrantWindow(event) and nothing is shifted (FR-007).
func (uc *GrantUseCase) resolveGrantWindow(ctx context.Context, event *PlanChangedEvent) (GrantWindow, error) {
	window := EventGrantWindow(event)

	prior, err := uc.aiCostRepo.GetNewestGrantForOrdering(ctx, event.UserID)
	if err != nil {
		// The read failing says nothing about the event's ordering, so it is a
		// retryable failure, never a stale refusal. Same classification the old
		// ordering check used for the same read.
		uc.metrics.RecordRejection(ReasonTransientFailure, event.UserID.String())
		return GrantWindow{}, uc.transient(err, ctx, event, "read newest grant for the window resolution")
	}
	if prior == nil {
		// No prior record means no floor to shift against. The event's window is
		// returned untouched.
		return window, nil
	}

	stale, fellBack := staleFloor(prior, time.Now())
	if fellBack {
		uc.warnStaleFloorFallback(prior)
	}
	if !event.OccurredAt.After(stale) {
		// FR-002: an event that is not strictly newer than the last write must not
		// resurrect an entitlement that was already revoked. "Newer" is strict, so
		// equality is refused too.
		uc.metrics.RecordRejection(ReasonStaleEvent, event.UserID.String())
		return GrantWindow{}, uc.reject(ReasonStaleEvent, event.UserID.String(),
			"Sự kiện đã cũ hơn hoặc bằng thời điểm của bản ghi mức quyền đang hiệu lực.")
	}

	orderFloor := stale
	if prior.EffectiveFrom.After(orderFloor) {
		orderFloor = prior.EffectiveFrom
	}

	if !window.EffectiveFrom.After(orderFloor) {
		shifted := orderFloor.Add(time.Microsecond)
		if window.EffectiveTo != nil && !shifted.Before(*window.EffectiveTo) {
			// FR-017: shifting produced an empty window. Refuse permanently rather
			// than write a row with no usable interval - the CHECK would reject it
			// anyway, and a customer cannot be given a period that does not exist.
			uc.metrics.RecordRejection(ReasonStaleEvent, event.UserID.String())
			return GrantWindow{}, uc.reject(ReasonStaleEvent, event.UserID.String(),
				"Cửa sổ mức quyền mới rỗng sau khi dịch mốc bắt đầu; sự kiện không thể áp dụng.")
		}
		window.EffectiveFrom = shifted
	}
	return window, nil
}
