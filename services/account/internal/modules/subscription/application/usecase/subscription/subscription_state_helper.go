package subscription

import (
	"context"
	"fmt"
	"time"

	subscriptionerrors "smart-wardrobe-be/services/account/internal/modules/subscription/application/errors"
	"smart-wardrobe-be/services/account/internal/modules/subscription/domain/constants/plankind"
	"smart-wardrobe-be/services/account/internal/modules/subscription/domain/entities"
	"smart-wardrobe-be/services/account/internal/modules/subscription/domain/repositories"
	"smart-wardrobe-be/shared/domain/constants/subscriptioneventtype"
	shared_repos "smart-wardrobe-be/shared/domain/repositories"

	"github.com/google/uuid"
)

type SubscriptionStateSupport struct {
	userSubRepo repositories.IUserSubscriptionRepository
	planRepo    repositories.ISubscriptionPlanRepository
	quotaRepo   repositories.IUserDailyQuotaRepository
	// 024 T049: default provisioning has to publish, so the state support holds
	// the same single publish point every other plan change uses. It brings the
	// history repository and the outbox publisher with it, which is why they do
	// not appear as separate dependencies here.
	planChangePublisher *PlanChangePublisher
	// 024 T126: provisioning writes THREE things - the subscription row, the
	// history row and the outbox row - and constitution §III requires the event
	// record to land in the same transaction as the business operation. Without
	// this dependency the three writes each fell back to the base handle and
	// committed on their own, which was the single remaining call site that broke
	// FR-013 + FR-015.
	uow shared_repos.IUnitOfWork
}

func NewSubscriptionStateSupport(
	userSubRepo repositories.IUserSubscriptionRepository,
	planRepo repositories.ISubscriptionPlanRepository,
	quotaRepo repositories.IUserDailyQuotaRepository,
	planChangePublisher *PlanChangePublisher,
	uow shared_repos.IUnitOfWork,
) *SubscriptionStateSupport {
	return &SubscriptionStateSupport{
		userSubRepo:         userSubRepo,
		planRepo:            planRepo,
		quotaRepo:           quotaRepo,
		planChangePublisher: planChangePublisher,
		uow:                 uow,
	}
}

// GetOrCreateUserSubscription returns the account's subscription, provisioning
// the default free plan when there is none.
//
// 024 T049: provisioning now writes a history row and publishes an event, and
// the plan kind comes from the plan row instead of a hardcoded
// plankind.DefaultFree. Before this, default provisioning was a third path that
// produced no history and no event, so a brand-new account had no AI policy
// grant at all — not even a free one — until something else changed its plan.
// That is the same root cause as the manual grant path, and it is why accounts
// created by an administrator were the worst affected: they received the free
// plan late and silently.
//
// 024 T126: the writes are now inside ONE transaction. The existence check stays
// outside it because it is the fast path for the overwhelmingly common case (the
// subscription already exists) and holding a transaction open for a read would
// serialise every subscription read in the system. Everything that WRITES -
// the row, the re-read that observes it, the history row and the outbox row -
// runs under `txCtx`, so a publish failure rolls the provisioning back instead
// of leaving an account with a plan that the AI service never heard about.
func (s *SubscriptionStateSupport) GetOrCreateUserSubscription(ctx context.Context, userID uuid.UUID) (*entities.UserSubscription, error) {
	sub, err := s.userSubRepo.GetByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if sub != nil {
		return sub, nil
	}

	defaultPlan, err := s.planRepo.GetDefaultPlan(ctx)
	if err != nil {
		return nil, err
	}
	if defaultPlan == nil {
		return nil, subscriptionerrors.ErrDefaultPlanConfigNotFound()
	}

	var provisioned *entities.UserSubscription
	err = s.uow.Execute(ctx, func(txCtx context.Context) error {
		sub, err := s.provisionDefaultOn(txCtx, userID, defaultPlan)
		if err != nil {
			return err
		}
		provisioned = sub
		return nil
	})
	if err != nil {
		return nil, err
	}
	return provisioned, nil
}

