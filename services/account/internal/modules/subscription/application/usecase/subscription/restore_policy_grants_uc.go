package subscription

import (
	"context"
	"fmt"
	"time"

	admin_dto "smart-wardrobe-be/services/account/dto/admin"
	sub_dto "smart-wardrobe-be/services/account/dto/subscription"
	"smart-wardrobe-be/services/account/internal/modules/subscription/application/mapper"
	"smart-wardrobe-be/services/account/internal/modules/subscription/domain/entities"
	"smart-wardrobe-be/services/account/internal/modules/subscription/domain/repositories"
	"smart-wardrobe-be/shared/domain/constants/adminaudit"
	"smart-wardrobe-be/shared/domain/constants/subscriptioneventtype"

	"github.com/google/uuid"
)

// restoreScanPageSize bounds one page of the scan. The repair reads every
// subscription in the table, so it must page rather than load it all: a system
// with a million customers would otherwise hold a million rows in memory during
// a repair nobody asked for.
const restoreScanPageSize = 200

// RestorePolicyGrants re-publishes the plan-change event for accounts whose AI
// entitlement went missing (024 T110/T111, FR-088).
//
// It is a REPAIR tool, not a grant tool. A grant changes what a customer gets;
// this changes nothing about the subscription - it only makes the AI service
// rebuild the entitlement it should already have derived. That is why it is a
// separate endpoint with its own audit action: an operator asking "who gave
// away Premium" must not find restores mixed in with decisions.
//
// FREE AND PAID ACCOUNTS ARE TREATED ALIKE (FR-088). A free account's
// entitlement is a `free-default` grant just as real as a Premium one: it is what
// carries the free per-day attempt cap, and the cost recorded against it is
// observation data, not a charge. An earlier version of this file skipped free
// plans on the assumption that they had no grant at all; that assumption was
// wrong, and skipping them would have left a free account whose grant went
// missing permanently unrepaired.
//
// EXPIRED ACCOUNTS ARE STILL SKIPPED, and that is a different rule. A lapsed
// subscription is handled by the renewal and downgrade paths; re-publishing here
// would hand a customer who has stopped paying an ACTIVE grant whose end date is
// in the past, and nothing downstream would notice.
//
// Account cannot see which accounts are actually missing an entitlement, because
// that table belongs to the AI service (constitution I). So it re-publishes for
// every subscribed account and lets the AI side de-duplicate on sourceEventId.
// The alternative - querying AI's table - would re-create exactly the cross-
// module read this feature removed.
//
// ⚠️ Running it repeatedly is safe. The event key is derived from the
// subscription's VERSION, which the repair does not change, so a second run
// finds the key already present and is skipped. An operator can therefore retry
// without bookkeeping.
//
// ⚠️ The WHOLE RUN is one transaction (constitution V). Scanning the
// subscriptions, resolving their plans, re-publishing their events and writing
// the audit record are all inside a single `IUnitOfWork.Execute`.
//
// That is deliberately coarser than the batch grant path, which isolates each
// item (FR-064). A repair is a different kind of operation: it exists to bring
// the table back into a known-good state, so a half-applied repair is worse than
// no repair - the next run would have to reason about which accounts were
// already touched. Here, either every account is re-published AND the audit
// record exists, or nothing happened at all.
//
// A per-account failure is still recorded in the outcome and does NOT abort the
// run: one unpublishable account must not prevent the other 499 from being
// repaired.
func (uc *SubscriptionUseCase) RestorePolicyGrants(ctx context.Context, actorAdminID uuid.UUID, dryRun bool, requestIP *string) (*sub_dto.RestorePolicyGrantsRes, error) {
	res := &sub_dto.RestorePolicyGrantsRes{DryRun: dryRun}

	err := uc.uow.Execute(ctx, func(txCtx context.Context) error {
		subs, err := uc.scanSubscriptions(txCtx)
		if err != nil {
			return err
		}
		res.Scanned = len(subs)

		plans, err := uc.loadRestorePlans(txCtx, subs)
		if err != nil {
			return err
		}

		uc.applyRestoreRun(txCtx, res, subs, plans, dryRun)

		// The audit record is INSIDE the transaction on purpose. Written outside,
		// a failure between the repairs and the audit would leave a repaired
		// system with no record of who ran the repair - which is exactly the
		// question SC-021 asks.
		return uc.recordRestoreAudit(txCtx, actorAdminID, dryRun, res, requestIP)
	})
	if err != nil {
		return nil, err
	}
	return res, nil
}

