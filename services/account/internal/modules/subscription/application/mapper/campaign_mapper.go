package mapper

import (
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
