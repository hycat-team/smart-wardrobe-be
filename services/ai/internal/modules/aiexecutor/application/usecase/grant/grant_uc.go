package grant

import (
	"context"
	"errors"
	"time"

	aiexecutor_dto "smart-wardrobe-be/services/ai/dto/aiexecutor"
	uc_interfaces "smart-wardrobe-be/services/ai/internal/modules/aiexecutor/application/interface/usecase"
	aiexecutor_entities "smart-wardrobe-be/services/ai/internal/modules/aiexecutor/domain/entities"
	"smart-wardrobe-be/services/ai/internal/modules/aiexecutor/domain/repositories"
	sharedevent "smart-wardrobe-be/shared/application/event"
	"smart-wardrobe-be/shared/domain/constants/subscriptioneventtype"
	shared_repos "smart-wardrobe-be/shared/domain/repositories"
	"smart-wardrobe-be/shared/pkg/logger"

	"go.uber.org/zap"

	"github.com/google/uuid"
)

// Reason codes recorded when an event cannot be applied. Stable strings: they
// are the keys of the observability counters, and an operator has to be able to
// alert on one of them.
//
// 024 T159 renamed `plan_not_in_catalog` to `ai_cost_policy_not_found`.
//
// The old name was wrong about what had been checked. There is no catalogue lookup
// here at all: the branch fires when `GetPolicyByID` returns nil for the policy id the
// event carried. Nothing ever looked a plan up, because the AI service must not read
// the Account service's tables (constitution §I, single owner per data domain) - and
// adding such a query to "fix" the name would have been the more serious defect of
// the two.
//
// The cost matters for more than tidiness. A name that refers to a table nobody reads
// is expensive when it is false: an operator investigating `plan_not_in_catalog` would
// go check the plan catalogue, find a perfectly valid plan row, and have nothing left
// to look at. The wording of the message matters for the same reason - it must not
// send them to the catalogue either.
//
// The rename is deliberate rather than a silent string swap: these are metric keys and
// alert names, so a dashboard or alert built on the old string has to be updated, and
// the tests below pin both the constant and the messages so the two cannot drift.
const (
	// ReasonAICostPolicyNotFound: the event named an AI cost policy that does not
	// exist in aiexecutor.ai_cost_policies. No entitlement is written - guessing a
	// policy would hand out something nobody priced.
	ReasonAICostPolicyNotFound  = "ai_cost_policy_not_found"
	ReasonStaleEvent            = "stale_event"
	ReasonPolicySnapshotUnavail = "policy_snapshot_unavailable"
	ReasonDuplicateSourceEvent  = "duplicate_source_event"
	ReasonTransientFailure      = "transient_failure"
)

// RejectionError is an event that was understood but not applied.
//
// 024 T152: the comment here used to say "the message is acknowledged", and that
// was wrong. Nothing ever acknowledged it. `RejectionError` had no `Unwrap`, so it
// matched neither broker marker and fell through to the `default` branch of
// `startConsumerLoop`, which NACKs without requeue - the message was dead-lettered
// every time. The outcome happened to be the intended one, but it was accidental:
// a requeue-only change anywhere in that switch would have started spinning
// poison messages forever.
//
// Rejections are now returned as `sharedevent.PermanentError` wrapping this type,
// so the intent is stated where the transport reads it. Note this type is still the
// reason carrier - it is what the metrics and the log describe.
type RejectionError struct {
	Reason  string
	Message string
}

func (e *RejectionError) Error() string { return e.Message }

type GrantUseCase struct {
	aiCostRepo repositories.IAICostRepository
	uow        shared_repos.IUnitOfWork
	logger     logger.Interface
	metrics    *GrantMetrics
}

func NewGrantUseCase(
	aiCostRepo repositories.IAICostRepository,
	uow shared_repos.IUnitOfWork,
	log logger.Interface,
	metrics *GrantMetrics,
) uc_interfaces.IGrantUseCase {
	return &GrantUseCase{
		aiCostRepo: aiCostRepo,
		uow:        uow,
		logger:     log,
		metrics:    metrics,
	}
}

