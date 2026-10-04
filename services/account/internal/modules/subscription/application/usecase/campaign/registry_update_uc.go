package campaign

import (
	"context"
	"errors"

	admin_dto "smart-wardrobe-be/services/account/dto/admin"
	sub_dto "smart-wardrobe-be/services/account/dto/subscription"
	subscriptionerrors "smart-wardrobe-be/services/account/internal/modules/subscription/application/errors"
	"smart-wardrobe-be/services/account/internal/modules/subscription/application/mapper"
	"smart-wardrobe-be/services/account/internal/modules/subscription/domain/entities"
	"smart-wardrobe-be/shared/domain/constants/adminaudit"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

// 026 T052, extended by T061 (US4) — the EDIT path of `CampaignRegistryUseCase`,
// split out of `registry_uc.go`.
//
// WHY A SEPARATE FILE, and the same answer `registry_close_uc.go` gives: this is the
// longest of the three operations — an ordered set of answers, a merge, a window check
// that needs the merged row, a conditional statement, a reload and an audit payload —
// and with it inline the file holding the shared type and `Create` sat past the
// production size limit (code-hygiene H1). US4 then needed to add a rule to it, which
// is exactly what that limit is asking you to prevent.
//
// IT IS STILL ONE TYPE AND ONE TRANSACTION. The split is a FILE boundary and nothing
// else: the three functions below are methods of `CampaignRegistryUseCase`, declared in
// `registry_uc.go`, and an edit still enters `uow.Execute` exactly once
// (constitution §V).
//
// THE PARTS THAT STAYED IN `registry_uc.go`, and why they are not here:
//
//   - `requireRegistryRow`, because `Close` asks the same 404 question and one
//     implementation is what stops the two operations from answering it differently;
//   - `mergeCampaignUpdate` and `validateMergedCampaignWindow`, because they are PURE —
//     a reader can check both by hand without a transaction, which is the property that
//     makes US4's rules reviewable at all.

// Update edits a campaign that already exists (026 T052, FR-020..FR-023, FR-055,
// contract §4).
//
// THE ORDER OF THE ANSWERS IS THE WHOLE METHOD, exactly as in `Close` and for the same
// reason (contract §4.3):
//
//  1. `GetByCode` INSIDE this transaction: a code nobody typed is 404 at once. The
//     conditional `UPDATE ... WHERE code = ? AND version = ?` reports ZERO ROWS for
//     both "no such code" and "wrong version", so an implementation that consulted only
//     the row count would answer 412 to an operator who mistyped a code.
//  2. NOTHING CHANGED ⇒ 200 as a NO-OP: no `UPDATE`, no version bump, no audit row
//     (FR-055's exception, US3 scenario 4). This is ABOVE the version check on
//     purpose — a screen that resubmits an unchanged form must succeed quietly, and a
//     history that recorded "quota changed from 120 to 120" would be worse than no
//     history at all.
//  3. THE MERGED WINDOW (026 T061, FR-013): `ends_at > starts_at` is re-decided on the
//     row the edit would leave behind, because an omitted timestamp contributes its
//     STORED value. It is a PURE check and runs before the ledger read below, so an
//     operator whose form is wrong is told so without the database being consulted
//     twice.
//  4. THE BUDGET IMMUTABILITY CHECK (FR-020/FR-021): grants exist AND one of
//     `planSlug`/`quota`/`reserve` differs ⇒ 422 with the message that names the way
//     out. This is the APPLICATION layer of the two FR-021 demands; the
//     `trg_campaigns_immutable_budget` trigger is the other, and it is the only one
//     a hand-written SQL UPDATE cannot walk around. The check reads the ledger INSIDE
//     this transaction, so a grant committing in between cannot slip past it.
//  5. `UPDATE ... WHERE version = ?`, zero rows ⇒ 412, then the audit outbox row in the
//     SAME transaction (FR-055, FR-060/FR-061).
//
// STEP 3 ABOVE STEP 4 IS A CHOICE, and it is the one an operator notices: a form with
// both a broken window and a frozen budget reports the WINDOW, because that is the
// field they typed most recently and the one they can fix without abandoning the
// promotion. It also costs nothing to put first — the ledger read is the only I/O among
// the two.
//
// WHAT THE RESPONSE CARRIES, AND WHY IT IS NOT THE REQUEST ECHOED. `version` comes
// from the row re-read AFTER the statement, because `version + 1` is computed by
// PostgreSQL and nothing in this process knows it until the row is read back. A client
// that received the version it sent would send it again and be refused with 412 for its
// own successful edit.
func (uc *CampaignRegistryUseCase) Update(
	ctx context.Context,
	actorAdminID uuid.UUID,
	campaignCode string,
	req *sub_dto.CampaignUpdateReq,
	requestIP *string,
) (*sub_dto.CampaignSummaryRes, error) {
	// 400 and 422 BEFORE ANY I/O, as in `Create` and `Close`: a rejected edit opens no
	// transaction, takes no row lock and writes no outbox row.
	edit, err := validateCampaignUpdate(req)
	if err != nil {
		return nil, err
	}

	var updated *entities.Campaign
	var changes campaignChanges
	err = uc.uow.Execute(ctx, func(txCtx context.Context) error {
		stored, err := uc.requireRegistryRow(txCtx, campaignCode)
		if err != nil {
			return err
		}
		// The candidate is built from the STORED row, not from an empty entity: an
		// omitted field must keep its current value, and the repository's UPDATE writes
		// all five mutable columns unconditionally (so it cannot be asked to write
		// "leave this alone").
		candidate, moved := mergeCampaignUpdate(stored, edit)
		// STEP 2 — the no-op answer. Taken BEFORE the version check and before the
		// immutability check, because a request that changes nothing has nothing to
		// conflict with and nothing to refuse.
		if moved.empty() {
			updated = stored
			return nil
		}
		// STEP 3 — FR-013 on the MERGED row (026 T061). A merged window equal to the
		// stored one cannot be illegal, because the CHECK admitted it when it was
		// written; so reaching this branch means the request really did describe a
		// window that never existed.
		if err := validateMergedCampaignWindow(candidate); err != nil {
			return err
		}
		changes = moved
		updated, err = uc.updateWithAudit(txCtx, actorAdminID, stored, candidate, moved, req, requestIP)
		return err
	})
	if err != nil {
		return nil, err
	}

	// TWO MESSAGES, because FR-055's no-op changes nothing and a log line saying
	// "updated" for a request that wrote nothing would send whoever reads the trail
	// looking for a change that did not happen — the same confusion the no-op rule
	// exists to prevent in the AUDIT trail, repeated in the log.
	if changes.empty() {
		uc.log.Info("campaign registry update was a no-op",
			zap.String("campaign_code", updated.Code),
			zap.String("actor_user_id", actorAdminID.String()),
			zap.Int("version", updated.Version),
		)
	} else {
		uc.log.Info("campaign registry row updated",
			zap.String("campaign_code", updated.Code),
			zap.String("actor_user_id", actorAdminID.String()),
			zap.Int("version", updated.Version),
			zap.Strings("changed_fields", changes.names()),
		)
	}

	return mapper.MapCampaignSummaryRes(
		updated,
		mapper.CampaignGrantCounts{},
		nil,
		deriveCampaignStatus(updated, 0, 0, uc.now()),
	), nil
}

// updateWithAudit is the BODY of `Update`'s single transaction: the immutability
// check, the conditional `UPDATE`, the reload, and the audit outbox row.
//
// IT IS A SEPARATE FUNCTION for the reason `insertWithAudit` and `closeWithAudit`
// are: the ORDER of those writes is FR-060/FR-061, and an order is only visible when
// the statements are neighbours. The reload sits between them for the same reason it
// does on close — `version + 1` belongs to the STATEMENT, so the only honest source
// for the number the client must send next is the row itself.
//
// `req.Version` IS THE ONE COMPARED, never `stored.Version`, for the reason
// `closeWithAudit` gives: comparing against what the read returned would make the
// conditional UPDATE unconditional and FR-055's 412 unreachable.
func (uc *CampaignRegistryUseCase) updateWithAudit(
	ctx context.Context,
	actorAdminID uuid.UUID,
	stored *entities.Campaign,
	candidate *entities.Campaign,
	changes campaignChanges,
	req *sub_dto.CampaignUpdateReq,
	requestIP *string,
) (*entities.Campaign, error) {
	// STEP 4 — FR-020's application layer. Asked only when a BUDGET column is among
	// the changes, so moving `endsAt` on a live campaign costs no ledger read and is
	// never refused (FR-023).
	if changes.touchesBudget() {
		granted, err := uc.campaignRepo.ExistsClaims(ctx, stored.Code)
		if err != nil {
			return nil, subscriptionerrors.ErrCampaignRegistryReadFailed()
		}
		if granted {
			return nil, subscriptionerrors.ErrCampaignBudgetLocked(stored.Code)
		}
	}

	written, err := uc.campaignRepo.Update(ctx, candidate, req.Version)
	if err != nil {
		// 412 and 404 were both ruled out above, so a failure here is the database.
		return nil, subscriptionerrors.ErrCampaignRegistryWriteFailed(stored.Code, err)
	}
	if !written {
		// STEP 5 — the version moved between this transaction's read and its UPDATE.
		// The campaign still exists (step 1 proved it here), so this is the 412.
		return nil, subscriptionerrors.ErrCampaignVersionConflict(stored.Code)
	}

	reloaded, err := uc.campaignRepo.GetByCodeForShare(ctx, stored.Code)
	if err != nil {
		return nil, subscriptionerrors.ErrCampaignRegistryReadFailed()
	}
	if reloaded == nil {
		return nil, subscriptionerrors.ErrCampaignRegistryWriteFailed(stored.Code, errRegistryReloadLostUpdate)
	}

	if err := uc.recordUpdateAudit(ctx, actorAdminID, stored, reloaded, changes, requestIP); err != nil {
		return nil, err
	}
	return reloaded, nil
}

// recordUpdateAudit writes the ONE outbox row an edit that changed something owes
// (FR-060, contract §6.3).
//
// `payload.before`/`payload.after` CARRY ONLY THE FIELDS THAT MOVED, and that is what
// makes them readable. A payload repeating the whole campaign would make "what did
// this administrator actually change" a diff every reader had to compute, which is the
// work the audit trail exists to have already done — and it would put unchanged values
// on both sides of a comparison, so a reader could no longer tell a field that was left
// alone from one that was rewritten with its own value. A CLEARED `endsAt` is in that
// set with `""` on the `after` side (US4): the key is present precisely because the
// value is gone, and omitting it would read as "unchanged".
//
// `version` IS IN BOTH SIDES even though it is not something an operator typed: it is
// the optimistic-concurrency token, it moved, and an audit line that could not say
// which revision of the row it describes would be unusable for the concurrent-edit
// question FR-055 exists to answer.
func (uc *CampaignRegistryUseCase) recordUpdateAudit(
	ctx context.Context,
	actorAdminID uuid.UUID,
	before *entities.Campaign,
	after *entities.Campaign,
	changes campaignChanges,
	requestIP *string,
) error {
	err := uc.auditLogContract.Record(ctx, admin_dto.RecordAuditLogInput{
		ActorUserID: actorAdminID,
		Action:      string(adminaudit.CampaignUpdate),
		TargetType:  string(adminaudit.Campaign),
		// Intentionally nil: FR-063, identically to create and close.
		TargetID: nil,
		Payload: map[string]any{
			"campaignCode": after.Code,
			"before":       changes.auditSide(before),
			"after":        changes.auditSide(after),
		},
		RequestIP: requestIP,
	})
	if err != nil {
		// FR-060/FR-061, and the same line as the other two operations: the outbox row
		// is part of THIS transaction, so returning the error rolls the edit back with
		// it. A row whose values and whose history disagree is what FR-062 forbids.
		return subscriptionerrors.ErrCampaignRegistryAuditFailed(after.Code, err)
	}
	return nil
}

// errRegistryReloadLostUpdate reports that the reload after a successful `Update`
// found no row.
//
// A PACKAGE-LEVEL SENTINEL for the same reason as `errRegistryReloadLostClose`
// (code-hygiene H8's sanctioned exception): it is a fact about the data layer rather
// than anything the operator did, so it has no HTTP status of its own and no
// `apperror` factory would be the right shape.
//
// UNREACHABLE on a healthy registry — the conditional UPDATE reported one row
// changed, and no delete path exists (FR-005). It is named so the branch reports what
// it saw instead of dereferencing a nil row on an administrator-facing endpoint.
var errRegistryReloadLostUpdate = errors.New("campaign registry: the updated row did not come back")
