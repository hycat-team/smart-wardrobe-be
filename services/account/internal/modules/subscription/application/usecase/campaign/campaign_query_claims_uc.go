package campaign

import (
	"context"

	identitydto "smart-wardrobe-be/services/account/dto/identity"
	sub_dto "smart-wardrobe-be/services/account/dto/subscription"
	subscriptionerrors "smart-wardrobe-be/services/account/internal/modules/subscription/application/errors"
	uc_interfaces "smart-wardrobe-be/services/account/internal/modules/subscription/application/interface/usecase"
	"smart-wardrobe-be/services/account/internal/modules/subscription/application/mapper"
	"smart-wardrobe-be/services/account/internal/modules/subscription/domain/entities"
	"smart-wardrobe-be/services/account/internal/modules/subscription/domain/repositories"

	"github.com/google/uuid"
)

// The two per-account reads of the operator look-up (025 T080, FR-066, FR-067,
// FR-068, FR-033). They are in their own file because they are the only part of
// the look-up that reasons about ONE account, and keeping them apart from the
// campaign-level reads means a reader looking for "what does the state machine
// do" does not have to walk past the eligibility rules to find it.

// ListClaims returns one page of a campaign's granted accounts (FR-066, FR-067).
//
// READ-ONLY, and provably so: this method reads the ledger, reads the account
// contract, and maps rows. It holds no unit of work, so there is no transaction
// here to write through, and it calls nothing that grants.
func (uc *CampaignQueryUseCase) ListClaims(
	ctx context.Context,
	campaignCode string,
	filter repositories.CampaignClaimsFilter,
	page, limit int,
) ([]*sub_dto.CampaignClaimRes, int64, error) {
	// FR-072: an unknown code is an error naming the code, NOT an empty page. An
	// empty page reads as "this campaign granted nobody", which is a different
	// fact and the one that would send an operator looking for a budget fault
	// that does not exist.
	if _, ok := uc.findCampaign(campaignCode); !ok {
		return nil, 0, subscriptionerrors.ErrCampaignNotFound(campaignCode)
	}
	if err := validateClaimsWindow(filter); err != nil {
		return nil, 0, err
	}

	claims, total, err := uc.claimRepo.ListPageByCampaign(ctx, campaignCode, filter, page, limit)
	if err != nil {
		return nil, 0, subscriptionerrors.ErrCampaignListClaimsFailed(campaignCode)
	}

	// The account join happens ONCE for the page, through the identity contract
	// (constitution §II). The campaign module does not own `identity.users`, so
	// reading it directly is not available and would be wrong if it were.
	//
	// `sliceutils.IndexBy` does the matching inside the mapper, so a page of twenty
	// claims against twenty accounts is twenty lookups rather than four hundred.
	accounts, err := uc.accountsFor(ctx, claims)
	if err != nil {
		return nil, 0, err
	}
	return mapper.MapCampaignClaimResList(claims, accounts), total, nil
}