// HandlePlanChanged applies one subscription.plan.changed event.
//
// The previous implementation branched on raw strings, returned nil for
// "EXTENDED" and "RENEWED", fell through to a default branch that created a
// grant for every unrecognised type, and closed the previous grant and created
// the new one as two separate database writes. So a crash between the two left
// the account with no entitlement at all, a renewal created nothing, and
// toggling auto-renew revoked and re-granted. All three are fixed here.
//
// 024 T152: every path that leaves this method now says, in the broker's own
// vocabulary, whether redelivering the message could plausibly change the answer.
//
// The distinction is not cosmetic. `startConsumerLoop` (`rabbitmq.go:120-129`)
// requeues ONLY a `RetryableError` and dead-letters everything else. A repository
// error used to be returned raw, which is neither marker, so it fell to `default`
// and was dead-lettered on the FIRST attempt. A two-second database blip therefore
// destroyed the event for a paying customer: the grant was never written, the
// queue never retried, and the account silently lost its entitlement. The quorum
// queue's `x-delivery-limit: 5`, configured specifically so transient failures
// survive long enough to be retried, never fired.
//
// This method is therefore the boundary where an error becomes a transport
// decision. It is deliberately the last thing before returning rather than
// something buried in the repository layer: only this method knows whether a
// failure is about the message or about the world.
func (uc *GrantUseCase) HandlePlanChanged(ctx context.Context, payload aiexecutor_dto.PlanChangedEventPayload) error {
	event, err := ParsePlanChangedPayload(uc.logger, payload)
	if err != nil {
		var parseErr *ParseError
		if errors.As(err, &parseErr) {
			uc.metrics.RecordRejection(parseErr.Reason, payload.UserID)
			uc.logger.Warn("rejected subscription.plan.changed event",
				zap.String("reason", parseErr.Reason),
				zap.String("message", parseErr.Message),
				zap.String("user_id", payload.UserID),
				zap.String("source_event_id", payload.SourceEventID),
				zap.String("correlation_id", sharedevent.CorrelationIdFromContext(ctx)),
			)
		}
		// A payload that failed validation will fail identically on every
		// redelivery, so it is refused outright rather than retried five times.
		return sharedevent.PermanentError{Err: err}
	}

	// A duplicate is not a failure. The outbox guarantees at-least-once
	// delivery, so redelivery is the normal case, and returning an error here
	// would burn the delivery limit on a message that was handled correctly.
	//
	// 024 T163: this check is only an OPTIMISATION now. It stays outside the
	// transaction because it is the common case - a redelivery should be acked
	// without taking a lock or opening a write transaction - but it cannot be the
	// thing that guarantees FR-033, because two deliveries of the same event can
	// both read "absent" and then both insert. The authoritative re-check runs
	// inside the transaction, after the per-account lock.
	existing, err := uc.aiCostRepo.GetGrantBySourceEventID(ctx, event.SourceEventID)
	if err != nil {
		uc.metrics.RecordRejection(ReasonTransientFailure, event.UserID.String())
		return uc.transient(err, ctx, event, "read existing grant")
	}
	if existing != nil {
		uc.recordDuplicateSkipped(event)
		return nil
	}

	if !CreatesNewGrant(event.EventType) {
		// A revoke, or a change that moves no entitlement at all. Still routed
		// through the transaction so the close and the delete are atomic.
		return uc.applyRevocation(ctx, event)
	}

	policy, err := uc.aiCostRepo.GetPolicyByID(ctx, event.AICostPolicy)
	if err != nil {
		uc.metrics.RecordRejection(ReasonTransientFailure, event.UserID.String())
		return uc.transient(err, ctx, event, "read AI cost policy")
	}
	if policy == nil {
		// 024 T159: this used to report `plan_not_in_catalog`, with a comment
		// describing it as "FR-036's catalogue check, which looks a plan up BY id".
		// No such lookup exists - and adding one would have violated constitution §I,
		// since the AI service must not read the Account service's tables. The only
		// thing actually consulted is this service's own ai_cost_policies, which is
		// why the reason and the message now say so.
		//
		// It is refused rather than skipped: a missing policy is a configuration
		// fault, and guessing one would hand out entitlements nobody priced.
		//
		// 024 T204: the refusal alone is not the whole answer, because this branch
		// sits in front of `retirePreviousGrants`. See `retireOnUnpriceableDowngrade`.
		if err := uc.retireOnUnpriceableDowngrade(ctx, event); err != nil {
			return err
		}
		uc.metrics.RecordRejection(ReasonAICostPolicyNotFound, event.UserID.String())
		return uc.reject(ReasonAICostPolicyNotFound, event.UserID.String(), msgAICostPolicyNotFound)
	}

	snapshot, err := BuildPolicySnapshot(BuildPolicySnapshotInput{
		Event:      event,
		Policy:     policy,
		OccurredAt: event.OccurredAt,
	})
	if err != nil {
		uc.metrics.RecordRejection(ReasonPolicySnapshotUnavail, event.UserID.String())
		return uc.reject(ReasonPolicySnapshotUnavail, event.UserID.String(),
			"Không dựng được ảnh chụp chính sách từ cấu hình hiện hành.")
	}
	document, err := snapshot.Marshal()
	if err != nil {
		uc.metrics.RecordRejection(ReasonPolicySnapshotUnavail, event.UserID.String())
		return uc.reject(ReasonPolicySnapshotUnavail, event.UserID.String(),
			"Không mã hoá được ảnh chụp chính sách.")
	}

	grant := uc.buildGrant(event, document)
	if grant == nil {
		// 024 T137: buildGrant refuses to assemble a row without a plan id (FR-023).
		// Recorded as a rejection rather than returned as a bare error, so the
		// event is traceable to a reason and the account is not silently left
		// without an entitlement.
		uc.metrics.RecordRejection(ReasonMissingRequiredField, event.UserID.String())
		return uc.reject(ReasonMissingRequiredField, event.UserID.String(),
			"Sự kiện không có mã gói đích nên không thể dựng bản ghi quyền.")
	}

	// Closing the previous grant and creating the new one MUST be one
	// transaction. Two writes mean a window in which the account holds no
	// entitlement at all — a user who just paid, being refused by the AI service.
	//
	// 024 T163 + T164: the transaction opens with a per-account lock, and the
	// duplicate check is REPEATED after it. Those two lines are what make FR-033 and
	// FR-034 hold under concurrency, and they belong together because they defend
	// the same thing from two sides:
	//
	//   - the lock serialises grant writes for one account, so two events cannot
	//     both decide they are the newest and both write;
	//   - the re-check catches the redelivery that the optimistic pre-check let
	//     through, so a duplicate event produces no second grant.
	//
	// Neither is sufficient alone, and both are needed. FR-034 rules out a unique
	// constraint at the data layer - an account may hold several `active` rows as
	// long as they do not overlap - so the invariant cannot be pushed down into the
	// schema.
	alreadyApplied := false
	err = uc.uow.Execute(ctx, func(txCtx context.Context) error {
		if err := uc.aiCostRepo.LockGrantsForAccount(txCtx, event.UserID); err != nil {
			return err
		}
		// Re-check under the lock. The pre-check above saw this before the lock was
		// held, so it said nothing about what another delivery may have written in
		// the meantime.
		duplicate, err := uc.aiCostRepo.GetGrantBySourceEventID(txCtx, event.SourceEventID)
		if err != nil {
			return err
		}
		if duplicate != nil {
			alreadyApplied = true
			return nil
		}
		// 024 T163 + T164: the out-of-order check runs HERE, under the account lock,
		// and not before the transaction. It used to run earlier, which is what let
		// two events for one account each decide they were the newest against a state
		// the other had already replaced.
		//
		// The failure it produced is not a duplicate row, it is two ACTIVE grants: the
		// later event commits first, then the earlier one takes the lock and runs
		// CloseActiveGrants with a cutoff taken from ITS OWN earlier timestamp. That
		// cutoff is before the newer grant's effective_from, so the newer grant is not
		// closed and the older one is inserted alongside it. The account then holds
		// two overlapping entitlements and the resolver cannot say which one applies.
		//
		// Deciding under the lock costs one read that was already being done, and it
		// is the only place the answer is true.
		//
		// 027 US1: resolveGrantWindow REPLACES rejectStaleEvent on this path. It runs
		// here for the same reason - under the lock - and it returns the ONE start
		// instant that both the new row and the truncation point use. Computing it
		// twice is the documented way to leave a 1µs overlap or a 1µs gap (contract
		// §3, ADR 014 §2.2). rejectStaleEvent is deliberately NOT kept alongside it:
		// that would run the ordering test twice against two different clocks, and
		// the old one still refuses permanently when the snapshot envelope is
		// unreadable - the very lock-up 027 removes.
		window, err := uc.resolveGrantWindow(txCtx, event)
		if err != nil {
			return err
		}
		// Only EffectiveFrom depends on stored state, so it is the only field
		// re-assigned here. buildGrant assembled the row OUTSIDE the transaction from
		// static event and policy data, which is still correct.
		grant.EffectiveFrom = window.EffectiveFrom
		if err := uc.retirePreviousGrants(txCtx, event, window.EffectiveFrom); err != nil {
			return err
		}
		return uc.aiCostRepo.CreateGrant(txCtx, grant)
	})
	if err != nil {
		// A stale refusal is a decision, not a failure, so it keeps its own
		// classification instead of being flattened into "transient" and requeued.
		// Requeueing it would redeliver an event that can never become current.
		var rejection *RejectionError
		if errors.As(err, &rejection) {
			return err
		}
		uc.metrics.RecordRejection(ReasonTransientFailure, event.UserID.String())
		return uc.transient(err, ctx, event, "write grant in transaction")
	}
	if alreadyApplied {
		// The loser of the race writes nothing at all, which is what FR-033 requires
		// ("MUST NOT ghi tác dụng phụ"). Nothing was retired and nothing was created.
		uc.recordDuplicateSkipped(event)
	}
	return nil
}