// EnsureDefaultSubscription provisions the default free plan for a brand-new
// account, writing the subscription row, the history row and the outbox record
// on the transaction ALREADY OPEN in ctx.
//
// 024 T196: this is the eager half of FR-013. Provisioning used to happen only
// on the first read of a subscription (SC-006 requires it within 60s of account
// creation with no read at all), so an administrator-created account reached the
// AI service with no policy grant for as long as nobody asked it anything.
//
// Why it does NOT open a unit of work: the caller is the identity use case that
// has just inserted the user row, and `user_subscriptions.user_id` is a foreign
// key onto `identity.users(id)`. A second `uow.Execute` would run on a different
// connection, where the uncommitted user row does not exist, so the insert
// would fail the FK check - or worse, succeed against a stale committed row and
// leave the entitlement attached to nothing. Sharing the caller's transaction is
// what makes "the account and its entitlement commit or roll back together" true
// rather than aspirational.
//
// Idempotent, and that is the property the whole eager path rests on: the lazy
// `GetOrCreateUserSubscription` still runs for accounts created any other way
// (rows that predate this task), and for accounts created before a retried
// request. Both doors open the same provisioning body, so a second run must
// produce no second record. `ProvisionDefault` is `ON CONFLICT (user_id) DO
// NOTHING` and the event key is `DEFAULT_PROVISION:{userID}:{version}` on a
// UNIQUE column, which is exactly the pair of guards `TestLazyProvisioningAfterEager`
// exercises.
//
// Returns nil without writing anything when the account already has a
// subscription, so an account that was granted a plan before this call is left
// alone.
func (s *SubscriptionStateSupport) EnsureDefaultSubscription(ctx context.Context, userID uuid.UUID) error {
	existing, err := s.userSubRepo.GetByUserID(ctx, userID)
	if err != nil {
		return err
	}
	if existing != nil {
		return nil
	}

	defaultPlan, err := s.planRepo.GetDefaultPlan(ctx)
	if err != nil {
		return err
	}
	if defaultPlan == nil {
		return subscriptionerrors.ErrDefaultPlanConfigNotFound()
	}

	_, err = s.provisionDefaultOn(ctx, userID, defaultPlan)
	return err
}

// provisionDefaultOn writes the subscription row, re-reads it and publishes the
// activation event, all on the transaction carried by ctx.
//
// It is the single body both provisioning doors run, so the eager and the lazy
// path cannot drift: same plan source, same benefit snapshot, same event key.
func (s *SubscriptionStateSupport) provisionDefaultOn(ctx context.Context, userID uuid.UUID, defaultPlan *entities.SubscriptionPlan) (*entities.UserSubscription, error) {
	now := time.Now()
	sub := &entities.UserSubscription{
		UserID:             userID,
		SubscriptionPlanID: defaultPlan.ID,
		CurrentPlanCode:    defaultPlan.Slug,
		CurrentTierRank:    defaultPlan.TierRank,
		// From the plan row, not a literal. Hardcoding DefaultFree here is what
		// made a re-pointed default plan silently produce a subscription whose
		// kind disagreed with its plan.
		CurrentPlanKind: defaultPlan.PlanKind,
		// 024 T170: this path wrote an empty object while the batch path wrote
		// BuildPlanBenefitSnapshot(defaultPlan), so the same default plan
		// produced two different snapshots depending on which door the account
		// came through.
		//
		// FR-070/FR-071 require the plan's benefits to be recorded at the moment
		// of the change, and `{}` records nothing. For most accounts that is the
		// only provision they ever get — this single-account path — so the empty
		// snapshot was the common case, not the edge one, and there was nothing to
		// reconcile an account back to what it was actually granted.
		CurrentBenefitSnapshot: BuildPlanBenefitSnapshot(defaultPlan),
		StartedAt:              now,
		UpdatedAt:              now,
		SubscriptionPlan:       defaultPlan,
	}
	if err := s.userSubRepo.ProvisionDefault(ctx, sub); err != nil {
		return nil, err
	}
	// Re-read on the SAME handle that wrote the row, so the version this
	// event is keyed on is the one the transaction actually committed.
	stored, err := s.userSubRepo.GetByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if stored == nil {
		return nil, subscriptionerrors.ErrUserSubscriptionNotFound()
	}

	// The event key is derived from the account and the provisioned version, so
	// a retried provisioning of the same account does not publish twice.
	activationEvent := &entities.UserSubscriptionEvent{
		EventKey:    fmt.Sprintf("DEFAULT_PROVISION:%s:%d", userID, stored.Version),
		UserID:      userID,
		EventType:   string(subscriptioneventtype.ACTIVATED),
		ToPlanCode:  &stored.CurrentPlanCode,
		ToTierRank:  &stored.CurrentTierRank,
		OccurredAt:  now,
		EffectiveAt: now,
	}
	// 024 T203: the plan handed to the payload builder must be the one the STORED
	// row points at, not the one this call tried to insert.
	//
	// `ProvisionDefault` is `ON CONFLICT (user_id) DO NOTHING` and the existence
	// check above is a plain read taken before the write, outside this body and
	// on a different snapshot on the lazy door (:72 vs :89). A row committed in
	// that window - the grant path creates one for an account that has none, see
	// admin_grant_batch_uc.go:401 - is therefore kept, `stored` is that row, and
	// the history row written just above correctly named ITS plan. Passing
	// `defaultPlan` here made `BuildPlanChangedPayload` fill `toPlanCode` and
	// `toPlanId` from the free plan instead, so the event said "moved to free" for
	// an account that is on premium, against FR-019's rule that the event
	// identity comes from the history row. The receiver writes
	// `ai_policy_grants.plan_id` from `toPlanId`, so the mismatch would have
	// granted the wrong policy rather than merely mislabelled the event.
	//
	// `GetByUserID` preloads `SubscriptionPlan`, so the re-read already carries
	// it; the fallback keeps the ordinary insert path working if a caller ever
	// reads without the relation.
	publishedPlan := stored.SubscriptionPlan
	if publishedPlan == nil {
		publishedPlan = defaultPlan
	}
	if _, err := s.planChangePublisher.PublishPlanChange(ctx, activationEvent, stored, publishedPlan); err != nil {
		return nil, err
	}
	return stored, nil
}

