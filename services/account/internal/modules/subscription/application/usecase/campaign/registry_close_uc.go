package campaign

import (
	"context"
	"errors"
	"time"

	admin_dto "smart-wardrobe-be/services/account/dto/admin"
	sub_dto "smart-wardrobe-be/services/account/dto/subscription"
	subscriptionerrors "smart-wardrobe-be/services/account/internal/modules/subscription/application/errors"
	"smart-wardrobe-be/services/account/internal/modules/subscription/application/mapper"
	"smart-wardrobe-be/services/account/internal/modules/subscription/domain/entities"
	"smart-wardrobe-be/shared/domain/constants/adminaudit"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

// 026 T041 — the FORCE-CLOSE path of `CampaignRegistryUseCase`, split out of
// `registry_uc.go`.
//
// WHY A SEPARATE FILE AND NOT A SECTION. `registry_uc.go` already held `Create`,
// the shared type and its constructor, and this operation is the longest in the
// registry: four ordered answers, a conditional statement, a reload and an audit
// payload. Adding it inline pushed the file past the 500-line production limit
// (code-hygiene H1), and the honest fix is the one the limit is asking for — a
// file named after the operation, so that "what does closing do" is answered by
// `registry_close_uc.go` without reading a create.
//
// IT IS STILL ONE TYPE AND ONE TRANSACTION. The split is a FILE boundary and
// nothing else: `Close`, `closeWithAudit` and `recordCloseAudit` are methods and
// helpers of `CampaignRegistryUseCase`, declared in `registry_uc.go`, and closing
// still enters `uow.Execute` exactly once (constitution §V).
//
// THE PARTS THAT STAYED IN `registry_uc.go`, and why they are not here:
//
//   - `requireRegistryRow`, because `Update` asks the same 404 question and one
//     implementation is what stops the two operations from answering it differently
//     (FR-054: one code per failure);
//   - `endsAtForAudit`, because the create audit needs it too — a helper used once
//     would be the `_helper.go` shape, a helper used by two operations is not.

// recordCloseAudit writes the ONE outbox row a successful close owes (FR-060).
//
// IT IS SEPARATE FROM `recordCreateAudit` rather than a shared helper with a flag,
// because the two payloads share almost nothing: a create has no "before" (the
// campaign did not exist) and a close has no configuration to describe (nothing
// about the plan or the budget changed). One function with a boolean deciding which
// of the two shapes to build is a function that has to be read twice to know which
// fields the per-campaign audit read endpoint will find.
//
// `before`/`after` ARE PRESENT HERE, and that is what makes a closure readable
// weeks later: `before.closedAt` empty against `after.closedAt` set is the whole
// change, side by side with the version that moved (contract §6.3). The reason
// travels beside them because "who stopped this promotion and why" is the question
// the screen exists to answer (SC-011).
func (uc *CampaignRegistryUseCase) recordCloseAudit(
	ctx context.Context,
	actorAdminID uuid.UUID,
	campaignCode string,
	reason string,
	beforeVersion int,
	closedAt time.Time,
	requestIP *string,
) error {
	err := uc.auditLogContract.Record(ctx, admin_dto.RecordAuditLogInput{
		ActorUserID: actorAdminID,
		Action:      string(adminaudit.CampaignClose),
		TargetType:  string(adminaudit.Campaign),
		// Intentionally nil: FR-063, identically to the create record.
		TargetID: nil,
		Payload: map[string]any{
			"campaignCode": campaignCode,
			"reason":       reason,
			"before": map[string]any{
				"closedAt": "",
				"version":  beforeVersion,
			},
			"after": map[string]any{
				// UTC and RFC3339 for the same reason `endsAtForAudit` uses it: the
				// payload is compared against another row later, on a screen written by
				// somebody else.
				"closedAt": closedAt.UTC().Format(time.RFC3339),
				"version":  beforeVersion + 1,
			},
		},
		RequestIP: requestIP,
	})
	if err != nil {
		// FR-060/FR-061, and the same line as the create: the outbox row is part of
		// THIS transaction, so returning the error rolls the closure back with it.
		// A registry whose `closed_at` and whose audit trail disagree is exactly what
		// FR-062 exists to prevent.
		return subscriptionerrors.ErrCampaignRegistryAuditFailed(campaignCode, err)
	}
	return nil
}

// Close force-closes a campaign and returns it as the read endpoints would render
// it (FR-030/FR-031/FR-034, contract §5).
//
// THE ORDER OF THE FOUR ANSWERS IS THE WHOLE METHOD, and it is not an
// implementation detail — contract §4.3 and §5.3 fix it, and each step exists
// because of a specific operator mistake:
//
//  1. `GetByCode` INSIDE this transaction, and a code nobody typed answers 404
//     immediately. `UPDATE ... WHERE code = ? AND version = ?` returns ZERO ROWS for
//     both "no such code" and "wrong version", so a method that looked only at the
//     row count would answer 412 to an operator who mistyped a code — telling them
//     to reload a campaign that does not exist.
//  2. ALREADY CLOSED ⇒ 200, no second audit row, and NO version comparison at all.
//     A retried request, a double-click, and two administrators closing the same
//     campaign at the same moment all arrive here, and all three must succeed
//     (FR-034). The version is deliberately not consulted: the row has moved on
//     since the client last read it, which is precisely the situation idempotence
//     exists to absorb.
//  3. `UPDATE ... WHERE version = ?` and zero rows means somebody else wrote in
//     between, which is 412 (FR-055) — refused rather than overwritten, so no
//     administrator's change is lost and no audit row describes a change that did
//     not happen.
//  4. The audit outbox row, INSIDE the same transaction (FR-060/FR-061).
//
// WHAT THE RESPONSE IS, AND WHAT IT IS NOT. It is the campaign's own registry row
// after the change, plus the derived `status`, built by the SAME mapper the read
// endpoints and `Create` use — so no screen has to reformat one shape into
// another. It is NOT a full operator summary: the ledger counts and the priority
// watermark are not read here, because the registry WRITER holds no claim
// repository and giving it one would make the write path a second reader of the
// ledger for three numbers the client refreshes anyway. The count fields therefore
// read as zero on this response and are correct on the next `GET`; `closedAt` and
// `status` — the two things this operation changed — are exact.
//
// `closed_at` IS WRITTEN BY THE DATABASE (`now()` inside the repository's
// statement), never by the value this process reads from its own clock. It answers
// "when was this campaign closed" to somebody reading the trail weeks later, and a
// caller is not a trustworthy witness to its own request.
func (uc *CampaignRegistryUseCase) Close(
	ctx context.Context,
	actorAdminID uuid.UUID,
	campaignCode string,
	req *sub_dto.CampaignCloseReq,
	requestIP *string,
) (*sub_dto.CampaignSummaryRes, error) {
	// 400 (the binder, for a missing `version` or `reason`) and 422 (a blank reason)
	// BEFORE ANY I/O: a rejected closure opens no transaction, takes no row lock and
	// writes no outbox row, so a typo cannot produce a trace saying somebody tried.
	reason, err := validateCampaignClose(req)
	if err != nil {
		return nil, err
	}

	var closed *entities.Campaign
	err = uc.uow.Execute(ctx, func(txCtx context.Context) error {
		stored, err := uc.requireRegistryRow(txCtx, campaignCode)
		if err != nil {
			return err
		}
		// STEP 2 — the idempotent answer, ABOVE the version check (contract §5.3).
		// Returning early is what makes "no second audit row" true rather than
		// aspirational: there is no code path below this line that can write one.
		if stored.ClosedAt != nil {
			closed = stored
			return nil
		}
		closed, err = uc.closeWithAudit(txCtx, actorAdminID, stored, req, reason, requestIP)
		return err
	})
	if err != nil {
		return nil, err
	}

	// FR-032: the response says `closed` rather than what the clock or the budget
	// would otherwise say. `deriveCampaignStatus` reads the reloaded row, so this is
	// the real stored state and not the pre-update one.
	uc.log.Info("campaign registry row closed",
		zap.String("campaign_code", closed.Code),
		zap.String("actor_user_id", actorAdminID.String()),
		zap.Int("version", closed.Version),
		zap.Time("closed_at", *closed.ClosedAt),
	)

	return mapper.MapCampaignSummaryRes(
		closed,
		mapper.CampaignGrantCounts{},
		nil,
		deriveCampaignStatus(closed, 0, 0, uc.now()),
	), nil
}

// closeWithAudit is the BODY of `Close`'s single transaction: the conditional
// `UPDATE`, the reload, and the audit outbox row — in that order, so that either
// all three stand or none does.
//
// IT IS A SEPARATE FUNCTION for the reason `insertWithAudit` is one: the ORDER of
// the two writes is FR-060/FR-061, and an order is only visible when the statements
// are neighbours. The reload sits between them deliberately — `closed_at` is the
// DATABASE's instant and the version is the STATEMENT's arithmetic, so neither is
// known to this process until the row is read back inside the same transaction.
func (uc *CampaignRegistryUseCase) closeWithAudit(
	ctx context.Context,
	actorAdminID uuid.UUID,
	existing *entities.Campaign,
	req *sub_dto.CampaignCloseReq,
	reason string,
	requestIP *string,
) (*entities.Campaign, error) {
	// `req.Version` IS THE ONE COMPARED, never `existing.Version`.
	//
	// Reading the row first and then comparing against what the read returned would
	// make the conditional UPDATE unconditional: it could never change zero rows for
	// a stale client, so FR-055's 412 would be unreachable and the second
	// administrator's close would silently overwrite the first one's state. The read
	// above exists to answer "does this campaign exist", not "what version is it" —
	// the two answers have to come from different places or one of them is theatre.
	updated, err := uc.campaignRepo.Close(ctx, existing.Code, req.Version)
	if err != nil {
		// The database refused. The operator's inputs were already correct — 412 and
		// 404 have both been ruled out above — so the cause goes to `Cause` and the
		// client is told to retry rather than invited to change the form.
		return nil, subscriptionerrors.ErrCampaignRegistryWriteFailed(existing.Code, err)
	}
	if !updated {
		// STEP 3 — the version moved between the read above and this statement. The
		// campaign still exists (step 1 proved it in this transaction), so this is
		// the 412 and not the 404 (FR-055, contract §4.3).
		return nil, subscriptionerrors.ErrCampaignVersionConflict(existing.Code)
	}

	// The reload, under the SHARE lock this transaction already holds from its own
	// UPDATE. It is what makes the response and the audit row carry the DATABASE's
	// closure instant and the STATEMENT's version instead of this process's guesses:
	// `closed_at = now()` is evaluated by PostgreSQL, and `version + 1` is computed
	// there too. Reading it back is one extra SELECT on a single-row primary key, on
	// an operation an operator performs a handful of times in a campaign's life.
	reloaded, err := uc.campaignRepo.GetByCodeForShare(ctx, existing.Code)
	if err != nil {
		return nil, subscriptionerrors.ErrCampaignRegistryReadFailed()
	}
	if reloaded == nil || reloaded.ClosedAt == nil {
		// Unreachable through this method: the UPDATE above reported one row changed,
		// and no delete path exists (FR-005). Reported rather than passed through,
		// because a nil dereference here would turn a data-layer surprise into a
		// crash on an operator-facing endpoint.
		return nil, subscriptionerrors.ErrCampaignRegistryWriteFailed(existing.Code, errRegistryReloadLostClose)
	}

	if err := uc.recordCloseAudit(
		ctx, actorAdminID, reloaded.Code, reason, existing.Version, *reloaded.ClosedAt, requestIP,
	); err != nil {
		return nil, err
	}
	return reloaded, nil
}

// errRegistryReloadLostClose reports that the reload after a successful `Close`
// found no row, or one without `closed_at`.
//
// It is a PACKAGE-LEVEL SENTINEL, the sanctioned exception to "no bare
// `errors.New`" (code-hygiene H8): it is a fact about the data layer rather than
// anything the operator did, so it has no HTTP status of its own and no
// `apperror` factory would be the right shape — the use case wraps it into
// `ErrCampaignRegistryWriteFailed` before it leaves this package.
//
// It is UNREACHABLE through `Close` on a healthy registry: the conditional UPDATE
// reported one row changed and no delete path exists (FR-005). It is named so that
// the unreachable branch reports what it saw instead of dereferencing a nil row on
// an administrator-facing endpoint.
var errRegistryReloadLostClose = errors.New("campaign registry: the closed row did not come back")