// recordDuplicateSkipped reports an event that was already applied (024 T163).
//
// Shared by the cheap pre-check and the authoritative re-check under the lock, so
// the counter and the log line are identical whichever one fired - otherwise a
// redelivery would look like two different outcomes depending on timing.
func (uc *GrantUseCase) recordDuplicateSkipped(event *PlanChangedEvent) {
	uc.metrics.RecordRejection(ReasonDuplicateSourceEvent, event.UserID.String())
	uc.logger.Info("subscription.plan.changed event already applied; skipping",
		zap.String("source_event_id", event.SourceEventID.String()),
		zap.String("user_id", event.UserID.String()),
	)
}

// transient marks a failure as worth retrying and records what it was.
//
// 024 T152. A `RetryableError` is the only thing that makes `startConsumerLoop`
// requeue instead of dead-letter, so without this wrapper a momentary database
// failure ends the customer's entitlement permanently on the first delivery. The
// underlying error is preserved through `Unwrap`, so the repository detail still
// reaches the log and `errors.Is`/`errors.As` keep working further down.
func (uc *GrantUseCase) transient(err error, ctx context.Context, event *PlanChangedEvent, operation string) error {
	uc.logger.Error("transient failure applying subscription.plan.changed; the message will be requeued",
		zap.String("operation", operation),
		zap.String("user_id", event.UserID.String()),
		zap.String("source_event_id", event.SourceEventID.String()),
		zap.String("event_type", string(event.EventType)),
		zap.String("correlation_id", sharedevent.CorrelationIdFromContext(ctx)),
		zap.Error(err),
	)
	return sharedevent.RetryableError{Err: err}
}