// GetAccountEligibility answers what happened to one account under one campaign
// (FR-068, FR-033).
//
// THE REQUIREMENT IS DIFFERENTIABILITY, NOT ACCURACY. Accounts in different
// states must get different answers, because those answers are different things to
// tell a customer:
//
//	granted                  → "you were given the plan"
//	eligible_pending         → "you qualified; the grant is still on its way"
//	eligible_but_exhausted   → "you qualified, but the giveaways ran out"
//	not_eligible             → "the campaign does not apply to you"
//
// An implementation that returned the same answer for two of them would satisfy
// every field-level assertion and still fail FR-068, which is why the test for it
// compares the RESPONSES WITH EACH OTHER rather than checking each against a
// constant.
//
// `eligible_pending` is deliberately NOT folded into `eligible_but_exhausted`:
// "we ran out" is the wrong thing to tell an account whose grant is minutes away
// (FR-079), and the contract gives it its own value for exactly that reason.
//
// THE VERDICT IS RECONSTRUCTED FROM THE SAME RULES THE GRANT PATH USES. Conditions
// 1 to 7 come from `campaignEvaluator` — the identical function `ProcessUserCreated`
// calls — and condition 8's budget verdict comes from `quotaDecider.chooseBucket`,
// which is a pure decision and writes nothing. FR-080's whole protection is that
// there is ONE implementation of "is this account eligible"; a look-up that
// re-derived it would be the second, and the drift between them would be invisible
// because both answers would look individually reasonable.
func (uc *CampaignQueryUseCase) GetAccountEligibility(
	ctx context.Context,
	campaignCode string,
	userID uuid.UUID,
) (*sub_dto.CampaignAccountEligibilityRes, error) {
	campaign, ok := uc.findCampaign(campaignCode)
	if !ok {
		return nil, subscriptionerrors.ErrCampaignNotFound(campaignCode)
	}

	account, err := uc.accounts.GetByID(ctx, userID)
	if err != nil {
		// Never downgraded to "not eligible": a read failure reported as a
		// business verdict would be written into a support ticket as fact, and
		// the customer would be told they were ineligible on the strength of a
		// database timeout.
		return nil, subscriptionerrors.ErrCampaignGetAccountFailed()
	}
	if account == nil {
		return nil, subscriptionerrors.ErrCampaignAccountNotFound()
	}

	now := uc.now()

	// The account's own grant record, WHATEVER campaign wrote it.
	//
	// "Whatever campaign" rather than "this campaign" is what makes
	// `already_claimed_other_campaign` answerable at all: FR-012 gives an account
	// one participation across the whole mechanism, so an account campaign A
	// granted is ineligible for campaign B, and a lookup scoped to B would report
	// "no claim" and let support tell a customer they were eligible.
	claim, claimed, err := uc.claimRepo.GetByUser(ctx, userID)
	if err != nil {
		return nil, subscriptionerrors.ErrCampaignGetClaimForAccountFailed()
	}

	// The campaign's watermark and budget, needed both for the answer and for the
	// response envelope: an operator asking "why did they miss out?" needs the
	// boundary the account was measured against.
	watermarkAt, watermarkSet, err := uc.claimRepo.GetWatermark(ctx, campaign.Code)
	if err != nil {
		return nil, subscriptionerrors.ErrCampaignReadWatermarkForLookupFailed(campaign.Code)
	}

	result := &sub_dto.CampaignAccountEligibilityRes{
		CampaignCode: campaign.Code,
		UserID:       userID,
	}
	if watermarkSet {
		result.WatermarkAt = &watermarkAt
	}
	registeredAt := account.CreatedAt
	result.UserRegisteredAt = &registeredAt

	// A claim FROM THIS CAMPAIGN is the one unambiguous "yes", and it is checked
	// first because it is the only answer that needs nothing else. It is also the
	// only one whose `reason` is empty: there is nothing to diagnose about a grant
	// that happened.
	if claimed && claim.CampaignCode == campaign.Code {
		result.Eligibility = sub_dto.CampaignEligibilityGranted
		result.Granted = true
		return result, nil
	}

	// A claim from ANOTHER campaign ends the evaluation here (FR-012).
	//
	// FR-012 gives an account one participation across the WHOLE mechanism, so an
	// account campaign A already granted is ineligible for campaign B whatever
	// campaign B's window says. It is its own eligibility value rather than a
	// flavour of `not_eligible` for the reason `created_by_admin` is: the account
	// is a perfectly good customer who arrived too late, and an operator reading a
	// wall of `not_eligible` would go looking for a configuration error. There is
	// none — there is simply an earlier campaign.
	if claimed {
		result.Eligibility = sub_dto.CampaignEligibilityAlreadyClaimedOtherCampaign
		result.Reason = string(SkipAlreadyClaimed)
		return result, nil
	}

	// Condition 3 (FR-091), read from the account's own record.
	//
	// Checked before the campaign is asleep or the account is out of window
	// because an administrator's account is a perfectly good customer the campaign
	// simply does not apply to, and an operator reading a wall of `not_eligible`
	// would go looking for a configuration error. There is none.
	if reason := adminSkipReason(account.CreatedByAdmin); reason != "" {
		result.Eligibility = sub_dto.CampaignEligibilityCreatedByAdmin
		result.Reason = string(reason)
		return result, nil
	}

	// Conditions 1 to 7, through the SAME evaluator the grant path calls (FR-080).
	sub, err := uc.subRepo.GetByUserID(ctx, userID)
	if err != nil {
		return nil, subscriptionerrors.ErrCampaignGetAccountFailed()
	}
	verdict := newCampaignEvaluator(uc.cfg.Campaigns, now).
		evaluate(ctx, userID, account.CreatedAt, account.CreatedByAdmin, sub)
	if verdict.Campaign == nil {
		result.Eligibility = sub_dto.CampaignEligibilityNotEligible
		result.Reason = string(verdict.Reason)
		return result, nil
	}
	// The evaluator picked the LATEST-starting campaign the account qualifies for,
	// which is not necessarily the one that was asked about. If it is a different
	// one, then this campaign does not apply to this account — a different answer
	// from "no campaign applies at all", and the reason says which.
	if verdict.Campaign.Code != campaign.Code {
		result.Eligibility = sub_dto.CampaignEligibilityNotEligible
		result.Reason = string(SkipOutsideWindow)
		return result, nil
	}

	// FR-010: the configured plan must exist and be active. Loading configuration
	// cannot check this, so it is checked here — and its answer is a per-account
	// verdict, because `plan_unavailable` is how an operator discovers that a
	// campaign's `plan_slug` is wrong. A campaign in that state grants nobody
	// anything, and every account looks identical until this says why.
	plan, err := uc.planRepo.GetBySlug(ctx, campaign.PlanSlug)
	if err != nil {
		return nil, subscriptionerrors.ErrCampaignLoadPlanFailed(campaign.PlanSlug)
	}
	if plan == nil {
		result.Eligibility = sub_dto.CampaignEligibilityNotEligible
		result.Reason = string(SkipPlanUnavailable)
		return result, nil
	}

	// Condition 8, as a DECISION and not as a write.
	//
	// `chooseBucket` is the pure half of the quota decider: it reads the counts it
	// is given and returns a verdict, and the half that writes is a different
	// method. Reusing it means the look-up cannot disagree with the grant path
	// about which accounts the reserve belongs to (FR-031), which is the specific
	// wrong answer that would cost a real customer a real plan.
	grantedMain, err := uc.claimRepo.CountByBucket(ctx, campaign.Code, entities.GrantBucketMain)
	if err != nil {
		return nil, subscriptionerrors.ErrCampaignCountGrantsFailed(campaign.Code, string(entities.GrantBucketMain))
	}
	grantedReserve, err := uc.claimRepo.CountByBucket(ctx, campaign.Code, entities.GrantBucketReserve)
	if err != nil {
		return nil, subscriptionerrors.ErrCampaignCountGrantsFailed(campaign.Code, string(entities.GrantBucketReserve))
	}
	outcome := newQuotaDecider(nil).chooseBucket(
		claimInput{
			Campaign:     *campaign,
			Plan:         plan,
			UserID:       userID,
			RegisteredAt: account.CreatedAt,
			Now:          now,
		},
		grantedMain, grantedReserve, watermarkAt, watermarkSet,
	)

	// The budget refused the account. This is the middle state FR-033 and FR-068
	// exist for, and it is distinct from `not_eligible` because the account met
	// every condition and lost on supply alone: FR-033 requires that difference to
	// be visible, and an operator who cannot see it will tell a qualifying
	// customer they were not eligible.
	//
	// `quota_anomaly` maps here too. The main budget is full and no watermark
	// exists, which should be impossible; the account is eligible and the ledger is
	// wrong, and calling that `not_eligible` would blame the customer for a data
	// fault.
	//
	// The two branches below are the two different ways an eligible account ends
	// up without a grant, and they are NOT the same answer: one lost on supply,
	// the other has not been processed yet.
	switch {
	case outcome.Granted:
		// ELIGIBLE PENDING. The account meets every condition AND the budget still
		// has room, so the grant has not happened yet — the message is in flight, or
		// the catch-up sweep has not reached it (FR-079).
		//
		// This used to answer `eligible_but_exhausted` with an empty reason, which
		// is wrong in the way that matters: an operator reading `exhausted` tells a
		// customer "we have run out of giveaways", and that is false in both senses
		// — the campaign still has quota, and this customer's grant is on its way.
		// It is the most common question on the phone ("I just signed up, where is
		// my plan?"), so it gets the value that says the true thing.
		//
		// The reason stays EMPTY, and the emptiness is meaningful: every other value
		// that reaches this switch names a supply shortage, and here there is none to
		// name. An invented code would send the operator to look at a budget that is
		// not the problem.
		result.Eligibility = sub_dto.CampaignEligibilityEligiblePending
		result.Reason = ""
	case outcome.Reason == SkipQuotaMain || outcome.Reason == SkipQuotaCompensation || outcome.Reason == SkipQuotaAnomaly:
		result.Eligibility = sub_dto.CampaignEligibilityEligibleButExhausted
		result.Reason = string(outcome.Reason)
	default:
		result.Eligibility = sub_dto.CampaignEligibilityNotEligible
		result.Reason = string(outcome.Reason)
	}
	return result, nil
}

