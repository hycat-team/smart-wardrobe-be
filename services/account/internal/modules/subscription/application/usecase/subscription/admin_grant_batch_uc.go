package subscription

import (
	"context"
	"errors"
	"fmt"
	"time"

	admin_dto "smart-wardrobe-be/services/account/dto/admin"
	sub_dto "smart-wardrobe-be/services/account/dto/subscription"
	notificationContract "smart-wardrobe-be/services/account/internal/modules/notification/presentation/contract"
	subscriptionerrors "smart-wardrobe-be/services/account/internal/modules/subscription/application/errors"
	"smart-wardrobe-be/services/account/internal/modules/subscription/application/mapper"
	"smart-wardrobe-be/services/account/internal/modules/subscription/domain/entities"
	"smart-wardrobe-be/shared/domain/constants/adminaudit"

	"go.uber.org/zap"

	"github.com/google/uuid"
)

// AdminGrantBatch is the ONE way an administrator grants a plan (024 T083,
// FR-060).
//
// A single-account grant is a list of one. It is deliberately not a second
// method: before this, the two paths had separate validation, separate audit and
// separate error handling, and the single-account one silently wrote no audit
// record — so there was no way to answer "who gave this customer Premium for
// free". Any rule stated for a batch has to hold for the one-item case too, and
// the only way to guarantee that is to have one implementation.
//
// FR-064 shapes this whole file: ONE BAD ITEM MUST NOT AFFECT THE OTHERS. Not one
// transaction for the batch, not one failure for the batch. Each item runs in its
// own transaction and reports its own status, so a list of 500 whose 37th
// identifier is a typo yields 499 grants and one error line.

