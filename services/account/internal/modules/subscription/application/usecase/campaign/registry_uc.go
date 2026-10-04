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

// endsAtForAudit renders the optional window end for an audit payload.
//
// UTC and RFC3339, and the reason is that the payload is read WEEKS LATER by somebody
// comparing two rows: a value that keeps the writer's offset (`+07:00`) would render
// differently depending on which administrator's browser produced it, and two audit
// lines for the same change would not compare equal. `""` rather than null for an
// open-ended campaign, so the key is present and a reader does not have to distinguish
// "absent" from "empty".
func endsAtForAudit(endsAt *time.Time) string {
	if endsAt == nil {
		return ""
	}
	return endsAt.UTC().Format(time.RFC3339)
}
