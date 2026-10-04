package usecase

import (
	"context"

	sub_dto "smart-wardrobe-be/services/account/dto/subscription"

	"github.com/google/uuid"
)

// ICampaignRegistryUseCase is the WRITE side of the campaign mechanism (026 T031,
// FR-052..FR-055, FR-060..FR-063).
//
// WHY IT IS A SEPARATE INTERFACE FROM BOTH EXISTING ONES, and this is the third
// interface in this package for the same domain — which needs a reason rather than a
// habit, so here it is.
//
//   - `ICampaignUseCase` has exactly ONE method, `ProcessUserCreated`, because there
//     is exactly one trigger: the `identity.user.created` message. It holds the rules
//     1-7 and opens the grant transaction. Adding an operator-facing create to it
//     would put an HTTP-shaped request struct next to the one method whose signature
//     is dictated by a message contract, and the trigger that decides whether a
//     CUSTOMER receives a plan would then be edited by somebody working on an admin
//     page.
//   - `ICampaignQueryUseCase` is read-only, and FR-069 made that structural: no type
//     reachable from it holds a write method. Adding a create to it would destroy the
//     strongest available statement of that guarantee — the absence of a unit of work
//     and a repository write behind a read interface — and it would give the look-up
//     a reason to be changed by an operator action.
//
// So the three have three different shapes of input, three different transactions and
// three different blast radii, and three interfaces is the shape that keeps each of
// them honest.
//
// THE WRITES ARE AUDITED INSIDE THE TRANSACTION (FR-060), which is why every method
// takes the ACTOR and the request IP rather than reading them from a context: the
// outbox row has to name who did it, and a background context would name nobody.
type ICampaignRegistryUseCase interface {
	// Create opens a new campaign (FR-052, contract
	// `campaign-registry-api.md` §3).
	//
	// IT IS NOT IDEMPOTENT, BY DESIGN. A duplicate code is refused with 409 rather
	// than silently ignored: an operator who typed an existing code has a mistake to
	// fix, and answering "created" would leave them believing a campaign exists that
	// they are not looking at (spec.md Câu 8).
	//
	// `plan_slug` is NOT checked for existence here. FR-014 keeps 025's decision that
	// this is a RUN-time check, because loading and writing must not reach the plan
	// catalogue as a precondition — a paused plan must not be able to block an
	// operator from opening a campaign, and a wrong slug must not be able to crash a
	// service. Each grant against a missing plan is skipped with `plan_unavailable`.
	//
	// The response is the same `CampaignSummaryRes` the read endpoints return, so a
	// client that just created a campaign renders it from the create response and
	// then refreshes rather than reformatting one shape into another.
	Create(
		ctx context.Context,
		actorAdminID uuid.UUID,
		req *sub_dto.CampaignCreateReq,
		requestIP *string,
	) (*sub_dto.CampaignSummaryRes, error)
}

// THE REMAINING METHODS ARE `Update` AND `CLOSE`, AND THEY ARE NOT HERE YET.
//
// This interface is introduced by Phase 3 (US1) and completed by Phase 4 (US2,
// `Update`/`Close` are T041/T052). Declaring methods no code implements would mean the
// concrete use case could not satisfy its own interface, so the constructor could not
// return it and Wire could not wire it — a build failure invented by an interface
// that was honest about its own future.
//
// Each of them is added HERE, not to the implementation, when it is implemented: the
// point of an interface in this codebase is that the presentation layer depends on
// the SET OF OPERATIONS IT MAY PERFORM, so an operator interface that listed a close
// nobody could invoke would be a promise the router does not keep.