// applyRestoreRun produces one outcome per account and fills the counters.
//
// It never returns an error: a per-account failure belongs in that account's
// line of the response, not in the transaction.
func (uc *SubscriptionUseCase) applyRestoreRun(
	txCtx context.Context,
	res *sub_dto.RestorePolicyGrantsRes,
	subs []*entities.UserSubscription,
	plans map[string]*entities.SubscriptionPlan,
	dryRun bool,
) {
	now := time.Now()
	for _, sub := range subs {
		outcome := sub_dto.RestoreResult{
			UserID:   sub.UserID.String(),
			PlanSlug: sub.CurrentPlanCode,
			Status:   sub_dto.RestoreStatusSuccess,
		}

		plan := plans[sub.CurrentPlanCode]
		switch {
		case plan == nil:
			// 024 T159: this reason really IS a catalogue check - `plans` is the plan
			// catalogue loaded above. The identically-worded reason on the AI service
			// was renamed in T159 because THAT side never reads a catalogue at all; it
			// only looks up an AI cost policy. Two different facts, two different names,
			// and this one keeps its name. Noted here so the two are not "aligned" later.
			outcome.Status = sub_dto.RestoreStatusError
			outcome.Reason = sub_dto.RestoreReasonPlanNotInCatalog
			outcome.Message = mapper.RestoreMessage(sub_dto.RestoreReasonPlanNotInCatalog)
		case sub.ExpiresAt != nil && sub.ExpiresAt.Before(now):
			// An expired subscription is left to the renewal and downgrade paths.
			// Re-publishing here would hand a lapsed customer an ACTIVE grant
			// with a past end date, and nothing downstream would notice.
			outcome.Status = sub_dto.RestoreStatusSkipped
			outcome.Reason = sub_dto.RestoreReasonSubscriptionExpired
			outcome.Message = mapper.RestoreMessage(sub_dto.RestoreReasonSubscriptionExpired)
		case !dryRun:
			alreadyDone, err := uc.publishRestoreEvent(txCtx, sub, plan)
			switch {
			case err != nil:
				outcome.Status = sub_dto.RestoreStatusError
				outcome.Reason = sub_dto.RestoreReasonPublishFailed
				outcome.Message = mapper.RestoreMessage(sub_dto.RestoreReasonPublishFailed)
			case alreadyDone:
				// A previous run already repaired this account. Reported as
				// skipped rather than restored, because a counter that keeps
				// climbing on every retry tells the operator a repair happened
				// when nothing was written.
				outcome.Status = sub_dto.RestoreStatusSkipped
				outcome.Reason = sub_dto.RestoreReasonAlreadyRestored
				outcome.Message = mapper.RestoreMessage(sub_dto.RestoreReasonAlreadyRestored)
			}
		}

		switch outcome.Status {
		case sub_dto.RestoreStatusSuccess:
			res.Restored++
		case sub_dto.RestoreStatusError:
			res.Failed++
		default:
			// An expired subscription, or one a previous run already repaired.
			// Neither is a failure, so neither is counted as one.
			res.Skipped++
		}
		res.Results = append(res.Results, outcome)
	}
}

// publishRestoreEvent writes the history row and the outbox record for one
// account, and reports whether it had already been done.
//
// It runs on the CALLER's transaction and must not open one of its own: the
// whole repair is already inside `IUnitOfWork.Execute` (constitution V), and a
// nested `Execute` would either create a savepoint nobody asked for or a second
// independent transaction that could commit without the audit record.
//
// ACTIVATED is used deliberately rather than a new event type: the AI side
// treats it as "derive a fresh entitlement from this plan", which is exactly
// what a repair needs, and it already closes the stale grant that is being
// replaced. A bespoke type would have to be added to the event contract and to
// every consumer to mean the same thing.
func (uc *SubscriptionUseCase) publishRestoreEvent(ctx context.Context, sub *entities.UserSubscription, plan *entities.SubscriptionPlan) (bool, error) {
	planSlug := plan.Slug
	tierRank := plan.TierRank
	// Version is part of the key and the repair does not change it, so a repeat
	// run produces the same key and is skipped rather than duplicating work.
	eventKey := fmt.Sprintf("RESTORE_GRANT:%s:%d", sub.UserID, sub.Version)
	event := &entities.UserSubscriptionEvent{
		EventKey:    eventKey,
		UserID:      sub.UserID,
		EventType:   string(subscriptioneventtype.ACTIVATED),
		ToPlanCode:  &planSlug,
		ToTierRank:  &tierRank,
		OccurredAt:  time.Now(),
		EffectiveAt: effectiveFromOf(sub),
		Metadata:    entities.JSONDocument(`{"source":"admin_restore"}`),
	}

	alreadyDone := false
	// Checked before writing so a repeated run does not re-emit an event the AI
	// service would then have to de-duplicate.
	existing, err := uc.eventRepo.GetByKey(ctx, eventKey)
	if err != nil {
		return false, err
	}
	if existing != nil {
		alreadyDone = true
		return alreadyDone, nil
	}
	_, err = uc.planChangePublisher.PublishPlanChange(ctx, event, sub, plan)
	return alreadyDone, err
}