func (uc *SubscriptionUseCase) AdminGrantBatch(
	ctx context.Context,
	actorAdminID uuid.UUID,
	items []sub_dto.AdminGrantBatchItem,
	identifierKey string,
	requestIP *string,
) (*sub_dto.AdminGrantBatchRes, error) {
	if len(items) == 0 {
		// 024 T129: constitution §V - business errors come from `apperror`
		// factories. The handler already rejects an empty list before reaching
		// here, so this is a guard for a caller that skips the handler, not the
		// normal path.
		return nil, subscriptionerrors.ErrAdminGrantEmptyBatch()
	}

	// 024 T143: FR-076's latency bounds, enforced in code rather than measured.
	//
	// The spec requires a one-item list to answer within 5 seconds and a full
	// 500-item list within 60, and it separately requires that the server's WRITE
	// deadline stay above 60 - already true (`write_timeout_seconds: 120`), because
	// a write deadline at or below 60 would kill the response mid-operation.
	//
	// These are constants, not configuration. A setting would let a deployment
	// relax the guarantee the requirement states, and the requirement exists
	// because an administrator is waiting on a synchronous response.
	//
	// Deliberately NOT the global request timeout, which is 300s - a list that ran
	// to it would exceed FR-076 by a factor of five and the client would already
	// have given up.
	//
	// 024 T150: opened BEFORE any I/O, not just before the write loop.
	//
	// It used to be created after `Resolve`, `resolveGrantPlans` and
	// `loadPlannedVersions`, which meant the budget it enforced covered only the
	// grants themselves. The three pre-loop steps ran on the unbounded request
	// context, so a requirement stated as "answer within 60 seconds" was in fact
	// unenforced for a large part of the work - the administrator waited on queries
	// that could not be cut short. The budget has to start where the request does.
	ctx, cancel := context.WithTimeout(ctx, grantBudgetFor(len(items)))
	defer cancel()

	// One resolution pass for the whole list (FR-050). Otherwise a batch of 500
	// becomes 500 round trips to the identity module inside one request.
	resolved, err := uc.adminUserResolver.Resolve(ctx, collectIdentifiers(items))
	if err != nil {
		return nil, subscriptionerrors.ErrAdminGrantResolveUsersFailed(err)
	}

	// Duplicate detection runs BEFORE any write, so a list naming the same
	// account twice grants once (FR-064). Checking per item as the loop advanced
	// would already have committed the first grant by the time the second was
	// noticed.
	//
	// 024 T132: it runs AFTER resolution and keys on the RESOLVED ACCOUNT, not on
	// the identifier text. Two consequences that the old key missed:
	//
	//   - The same account written twice with DIFFERENT plan slugs is now
	//     detected. `quickstart.md` lists that combination explicitly, and before
	//     this task both items were granted, extending the period twice.
	//   - The same account reached through two different identifier forms (its
	//     email in one line, its UUID in the next) is now detected too.
	//
	// Known trade-off, stated rather than left to be discovered: if the FIRST line
	// for an account has a bad plan slug, it fails with `plan_not_found` and the
	// later well-formed line for the same account is still skipped. FR-066 asks
	// for "first line only", and one line per account is the predictable rule for
	// an operator reading a 500-line result list.
	duplicates := markDuplicateAccounts(items, resolved)

	// Distinct plan slugs are loaded once each, for the same reason.
	plans, planIssues := uc.resolveGrantPlans(ctx, items)

	// FR-069: the subscription versions are read ONCE, up front, for the whole
	// batch. Every item then compares what it planned against what it finds
	// inside its own transaction.
	//
	// One query for 500 accounts rather than 500: a pre-pass per item would be a
	// second round of I/O that buys nothing. It also makes the conflict real
	// rather than theoretical — the last item of a long list is planned early
	// and written late, so a purchase in between has to be detected instead of
	// silently overwritten.
	plannedVersions, err := uc.loadPlannedVersions(ctx, items, resolved)
	if err != nil {
		return nil, err
	}

	results := make([]sub_dto.AdminGrantBatchResult, len(items))
	for idx, item := range items {
		result := sub_dto.AdminGrantBatchResult{
			Index:          idx,
			UserIdentifier: item.UserIdentifier,
			PlanSlug:       item.PlanSlug,
		}

		// FR-076: on running out of time the system MUST NOT keep granting. The
		// items already granted stay granted - they were completed, and the
		// administrator is told to resubmit with the SAME Idempotency-Key, which
		// FR-068 turns into a no-op for those lines.
		//
		// The check is explicit rather than left to the database driver. Relying on
		// a cancelled context to fail each remaining write worked, but every one of
		// those items was then reported as `concurrent_modification`, which told
		// the operator the account had been changed by someone else.
		if ctxErr := ctx.Err(); ctxErr != nil {
			uc.log.Warn("admin grant batch stopped on its time budget",
				zap.Int("grantedSoFar", idx),
				zap.Int("total", len(items)),
				zap.Error(ctxErr),
			)
			markRemainingAsTimedOut(results, idx, items)
			return summarizeGrantResults(results), nil
		}

		switch {
		case duplicates[idx]:
			// FR-066 says "đánh dấu ... là bỏ qua có lý do": SKIPPED, not error.
			// The distinction is the whole point of the status field - `error`
			// means something is wrong with this line and the operator should fix
			// it, while `skipped` means the request was understood and this line
			// was deliberately not acted on. Reporting a duplicate as an error
			// makes a correct request look broken.
			result.Status = sub_dto.AdminGrantStatusSkipped
			result.Reason = sub_dto.AdminGrantReasonDuplicateInRequest
			result.Message = mapper.MsgAdminGrantDuplicateInRequest

		default:
			reason, message, failed := classifyGrantIdentifier(item.UserIdentifier, resolved)
			if failed {
				result.Status = sub_dto.AdminGrantStatusError
				result.Reason = reason
				result.Message = message
				break
			}
			if issue, has := planIssues[idx]; has {
				result.Status = sub_dto.AdminGrantStatusError
				result.Reason = issue.reason
				result.Message = issue.message
				break
			}

			user := resolved[normalizeIdentifier(item.UserIdentifier)]
			plan := plans[idx]

			// FR-056: an administrator is never a grant target. Granting Premium
			// to staff would be charging the company, and the check has to be per
			// item because one list can name any mix of accounts.
			if IsAdministratorTarget(user.User) {
				result.Status = sub_dto.AdminGrantStatusError
				result.Reason = sub_dto.AdminGrantReasonTargetIsAdmin
				result.Message = mapper.MsgAdminGrantTargetIsAdmin
				break
			}
			// FR-049: a soft-deleted account is refused and REPORTED as deleted,
			// so the administrator does not go hunting for a typo that is not
			// there.
			if user.User != nil && user.User.IsDeleted {
				result.Status = sub_dto.AdminGrantStatusError
				result.Reason = sub_dto.AdminGrantReasonUserSoftDeleted
				result.Message = mapper.MsgAdminGrantUserSoftDeleted
				break
			}

			granted, gerr := uc.grantOne(ctx, grantRequest{
				actorAdminID:   actorAdminID,
				requestIP:      requestIP,
				userID:         user.UserID,
				plan:           plan,
				identifierKey:  identifierKey,
				itemIndex:      idx,
				plannedVersion: plannedVersions[user.UserID].version,
				plannedFound:   plannedVersions[user.UserID].found,
			})
			if gerr != nil {
				// 024 T129: only a genuine version conflict may be reported as
				// `concurrent_modification`. Before this, EVERY error from the
				// item's transaction - a dead database, a failed audit write -
				// was labelled "the account was just changed by someone else".
				// That is not merely imprecise: it names a cause the operator
				// cannot act on and tells them to retry, when retrying a failing
				// dependency just fails again.
				if errors.Is(gerr, subscriptionerrors.ErrAdminGrantConcurrentModification()) {
					result.Status = sub_dto.AdminGrantStatusError
					result.Reason = sub_dto.AdminGrantReasonConcurrentModif
					result.Message = mapper.MsgAdminGrantConcurrentModif
					break
				}
				// Anything else is an infrastructure or invariant failure. It
				// stays an item-level error so the other items still proceed
				// (FR-064), but it carries no reason code the spec does not
				// define, and the cause is logged rather than shown.
				uc.log.Error("admin grant item failed",
					zap.Int("itemIndex", idx),
					zap.String("userId", user.UserID.String()),
					zap.Error(gerr),
				)
				result.Status = sub_dto.AdminGrantStatusError
				result.Reason = sub_dto.AdminGrantReasonGrantFailed
				result.Message = mapper.MsgAdminGrantFailed
				break
			}
			if granted.replayed {
				// FR-068: this exact (key, index) was already granted. Reported as
				// `skipped`, not `success`, so the administrator can tell a repeat
				// from a fresh grant - and the expiry is NOT extended a second
				// time.
				result.Status = sub_dto.AdminGrantStatusSkipped
				result.Reason = sub_dto.AdminGrantReasonDuplicateRequest
				result.Message = mapper.MsgAdminGrantDuplicateRequest
				result.PlanName = plan.Name
				result.ExpiresAt = granted.expiresAt
				resolvedID := user.UserID
				result.ResolvedUserID = &resolvedID
				break
			}

			result.Status = sub_dto.AdminGrantStatusSuccess
			result.PlanName = plan.Name
			result.ExpiresAt = granted.expiresAt
			resolvedID := user.UserID
			result.ResolvedUserID = &resolvedID
			result.Message = grantSuccessMessage(plan, granted.expiresAt)
			results[idx] = result

			// FR-073: the cached overview still describes the OLD plan, so it is
			// dropped after the commit. Doing it inside the transaction would be
			// wrong in the other direction - a rollback would leave the customer
			// with no cached overview at all, which is harmless but pointless,
			// whereas a stale entry that survives a successful grant is not.
			//
			// The write to the database above is unconditional and does not consult
			// this cache: the grant reads the row under FOR UPDATE precisely so a
			// stale cache can never decide what a customer is entitled to.
			uc.invalidateOverviewCache(ctx, user.UserID)

			// FR-083: the notification goes out AFTER the commit, and only when
			// the account has an address. A recipient must never be told about a
			// grant that was rolled back.
			uc.notifyGrant(ctx, user, plan, granted.expiresAt)
			continue
		}

		results[idx] = result
	}

	return summarizeGrantResults(results), nil
}