// reject marks a failure as one that redelivery cannot fix.
//
// 024 T152. This is the explicit counterpart to `transient`. `RejectionError` has no
// `Unwrap`, so it never matched either broker marker and fell through to `default` -
// which happened to dead-letter it, the right outcome for the wrong reason. Stating
// it means the behaviour survives a change to that switch, and it stops the
// consumer's log claiming a message "will retry" when it is being discarded.
func (uc *GrantUseCase) reject(reason, userID, message string) error {
	return sharedevent.PermanentError{Err: &RejectionError{
		Reason:  reason,
		Message: message,
	}}
}

const (
	// 024 T172: the refusal names the table that is actually consulted — this
	// service's own cost policies — and not a "catalogue".
	//
	// "Danh mục" (catalogue) is the wrong word for the same reason it was wrong in the
	// reason code T159 renamed. An operator reading "không tồn tại trong danh mục" goes
	// to look for a plan catalogue, finds a perfectly valid plan row there, and has
	// nothing left to look at. The one thing worth naming is the table to go and
	// check: ai_cost_policies, by id.
	//
	// It is a constant so the test asserts the SAME string production uses. The test
	// used to carry its own copy of this sentence, which is precisely how the two
	// drift apart — and they had.
	msgAICostPolicyNotFound = "Chính sách phí AI không tồn tại trong bảng chính sách phí của dịch vụ AI."
)