// recordRestoreAudit writes one record for the WHOLE run.
//
// One row rather than one per account: the per-account detail belongs in the
// response, while the audit trail answers the operator's question - who ran a
// repair, when, from where, and how much it touched.
//
// requestIP is recorded because SC-021 requires every restore to be traceable
// to a connection, not merely to an account. The batch grant path already does
// this; without it here, a repair would be the one administrative write on this
// feature with no way to tell which operator ran it from where.
func (uc *SubscriptionUseCase) recordRestoreAudit(ctx context.Context, actorAdminID uuid.UUID, dryRun bool, res *sub_dto.RestorePolicyGrantsRes, requestIP *string) error {
	return uc.auditLogContract.Record(ctx, admin_dto.RecordAuditLogInput{
		ActorUserID: actorAdminID,
		Action:      string(adminaudit.SubscriptionGrantRestore),
		TargetType:  string(adminaudit.UserSubscription),
		Payload: map[string]any{
			"dryRun":   dryRun,
			"scanned":  res.Scanned,
			"restored": res.Restored,
			"skipped":  res.Skipped,
			"failed":   res.Failed,
		},
		RequestIP: requestIP,
	})
}

// scanSubscriptions pages through every subscription.
//
// Keyset paging on user_id rather than OFFSET: an OFFSET scan re-reads every row
// it skipped on each page, so a repair over a large table would degrade
// quadratically and take long enough that an operator would run it twice.
func (uc *SubscriptionUseCase) scanSubscriptions(ctx context.Context) ([]*entities.UserSubscription, error) {
	out := make([]*entities.UserSubscription, 0, restoreScanPageSize)
	for page := 1; ; page++ {
		pageItems, total, err := uc.userSubRepo.ListAdmin(ctx, repositories.AdminUserSubscriptionFilter{}, page, restoreScanPageSize)
		if err != nil {
			return nil, fmt.Errorf("quét danh sách gói đăng ký: %w", err)
		}
		out = append(out, pageItems...)
		if len(out) >= int(total) || len(pageItems) == 0 {
			return out, nil
		}
	}
}

// loadRestorePlans loads every plan referenced by the scan, once each.
//
// A plan missing from the catalogue is an ERROR rather than a skip: the
// subscription names a plan the system no longer sells, and the operator needs
// to see that as a problem instead of quietly repairing nothing.
func (uc *SubscriptionUseCase) loadRestorePlans(ctx context.Context, subs []*entities.UserSubscription) (map[string]*entities.SubscriptionPlan, error) {
	out := make(map[string]*entities.SubscriptionPlan)
	for _, sub := range subs {
		if sub.CurrentPlanCode == "" {
			continue
		}
		if _, ok := out[sub.CurrentPlanCode]; ok {
			continue
		}
		plan, err := uc.planRepo.GetBySlugAnyStatus(ctx, sub.CurrentPlanCode)
		if err != nil {
			return nil, fmt.Errorf("tải gói %q: %w", sub.CurrentPlanCode, err)
		}
		out[sub.CurrentPlanCode] = plan
	}
	return out, nil
}

// effectiveFromOf anchors the restored window on the day the current period
// started. Using "now" would leave the AI grant beginning in the future relative
// to the period it is meant to cover.
func effectiveFromOf(sub *entities.UserSubscription) time.Time {
	if sub == nil || sub.StartedAt.IsZero() {
		return time.Now()
	}
	if sub.ExpiresAt != nil && sub.StartedAt.After(*sub.ExpiresAt) {
		return time.Now()
	}
	return sub.StartedAt
}
