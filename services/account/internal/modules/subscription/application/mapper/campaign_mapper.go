package mapper

import (
	"time"

	identitydto "smart-wardrobe-be/services/account/dto/identity"
	sub_dto "smart-wardrobe-be/services/account/dto/subscription"
	"smart-wardrobe-be/services/account/internal/modules/subscription/domain/entities"
	"smart-wardrobe-be/shared/pkg/utils/sliceutils"

	"github.com/google/uuid"
)

// 025 T078: entity → DTO for the operator look-up (FR-065, FR-066, FR-068).
//
// WHY THESE LIVE HERE AND NOT IN THE USE CASE. The constitution requires mapper
// functions converting entities to DTOs to live in `application/mapper`, never in
// a use case file. That is not a filing preference: the use case is where the
// RULES live, and a mapping buried between two rules is a mapping that gets
// edited while somebody is changing a rule.
//
// WHY THE JOIN LIVES HERE AND NOT IN THE USE CASE. `CampaignClaimRes.Username`
// needs account data the campaign module does not own, so it comes through
// `IUserBriefContract.GetByIDs` (constitution §II). Joining a page of twenty
// claims against twenty accounts with a nested loop is O(n²) on the one endpoint
// an operator hammers while answering a ticket, and the constitution names
// `sliceutils.IndexBy` as the required form for exactly this. Putting the index
// here means there is ONE place that performs the join, so a second use case
// cannot quietly build its own map with a different key.
//
// THE CAMPAIGN SUMMARY AND THE ACCOUNT VERDICT ARE NOT IN THIS FILE, and that is
// deliberate rather than an omission. Neither is an entity: one is derived from
// configuration, the ledger, the clock and the sweep table together, and the other
// is a decision. Neither has a source row to map FROM. They are built where the
// inputs live — `campaign_query_uc.go` — and this file stays the one place that
// converts persisted rows into response shapes.

// CampaignGrantCounts carries the two ledger counts a campaign summary needs
// (026 T032).
//
// A STRUCT rather than two positional `int64`s, because the two numbers are the ones
// FR-043's rule reasons about as a pair — "the main part is full and the reserve is
// not" is the difference between `compensation` and `running` — and two bare arguments
// at a call site are indistinguishable to a reader who has forgotten which is which.
//
// They are two SEPARATE counts and never a total: deriving the reserve as
// `total - main` is the mistake research.md R2b exists to forbid, and a shape that
// makes people reach for a total is a shape that invites it.
type CampaignGrantCounts struct {
	// Main is `COUNT(*)` of claims in the `main` bucket: the grants that drew on
	// `quota`.
	Main int64
	// Compensation is `COUNT(*)` of claims in the `reserve` bucket: the grants that
	// drew on the ceiling above `quota`.
	Compensation int64
}