// rejectStaleEvent refuses an event that is not newer than the account's newest
// stored grant.
//
// Without this, a renewal that arrives after a downgrade — a normal out-of-order
// delivery — would resurrect a revoked Premium entitlement.
//
// 024 T164: this reads the account's NEWEST grant, not the one in force. The two are
// different questions and answering the wrong one is what allowed two overlapping
// active entitlements to be written. The caller must hold the account lock, because
// the answer it compares against has to be the one no other writer can change.
//
// 027 FR-001/FR-004a/FR-019: the ordering floor comes from the grant's OWN
// created_at - falling back to effective_from when that column is zero or in the
// future - and NOT from occurred_at in the policy snapshot. The snapshot envelope
// is unreadable on rows written before it existed, and reading it made this check
// refuse those rows permanently: the lock-up 027 exists to remove. It therefore no
// longer calls ExtractOccurredAt at all. The shared `staleFloor` is the single
// definition of the floor, so this path and the create path cannot drift apart.
//
// This is now the only caller of the ordering check besides resolveGrantWindow. It
// stays for the unpriceable-downgrade path, which retires a grant without creating
// a row and so has no window to shift.
func (uc *GrantUseCase) rejectStaleEvent(ctx context.Context, event *PlanChangedEvent) error {
	current, err := uc.aiCostRepo.GetNewestGrantForOrdering(ctx, event.UserID)
	if err != nil {
		// 024 T152: this read failing is not a statement about the event's ordering,
		// so it must not reach the broker as a bare error and be dead-lettered. A
		// requeue gets the ordering check another chance once the database recovers.
		uc.metrics.RecordRejection(ReasonTransientFailure, event.UserID.String())
		return uc.transient(err, ctx, event, "read newest grant for the stale-event check")
	}
	if current == nil {
		return nil
	}
	floor, fellBack := staleFloor(current, time.Now())
	if fellBack {
		uc.warnStaleFloorFallback(current)
	}
	if !event.OccurredAt.After(floor) {
		uc.metrics.RecordRejection(ReasonStaleEvent, event.UserID.String())
		return uc.reject(ReasonStaleEvent, event.UserID.String(),
			"Sự kiện đã cũ hơn hoặc bằng thời điểm của bản ghi mức quyền đang hiệu lực.")
	}
	return nil
}