// grantRequest is the input of one item's transaction.

func adminGrantEventKey(identifierKey string, itemIndex int) string {
	return fmt.Sprintf("ADMIN_GRANT:%s:%d", identifierKey, itemIndex)
}

// 024 T133: checkGrantableState is GONE, and this is the note explaining why.
//
// It used to refuse a grant when the subscription had expired AND auto-renew was
// on, reporting `concurrent_modification` on the theory that "the cron worker owns
// this row, granting over it races the renewal". That reasoning was wrong on its
// own terms: the row has already been taken with FOR UPDATE one call earlier, so
// the renewal worker is either blocked behind us or has already finished. There
// is no race left to refuse.
//
// FR-053 requires the opposite behaviour: granting a different plan, or granting
// when the current period has expired, MUST start a new period from the moment of
// the grant. An expired Premium subscriber whose renewal failed is precisely the
// customer an administrator is trying to fix, and the old code refused that grant
// while telling them the account had been changed by someone else - a reason code
// that sends the operator down the wrong path.
//
// What still protects the row is the FR-069 version comparison in checkPlannedVersion,
// which is the check that actually matters: it catches a real change made between
// the batch's read and this item's write, whoever made it.

// replayIfAlreadyGranted reports whether this exact (key, index) has already been
// granted, and fills `out` for a replay when it has.
//
// It is called twice: once before the account row is locked, so an ordinary retry
// costs nothing, and once after, so a retry that raced another one is recognised
// instead of being reported as an infrastructure failure (024 T169).
func (uc *SubscriptionUseCase) replayIfAlreadyGranted(
	ctx context.Context,
	req grantRequest,
	out *grantedItem,
) (bool, error) {
	existing, err := uc.eventRepo.GetByKey(ctx, adminGrantEventKey(req.identifierKey, req.itemIndex))
	if err != nil {
		return false, err
	}
	if existing == nil {
		return false, nil
	}

	// Read the CURRENT expiry rather than replaying a stored one: the response has to
	// describe the account as it is now, not as it was when the original grant landed.
	current, err := uc.userSubRepo.GetByUserID(ctx, req.userID)
	if err != nil {
		return false, err
	}
	out.replayed = true
	out.expiresAt = nil
	if current != nil {
		out.expiresAt = current.ExpiresAt
	}
	return true, nil
}

