package campaign

import (
	"context"
	"time"

	admincontract "smart-wardrobe-be/services/account/contract/admin"
	identitydto "smart-wardrobe-be/services/account/dto/identity"
	notificationcontract "smart-wardrobe-be/services/account/internal/modules/notification/presentation/contract"
	subscriptionerrors "smart-wardrobe-be/services/account/internal/modules/subscription/application/errors"
	subscriptioncaching "smart-wardrobe-be/services/account/internal/modules/subscription/application/interface/caching"
	uc_interfaces "smart-wardrobe-be/services/account/internal/modules/subscription/application/interface/usecase"
	"smart-wardrobe-be/services/account/internal/modules/subscription/application/usecase/subscription"
	"smart-wardrobe-be/services/account/internal/modules/subscription/domain/entities"
	"smart-wardrobe-be/services/account/internal/modules/subscription/domain/repositories"
	sharedevent "smart-wardrobe-be/shared/application/event"
	sharedconfig "smart-wardrobe-be/shared/config"
	sharedrepos "smart-wardrobe-be/shared/domain/repositories"
	"smart-wardrobe-be/shared/pkg/logger"
	"smart-wardrobe-be/shared/pkg/utils/timeutils"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

// CampaignUseCase is the application service behind the signup-campaign
// mechanism (025 T044).
//
// ONE method, because there is exactly one trigger: the `identity.user.created`
// message. The catch-up sweep of Phase 10 re-enters through this same method
// rather than a second one, so the eligibility rules have a single implementation
// and the two paths cannot drift apart (FR-080). That is the same reasoning as
// `PlanChangePublisher` (FR-011/FR-012): make the wrong thing a wiring error
// rather than a code-review convention.
//
// LAYERING. Conditions 1-7 live in `admission_uc.go` and need no lock;
// conditions 8-9 live in `quota_uc.go` and must run inside a transaction. This
// file is the only place that opens one, and it opens exactly ONE.
type CampaignUseCase struct {
	cfg                 *sharedconfig.Config
	claimRepo           repositories.ICampaignClaimRepository
	subRepo             repositories.IUserSubscriptionRepository
	planRepo            repositories.ISubscriptionPlanRepository
	planChangePublisher *subscription.PlanChangePublisher
	auditLogContract    admincontract.IAuditLogContract
	notifContract       notificationcontract.INotificationContract
	overviewCache       subscriptioncaching.ISubscriptionOverviewCache
	uow                 sharedrepos.IUnitOfWork
	log                 logger.Interface
	// now is the clock. A field rather than a direct `timeutils.GetNow` call so
	// that a test can pin "now" to the campaign's window; production leaves it nil
	// and every read goes through the configured zone (R6, constitution §VIII).
	now func() time.Time
}

func NewCampaignUseCase(
	cfg *sharedconfig.Config,
	claimRepo repositories.ICampaignClaimRepository,
	subRepo repositories.IUserSubscriptionRepository,
	planRepo repositories.ISubscriptionPlanRepository,
	planChangePublisher *subscription.PlanChangePublisher,
	auditLogContract admincontract.IAuditLogContract,
	notifContract notificationcontract.INotificationContract,
	// overviewCache is the full cache interface, not a one-method slice of it, and
	// that is forced by `InvalidateOverviewCache` — the shared post-commit helper
	// the boundary test requires every plan-change path to use. The narrow
	// interface that would have been nicer to declare cannot be, because the helper
	// is the thing T058 requires and it takes the wide one.
	//
	// The consequence is that this type COULD read the cache, and the protection
	// against it doing so is that the grant reads the subscription row under
	// `FOR UPDATE` rather than consulting the cache at all: a stale entry can never
	// decide what a customer is entitled to, whichever interface it arrives through.
	overviewCache subscriptioncaching.ISubscriptionOverviewCache,
	uow sharedrepos.IUnitOfWork,
	log logger.Interface,
) uc_interfaces.ICampaignUseCase {
	return &CampaignUseCase{
		cfg:                 cfg,
		claimRepo:           claimRepo,
		subRepo:             subRepo,
		planRepo:            planRepo,
		planChangePublisher: planChangePublisher,
		auditLogContract:    auditLogContract,
		notifContract:       notifContract,
		overviewCache:       overviewCache,
		uow:                 uow,
		log:                 log,
		now:                 func() time.Time { return timeutils.GetNow(cfg.Database.TimeZone) },
	}
}

// ProcessUserCreated evaluates one account-creation message.
//
// A SKIP IS A SUCCESS. Returning nil for "not eligible, and here is why" is what
// lets a campaign that is switched off, or full, or does not apply to the account
// behave normally: the message is acknowledged, the queue keeps moving, and
// nothing anywhere reports a failure. A non-nil error means the decision could
// not be made at all, and the message must be retried or dead-lettered.
func (uc *CampaignUseCase) ProcessUserCreated(ctx context.Context, payload identitydto.UserCreatedEventPayload) error {
	// A message with no account or no registration instant cannot be evaluated:
	// condition 7 compares `registered_at` against the campaign window, and the
	// zero time is 2026 years before every campaign. Redelivering it would produce
	// the same wrong answer every time, so it is dead-lettered rather than retried
	// (FR-062: a message that cannot be fixed must not block the queue).
	if payload.UserID == uuid.Nil {
		return sharedevent.PermanentError{Err: subscriptionerrors.ErrCampaignMessageMalformed("userId")}
	}
	if payload.RegisteredAt.IsZero() {
		return sharedevent.PermanentError{Err: subscriptionerrors.ErrCampaignMessageMalformed("registeredAt")}
	}

	// ONE clock read for the whole evaluation. Conditions 1-7 must all see the
	// same instant or a message landing exactly on `ends_at` gets two different
	// answers from two different rules.
	now := uc.now()

	// Conditions 1-7: cheap, read-only, no lock, no transaction.
	sub, err := uc.subRepo.GetByUserID(ctx, payload.UserID)
	if err != nil {
		return sharedevent.RetryableError{Err: subscriptionerrors.ErrCampaignReadSubscriptionFailed(payload.UserID)}
	}
	evaluator := newCampaignEvaluator(uc.cfg.Campaigns, now)
	verdict := evaluator.evaluate(ctx, payload.UserID, payload.RegisteredAt, payload.CreatedByAdmin, sub)
	if verdict.Campaign == nil {
		uc.logSkip(payload.UserID, "", verdict.Reason)
		return nil
	}

	campaign := *verdict.Campaign

	// FR-010: the configured plan must exist AND be active. Loading configuration
	// cannot check this — it must not reach into the database — so it is checked
	// here, where the plan row is read anyway. Granting an inactive plan grants
	// something with no commercial value.
	plan, err := uc.planRepo.GetBySlug(ctx, campaign.PlanSlug)
	if err != nil {
		return sharedevent.RetryableError{Err: subscriptionerrors.ErrCampaignLoadPlanFailed(campaign.PlanSlug)}
	}
	if plan == nil {
		uc.logSkip(payload.UserID, campaign.Code, SkipPlanUnavailable)
		return nil
	}

	// Conditions 8-9 and the grant: all inside ONE transaction.
	//
	// NOT NESTED (R3). `GormUnitOfWork.Execute` calls `db.Transaction` on the BASE
	// handle (shared/infrastructure/db/uow.go:69), not on whatever handle the
	// context carries. A nested `Execute` therefore opens a SECOND transaction on a
	// SECOND connection: the advisory lock would be taken in the inner one, the
	// counts read in the outer one could not see the inner one's rows, and the
	// ceiling would be enforced against a stale number. This is the single most
	// dangerous mistake available in this codebase, so it happens in exactly one
	// place and the helpers below take a plain `context.Context`.
	var outcome quotaOutcome
	err = uc.uow.Execute(ctx, func(txCtx context.Context) error {
		decider := newQuotaDecider(uc.claimRepo)
		decided, derr := decider.decideAndClaim(txCtx, claimInput{
			Campaign:     campaign,
			Plan:         plan,
			UserID:       payload.UserID,
			RegisteredAt: payload.RegisteredAt,
			Now:          now,
		})
		if derr != nil {
			return derr
		}
		if !decided.Granted {
			uc.logSkip(payload.UserID, campaign.Code, decided.Reason)
			return nil
		}
		outcome = decided

		return uc.applyGrant(txCtx, payload, campaign, plan, decided, now)
	})
	if err != nil {
		// FR-039: the transaction rolled back, so there is no claim and no plan.
		// A claim without a plan would be found by the next catch-up sweep and
		// granted then; a plan without a claim would be granted a SECOND time.
		return sharedevent.RetryableError{Err: err}
	}

	if !outcome.Granted {
		return nil
	}
	uc.logGranted(ctx, payload.UserID, campaign, plan, outcome)

	// FR-040/T058: the cached overview still describes the OLD plan, so it is
	// dropped HERE — after the transaction committed. Doing it inside the
	// transaction would be wrong in the other direction: a rollback would leave the
	// customer with no cached overview at all, which is harmless but pointless,
	// whereas a stale entry that survives a successful grant is not.
	//
	// The caller's own context is passed, not discarded. The shared invalidator
	// applies `context.WithoutCancel` internally and gives the cache its own
	// short-lived context, so a cancelled caller context cannot silently skip the
	// invalidation — which is what T058's "no `_ =` on the caller's ctx" is about.
	uc.invalidateOverviewAfterCommit(ctx, payload.UserID)
	return nil
}

// logSkip records why an account was not granted (FR-061).
//
// Every skip names a reason code, and the code is stable text rather than an
// index, because these lines are read by an operator deciding whether to act and
// they outlive the binary that wrote them (FR-062).
func (uc *CampaignUseCase) logSkip(userID uuid.UUID, campaignCode string, reason SkipReason) {
	fields := []zap.Field{
		zap.String("user_id", userID.String()),
		zap.String("reason", string(reason)),
	}
	if campaignCode != "" {
		fields = append(fields, zap.String("campaign_code", campaignCode))
	}
	uc.log.Info("campaign account skipped", fields...)
}

// logGranted records a successful grant with the facts an operator needs later
// (FR-060): which campaign, which plan, which account, when.
//
// `grant_trigger` says which door the account came through — the queue or the
// catch-up sweep. FR-060's fields are identical on both paths by design (FR-080),
// so without this one field a grant recovered by the sweep is indistinguishable
// from one the message path handled on time, and the FR-082 latency signal cannot
// be read from the logs (see `grant_uc.go`).
func (uc *CampaignUseCase) logGranted(ctx context.Context, userID uuid.UUID, campaign sharedconfig.CampaignConfig, plan *entities.SubscriptionPlan, outcome quotaOutcome) {
	uc.log.Info("campaign plan granted",
		zap.String("user_id", userID.String()),
		zap.String("campaign_code", campaign.Code),
		zap.String("plan_slug", plan.Slug),
		zap.String("grant_bucket", string(outcome.Bucket)),
		zap.String("grant_trigger", string(grantOriginOf(ctx))),
		zap.Time("granted_at", outcome.ExpiresAt),
	)
}