func (uc *GrantUseCase) applyRevocation(ctx context.Context, event *PlanChangedEvent) error {
	if event.EventType == subscriptioneventtype.AUTO_RENEW_UPDATED {
		// Published, received, and deliberately a no-op: toggling auto-renew does
		// not change the plan. The receiver is the right place for that decision,
		// which is why the sender publishes it at all.
		return nil
	}
	// 024 T164: the same per-account lock as the grant path. A revoke racing a grant
	// for one account would otherwise interleave - the revoke could close the row the
	// grant had just inserted, or the grant could close the row the revoke had just
	// inserted, and the account ends up with neither.
	err := uc.uow.Execute(ctx, func(txCtx context.Context) error {
		if err := uc.aiCostRepo.LockGrantsForAccount(txCtx, event.UserID); err != nil {
			return err
		}
		// 027 T010c: the revocation path never creates a row, so it has no resolved
		// window to pass. It hands retirePreviousGrants the window the EVENT declares,
		// exactly as before - the CloseActiveGrants branch ignores the cutoff anyway.
		return uc.retirePreviousGrants(txCtx, event, EventGrantWindow(event).EffectiveFrom)
	})
	if err != nil {
		uc.metrics.RecordRejection(ReasonTransientFailure, event.UserID.String())
		return uc.transient(err, ctx, event, "write revocation in transaction")
	}
	return nil
}

