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

// ---- US2 and US3: the other two operations, each in a file of its own ----
//
// `recordCloseAudit`, `Close` and `closeWithAudit` live in `registry_close_uc.go`, and
// `Update`, `updateWithAudit` and `recordUpdateAudit` in `registry_update_uc.go`. The
// force-close path was moved first because with it inline this file sat past the
// production size limit (code-hygiene H1) before US3 added a single line, and the edit
// path followed for the same reason: it is the longest of the three operations, and
// US4 then had to add a rule to it. Reading one operation per file is also what lets a
// reader answer "what does closing do" without reading a create.
//
// The splits are FILE boundaries and nothing else — one type, and exactly one
// transaction per operation either way. `requireRegistryRow` stayed HERE because
// `Close` and `Update` ask the same 404 question and one implementation is what stops
// the two from answering it differently (FR-054: one code per failure).

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