// validateClaimsWindow refuses a reversed `from`/`to` pair (FR-067).
//
// It is a bad request from the CALLER, and answering it with an empty list would
// be the worse of the two failures: the caller believes the range is empty, nobody
// looks at the range, and the gap is discovered later as a missing grant that was
// reported to a customer as never having happened.
func validateClaimsWindow(filter repositories.CampaignClaimsFilter) error {
	if filter.From == nil || filter.To == nil {
		return nil
	}
	if filter.From.After(*filter.To) {
		return subscriptionerrors.ErrCampaignClaimWindowInvalid()
	}
	return nil
}

// accountsFor fetches the brief for every account in a page of claims.
//
// ONE call for the whole page rather than one per claim: this is the endpoint an
// operator hammers while working down a ticket, and twenty accounts in a page
// would otherwise be twenty round trips to a contract that can answer all twenty
// at once.
func (uc *CampaignQueryUseCase) accountsFor(
	ctx context.Context,
	claims []*entities.CampaignClaim,
) ([]*identitydto.UserBriefContractDTO, error) {
	if len(claims) == 0 {
		return nil, nil
	}
	userIDs := make([]uuid.UUID, 0, len(claims))
	for _, claim := range claims {
		if claim != nil {
			userIDs = append(userIDs, claim.UserID)
		}
	}
	accounts, err := uc.accounts.GetByIDs(ctx, userIDs)
	if err != nil {
		return nil, subscriptionerrors.ErrCampaignGetAccountFailed()
	}
	return accounts, nil
}

// unusedInterface keeps the interface import honest: the constructor in
// `campaign_query_uc.go` returns this interface, and this file implements the
// remaining methods on the same type.
var _ uc_interfaces.ICampaignQueryUseCase = (*CampaignQueryUseCase)(nil)