// retireOnUnpriceableDowngrade still retires the prior entitlement when a
// downgrade's target policy cannot be read.
//
// 024 T204. This is a consequence of T188, and T188 could not have seen it.
//
// Before T188 a downgrade took the revoke path, which never read a policy, so
// nothing in the policy catalogue could prevent the Premium grant from being
// closed. T188 moved the downgrade onto the create path — correctly, because a
// free entitlement is a grant row like any other — and `GetPolicyByID` sits in
// front of `retirePreviousGrants` on that path. So a free plan whose
// `ai_cost_policy_id` cannot be resolved now refuses the event permanently
// (`RejectionError`, dead-lettered, never redelivered) and the Premium grant it
// was supposed to replace stays active and in force. A customer who cancelled
// keeps Premium entitlements and nobody is told why: the counters record a
// configuration fault, and the row that grants the access is untouched.
//
// The reachability is not hypothetical. `subscription.subscription_plans.ai_cost_policy_id`
// is `NOT NULL` but carries NO foreign key to `aiexecutor.ai_cost_policies` in the
// shipped schema — the column is declared at
// `database/scripts/closy_db-schema.sql:998` and no matching constraint appears
// in the `ADD CONSTRAINT` block at :1181-1189, although
// `20260622131631_add_ai_cost_policy.sql:25` created one and
// `20260801000000_separate_module_schemas.sql:180-181` re-created it against
// `public.ai_cost_policies`. So nothing at the data layer stops a hand-edited,
// copied or environment-specific value from naming a policy that does not exist
// here. Nothing on the producer side stops it either: `BuildPlanChangedPayload`
// fills `aiCostPolicyId` from `plan.AICostPolicyID.String()` and the parser only
// checks that the value PARSES (`grant_payload_parser.go:157`), while
// `ValidatePlanChangedPayload` checks `toPlanId` and `sourceEventId` but not this
// field. The refusal is therefore the correct answer to a well-formed event, and
// it is exactly this branch that has to stop it from also being the end of the
// customer's downgrade.
//
// The rejection is deliberately NOT weakened. No grant is written — there is no
// policy to price it from, and inventing one is what FR-085's "name the specific
// reason" exists to prevent — and the reason is still recorded on the counter
// and in the log (FR-086). Only the side effect is added: the entitlement the
// customer has already given up is closed regardless of whether the new one can
// be written.
//
// Scoped to the downgrade on purpose. For an UPGRADED that cannot be priced the
// customer has lost nothing: they still hold the tier they had, and closing that
// row would take away an entitlement they are still entitled to and leave the
// fallback, whose limits are lower. The two directions are not symmetric, and a
// rule that treated them the same would be wrong half the time.
//
// 024 T210: this path also had to answer the ORDERING question the priced path
// asks, because a refusal on one path and an application on the other for the
// same event is not a defensible pair of outcomes. The retirement below is now
// gated by `rejectStaleEvent`, exactly where the priced path puts it.
func (uc *GrantUseCase) retireOnUnpriceableDowngrade(ctx context.Context, event *PlanChangedEvent) error {
	if event.EventType != subscriptioneventtype.SUBSCRIPTION_DOWNGRADED_TO_FREE {
		return nil
	}
	err := uc.uow.Execute(ctx, func(txCtx context.Context) error {
		// Same per-account lock as every other write path, so this retirement
		// cannot interleave with a grant for the same account.
		if err := uc.aiCostRepo.LockGrantsForAccount(txCtx, event.UserID); err != nil {
			return err
		}
		// 024 T210: the ordering check, in the same position it occupies on the
		// priced create path - under the account lock, before anything is retired.
		//
		// Without it this path answered a different question than the path it
		// shadows. A downgrade that arrived late would still take the Premium
		// entitlement away here, while the identical event with a resolvable
		// policy would be refused as `stale_event`. Whether the customer loses
		// access would then depend on whether the free plan's
		// `ai_cost_policy_id` resolves - a property of the plan catalogue, not of
		// the event's ordering. One event, two conclusions.
		//
		// The scope T204 chose is unchanged: this only decides whether to retire
		// at all. A current downgrade still retires, which is the whole point of
		// the branch.
		if err := uc.rejectStaleEvent(txCtx, event); err != nil {
			return err
		}
		// 027 T010c: no row is created here, so there is no resolved window to pass.
		// The event's declared start is what the truncation point would have been,
		// and the CloseActiveGrants branch ignores the cutoff for the downgrade type.
		return uc.retirePreviousGrants(txCtx, event, EventGrantWindow(event).EffectiveFrom)
	})
	if err != nil {
		// A refusal is a decision, not a failure, and it must keep its own
		// classification. The stale refusal in particular must not be reclassified
		// as transient and requeued: it can never become current, so requeueing it
		// spends the delivery limit on an event whose answer is already known.
		// Same shape as the priced create path.
		var rejection *RejectionError
		if errors.As(err, &rejection) {
			return err
		}
		// Requeued rather than refused: this write did not land, so the
		// entitlement is still there and another delivery has to try again.
		uc.metrics.RecordRejection(ReasonTransientFailure, event.UserID.String())
		return uc.transient(err, ctx, event, "retire grants for a downgrade with no readable policy")
	}
	uc.logger.Error("downgrade could not be priced; the previous entitlement was still retired",
		zap.String("user_id", event.UserID.String()),
		zap.String("source_event_id", event.SourceEventID.String()),
		zap.String("ai_cost_policy_id", event.AICostPolicy.String()),
		zap.String("to_plan_code", event.ToPlanCode),
		zap.String("correlation_id", sharedevent.CorrelationIdFromContext(ctx)),
	)
	return nil
}