// GetOrCreateUserDailyQuota returns the account's daily counters, creating them on
// first use.
//
// 024 T148: this method is reached from several request handlers at once, and a
// brand-new account is exactly when every one of them is asked for something. The
// read-then-insert it used to do has a window between them, and since `user_id` is
// the primary key the second writer failed. Measured on the development database:
// 2 of 12 parallel requests returned HTTP 500 with
// `duplicate key value violates unique constraint "user_daily_quotas_pkey"`.
//
// The insert is what serialises the race, not a check that runs before it. The
// loser blocks on the winner's key, comes back with `inserted == false`, and
// re-reads - so both callers observe the SAME quota instead of one of them
// failing. If the winner later rolls back, the loser's insert succeeds instead,
// which is also correct: there is no row to lose.
func (s *SubscriptionStateSupport) GetOrCreateUserDailyQuota(ctx context.Context, userID uuid.UUID) (*entities.UserDailyQuota, error) {
	quota, err := s.quotaRepo.GetByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if quota != nil {
		return quota, nil
	}

	_, err = s.quotaRepo.CreateIfAbsent(ctx, &entities.UserDailyQuota{
		UserID:               userID,
		OutfitRecommendCount: 0,
		AiUsageCount:         0,
		LastResetDate:        time.Now(),
	})
	if err != nil {
		return nil, err
	}

	// Read back unconditionally rather than returning the in-memory struct. On the
	// insert path the two are equivalent; on the conflict path only the stored row is
	// the truth, because it may already carry counters another request moved.
	stored, err := s.quotaRepo.GetByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if stored == nil {
		// Unreachable: the conflict proves a row exists. Failing loudly beats handing
		// back a quota that was never persisted.
		return nil, subscriptionerrors.ErrUserDailyQuotaNotFound()
	}
	return stored, nil
}

func (s *SubscriptionStateSupport) LoadPlanForSubscription(ctx context.Context, sub *entities.UserSubscription) (*entities.SubscriptionPlan, error) {
	if sub.CurrentPlanKind == plankind.Finite && sub.ExpiresAt != nil && !sub.ExpiresAt.After(time.Now()) {
		if sub.FallbackPlanID != nil {
			plan, err := s.planRepo.GetByID(ctx, *sub.FallbackPlanID)
			if err != nil {
				return nil, err
			}
			if plan != nil {
				return plan, nil
			}
		}
		plan, err := s.planRepo.GetDefaultPlan(ctx)
		if err != nil {
			return nil, err
		}
		if plan == nil {
			return nil, subscriptionerrors.ErrDefaultPlanConfigNotFound()
		}
		return plan, nil
	}
	if sub.SubscriptionPlan != nil {
		return sub.SubscriptionPlan, nil
	}

	plan, err := s.planRepo.GetByID(ctx, sub.SubscriptionPlanID)
	if err != nil {
		return nil, err
	}
	if plan == nil {
		return nil, subscriptionerrors.ErrSubscriptionPlanNotFound()
	}
	return plan, nil
}

func (s *SubscriptionStateSupport) ResolveSubscriptionPlans(ctx context.Context, subs []*entities.UserSubscription) (map[uuid.UUID]*entities.SubscriptionPlan, error) {
	planByID := make(map[uuid.UUID]*entities.SubscriptionPlan, len(subs))
	missingPlanIDs := make([]uuid.UUID, 0)
	seenMissingPlanIDs := make(map[uuid.UUID]struct{})

	for _, sub := range subs {
		if sub == nil {
			continue
		}
		if sub.SubscriptionPlan != nil {
			planByID[sub.SubscriptionPlanID] = sub.SubscriptionPlan
		} else if _, exists := seenMissingPlanIDs[sub.SubscriptionPlanID]; !exists {
			seenMissingPlanIDs[sub.SubscriptionPlanID] = struct{}{}
			missingPlanIDs = append(missingPlanIDs, sub.SubscriptionPlanID)
		}
		if sub.FallbackPlanID != nil {
			if _, exists := seenMissingPlanIDs[*sub.FallbackPlanID]; !exists {
				seenMissingPlanIDs[*sub.FallbackPlanID] = struct{}{}
				missingPlanIDs = append(missingPlanIDs, *sub.FallbackPlanID)
			}
		}
	}

	if len(missingPlanIDs) == 0 {
		return planByID, nil
	}

	plans, err := s.planRepo.GetByIDs(ctx, missingPlanIDs)
	if err != nil {
		return nil, err
	}
	for _, plan := range plans {
		if plan == nil {
			continue
		}
		planByID[plan.ID] = plan
	}

	return planByID, nil
}