// MapCampaignSummaryRes maps one registry row plus its ledger counts into the
// operator's summary shape (026 T032, FR-051).
//
// IT IS THE ONLY PLACE A `CampaignSummaryRes` IS BUILT, and that is the reason it
// takes the counts instead of reading them. Both callers — the read endpoints and the
// write endpoints that return the campaign they just changed — must produce the SAME
// shape, and two builders for one response is how a `remainingMain` starts disagreeing
// with a `hardCap` on two screens an operator compares side by side.
//
// WHAT COMES FROM WHERE, and the split is deliberate:
//
//   - the row supplies identity and configuration: `code`, `planSlug`, `quota`,
//     `reserve`, `startsAt`, `endsAt`;
//   - the counts supply consumption, and everything DERIVED from it — the total, the
//     two remainders and the hard ceiling. Deriving them here rather than at each call
//     site is what makes `grantedTotalCount == grantedMainCount +
//     grantedCompensationCount` and `hardCap == quota + reserve` structural instead of
//     repeated;
//   - `watermarkAt` is the claim that FILLED the main budget, which no arithmetic on
//     the counts can recover, so it is passed in as the instant the ledger reported
//     (nil until the main budget fills);
//   - `status` is passed in because deriving it is a RULE, and rules live beside the
//     other rules — see `deriveCampaignStatus` in `campaign_query_uc.go`, whose header
//     explains why it must not move here.
//
// NOTHING ELSE IS SET. `PlanName`, `Degraded`, `DegradedReason` and `LastSweepAt` are
// filled by the caller because each needs a table this mapper must not know about —
// the plan catalogue, the sweep health table — and `budgetLocked`/`version` arrive
// with the next read-API phase (T050). A mapper that went looking for them would
// turn one pure conversion into a fan-out of four queries.
func MapCampaignSummaryRes(
	campaign *entities.Campaign,
	counts CampaignGrantCounts,
	watermarkAt *time.Time,
	status string,
) *sub_dto.CampaignSummaryRes {
	if campaign == nil {
		return nil
	}
	quota := int64(campaign.Quota)
	reserve := int64(campaign.Reserve)
	grantedTotal := counts.Main + counts.Compensation

	return &sub_dto.CampaignSummaryRes{
		Code:                     campaign.Code,
		PlanSlug:                 campaign.PlanSlug,
		Quota:                    campaign.Quota,
		Reserve:                  campaign.Reserve,
		GrantedMainCount:         counts.Main,
		GrantedCompensationCount: counts.Compensation,
		GrantedTotalCount:        grantedTotal,
		// `max(0, …)` rather than the bare subtraction. A campaign whose budget was
		// LOWERED below what it already granted would otherwise answer with a negative
		// number of slots left, and a page an operator reads at a glance would show
		// "-3 suất còn lại" on a field no client is expected to handle being negative.
		RemainingMain:         remainingOf(quota, counts.Main),
		RemainingCompensation: remainingOf(reserve, counts.Compensation),
		HardCap:               quota + reserve,
		StartsAt:              campaign.StartsAt,
		EndsAt:                campaign.EndsAt,
		// FR-051's `closedAt`, copied straight from the row. T039 added the field to
		// the response and this is the one place a registry row becomes a response,
		// so copying it here is what keeps the read endpoints and the write endpoints
		// showing the same closure instant — and nil here genuinely means "never
		// force-closed", because `closed_at` is nullable for exactly that reason.
		ClosedAt:    campaign.ClosedAt,
		WatermarkAt: watermarkAt,
		Status:      status,
	}
}

// remainingOf is `max(0, budget - granted)` for the two derived remainders.
func remainingOf(budget, granted int64) int64 {
	if granted > budget {
		return 0
	}
	return budget - granted
}

// MapCampaignClaimRes maps one grant row to its response shape
// (contracts/campaign-query-api.md §3.3).
//
// `account` is the brief the join produced, or nil when the account is no longer
// readable. A nil account yields an EMPTY USERNAME rather than an error: the claim
// is still the truth about what was granted, and refusing to show it because the
// account was soft-deleted would hide the very grant the operator is asking about.
func MapCampaignClaimRes(claim *entities.CampaignClaim, account *identitydto.UserBriefContractDTO) *sub_dto.CampaignClaimRes {
	if claim == nil {
		return nil
	}
	res := &sub_dto.CampaignClaimRes{
		UserID:       claim.UserID,
		PlanSlug:     claim.PlanSlug,
		ExpiresAt:    claim.ExpiresAt,
		GrantedAt:    claim.GrantedAt,
		RegisteredAt: claim.RegisteredAt,
		Source:       string(claim.GrantBucket),
	}
	if account != nil {
		res.Username = account.Username
	}
	return res
}

// MapCampaignClaimResList maps a page of grants, joining usernames in one pass.
//
// `accounts` is the slice `GetByIDs` returned, unfiltered. It is indexed here
// rather than in the use case so that the O(1) requirement is enforced by the ONE
// function that performs the join.
//
// The map is built with `sliceutils.IndexBy` (constitution §II), keyed on the
// account ID. A nil or absent entry is not an error: see `MapCampaignClaimRes`.
func MapCampaignClaimResList(
	claims []*entities.CampaignClaim,
	accounts []*identitydto.UserBriefContractDTO,
) []*sub_dto.CampaignClaimRes {
	byID := sliceutils.IndexBy(accounts, func(account *identitydto.UserBriefContractDTO) uuid.UUID {
		return account.ID
	})
	result := make([]*sub_dto.CampaignClaimRes, 0, len(claims))
	for _, claim := range claims {
		if claim == nil {
			continue
		}
		result = append(result, MapCampaignClaimRes(claim, byID[claim.UserID]))
	}
	return result
}
