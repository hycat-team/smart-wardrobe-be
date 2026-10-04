package usecase

import (
	"context"

	sub_dto "smart-wardrobe-be/services/account/dto/subscription"
	"smart-wardrobe-be/services/account/internal/modules/subscription/domain/repositories"

	"github.com/google/uuid"
)

// ICampaignQueryUseCase is the READ side of the campaign mechanism (025 T079,
// FR-065 to FR-072).
//
// IT IS A SEPARATE INTERFACE FROM `ICampaignUseCase`, and that separation is the
// whole design rather than a stylistic choice. `ICampaignUseCase` has exactly one
// method — `ProcessUserCreated` — because there is exactly one TRIGGER: the
// `identity.user.created` message. Adding three read methods to it would say the
// grant path has three more entry points when it has none, and it would put an
// HTTP-shaped, pagination-shaped, filtering-shaped surface next to the one method
// whose signature is dictated by a message contract. A change to either would then
// touch the other, and the trigger that decides whether a customer receives a plan
// would be edited by somebody working on a reporting page.
//
// It also makes the read-only guarantee structural rather than a matter of
// restraint. FR-069 forbids this feature from changing any data, and the strongest
// statement of that is that no type reachable from this interface holds a write
// method: no unit of work, no claim repository's `Create`, no notification
// contract. There is nothing here to reach for, so there is nothing to reach for
// by mistake.
type ICampaignQueryUseCase interface {
	// ListCampaigns returns every configured campaign with its DERIVED state
	// (FR-065).
	//
	// Read-only in the strong sense: every count is computed from the claim
	// ledger at the moment of the call, and none of it is stored. A stored count
	// would need synchronising by hand and would be wrong the moment a grant
	// committed without touching it — and a wrong budget number is worse than no
	// number, because an operator acts on it.
	ListCampaigns(ctx context.Context, page, limit int) ([]*sub_dto.CampaignSummaryRes, int64, error)

	// GetCampaign returns one campaign by code, or a not-found error naming the
	// code that was asked for (FR-072).
	//
	// A not-found error rather than an empty list: an operator who typed the code
	// wrong must be told so, and an empty list reads as "this campaign granted
	// nothing", which is a completely different fact and the one that would send
	// them looking for a budget problem that does not exist.
	GetCampaign(ctx context.Context, campaignCode string) (*sub_dto.CampaignSummaryRes, error)

	// ListClaims returns one page of a campaign's granted accounts, filtered by
	// the grant instant (FR-066, FR-067).
	//
	// `filter` narrows on `granted_at`, half-open, matching the sweep's own
	// window query so an operator exporting a time range gets each grant in
	// exactly one export. A `from` after `to` is REFUSED rather than answered
	// with an empty page: a caller that writes that has a bug, and an empty list
	// would hide it until somebody noticed a missing grant.
	ListClaims(ctx context.Context, campaignCode string, filter repositories.CampaignClaimsFilter, page, limit int) ([]*sub_dto.CampaignClaimRes, int64, error)

	// GetAccountEligibility answers what happened to ONE account under ONE
	// campaign (FR-068, FR-033, FR-079).
	//
	// This is the method the whole phase exists for. Its requirement is not "be
	// correct" but "be DIFFERENTIABLE": granted, eligible-pending,
	// eligible-but-exhausted and not-eligible must be distinguishable answers,
	// because they are different things to tell a customer. A response that
	// returned the same answer for two of them would satisfy every field-level
	// check and fail the requirement, which is why the test asserts on the
	// responses differing from each other rather than on any single value.
	//
	// `eligible_pending` (eligible, quota left, grant not yet made) is separate
	// from `eligible_but_exhausted` (eligible, no quota left) because conflating
	// them tells a customer the giveaways ran out when the campaign still has
	// room and their grant is on its way.
	GetAccountEligibility(ctx context.Context, campaignCode string, userID uuid.UUID) (*sub_dto.CampaignAccountEligibilityRes, error)

	// ListCampaignAudit returns one page of ONE campaign's audit trail — who changed it,
	// when, and what moved (026 T067, FR-063, contract §6).
	//
	// IT LIVES ON THIS INTERFACE rather than beside `CampaignRegistryUseCase` for two
	// reasons. It is a READ, so putting it on the writer would mean the write path could
	// answer it — and FR-069's guarantee is that the look-up side holds no unit of work
	// and no writer. And the CAMPAIGN'S EXISTENCE has to be answered from the registry,
	// which is what this use case already reads; a new type for one method would mean a
	// second reader of `subscription.campaigns` for no gain.
	//
	// THE 404 IS DECIDED HERE, BEFORE THE TRAIL IS ASKED FOR, and that ordering is the
	// contract rather than an implementation detail. The audit trail lives in the `admin`
	// module, which does not own `subscription.campaigns` and must not read it
	// (constitution §II), so an unknown code answers it with an EMPTY PAGE — and an
	// operator who mistyped a code would read "this campaign has never been changed" as a
	// fact about the audit mechanism rather than as a fact about their typing (FR-072
	// makes the same argument for the other three endpoints).
	//
	// Read-only in the strong sense, like everything else here: it neither spends a slot,
	// nor grants a plan, nor sends an email, and the trail it reads was written
	// asynchronously by a consumer some time after each operation succeeded (FR-060).
	ListCampaignAudit(
		ctx context.Context,
		campaignCode string,
		page, limit int,
	) ([]*sub_dto.CampaignAuditLogRes, int64, error)
}

// campaignPaginationMaxLimit is the shared ceiling for the two paginated reads.
//
// It is the same 100 the shared `PaginationQuery.Normalize` clamps to, restated
// here so the use case's own guard is not a coincidence of the HTTP layer: the use
// case is also reachable from a future non-HTTP caller, and a bound that exists
// only in the handler is a bound that a caller can walk past.
const campaignPaginationMaxLimit = 100

// NormalizeCampaignPagination keeps a caller-supplied page and limit usable.
//
// It is EXPORTED because the handler, the use case and any future non-HTTP caller
// all have to agree on the ceiling, and three private copies of the same constant
// is three chances for one of them to differ.
//
// A limit above the ceiling is CLAMPED rather than refused. The refusal reading
// has teeth: refusing would make an operator who asked for 200 rows get an error
// instead of two pages of results, and the correct answer to "give me more than
// the maximum" is the maximum plus a total that tells them there is more.
func NormalizeCampaignPagination(page, limit int) (int, int) {
	if page <= 0 {
		page = 1
	}
	if limit <= 0 {
		limit = 20
	}
	if limit > campaignPaginationMaxLimit {
		limit = campaignPaginationMaxLimit
	}
	return page, limit
}