// retirePreviousGrants closes active rows and drops pending ones, according to
// the change type. A pending row is deleted rather than closed because a closed
// row would still be returned by a status-agnostic query.
//
// 027 T008: cutoff is an INPUT now, not something this function recomputes from
// the event. The create path passes the window resolveGrantWindow settled under
// the account lock, so the truncation point and the new row's start are the same
// value to the microsecond (contract §3, ADR 014 §2.2). The two paths that create
// no row - revocation and unpriceable downgrade - pass the event's own effective
// time, which for those types is exactly what EventGrantWindow(event).EffectiveFrom
// returns, so their behaviour is unchanged.
//
// BOTH branches cut at `cutoff`, and that is the point. The close branch used to
// cut at event.EffectiveAt, which is the UNSHIFTED event time. When a
// ClosesPriorGrants event arrives late with event.EffectiveAt earlier than the
// existing active row's effective_from, resolveGrantWindow shifts the new row up
// to orderFloor+1us but the old close cutoff stayed behind: CloseActiveGrants
// filters `effective_from <= cutoff`, so it matched nothing and left the prior row
// active alongside the new one - two overlapping active windows, which FR-008
// forbids. Cutting at the resolved `cutoff` closes the prior row at exactly the
// instant the new one starts, whether or not the window was shifted.
func (uc *GrantUseCase) retirePreviousGrants(ctx context.Context, event *PlanChangedEvent, cutoff time.Time) error {
	if DeletesPendingGrants(event.EventType) {
		if err := uc.aiCostRepo.DeleteFutureGrants(ctx, event.UserID); err != nil {
			return err
		}
	}
	if !ClosesPriorGrants(event.EventType) {
		// EXTENDED only moves the boundary, and the boundary is the START of the
		// new period — not the moment the event arrived. Cutting at the event
		// time would leave the account with no entitlement from now until the
		// new period begins, which is precisely the gap this rule exists to
		// prevent: a customer who renews early keeps the days they already paid
		// for only if the old period runs right up to the new one.
		return uc.aiCostRepo.TruncateGrantsBefore(ctx, event.UserID, cutoff, event.OccurredAt)
	}
	// Close at the same resolved instant the new row starts. The prior row is
	// closed rather than deleted because its window has ended, not because it never
	// existed - a closed row is still the newest record of what the account was
	// told, which the ordering check relies on.
	return uc.aiCostRepo.CloseActiveGrants(ctx, event.UserID, cutoff, event.OccurredAt)
}

// buildGrant assembles the grant row from a validated event.
//
// 024 T137: it now REQUIRES a plan id rather than trusting the caller. The parser
// already refuses an event without one, so this is the second of two guards - and
// the one that matters, because `plan_id` is `NOT NULL` and a nil UUID satisfies
// that constraint.
//
// 024 T172: this comment used to say such a row "is skipped by FR-036's catalogue
// check (which looks up BY id)". No such check exists, and the assertion was the
// dangerous part — it implied something downstream would catch the bad row. Nothing
// does. The AI service validates the COST POLICY id and deliberately does not look
// plans up at all, because reading the Account service's plan catalogue would
// violate constitution §I. So a grant pointing at the nil plan is a row that looks
// valid, is never inspected, and silently grants nothing usable. Nothing catches it
// later, which is why it is caught here.
//
// Returns nil rather than a broken row so the caller cannot accidentally persist
// it.
func (uc *GrantUseCase) buildGrant(
	event *PlanChangedEvent,
	snapshot aiexecutor_entities.JSONDocument,
) *aiexecutor_entities.AIPolicyGrant {
	// FR-023: the plan identifier is required, and the AI service must not derive
	// it by reading the Account service's subscription table (FR-024). There is
	// nothing to fall back to.
	if event == nil || event.ToPlanID == uuid.Nil {
		return nil
	}
	window := EventGrantWindow(event)
	return &aiexecutor_entities.AIPolicyGrant{
		UserID:                     event.UserID,
		PolicyID:                   event.AICostPolicy,
		PlanID:                     event.ToPlanID,
		PlanCode:                   event.ToPlanCode,
		TierRank:                   event.ToTierRank,
		PolicySnapshot:             snapshot,
		EffectiveFrom:              window.EffectiveFrom,
		EffectiveTo:                window.EffectiveTo,
		Status:                     window.Status,
		SourceEventID:              &event.SourceEventID,
		SourceDepositTransactionID: event.SourceDepositTransactionID,
	}
}
