package grant

import (
	"context"
	"time"

	aiexecutor_entities "smart-wardrobe-be/services/ai/internal/modules/aiexecutor/domain/entities"

	"go.uber.org/zap"
)

// clockSource names which column the stale floor was taken from. It is returned
// alongside the floor so each caller records the trace that matches the level it
// landed on: level 2 records FR-003, level 3 records FR-018, and level 1 records
// nothing.
type clockSource int

const (
	// clockOccurredAt: occurred_at read from the policy snapshot's envelope. This
	// is the BUSINESS time of the change that produced the stored grant, so it is
	// the preferred clock and needs no warning.
	clockOccurredAt clockSource = iota
	// clockCreatedAt: the row's own created_at, used when the snapshot carries no
	// readable occurred_at. It is the time the row was WRITTEN, not the time the
	// change happened, so using it degrades the clock (FR-003).
	clockCreatedAt
	// clockEffectiveFrom: the row's effective_from, the last resort when neither
	// occurred_at nor created_at is usable. It carries the CHECK constraint and is
	// the weakest clock, so it always warns (FR-018).
	clockEffectiveFrom
)

// staleFloor derives the ordering floor from one already-read grant, in three
// levels of decreasing trust (FR-001, FR-004a):
//
//  1. occurred_at, read from the policy snapshot's envelope via ExtractOccurredAt.
//     This is the business time of the change and the preferred clock; using it
//     logs nothing. The order matters when the queue is delayed: a later event can
//     occur after an earlier one yet be written before the earlier row's
//     created_at, so a created_at floor would refuse it wrongly (FR-001).
//  2. created_at, when it is readable (non-zero) and not in the future relative to
//     the application clock. It is the time the row was written, which lags the
//     business time, so every use is recorded as FR-003.
//  3. effective_from. It is the last resort because created_at is either zero or
//     in the future - a clock skew or a hand-edited value could otherwise block
//     every later grant for the account - so the floor falls back to a column that
//     carries the CHECK constraint (ADR 014 §2, R-003). Every use is recorded as
//     FR-018.
//
// It is PURE on purpose: it takes a grant and a clock, reads no database and takes
// no context. That is what lets both ordering checks share it - the create path's
// resolveGrantWindow and the unpriceable-downgrade path's rejectStaleEvent - so
// neither can keep the old "refuse when the snapshot envelope is unreadable"
// behaviour (FR-019).
//
// The returned clockSource tells the caller which warning to record, because the
// two fallbacks are different degrees of degradation and an operator must be able
// to count each separately.
//
// This function answers ONLY "is this event newer than the last change". The
// second floor - max(staleFloor, prior.EffectiveFrom) - lives in
// resolveGrantWindow, because a `future` grant has created_at in the past and
// effective_from in the future, and a floor taken from created_at alone cannot see
// it (FR-004a).
func staleFloor(grant *aiexecutor_entities.AIPolicyGrant, now time.Time) (time.Time, clockSource) {
	if grant == nil {
		return time.Time{}, clockEffectiveFrom
	}
	if occurredAt, ok := ExtractOccurredAt(grant); ok {
		return occurredAt, clockOccurredAt
	}
	if !grant.CreatedAt.IsZero() && !grant.CreatedAt.After(now) {
		return grant.CreatedAt, clockCreatedAt
	}
	return grant.EffectiveFrom, clockEffectiveFrom
}

// warnOccurredAtFallback records FR-003 when the ordering floor had to fall back
// from the snapshot's occurred_at to the row's created_at.
//
// It carries user_id AND grant_id deliberately, the same way warnStaleFloorFallback
// does: an operator reading the line can tell which account is running on the
// weaker clock and can count how many grants are in that state. The message is
// deliberately DIFFERENT from the FR-018 one, because an operator has to be able to
// tell the two degrees of degradation apart.
func (uc *GrantUseCase) warnOccurredAtFallback(grant *aiexecutor_entities.AIPolicyGrant) {
	uc.logger.Warn("policy snapshot has no readable occurred_at; falling back to created_at for ordering",
		zap.String("user_id", grant.UserID.String()),
		zap.String("grant_id", grant.ID.String()),
	)
}

// warnStaleFloorFallback records FR-018 once per time the stale floor had to fall
// back to effective_from, because created_at was zero or in the future.
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

// warnFloorFallback records the warning that matches the chosen floor source.
// clockOccurredAt is the healthy case and records nothing, which is the point of
// returning the source rather than a bool.
func (uc *GrantUseCase) warnFloorFallback(grant *aiexecutor_entities.AIPolicyGrant, source clockSource) {
	switch source {
	case clockCreatedAt:
		uc.warnOccurredAtFallback(grant)
	case clockEffectiveFrom:
		uc.warnStaleFloorFallback(grant)
	}
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
//     change". It has THREE levels - occurred_at from the snapshot, then
//     created_at, then effective_from - and FR-002 refuses on it. It is the only
//     floor the unpriceable-downgrade path needs, and each fallback level records
//     its own warning (FR-003, FR-018).
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

	stale, source := staleFloor(prior, time.Now())
	uc.warnFloorFallback(prior, source)
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