// grantOne applies a single item inside its own transaction.
//
// FR-064 again: this is why the unit of work is entered HERE and not around the
// whole loop. Rolling back item 37 must not undo items 1..36 and 38..500.
func (uc *SubscriptionUseCase) grantOne(ctx context.Context, req grantRequest) (*grantedItem, error) {
	out := &grantedItem{}
	err := uc.uow.Execute(ctx, func(txCtx context.Context) error {
		now := time.Now()

		// FR-068: this exact (key, index) has been granted before, so the request
		// is a RETRY. The check is first, before anything is locked or written,
		// because extending the expiry twice is the whole failure being
		// prevented: a batch retry would silently hand the customer two months.
		//
		// It is NOT the only check - see the re-check after the lock below.
		replayed, err := uc.replayIfAlreadyGranted(txCtx, req, out)
		if err != nil {
			return err
		}
		if replayed {
			return nil
		}

		// FR-053/FR-054: current state is read under FOR UPDATE. Without the lock,
		// two concurrent grants to the same account both read the same expiry,
		// both compute "now + 30 days", and the second write silently discards
		// the first — the customer's paid days vanish with no error anywhere.
		//
		// 024 T133: there is deliberately NO check of the SUBSCRIPTION STATE after
		// this. An expired subscription is grantable and starts a new period
		// (FR-053); the lock above already removed the race the old check was
		// defending against. The IDEMPOTENCY key is a different matter and is
		// re-checked below.
		current, err := uc.userSubRepo.GetByUserIDWithLock(txCtx, req.userID)
		if err != nil {
			return err
		}

		// 024 T169: re-check the idempotency key now that the account row is locked.
		//
		// Two administrators — or one administrator whose client retried — can send
		// the same Idempotency-Key at the same moment. Both passed the check above,
		// because both read "not granted yet".
		//
		// Holding the account row serialises the two transactions, so by the time
		// this runs the winner's event row is committed and visible and this is a
		// plain duplicate — exactly what FR-068 calls a retry.
		//
		// What this prevents is a DOUBLE GRANT, not only a misleading reason code.
		// The convergence note recorded it as "the reason code was wrong", and that
		// understates it. For an account that already has a subscription row — the
		// ordinary case, since administrators re-grant to existing customers —
		// nothing fails without this re-check: the loser simply blocks on the row
		// lock, wakes up holding the winner's NEW expiry, and extends it again. The
		// customer receives two months for one decision, which is the exact failure
		// FR-068 exists to prevent, and it is silent. Removing this call makes the
		// test in admin_grant_idempotency_race_test.go show an expiry that has moved
		// twice.
		//
		// The wrong-reason case still exists, for an account with no row yet: there is
		// nothing to lock, both proceed, and the primary key rejects the second
		// insert — so the loser falls through to `grant_failed` and the
		// administrator is sent to look for a fault that does not exist.
		replayed, err = uc.replayIfAlreadyGranted(txCtx, req, out)
		if err != nil {
			return err
		}
		if replayed {
			return nil
		}

		// FR-069: the row is compared against what the batch planned, not merely
		// locked. The lock stops two GRANTS from colliding; it does not stop a
		// customer purchase that landed between the batch's read and this
		// transaction. Overwriting that would discard a real payment.
		if err := checkPlannedVersion(req, current); err != nil {
			return err
		}

		// The apply also refreshes current_benefit_snapshot on an EXISTING
		// subscription (FR-070) and resets fallback_* plus is_auto_renew_enabled
		// when the plan changes (FR-071). Resetting the auto-renew flag IS the
		// "cancel the related auto-renewal" step: renewal is driven by that
		// column through the cron worker, so there is no separate job object to
		// cancel, and leaving the flag set would renew a plan the administrator
		// had just replaced.
		event, err := ApplySubscriptionPlanWithEvent(txCtx, uc.userSubRepo, req.userID, req.plan, now)
		if err != nil {
			return err
		}
		// The batch keys the event by Idempotency-Key + index, which is what makes
		// a retried request a no-op on the receiver side too.
		event.EventKey = adminGrantEventKey(req.identifierKey, req.itemIndex)
		event.ActorAdminID = actorAdminIDOf(&req.actorAdminID)

		if err := uc.resetGrantUsageQuota(txCtx, req.userID, now); err != nil {
			return err
		}

		// Re-read so the payload carries the NEW expiry. Passing the pre-apply row
		// would make the receiver place the renewal window from the old date.
		updated, err := uc.userSubRepo.GetByUserID(txCtx, req.userID)
		if err != nil {
			return err
		}

		// FR-015/FR-081: the plan-change event AND the audit record are written in
		// THIS transaction, so the trail can never claim a grant that was rolled
		// back. Both actors are recorded: who received the plan, and which
		// administrator gave it.
		if _, err := uc.planChangePublisher.PublishPlanChange(txCtx, event, updated, req.plan); err != nil {
			return err
		}
		if err := uc.auditLogContract.Record(txCtx, admin_dto.RecordAuditLogInput{
			ActorUserID: req.actorAdminID,
			Action:      string(adminaudit.SubscriptionGrant),
			TargetType:  string(adminaudit.UserSubscription),
			TargetID:    &req.userID,
			Payload: map[string]any{
				"identifierKey":    req.identifierKey,
				"itemIndex":        req.itemIndex,
				"planSlug":         req.plan.Slug,
				"planName":         req.plan.Name,
				"targetUserId":     req.userID.String(),
				"actorAdminId":     req.actorAdminID.String(),
				"previousPlanCode": previousPlanCodeOf(current),
				"expiresAt":        expiresAtText(updated),
				"isSelfService":    false,
			},
			RequestIP: req.requestIP,
		}); err != nil {
			// 024 T129: the audit row is what makes the grant answerable later, so
			// its failure aborts this item's transaction instead of producing a
			// grant nobody can attribute.
			return subscriptionerrors.ErrAdminGrantRecordAuditFailed(err)
		}

		out.expiresAt = updated.ExpiresAt
		return nil
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

// plannedVersion is what the batch observed before it wrote anything.

func (uc *SubscriptionUseCase) resetGrantUsageQuota(ctx context.Context, userID uuid.UUID, now time.Time) error {
	quota, err := uc.quotaRepo.GetByUserID(ctx, userID)
	if err != nil {
		return err
	}
	if quota == nil {
		return nil
	}
	quota.OutfitRecommendCount = 0
	quota.AiUsageCount = 0
	quota.LastResetDate = now
	return uc.quotaRepo.Update(ctx, quota)
}

func (uc *SubscriptionUseCase) invalidateOverviewCache(ctx context.Context, userID uuid.UUID) {
	InvalidateOverviewCache(ctx, uc.log, uc.overviewCache, userID)
}

// notifyGrant emails the recipient, and only when there is an address to send it
// to (FR-083). A missing address is NOT a failed grant: the plan is applied and
// the administrator already holds a success line.
func (uc *SubscriptionUseCase) notifyGrant(ctx context.Context, resolved *ResolvedUser, plan *entities.SubscriptionPlan, expiresAt *time.Time) {
	if resolved == nil || resolved.User == nil || resolved.User.Email == "" {
		return
	}
	expiresText := "Không thời hạn"
	if expiresAt != nil {
		expiresText = expiresAt.Local().Format("02/01/2006")
	}
	// The send error is dropped on purpose: reporting a failed email as a failed
	// grant would make the administrator re-grant a plan that was already
	// applied. The email outbox holds the delivery record.
	_ = uc.notifContract.Send(ctx, notificationContract.ChannelEmail, notificationContract.TemplateAdminGrant, map[string]any{
		"PlanName":  plan.Name,
		"ExpiresAt": expiresText,
	}, resolved.User.Email)
}

// resolveGrantPlans loads each distinct plan slug once (FR-050) and records a
// per-item reason for the ones that cannot be used.

func previousPlanCodeOf(sub *entities.UserSubscription) string {
	if sub == nil {
		return ""
	}
	return sub.CurrentPlanCode
}

func expiresAtText(sub *entities.UserSubscription) string {
	if sub == nil || sub.ExpiresAt == nil {
		return ""
	}
	return sub.ExpiresAt.UTC().Format(time.RFC3339)
}
