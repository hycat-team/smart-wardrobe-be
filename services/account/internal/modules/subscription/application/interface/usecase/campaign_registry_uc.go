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

	// Close force-closes a campaign and is ONE-WAY (FR-030/FR-031/FR-034, contract
	// `campaign-registry-api.md` §5).
	//
	// IT IS THE ONLY OPERATION HERE THAT TAKES THE CODE AS AN ARGUMENT rather than
	// reading it out of the body, and that asymmetry is the contract's: `Create`
	// names the campaign it creates, while every operation on an existing one names
	// it in the path. It is also why `code` is not a field of `CampaignCloseReq` —
	// a body that could contradict the URL is a body whose authority is ambiguous.
	//
	// IDEMPOTENT, AND THAT IS A CONTRACT RATHER THAN A CONVENIENCE (FR-034). A
	// retried request, a double-click, and two administrators closing the same
	// campaign at the same moment all arrive, and all three must succeed with 200
	// and NO second audit row: a closure recorded twice would make "when was this
	// stopped, and why" ambiguous, and FR-060 permits exactly one row per change.
	//
	// The idempotent answer is produced BEFORE the version is compared, so a client
	// that retries with the version it held before the first attempt still succeeds.
	// A code that does not exist is 404 and a version that moved is 412, in that
	// order — see `Close`'s own header, which is where the four answers are ordered
	// for a reason.
	Close(
		ctx context.Context,
		actorAdminID uuid.UUID,
		campaignCode string,
		req *sub_dto.CampaignCloseReq,
		requestIP *string,
	) (*sub_dto.CampaignSummaryRes, error)
}

// `Update` IS NOT HERE YET, AND THAT IS ITS OWN STATEMENT.
//
// This interface was introduced by Phase 3 (US1) with `Create` alone and completed
// by Phase 4 (US2, `Close`). `Update` arrives with US3 (T052), and it will be
// declared HERE, in this interface, when it is implemented.
//
// Declaring a method no code implements would mean the concrete use case could not
// satisfy its own interface, so the constructor could not return it and Wire could
// not wire it — a build failure invented by an interface that was honest about its
// own future. The point of an interface in this codebase is that the presentation
// layer depends on the SET OF OPERATIONS IT MAY PERFORM, so an operator interface
// that listed an edit nobody could invoke would be a promise the router does not
// keep.
