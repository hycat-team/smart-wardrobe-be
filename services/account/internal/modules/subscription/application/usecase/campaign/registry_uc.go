package campaign

import (
	"context"
	"errors"
	"time"

	admincontract "smart-wardrobe-be/services/account/contract/admin"
	admin_dto "smart-wardrobe-be/services/account/dto/admin"
	sub_dto "smart-wardrobe-be/services/account/dto/subscription"
	subscriptionerrors "smart-wardrobe-be/services/account/internal/modules/subscription/application/errors"
	uc_interfaces "smart-wardrobe-be/services/account/internal/modules/subscription/application/interface/usecase"
	"smart-wardrobe-be/services/account/internal/modules/subscription/application/mapper"
	"smart-wardrobe-be/services/account/internal/modules/subscription/domain/entities"
	"smart-wardrobe-be/services/account/internal/modules/subscription/domain/repositories"
	sharedconfig "smart-wardrobe-be/shared/config"
	"smart-wardrobe-be/shared/domain/constants/adminaudit"
	sharedrepos "smart-wardrobe-be/shared/domain/repositories"
	"smart-wardrobe-be/shared/pkg/logger"
	"smart-wardrobe-be/shared/pkg/utils/timeutils"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

// CampaignRegistryUseCase is the WRITE side of the campaign registry (026 T033,
// FR-052..FR-055, FR-060..FR-063).
//
// THIS IS THE FEATURE. Everything else in 025 and 026 exists so that this type can
// exist: a campaign is a row of `subscription.campaigns` written by an administrator
// through an audited API, so opening one is a request instead of a commit, a tag and a
// deploy (spec.md §2.2). Until this method exists, the only way to open a campaign is
// to ship binaries — which is the whole problem the feature was written to remove.
//
// IT IS A SEPARATE TYPE FROM `CampaignUseCase` AND FROM `CampaignQueryUseCase`, and
// `ICampaignRegistryUseCase`'s header gives the full argument. One line of it belongs
// here: this type holds a unit of work and an audit contract, and neither of the other
// two does — so FR-069's read-only guarantee cannot be weakened by a change made here,
// and a grant can never be routed through an operator action.
//
// ONE TRANSACTION PER OPERATION, NEVER NESTED (constitution §V, 025 R3). Each method
// below enters `uow.Execute` exactly once and nothing it calls opens another:
// `GormUnitOfWork.Execute` opens a transaction on the BASE handle, so a nested call
// would take a second connection, and the row read in the outer transaction would not
// see the inner one's writes. That is the single most dangerous mistake available in
// this codebase, so the transaction appears once per method and nowhere else.
type CampaignRegistryUseCase struct {
	cfg              *sharedconfig.Config
	campaignRepo     repositories.ICampaignRepository
	auditLogContract admincontract.IAuditLogContract
	uow              sharedrepos.IUnitOfWork
	log              logger.Interface
	// now is the clock, for the same reason it is a field on every other campaign
	// type: one read per call, pin-able by a test, never `time.Now()` (R6,
	// constitution §VIII).
	now func() time.Time
}

// NewCampaignRegistryUseCase builds the registry writer.
//
// IT TAKES NO PLAN REPOSITORY, and that omission is a requirement rather than a
// simplification. FR-014 keeps 025's decision that `plan_slug` is checked at RUN time
// — a write must not need the plan catalogue to be readable, and a paused plan must not
// be able to block an operator from opening a campaign (FR-003 forbids configuration
// and now this write path from treating the catalogue as a precondition). The grant
// path already performs that check, where the plan row is read anyway, and skips with
// `plan_unavailable` when it is absent.
func NewCampaignRegistryUseCase(
	cfg *sharedconfig.Config,
	campaignRepo repositories.ICampaignRepository,
	auditLogContract admincontract.IAuditLogContract,
	uow sharedrepos.IUnitOfWork,
	log logger.Interface,
) uc_interfaces.ICampaignRegistryUseCase {
	return &CampaignRegistryUseCase{
		cfg:              cfg,
		campaignRepo:     campaignRepo,
		auditLogContract: auditLogContract,
		uow:              uow,
		log:              log,
		now:              func() time.Time { return timeutils.GetNow(cfg.Database.TimeZone) },
	}
}

// Create opens a campaign and returns it as the read endpoints would render it
// (contract `campaign-registry-api.md` §3, FR-052).
//
// FR-060 AND FR-061 ARE THE SAME LINE OF CODE. `IAuditLogContract.Record` publishes
// through the transactional OUTBOX (constitution §III), so its row joins this
// transaction; returning its error rolls the campaign row back. There is therefore no
// successful creation without an audit trail, and no audit trail without a creation —
// neither outcome can be produced by accident, which is the only way FR-061's "a failed
// write leaves no audit record" can be true.
//
// NOTE WHAT IS NOT HERE. No plan lookup, no cache invalidation, no notification, and no
// event about the campaign itself: opening a campaign grants nobody anything, so it has
// no subscriber to tell (FR-033). Adding a `subscription.plan.changed` publish here
// would claim a customer's rights changed when none did.
func (uc *CampaignRegistryUseCase) Create(
	ctx context.Context,
	actorAdminID uuid.UUID,
	req *sub_dto.CampaignCreateReq,
	requestIP *string,
) (*sub_dto.CampaignSummaryRes, error) {
	// 400 / 422 BEFORE ANY I/O: a rejected campaign opens no transaction and writes no
	// outbox row, so a typo cannot produce a trace that says somebody tried.
	input, err := validateCampaignCreate(req)
	if err != nil {
		return nil, err
	}

	var created *entities.Campaign
	err = uc.uow.Execute(ctx, func(txCtx context.Context) error {
		inserted, err := uc.insertWithAudit(txCtx, actorAdminID, input, requestIP)
		if err != nil {
			return err
		}
		created = inserted
		return nil
	})
	if err != nil {
		return nil, err
	}

	uc.log.Info("campaign registry row created",
		zap.String("campaign_code", created.Code),
		zap.String("plan_slug", created.PlanSlug),
		zap.Int("quota", created.Quota),
		zap.Int("reserve", created.Reserve),
		zap.String("actor_user_id", actorAdminID.String()),
	)

	// The response is the SAME SHAPE the read endpoints return, built by the SAME
	// mapper (026 T032), so an admin screen renders the campaign it just created and
	// then refreshes instead of reformatting one shape into another.
	//
	// Zero counts, because a campaign nobody has been granted from has granted nobody.
	// `status` is DERIVED FROM THE CLOCK rather than echoed as "running": a campaign
	// opened with `starts_at` in the past is running, one opened for next month is
	// `not_started`, and answering 201 with a hard-coded state would make the operator
	// reload to discover which they got.
	return mapper.MapCampaignSummaryRes(
		created,
		mapper.CampaignGrantCounts{},
		nil,
		deriveCampaignStatus(created, 0, 0, uc.now()),
	), nil
}

// insertWithAudit is the BODY of `Create`'s single transaction: write the row, then the
// audit outbox row, and let either failure undo both.
//
// IT IS A SEPARATE FUNCTION, and not a cosmetic one, because the ordering between the
// two writes is the requirement (FR-060/FR-061) and an ordering is only visible when
// the two statements are neighbours. A closure that also validated the request, opened
// the transaction, logged and built the response would spread the order across eighty
// lines, and the next operation added to this file would have to re-derive it.
func (uc *CampaignRegistryUseCase) insertWithAudit(
	ctx context.Context,
	actorAdminID uuid.UUID,
	input campaignWrite,
	requestIP *string,
) (*entities.Campaign, error) {
	// `Version: 1` IS WRITTEN RATHER THAN LEFT TO THE COLUMN DEFAULT, and the response
	// depends on it: the client stores this value and sends it back as the If-Match of
	// the next write (FR-055). Reading it back to learn the version would be a second
	// query for a number this statement already decided, and it would leave the client
	// holding a version the row might no longer have.
	campaign := &entities.Campaign{
		Code:     input.code,
		PlanSlug: input.planSlug,
		Quota:    input.quota,
		Reserve:  input.reserve,
		StartsAt: input.startsAt,
		EndsAt:   input.endsAt,
		Version:  1,
	}
	if err := uc.campaignRepo.Create(ctx, campaign); err != nil {
		// 409 IS THE CONSTRAINT'S ANSWER, NOT A PRE-READ'S. `GetByCode` followed by
		// an insert has a window between the two statements, and that window is
		// exactly the case FR-054 assigns 409 to — two administrators opening the
		// same code at the same moment.
		if errors.Is(err, repositories.ErrCampaignCodeExists) {
			return nil, subscriptionerrors.ErrCampaignDuplicateCode(input.code)
		}
		// Anything else reached the database and the database said no. The message
		// must not invite the operator to change an input that was already correct
		// (data-model.md §2.1), so the cause goes to `Cause` and the client is told
		// to retry.
		return nil, subscriptionerrors.ErrCampaignRegistryWriteFailed(input.code, err)
	}

	if err := uc.recordCreateAudit(ctx, actorAdminID, campaign, requestIP); err != nil {
		return nil, err
	}
	return campaign, nil
}

// recordCreateAudit writes the ONE outbox row FR-060 requires, inside the caller's
// transaction.
//
// FR-063'S THREE FACTS ARE ALL IN HERE, and each is a constant rather than a literal
// because the read endpoint filters on them:
//
//   - `targetType = "campaign"`, its own target type: a campaign open moves nobody's
//     rights (FR-033), so filing it under `subscription_plan` or `user_subscription`
//     would put it in the answer to "who gave away Premium" when it answered nothing of
//     the sort;
//   - `targetId` is NIL, because the existing audit filter takes a UUID while a campaign
//     is keyed by a text code;
//   - the code itself travels in `payload.campaignCode`, which is what makes
//     `GET /admin/campaigns/{code}/audit` able to find the row at all.
//
// `payload.before`/`payload.after` ARE ABSENT HERE, and that is not an omission: there
// is no "before" for a campaign that did not exist. They appear on update and close,
// where there is something on both sides (contract §6.3).
func (uc *CampaignRegistryUseCase) recordCreateAudit(
	ctx context.Context,
	actorAdminID uuid.UUID,
	campaign *entities.Campaign,
	requestIP *string,
) error {
	err := uc.auditLogContract.Record(ctx, admin_dto.RecordAuditLogInput{
		ActorUserID: actorAdminID,
		Action:      string(adminaudit.CampaignCreate),
		TargetType:  string(adminaudit.Campaign),
		// Intentionally nil: FR-063. See the header above.
		TargetID: nil,
		Payload: map[string]any{
			"campaignCode": campaign.Code,
			"planSlug":     campaign.PlanSlug,
			"quota":        campaign.Quota,
			"reserve":      campaign.Reserve,
			"startsAt":     campaign.StartsAt.UTC().Format(time.RFC3339),
			"endsAt":       endsAtForAudit(campaign.EndsAt),
		},
		RequestIP: requestIP,
	})
	if err != nil {
		// FR-060/FR-061: the outbox row is part of THIS transaction, so returning the
		// error rolls the campaign row back with it. Continuing here would produce a
		// campaign nobody can be shown to have opened — the exact thing FR-062 says
		// must not be writable outside an audited path.
		return subscriptionerrors.ErrCampaignRegistryAuditFailed(campaign.Code, err)
	}
	return nil
}

// ---- US3: the update path ----
//
// `recordCloseAudit`, `Close` and `closeWithAudit` were moved OUT of this file to
// `registry_close_uc.go`: they are the force-close path, and with them inline this
// file sat past the production size limit (code-hygiene H1) before US3 added a
// single line. The split is a FILE boundary only — one type, and one transaction per
// operation either way.

// requireRegistryRow is the 404 half of FR-054's ordering, asked once so `Close`
// and `Update` cannot answer it differently.
//
// `nil, nil` from the repository is "absent"; an error is a database fault. The
// two must not collapse into one another: an outage reported as "this campaign does
// not exist" sends an operator looking for a code that has been running for months
// (the same argument `campaign_query_uc.requireCampaign` makes for the look-up).
func (uc *CampaignRegistryUseCase) requireRegistryRow(
	ctx context.Context,
	campaignCode string,
) (*entities.Campaign, error) {
	campaign, err := uc.campaignRepo.GetByCode(ctx, campaignCode)
	if err != nil {
		return nil, subscriptionerrors.ErrCampaignRegistryReadFailed()
	}
	if campaign == nil {
		return nil, subscriptionerrors.ErrCampaignNotFound(campaignCode)
	}
	return campaign, nil
}

// Update edits a campaign that already exists (026 T052, FR-020..FR-023, FR-055,
// contract §4).
//
// THE ORDER OF THE FOUR ANSWERS IS THE WHOLE METHOD, exactly as in `Close` and for
// the same reason (contract §4.3):
//
//  1. `GetByCode` INSIDE this transaction: a code nobody typed is 404 at once. The
//     conditional `UPDATE ... WHERE code = ? AND version = ?` reports ZERO ROWS for
//     both "no such code" and "wrong version", so an implementation that consulted
//     only the row count would answer 412 to an operator who mistyped a code.
//  2. NOTHING CHANGED ⇒ 200 as a NO-OP: no `UPDATE`, no version bump, no audit row
//     (FR-055's exception, US3 scenario 4). This is ABOVE the version check on
//     purpose — a screen that resubmits an unchanged form must succeed quietly, and
//     a history that recorded "quota changed from 120 to 120" would be worse than no
//     history at all.
//  3. THE BUDGET IMMUTABILITY CHECK (FR-020/FR-021): grants exist AND one of
//     `planSlug`/`quota`/`reserve` differs ⇒ 422 with the message that names the way
//     out. This is the APPLICATION layer of the two FR-021 demands; the
//     `trg_campaigns_immutable_budget` trigger is the other, and it is the only one
//     a hand-written SQL UPDATE cannot walk around. The check reads the ledger
//     INSIDE this transaction, so a grant committing in between cannot slip past it.
//  4. `UPDATE ... WHERE version = ?`, zero rows ⇒ 412, then the audit outbox row in
//     the SAME transaction (FR-055, FR-060/FR-061).
//
// WHAT THE RESPONSE CARRIES, AND WHY IT IS NOT THE REQUEST ECHOED. `version` comes
// from the row re-read AFTER the statement, because `version + 1` is computed by
// PostgreSQL and nothing in this process knows it until the row is read back. A
// client that received the version it sent would send it again and be refused with
// 412 for its own successful edit.
//
// `ends_at > starts_at` IS NOT RE-CHECKED HERE AGAINST THE MERGED WINDOW. Merging
// the two timestamps with the stored ones and re-deciding the rule is US4's task
// (T061), together with `endsAt: null` clearing the window; until it lands an absent
// timestamp keeps the stored value, which is the safe half — it cannot silently end a
// campaign that is running.
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
		// omitted field must keep its current value, and the repository's UPDATE
		// writes all five mutable columns unconditionally (so it cannot be asked to
		// write "leave this alone").
		candidate, moved := mergeCampaignUpdate(stored, edit)
		// STEP 2 — the no-op answer. Taken BEFORE the version check and before the
		// immutability check, because a request that changes nothing has nothing to
		// conflict with and nothing to refuse.
		if moved.empty() {
			updated = stored
			return nil
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
	// STEP 3 — FR-020's application layer. Asked only when a BUDGET column is among
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
		// STEP 4 — the version moved between this transaction's read and its UPDATE.
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
// this administrator actually change" a diff every reader had to compute, which is
// the work the audit trail exists to have already done — and it would put unchanged
// values on both sides of a comparison, so a reader could no longer tell a field that
// was left alone from one that was rewritten with its own value.
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
